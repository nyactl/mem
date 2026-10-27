package main

import (
	"fmt"
	"os"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var (
	lsTag string
	lsAll bool
)

var lsCmd = &cobra.Command{
	Use:   "ls [<slug>]",
	Short: "Browse or view notes (fzf picker if no slug given)",
	Long: `Shows a note: rendered in a terminal, with mermaid diagrams drawn in box characters, and as
the plain file when piped. Without a slug, picks one in fzf with a rendered preview; there
ctrl-e edits the note under the cursor, ctrl-x deletes it after asking, and ctrl-r edits a
copy saved as a new note that supersedes it (see mem new --supersedes).

A note superseded by a newer one is left out unless --all.`,
	ValidArgsFunction: slugCompleter,
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if len(args) == 1 {
			n, err := note.FindBySlug(cfg.NotesDir, args[0])
			if err != nil {
				return err
			}
			return showNote(n.Path)
		}

		all, err := note.List(cfg.NotesDir)
		if err != nil {
			return err
		}
		q := note.Query{All: lsAll}
		if lsTag != "" {
			q.Tags = []string{lsTag}
		}
		var notes []note.Note
		for _, m := range note.Search(all, q) {
			notes = append(notes, m.Note)
		}

		if len(notes) == 0 {
			fmt.Fprintln(os.Stderr, "no notes found")
			return nil
		}

		n, err := pickNote(notes)
		if err != nil || n.Path == "" {
			return err
		}
		return showNote(n.Path)
	},
}

func init() {
	lsCmd.Flags().StringVarP(&lsTag, "tag", "t", "", "filter by tag")
	lsCmd.Flags().BoolVar(&lsAll, "all", false, "include notes a newer note supersedes")
	lsCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	root.AddCommand(lsCmd)
}
