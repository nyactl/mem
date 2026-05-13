package main

import (
	"fmt"
	"os"
	"strings"

	"ref-cli/internal/config"
	"ref-cli/internal/note"

	"github.com/spf13/cobra"
)

var lsTag string

var lsCmd = &cobra.Command{
	Use:               "ls",
	Short:             "Browse notes interactively via fzf",
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

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
