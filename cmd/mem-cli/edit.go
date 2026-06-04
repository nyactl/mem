package main

import (
	"os"

	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:               "edit [<id>]",
	Short:             "Open a note in $EDITOR (fzf picker if no id given)",
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
			n, err = note.FindByIdentifier(cfg.NotesDir, args[0])
			if err != nil {
				return err
			}
		}

		tmpPath, tags, from, attachments, err := note.BeginEdit(n.Path)
		if err != nil {
			return err
		}

		if err := openEditor(tmpPath); err != nil {
			os.Remove(tmpPath)
			if isExitError(err) {
				return nil
			}
			return err
		}

		if _, err := note.CommitEdit(n.Path, tmpPath, tags, from, attachments); err != nil {
			return err
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	root.AddCommand(editCmd)
}
