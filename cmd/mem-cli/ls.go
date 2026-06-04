package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"mem-cli/internal/config"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var lsTag string
var lsFrom string
var lsUnnamed bool

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

		if lsFrom != "" {
			var filtered []note.Note
			for _, n := range notes {
				for _, f := range n.From {
					if strings.EqualFold(f, lsFrom) {
						filtered = append(filtered, n)
						break
					}
				}
			}
			notes = filtered
		}

		if lsUnnamed {
			var filtered []note.Note
			for _, n := range notes {
				if n.IsUnnamed() {
					filtered = append(filtered, n)
				}
			}
			// Sort oldest first for the processing queue view.
			sort.Slice(filtered, func(i, j int) bool {
				return filtered[i].Created.Before(filtered[j].Created)
			})
			notes = filtered
		}

		if len(notes) == 0 {
			fmt.Fprintln(os.Stderr, "No notes yet — run: mem new")
			return nil
		}

		n, err := pickNote(notes)
		if err != nil || n.Path == "" {
			return err
		}
		return viewBody(n.Path)
	},
}

func init() {
	lsCmd.Flags().StringVarP(&lsTag, "tag", "t", "", "filter by tag")
	lsCmd.Flags().StringVarP(&lsFrom, "from", "f", "", "filter by from value")
	lsCmd.Flags().BoolVar(&lsUnnamed, "unnamed", false, "show only unnamed notes, oldest first")
	lsCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	lsCmd.RegisterFlagCompletionFunc("from", fromCompleter)
	root.AddCommand(lsCmd)
}
