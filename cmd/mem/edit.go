package main

import (
	"fmt"
	"os"
	"strings"

	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:               "edit [<slug>]",
	Short:             "Edit a note's body in $EDITOR (fzf picker if no slug given)",
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

		_, err = editNote(cfg.NotesDir, n)
		return err
	},
}

// editNote edits n's body in $EDITOR. Emptying it deletes the note. It reports whether n changed.
func editNote(notesDir string, n note.Note) (bool, error) {
	body, err := editBody(n.Body)
	if err != nil || body == strings.TrimRight(n.Body, "\n \t") {
		return false, err
	}
	if body == "" {
		if err := os.Remove(n.Path); err != nil {
			return false, fmt.Errorf("delete: %w", err)
		}
	} else if err := note.SetBody(n.Path, body); err != nil {
		return false, err
	}
	_, _ = index.Rebuild(notesDir)
	return true, nil
}

// editBody opens body in $EDITOR and returns what was saved, without trailing blank lines. The
// editor gets a copy, so the note's frontmatter stays out of its reach and is untouched when it
// fails.
func editBody(body string) (string, error) {
	f, err := os.CreateTemp("", "mem-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, err = f.WriteString(strings.TrimRight(body, "\n") + "\n")
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	if err := openEditor(f.Name()); err != nil {
		return "", err
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n \t"), nil
}

func init() {
	root.AddCommand(editCmd)
}
