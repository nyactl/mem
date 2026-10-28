package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"mem/internal/attachment"
	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var newTitle string
var newLabels []string
var newSources []string
var newFiles []string
var newDate string
var newSupersedes []string

var newCmd = &cobra.Command{
	Use:   "new <content>  |  new <title> <content>",
	Short: "Capture a note from the command line",
	Long: `Creates a note instantly without opening an editor.

  mem new "ran into an interesting paper on indexing #research"
  mem new "kafka rebalance" "consumer group blocks all partitions for ~2min"
  mem new --title "C++" "templates are Turing-complete"
  mem new "meeting went sideways" -t work -t ops -d 2024-03-15
  mem new "kafka rebalance" "now about 30s with cooperative rebalancing" --supersedes kafka-rebalance

One argument:  slug is derived from the first few words of the content.
Two arguments, or --title: the title is kept as typed in the note and sets the slug,
the file name keeps only its letters and digits.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		title, body := newTitle, args[0]
		if len(args) == 2 {
			if title != "" {
				return fmt.Errorf("give the title either as --title or as the first argument")
			}
			title, body = args[0], args[1]
		}
		n, err := addNote(config.Load(), title, body, newLabels, newSources, newFiles, newDate, newSupersedes)
		if err != nil {
			return err
		}
		fmt.Printf("%s  %s\n", n.Created.Format("15:04:05"), n.DisplayTitle())
		return nil
	},
}

// addNote writes a new note. Without a title its slug comes from the first words of the body.
func addNote(cfg config.Config, title, body string, tags, sources, files []string, date string, supersedes []string) (note.Note, error) {
	ts := time.Now()
	title = note.CleanTitle(title)
	body = strings.TrimSpace(body)

	slug := note.Slugify(title)
	if title == "" {
		slug = note.Slugify(firstWords(stripInlineMarkers(body), 4))
	}
	if slug == "" {
		slug = "note"
	}
	if body == "" {
		return note.Note{}, fmt.Errorf("content cannot be empty")
	}

	var attachmentPaths []string
	for _, f := range files {
		dst, err := attachment.Store(f, ts, cfg.AttachmentsDir)
		if err != nil {
			return note.Note{}, fmt.Errorf("attach %s: %w", f, err)
		}
		attachmentPaths = append(attachmentPaths, dst)
		fmt.Fprintf(os.Stderr, "attached → %s\n", dst)
	}

	sluggedSources := make([]string, 0, len(sources))
	for _, s := range sources {
		if sl := note.Slugify(s); sl != "" {
			sluggedSources = append(sluggedSources, sl)
		}
	}

	for _, old := range supersedes {
		n, err := note.FindByIDOrSlug(cfg.NotesDir, old)
		if err != nil {
			return note.Note{}, fmt.Errorf("--supersedes: %w", err)
		}
		sluggedSources = append(sluggedSources, note.SupersedesPrefix+strings.ToLower(n.ID()))
	}

	allTags := note.MergeTags(tags, note.ExtractInlineTags(body))
	allSources := note.MergeSources(sluggedSources, note.ExtractInlineSources(body))

	var displayDate *time.Time
	if date != "" {
		parsed, ok := note.ParseDateTime(date)
		if !ok {
			return note.Note{}, fmt.Errorf("--date %q: unrecognised datetime format", date)
		}
		displayDate = parsed
	}

	path := uniquePath(cfg.NotesDir, ts, slug)
	content := note.BuildFrontmatter(title, allTags, allSources, attachmentPaths, displayDate) + body + "\n"
	if err := note.WriteRaw(path, content); err != nil {
		return note.Note{}, err
	}
	_, _ = index.Rebuild(cfg.NotesDir)
	return note.Parse(path)
}

// uniquePath is the path for a new note, its slug numbered when another note took it that second.
func uniquePath(dir string, ts time.Time, slug string) string {
	slug2 := slug
	for i := 2; note.Exists(note.FilePath(dir, ts, slug2)); i++ {
		slug2 = fmt.Sprintf("%s-%d", slug, i)
	}
	return note.FilePath(dir, ts, slug2)
}

var reInlineMarkers = regexp.MustCompile(`[#@][a-zA-Z][a-zA-Z0-9-]*`)

func stripInlineMarkers(s string) string {
	return strings.TrimSpace(reInlineMarkers.ReplaceAllString(s, ""))
}

func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}

func init() {
	newCmd.Flags().StringVar(&newTitle, "title", "", "title, kept as typed; the slug is made from it")
	newCmd.Flags().StringArrayVarP(&newLabels, "tag", "t", nil, "tag, repeatable")
	newCmd.Flags().StringArrayVarP(&newSources, "source", "s", nil, "source, repeatable")
	newCmd.Flags().StringArrayVarP(&newFiles, "file", "f", nil, "attach file, repeatable")
	newCmd.Flags().StringVarP(&newDate, "date", "d", "", "override display datetime, e.g. 2024-03-15T09:00 or 2024-03-15")
	newCmd.Flags().StringArrayVar(&newSupersedes, "supersedes", nil, "ID or slug (the newest note with it) of an older note this one replaces; search then leaves it out")
	newCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	newCmd.RegisterFlagCompletionFunc("source", sourceCompleter)
	root.AddCommand(newCmd)
}
