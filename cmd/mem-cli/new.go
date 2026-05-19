package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mem-cli/internal/attachment"
	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var newLabels []string
var newSource string
var newFiles []string

var newCmd = &cobra.Command{
	Use:   "new [<title>]",
	Short: "Create a new mem note",
	Long: `Title is optional. When omitted the editor opens immediately; the first
# Heading becomes the slug. Inline #tags and @source are extracted from the body.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		ts := time.Now()

		var attachmentPaths []string
		for _, f := range newFiles {
			dst, err := attachment.Store(f, ts, cfg.AttachmentsDir)
			if err != nil {
				return fmt.Errorf("attach %s: %w", f, err)
			}
			attachmentPaths = append(attachmentPaths, dst)
			fmt.Fprintf(os.Stderr, "attached → %s\n", dst)
		}

		var path string
		var err error

		if len(args) == 0 {
			path, err = note.CreateDraft(cfg.NotesDir, ts)
		} else {
			title := strings.Join(args, " ")
			slug := note.Slugify(title)
			if slug == "" {
				return fmt.Errorf("title %q produces an empty slug", title)
			}
			path, err = note.Create(cfg.NotesDir, ts, slug, newLabels, newSource, attachmentPaths)
		}
		if err != nil {
			return err
		}

		if err := openEditor(path); err != nil {
			return err
		}

		finalPath, err := note.FinalizeNote(path, newLabels, newSource)
		if err != nil {
			return err
		}
		if finalPath == "" {
			// Empty draft discarded.
			return nil
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	newCmd.Flags().StringArrayVarP(&newLabels, "label", "l", nil, "tag, repeatable: -l kafka -l backend")
	newCmd.Flags().StringVarP(&newSource, "source", "s", "", "source: person name or URL")
	newCmd.Flags().StringArrayVarP(&newFiles, "file", "f", nil, "attach file, repeatable")
	newCmd.RegisterFlagCompletionFunc("label", tagCompleter)
	newCmd.RegisterFlagCompletionFunc("source", sourceCompleter)
	root.AddCommand(newCmd)
}
