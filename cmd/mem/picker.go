package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"mem/internal/config"
	"mem/internal/note"
	"mem/internal/render"

	"github.com/spf13/cobra"
)

// pickerCmd runs what the fzf picker binds to its preview and keys; it is not for typing.
var pickerCmd = &cobra.Command{
	Use:    "picker",
	Hidden: true,
}

var pickerPreviewCmd = &cobra.Command{
	Use:  "preview <path>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := note.Parse(args[0])
		if err != nil {
			return err
		}
		width, _ := strconv.Atoi(os.Getenv("FZF_PREVIEW_COLUMNS"))
		if width <= 0 {
			width = 80
		}
		out, err := render.Note(n, width, true)
		if err != nil {
			raw, rerr := os.ReadFile(args[0])
			if rerr != nil {
				return err
			}
			out = string(raw)
		}
		fmt.Print(out)
		return nil
	},
}

var pickerShowCmd = &cobra.Command{
	Use:  "show <path>",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := showNote(args[0], lessPicker); err != nil {
			fmt.Fprintln(os.Stderr, err)
			waitForKey()
		}
		return nil
	},
}

func pickerActionCmd(action string, run func(notesDir string, n note.Note) (*note.Note, error)) *cobra.Command {
	return &cobra.Command{
		Use:  action + " <path> <list>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := note.Parse(args[0])
			if err != nil {
				return err
			}
			replacement, err := run(config.Load().NotesDir, n)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				waitForKey()
				return nil
			}
			return replaceLine(args[1], n.Path, replacement)
		},
	}
}

// replaceLine rewrites the picker's list file: the line for path becomes replacement's line, or
// goes when replacement is nil.
func replaceLine(list, path string, replacement *note.Note) error {
	data, err := os.ReadFile(list)
	if err != nil {
		return err
	}
	var out []string
	for _, l := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if p, _, _ := strings.Cut(l, "\t"); p != path {
			out = append(out, l)
		} else if replacement != nil {
			out = append(out, pickerLine(*replacement))
		}
	}
	text := ""
	if len(out) > 0 {
		text = strings.Join(out, "\n") + "\n"
	}
	return note.WriteRaw(list, text)
}

// waitForKey keeps an error on screen until Enter: fzf redraws over it as soon as the action ends.
func waitForKey() {
	fmt.Fprint(os.Stderr, "press enter to go back ")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
}

func init() {
	pickerCmd.AddCommand(pickerPreviewCmd)
	pickerCmd.AddCommand(pickerShowCmd)
	pickerCmd.AddCommand(pickerActionCmd("edit", func(dir string, n note.Note) (*note.Note, error) {
		if _, err := editNote(dir, n); err != nil {
			return &n, err
		}
		if !note.Exists(n.Path) {
			return nil, nil
		}
		nn, err := note.Parse(n.Path)
		return &nn, err
	}))
	pickerCmd.AddCommand(pickerActionCmd("delete", func(dir string, n note.Note) (*note.Note, error) {
		deleted, err := deleteNote(dir, n)
		if err != nil || !deleted {
			return &n, err
		}
		return nil, nil
	}))
	pickerCmd.AddCommand(pickerActionCmd("supersede", func(dir string, n note.Note) (*note.Note, error) {
		nn, err := supersedeNote(dir, n)
		if err != nil || nn.Path == "" {
			return &n, err
		}
		return &nn, nil
	}))
	root.AddCommand(pickerCmd)
}
