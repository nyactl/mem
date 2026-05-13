package main

import (
	"fmt"
	"os"
	"time"

	"ref-cli/internal/attachment"
	"ref-cli/internal/config"
	"ref-cli/internal/note"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:               "attach <slug> <file> [<file>...]",
	Short:             "Attach one or more files to an existing note",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: slugCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		n, err := note.FindBySlug(cfg.NotesDir, args[0])
		if err != nil {
			return err
		}

		ts := time.Now()
		var newPaths []string
		for _, f := range args[1:] {
			dst, err := attachment.Store(f, ts, cfg.AttachmentsDir)
			if err != nil {
				return fmt.Errorf("attach %s: %w", f, err)
			}
			newPaths = append(newPaths, dst)
			fmt.Fprintf(os.Stderr, "attached → %s\n", dst)
		}

		return note.UpdateAttachments(n.Path, newPaths)
	},
}

func init() {
	root.AddCommand(attachCmd)
}
