package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"
	"mem/internal/render"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

var root = &cobra.Command{
	Use:          "mem",
	Short:        "Atomic notes captured fast, found faster",
	Version:      version,
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

// showNote prints a note: rendered and paged in a terminal, as the plain file otherwise. less
// gets the options in less unless LESS is set.
func showNote(path, less string) error {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(raw)
		return err
	}
	n, err := note.Parse(path)
	if err != nil {
		return err
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		width = 80
	}
	out, err := render.Note(n, width, false)
	if err != nil {
		return err
	}
	return page(out, less)
}

// Options for less: FRX prints a short note and leaves it on the screen, the way git does. Shown
// from the picker, a note stays until q: fzf would redraw over it at once.
const (
	lessShow   = "FRX"
	lessPicker = "R"
)

// page shows s in $PAGER, less by default with the options in less unless LESS is set.
func page(s, less string) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}
	parts := strings.Fields(pager)
	if _, err := exec.LookPath(parts[0]); err != nil {
		_, err := io.WriteString(os.Stdout, s)
		return err
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	if os.Getenv("LESS") == "" {
		cmd.Env = append(os.Environ(), "LESS="+less)
	}
	cmd.Stdin = strings.NewReader(s)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// pickerKeys act on the note under the picker's cursor through the hidden picker command.
// Their keys are ones fzf leaves free, except ctrl-e, whose end-of-line the End key still does.
var pickerKeys = []struct{ key, hint, action string }{
	{"ctrl-e", "^E", "edit"},
	{"ctrl-x", "^X", "delete"},
	{"ctrl-r", "^R", "supersede"},
}

// pickNote presents notes in fzf with a rendered preview and returns the selected note. Returns a
// zero Note (empty Path) if the user cancelled.
func pickNote(notes []note.Note) (note.Note, error) {
	return runPicker(notes, false)
}

// browseNotes presents notes in fzf until the user leaves it; enter shows a note and comes back.
func browseNotes(notes []note.Note) error {
	_, err := runPicker(notes, true)
	return err
}

// runPicker runs fzf over notes. The pickerKeys change a note in place: the list lives in a
// temporary file that the action rewrites and fzf reloads, so the picker stays open.
func runPicker(notes []note.Note, browse bool) (note.Note, error) {
	if len(notes) == 0 {
		return note.Note{}, fmt.Errorf("no notes found")
	}

	var lines []string
	for _, n := range notes {
		lines = append(lines, pickerLine(n))
	}

	exe, err := os.Executable()
	if err != nil {
		return note.Note{}, err
	}
	dir, err := os.MkdirTemp("", "mem-pick-")
	if err != nil {
		return note.Note{}, err
	}
	defer os.RemoveAll(dir)
	list := filepath.Join(dir, "list")
	if err := os.WriteFile(list, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return note.Note{}, err
	}

	var hints []string
	args := []string{
		"--delimiter=\t",
		"--with-nth=2",
		"--ansi",
		"--no-sort",
		"--preview", shellQuote(exe) + " picker preview {1}",
		"--preview-window", "right,60%,wrap,<120(down,60%,wrap)",
	}
	if browse {
		hints = append(hints, "enter show")
		args = append(args, "--bind", fmt.Sprintf("enter:execute(%s picker show {1})", shellQuote(exe)))
	}
	for _, k := range pickerKeys {
		hints = append(hints, k.hint+" "+k.action)
		args = append(args, "--bind", fmt.Sprintf("%s:execute(%s picker %s {1} %s)+reload(cat %s)+refresh-preview",
			k.key, shellQuote(exe), k.action, shellQuote(list), shellQuote(list)))
	}
	args = append(args, "--header", strings.Join(hints, "  "))

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	// A closed terminal hangs up on fzf and on mem alike. Outliving fzf lets mem remove the list.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGTERM)
	err = cmd.Run()
	signal.Stop(signals)
	if err != nil {
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

// pickerLine is a note's line in the picker: its path, which fzf hides, then its date (the one
// notes are sorted by), slug and dimmed tags.
func pickerLine(n note.Note) string {
	line := n.Path + "\t" + n.DisplayTime().Format("2006-01-02") + " " + n.DisplayTitle()
	if len(n.Tags) > 0 {
		line += " \x1b[2m#" + strings.Join(n.Tags, " #") + "\x1b[0m"
	}
	return line
}

// shellQuote quotes s for the shell fzf runs its commands in.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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
		if n.Title != "" {
			meta = n.Title + " " + meta
		}
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
