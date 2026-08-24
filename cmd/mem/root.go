package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:          "mem",
	Short:        "Shared external memory for human and AI",
	SilenceUsage: true,
}

// openEditor opens path in $EDITOR (nvim fallback).
func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nvim"
	}
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// viewFile opens path in $PAGER (bat → less → cat fallback).
func viewFile(path string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		if _, err := exec.LookPath("bat"); err == nil {
			pager = "bat"
		} else {
			pager = "less"
		}
	}
	parts := strings.Fields(pager)
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Print(string(raw))
	}
	return nil
}

// pickNote presents notes in fzf with a bat preview pane and returns the
// selected note. Returns a zero Note (empty Path) if the user cancelled.
func pickNote(notes []note.Note) (note.Note, error) {
	if len(notes) == 0 {
		return note.Note{}, fmt.Errorf("no notes found")
	}

	var lines []string
	for _, n := range notes {
		ts := n.Created.Format("2006-01-02 15:04:05")
		tags := ""
		for _, t := range n.Tags {
			tags += "#" + t + " "
		}
		lines = append(lines, n.Path+"\t"+ts+"\t"+n.Slug+"\t"+strings.TrimRight(tags, " "))
	}

	previewCmd := "bat --color=always --style=plain --language=markdown {1}"
	if _, err := exec.LookPath("bat"); err != nil {
		previewCmd = "cat {1}"
	}

	args := []string{
		"--delimiter=\t",
		"--with-nth=2,3,4",
		"--ansi",
		"--no-sort",
		"--preview", previewCmd,
		"--preview-window", "right:60%:wrap",
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
			return note.Note{}, nil
		}
		return note.Note{}, fmt.Errorf("fzf: %w", err)
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return note.Note{}, nil
	}
	path := strings.SplitN(selected, "\t", 2)[0]
	return note.Parse(path)
}

func slugCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg := config.Load()
	notes, err := note.List(cfg.NotesDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, len(notes))
	for i, n := range notes {
		meta := n.CreatedStr() + " [" + strings.Join(n.Tags, ", ") + "]"
		if len(n.Sources) > 0 {
			meta += " @" + strings.Join(n.Sources, ", @")
		}
		out[i] = n.Slug + "\t" + meta
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func sourceCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Load()
	idx, err := index.Load(cfg.NotesDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return idx.Sources, cobra.ShellCompDirectiveNoFileComp
}

func tagCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Load()
	idx, err := index.Load(cfg.NotesDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return idx.Tags, cobra.ShellCompDirectiveNoFileComp
}
