package main

import (
	"mem-cli/internal/config"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:               "get [<slug>]",
	Short:             "View a note (fzf picker if no slug given)",
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

		return viewFile(n.Path)
	},
}

func init() {
	root.AddCommand(getCmd)
}
