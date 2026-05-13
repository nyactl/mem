package main

import (
	"mem-cli/internal/config"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:               "edit [<slug>]",
	Short:             "Open a note in $EDITOR (fzf picker if no slug given)",
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

		return openEditor(n.Path)
	},
}

func init() {
	root.AddCommand(editCmd)
}
