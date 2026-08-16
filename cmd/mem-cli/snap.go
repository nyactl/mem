package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var snapLabels []string
var snapSources []string

var snapCmd = &cobra.Command{
	Use:   "snap <text>",
	Short: "Capture a moment without opening an editor",
	Long: `Writes a timestamped note directly from the command line.
Inline #tags and @sources in the text are extracted automatically.

  mem snap "ran into an interesting paper on indexing #research"
  mem snap "meeting went sideways" -l work -l ops`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		ts := time.Now()

		body := strings.TrimSpace(strings.Join(args, " "))
		if body == "" {
			return fmt.Errorf("snap requires non-empty text")
		}

		allTags := note.MergeTags(snapLabels, note.ExtractInlineTags(body))
		sluggedSources := make([]string, 0, len(snapSources))
		for _, s := range snapSources {
			if sl := note.Slugify(s); sl != "" {
				sluggedSources = append(sluggedSources, sl)
			}
		}
		allSources := note.MergeSources(sluggedSources, note.ExtractInlineSources(body))

		slug := note.Slugify(firstWords(stripInlineMarkers(body), 4))
		if slug == "" {
			slug = "snap"
		}

		// ensure unique filename if multiple snaps land in the same second
		slug2 := slug
		for i := 2; note.Exists(note.FilePath(cfg.NotesDir, ts, slug2)); i++ {
			slug2 = fmt.Sprintf("%s-%d", slug, i)
		}

		path := note.FilePath(cfg.NotesDir, ts, slug2)
		content := note.BuildFrontmatter(allTags, allSources, nil) + body + "\n"
		if err := note.WriteRaw(path, content); err != nil {
			return err
		}

		n, _ := note.Parse(path)
		fmt.Printf("%s  %s\n", n.Created.Format("15:04:05"), n.Slug)

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

// firstWords returns the first n words of s joined by spaces.
func firstWords(s string, n int) string {
	words := strings.Fields(s)
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}

// stripInlineMarkers removes #tag and @source tokens so they don't bleed into slugs.
var reMarkers = regexp.MustCompile(`[#@][a-zA-Z][a-zA-Z0-9-]*`)

func stripInlineMarkers(s string) string {
	return strings.TrimSpace(reMarkers.ReplaceAllString(s, ""))
}

func init() {
	snapCmd.Flags().StringArrayVarP(&snapLabels, "label", "l", nil, "tag, repeatable")
	snapCmd.Flags().StringArrayVarP(&snapSources, "source", "s", nil, "source, repeatable")
	snapCmd.RegisterFlagCompletionFunc("label", tagCompleter)
	snapCmd.RegisterFlagCompletionFunc("source", sourceCompleter)
	root.AddCommand(snapCmd)
}
