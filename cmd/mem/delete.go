package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:               "delete [<slug>]",
	Short:             "Delete a note (fzf picker if no slug given)",
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

		fmt.Fprintf(os.Stderr, "delete %s? [y/N] ", n.Slug)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(line)) != "y" {
			fmt.Fprintln(os.Stderr, "cancelled")
			return nil
		}

		if err := os.Remove(n.Path); err != nil {
			return fmt.Errorf("delete: %w", err)
		}

		// Sync state entry is intentionally left intact so that
		// mem sync push propagates the deletion to the server.

		_, _ = index.Rebuild(cfg.NotesDir)
		fmt.Fprintf(os.Stderr, "deleted %s\n", n.Slug)
		return nil
	},
}

func init() {
	root.AddCommand(deleteCmd)
}
