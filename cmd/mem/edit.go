package main

import (
	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:               "edit [<slug>]",
	Short:             "Edit a note's body in $EDITOR (fzf picker if no slug given)",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: slugCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		var n note.Note
		var err error

		if len(args) == 0 {
			notes, err := note.List(cfg.NotesDir)
			if err != nil {
				return err
			}
			n, err = pickNote(notes)
			if err != nil || n.Path == "" {
				return err
			}
		} else {
			n, err = note.FindBySlug(cfg.NotesDir, args[0])
			if err != nil {
				return err
			}
		}

		// Strip frontmatter before opening — human only sees body.
		tags, sources, attachments, err := note.StripFrontmatter(n.Path)
		if err != nil {
			return err
		}

		if err := openEditor(n.Path); err != nil {
			return err
		}

		// Restore frontmatter, merging any inline changes from the edit.
		if _, err := note.FinalizeNote(n.Path, tags, sources, attachments); err != nil {
			return err
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	root.AddCommand(editCmd)
}
