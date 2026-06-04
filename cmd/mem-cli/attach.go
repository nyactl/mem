package main

import (
	"fmt"
	"os"
	"time"

	"mem-cli/internal/attachment"
	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:               "attach <id> <file> [<file>...]",
	Short:             "Attach one or more files to an existing note",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: slugCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		n, err := note.FindByIdentifier(cfg.NotesDir, args[0])
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

		if err := note.UpdateAttachments(n.Path, newPaths); err != nil {
			return err
		}
		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	root.AddCommand(attachCmd)
}
