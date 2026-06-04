package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mem-cli/internal/config"
	"mem-cli/internal/index"
	"mem-cli/internal/note"

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

// isExitError returns true if err is a non-zero exit from an exec'd process.
func isExitError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*exec.ExitError)
	return ok
}

// viewBody reads the body via note.Body and pipes to bat or less or prints directly.
func viewBody(path string) error {
	body, err := note.Body(path)
	if err != nil {
		return err
	}

	if _, err := exec.LookPath("bat"); err == nil {
		cmd := exec.Command("bat", "--language=markdown", "-")
		cmd.Stdin = strings.NewReader(body)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	if _, err := exec.LookPath("less"); err == nil {
		cmd := exec.Command("less")
		cmd.Stdin = strings.NewReader(body)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	fmt.Print(body)
	return nil
}

// pickNote presents notes in fzf and returns the selected note.
// Returns a zero Note (empty Path) if the user cancelled.
func pickNote(notes []note.Note) (note.Note, error) {
	if len(notes) == 0 {
		return note.Note{}, fmt.Errorf("no notes found")
	}

	var lines []string
	for _, n := range notes {
		display := n.Slug
		if display == "" {
			display = "(unnamed)"
		}
		lines = append(lines, n.Path+"\t"+display+"\t"+n.CreatedStr())
	}

	previewCmd := `awk 'BEGIN{f=0}/^---$/{f++;next}f==1{next}{print}' {1}`
	if _, err := exec.LookPath("bat"); err == nil {
		previewCmd = `awk 'BEGIN{f=0}/^---$/{f++;next}f==1{next}{print}' {1} | bat --language=markdown --color=always -`
	}

	args := []string{
		"--delimiter=\t",
		"--with-nth=2,3",
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
		if len(n.From) > 0 {
			meta += " @" + strings.Join(n.From, ", @")
		}
		out[i] = n.Identifier() + "\t" + meta
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func fromCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Load()
	idx, err := index.Load(cfg.NotesDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return idx.From, cobra.ShellCompDirectiveNoFileComp
}

func tagCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg := config.Load()
	idx, err := index.Load(cfg.NotesDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return idx.Tags, cobra.ShellCompDirectiveNoFileComp
}
