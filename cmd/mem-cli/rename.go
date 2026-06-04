package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

	"github.com/spf13/cobra"
)

var renameYes bool

var renameCmd = &cobra.Command{
	Use:               "rename <old> <new-title>",
	Short:             "Rename a note slug",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: slugCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		oldID := args[0]
		newSlug := note.Slugify(strings.Join(args[1:], " "))
		if newSlug == "" {
			return fmt.Errorf("new title %q produces an empty slug", args[1])
		}

		n, err := note.FindByIdentifier(cfg.NotesDir, oldID)
		if err != nil {
			return err
		}

		oldSlug := n.Identifier()
		if oldSlug == newSlug {
			fmt.Fprintln(os.Stderr, "nothing to rename: slugs are identical")
			return nil
		}

		// Check for collision before doing anything.
		existing, err := note.FindByIdentifier(cfg.NotesDir, newSlug)
		if err == nil && existing.Path != "" {
			return fmt.Errorf("note %q already exists", newSlug)
		}

		// Check for [[oldSlug]] references to decide whether to prompt.
		notes, err := note.List(cfg.NotesDir)
		if err != nil {
			return err
		}
		oldRef := "[[" + oldSlug + "]]"
		var refFiles []string
		for _, ref := range notes {
			data, err := os.ReadFile(ref.Path)
			if err != nil {
				continue
			}
			if strings.Contains(string(data), oldRef) {
				refFiles = append(refFiles, ref.Path)
			}
		}

		if len(refFiles) > 0 && !renameYes {
			fmt.Fprintf(os.Stderr, "rename %q → %q will update %d file(s):\n", oldSlug, newSlug, len(refFiles))
			for _, f := range refFiles {
				fmt.Fprintf(os.Stderr, "  %s\n", f)
			}
			fmt.Fprint(os.Stderr, "Continue? [y/N] ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(os.Stderr, "aborted")
				return nil
			}
		}

		modified, err := note.RenameSlug(cfg.NotesDir, oldSlug, newSlug)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "renamed: %s → %s\n", oldSlug, newSlug)
		if len(modified) > 0 {
			fmt.Fprintf(os.Stderr, "updated %d reference(s)\n", len(modified))
		}

		_, _ = index.Rebuild(cfg.NotesDir)
		return nil
	},
}

func init() {
	renameCmd.Flags().BoolVarP(&renameYes, "yes", "y", false, "skip confirmation prompt")
	root.AddCommand(renameCmd)
}
