package main

import (
	"fmt"
	"os"
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

var newCmd = &cobra.Command{
	Use:   "new [<title>]",
	Short: "Create a new mem note",
	Long: `Title is optional. The editor opens with natural text — no frontmatter.
Write your note, use #tags and @sources inline. Everything is derived on save.`,
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

		sluggedSources := make([]string, 0, len(newSources))
		for _, s := range newSources {
			if slug := note.Slugify(s); slug != "" {
				sluggedSources = append(sluggedSources, slug)
			}
		}

		var path string
		var err error

		if len(args) == 0 {
			path, err = note.CreateDraft(cfg.NotesDir, ts)
		} else {
			slug := note.Slugify(strings.Join(args, " "))
			if slug == "" {
				return fmt.Errorf("title %q produces an empty slug", strings.Join(args, " "))
			}
			path, err = note.Create(cfg.NotesDir, ts, slug)
		}
		if err != nil {
			return err
		}

		if err := openEditor(path); err != nil {
			return err
		}

		finalPath, err := note.FinalizeNote(path, newLabels, sluggedSources, attachmentPaths)
		if err != nil {
			return err
		}
		if finalPath == "" {
			return nil
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	newCmd.Flags().StringArrayVarP(&newLabels, "tag", "t", nil, "tag, repeatable: -t kafka -t backend")
	newCmd.Flags().StringArrayVarP(&newSources, "source", "s", nil, "source, repeatable: -s kate -s thomas-mueller")
	newCmd.Flags().StringArrayVarP(&newFiles, "file", "f", nil, "attach file, repeatable")
	newCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	newCmd.RegisterFlagCompletionFunc("source", sourceCompleter)
	root.AddCommand(newCmd)
}
