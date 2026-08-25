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

var newLabels []string
var newSources []string
var newFiles []string
var newDate string

var newCmd = &cobra.Command{
	Use:   "new <content>  |  new <title> <content>",
	Short: "Capture a note from the command line",
	Long: `Creates a note instantly without opening an editor.

  mem new "ran into an interesting paper on indexing #research"
  mem new "kafka rebalance" "consumer group blocks all partitions for ~2min"
  mem new "meeting went sideways" -t work -t ops -d 2024-03-15

One argument:  slug is derived from the first few words of the content.
Two arguments: first is the title (sets the slug), second is the body.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		ts := time.Now()

		var slug, body string
		if len(args) == 1 {
			body = strings.TrimSpace(args[0])
			slug = note.Slugify(firstWords(stripInlineMarkers(body), 4))
		} else {
			slug = note.Slugify(args[0])
			body = strings.TrimSpace(args[1])
		}

		if slug == "" {
			slug = "note"
		}
		if body == "" {
			return fmt.Errorf("content cannot be empty")
		}

		var attachmentPaths []string
		for _, f := range newFiles {
			dst, err := attachment.Store(f, ts, cfg.AttachmentsDir)
			if err != nil {
				return fmt.Errorf("attach %s: %w", f, err)
			}
			attachmentPaths = append(attachmentPaths, dst)
			fmt.Fprintf(os.Stderr, "attached → %s\n", dst)
		}

		sluggedSources := make([]string, 0, len(newSources))
		for _, s := range newSources {
			if sl := note.Slugify(s); sl != "" {
				sluggedSources = append(sluggedSources, sl)
			}
		}

		allTags := note.MergeTags(newLabels, note.ExtractInlineTags(body))
		allSources := note.MergeSources(sluggedSources, note.ExtractInlineSources(body))

		slug2 := slug
		for i := 2; note.Exists(note.FilePath(cfg.NotesDir, ts, slug2)); i++ {
			slug2 = fmt.Sprintf("%s-%d", slug, i)
		}

		var date *time.Time
		if newDate != "" {
			formats := []string{
				time.RFC3339,
				"2006-01-02T15:04:05",
				"2006-01-02T15:04",
				"2006-01-02 15:04:05",
				"2006-01-02 15:04",
				"2006-01-02",
			}
			var parsed time.Time
			var perr error
			for _, f := range formats {
				if parsed, perr = time.ParseInLocation(f, newDate, time.Local); perr == nil {
					break
				}
			}
			if perr != nil {
				return fmt.Errorf("--date %q: unrecognised datetime format", newDate)
			}
			date = &parsed
		}

		path := note.FilePath(cfg.NotesDir, ts, slug2)
		content := note.BuildFrontmatter(allTags, allSources, attachmentPaths, date) + body + "\n"
		if err := note.WriteRaw(path, content); err != nil {
			return err
		}

		n, _ := note.Parse(path)
		fmt.Printf("%s  %s\n", n.Created.Format("15:04:05"), n.Slug)

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
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
	newCmd.Flags().StringArrayVarP(&newLabels, "tag", "t", nil, "tag, repeatable")
	newCmd.Flags().StringArrayVarP(&newSources, "source", "s", nil, "source, repeatable")
	newCmd.Flags().StringArrayVarP(&newFiles, "file", "f", nil, "attach file, repeatable")
	newCmd.Flags().StringVarP(&newDate, "date", "d", "", "override display datetime, e.g. 2024-03-15T09:00 or 2024-03-15")
	newCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	newCmd.RegisterFlagCompletionFunc("source", sourceCompleter)
	root.AddCommand(newCmd)
}
