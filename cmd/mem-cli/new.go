package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"mem-cli/internal/attachment"
	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var newTags []string
var newFrom []string
var newFiles []string
var newBody string

var newCmd = &cobra.Command{
	Use:   "new [<title>]",
	Short: "Create a new mem note",
	Long: `Title is optional. The editor opens with natural text — no frontmatter.
Write your note, use #tags and @from inline. Everything is derived on save.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
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

		sluggedFrom := make([]string, 0, len(newFrom))
		for _, f := range newFrom {
			if slug := note.Slugify(f); slug != "" {
				sluggedFrom = append(sluggedFrom, slug)
			}
		}

		var slug string
		if len(args) > 0 {
			slug = note.Slugify(strings.Join(args, " "))
			if slug == "" {
				return fmt.Errorf("title %q produces an empty slug", strings.Join(args, " "))
			}
		}

		// Determine body source.
		var body string
		interactive := true

		if newBody != "" {
			body = newBody
			interactive = false
		} else {
			// Check if stdin is piped.
			stat, err := os.Stdin.Stat()
			if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				body = string(data)
				interactive = false
			}
		}

		var finalPath string

		if !interactive {
			tmpPath, err := note.NewTemp()
			if err != nil {
				return err
			}
			if err := os.WriteFile(tmpPath, []byte(body), 0600); err != nil {
				os.Remove(tmpPath)
				return err
			}
			finalPath, err = note.CommitNew(cfg.NotesDir, ts, slug, tmpPath, newTags, sluggedFrom, attachmentPaths)
			if err != nil {
				return err
			}
		} else {
			tmpPath, err := note.NewTemp()
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
			finalPath, err = note.CommitNew(cfg.NotesDir, ts, slug, tmpPath, newTags, sluggedFrom, attachmentPaths)
			if err != nil {
				return err
			}
		}

		if finalPath == "" {
			return nil
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		n, err := note.Parse(finalPath)
		if err != nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "created: %s\n", n.Identifier())
		return nil
	},
}

func init() {
	newCmd.Flags().StringArrayVarP(&newTags, "tag", "t", nil, "tag, repeatable: -t kafka -t backend")
	newCmd.Flags().StringArrayVarP(&newFrom, "from", "f", nil, "from, repeatable: -f alice -f bob")
	newCmd.Flags().StringArrayVarP(&newFiles, "attach", "a", nil, "attach file, repeatable")
	newCmd.Flags().StringVar(&newBody, "body", "", "note body (skips editor)")
	newCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	newCmd.RegisterFlagCompletionFunc("from", fromCompleter)
	root.AddCommand(newCmd)
}
