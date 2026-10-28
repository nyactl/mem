package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mem/internal/note"
)

// useEditor makes $EDITOR a script running body with the file to edit as $1.
func useEditor(t *testing.T, body string) {
	t.Helper()
	script := filepath.Join(t.TempDir(), "editor")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EDITOR", script)
}

// dated writes a note with a title, a date, tags, a source, an attachment and a supersedes source, as a
// finding has them.
func dated(t *testing.T, dir string) note.Note {
	t.Helper()
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	path := note.FilePath(dir, time.Date(2026, 10, 27, 8, 0, 0, 0, time.Local), "finding")
	fm := note.BuildFrontmatter("Finding: C++", []string{"kb", "work"}, []string{"agentdeck", "supersedes-20261001t000000-older"}, []string{"x.png"}, &day)
	if err := note.WriteRaw(path, fm+"first line\nsecond line\n"); err != nil {
		t.Fatal(err)
	}
	n, err := note.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestEditNoteKeepsDate(t *testing.T) {
	dir := t.TempDir()
	n := dated(t, dir)
	useEditor(t, `printf 'third line #extra\n' >> "$1"`)

	changed, err := editNote(dir, n)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got, err := note.Parse(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Date == nil || !got.Date.Equal(*n.Date) {
		t.Errorf("date = %v, want %v", got.Date, n.Date)
	}
	if got.Body != "first line\nsecond line\nthird line #extra" {
		t.Errorf("body = %q", got.Body)
	}
	if strings.Join(got.Tags, ",") != "kb,work,extra" || strings.Join(got.Attachments, ",") != "x.png" {
		t.Errorf("tags %v attachments %v", got.Tags, got.Attachments)
	}
	if got.Title != "Finding: C++" {
		t.Errorf("title = %q", got.Title)
	}
}

func TestEditNoteLeavesNoteWhenEditorFails(t *testing.T) {
	dir := t.TempDir()
	n := dated(t, dir)
	before, _ := os.ReadFile(n.Path)
	useEditor(t, `printf 'half written' > "$1"; exit 1`)

	if _, err := editNote(dir, n); err == nil {
		t.Fatal("no error from a failing editor")
	}
	after, _ := os.ReadFile(n.Path)
	if string(after) != string(before) {
		t.Errorf("note changed:\n%s", after)
	}
}

func TestEditNoteUnchanged(t *testing.T) {
	dir := t.TempDir()
	n := dated(t, dir)
	before, _ := os.ReadFile(n.Path)
	useEditor(t, `true`)

	if changed, err := editNote(dir, n); err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if after, _ := os.ReadFile(n.Path); string(after) != string(before) {
		t.Errorf("note rewritten:\n%s", after)
	}
}

func TestEditNoteEmptiedDeletes(t *testing.T) {
	dir := t.TempDir()
	n := dated(t, dir)
	useEditor(t, `: > "$1"`)

	if changed, err := editNote(dir, n); err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if note.Exists(n.Path) {
		t.Error("emptied note still there")
	}
}
