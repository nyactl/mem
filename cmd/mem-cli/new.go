package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mem-cli/internal/attachment"
	"mem-cli/internal/config"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var newLabels []string
var newSource string
var newFiles []string

var newCmd = &cobra.Command{
	Use:   "new <title>",
	Short: "Create a new mem note",
	Long:  `Title words can be quoted or unquoted: mem new kafka rebalance blocks partitions`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		title := strings.Join(args, " ")
		slug := note.Slugify(title)
		if slug == "" {
			return fmt.Errorf("title %q produces an empty slug", title)
		}
		ts := time.Now()

		var attachmentPaths []string
		for _, f := range newFiles {
			dst, err := attachment.Store(f, ts, cfg.AttachmentsDir)
			if err != nil {
				return fmt.Errorf("attach %s: %w", f, err)
			}
			attachmentPaths = append(attachmentPaths, dst)
			fmt.Fprintf(os.Stderr, "attached → %s\n", dst)
		}

		path, err := note.Create(cfg.NotesDir, ts, slug, newLabels, newSource, attachmentPaths)
		if err != nil {
			return err
		}
		return openEditor(path)
	},
}

func init() {
	newCmd.Flags().StringArrayVarP(&newLabels, "label", "l", nil, "tag, repeatable: -l kafka -l backend")
	newCmd.Flags().StringVarP(&newSource, "source", "s", "", "source: person name or URL")
	newCmd.Flags().StringArrayVarP(&newFiles, "file", "f", nil, "attach file, repeatable")
	newCmd.RegisterFlagCompletionFunc("label", tagCompleter)
	root.AddCommand(newCmd)
}
