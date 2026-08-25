package main

import (
	"fmt"
	"os"
	"strings"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var lsTag string

var lsCmd = &cobra.Command{
	Use:               "ls [<slug>]",
	Short:             "Browse or view notes (fzf picker if no slug given)",
	ValidArgsFunction: slugCompleter,
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if len(args) == 1 {
			n, err := note.FindBySlug(cfg.NotesDir, args[0])
			if err != nil {
				return err
			}
			return viewFile(n.Path)
		}

		notes, err := note.List(cfg.NotesDir)
		if err != nil {
			return err
		}

		if lsTag != "" {
			var filtered []note.Note
			for _, n := range notes {
				for _, t := range n.Tags {
					if strings.EqualFold(t, lsTag) {
						filtered = append(filtered, n)
						break
					}
				}
			}
			notes = filtered
		}

		if len(notes) == 0 {
			fmt.Fprintln(os.Stderr, "no notes found")
			return nil
		}

		n, err := pickNote(notes)
		if err != nil || n.Path == "" {
			return err
		}
		return viewFile(n.Path)
	},
}

func init() {
	lsCmd.Flags().StringVarP(&lsTag, "tag", "t", "", "filter by tag")
	lsCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	root.AddCommand(lsCmd)
}
