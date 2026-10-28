package main

import (
	"path/filepath"
	"testing"

	"mem/internal/config"
	"mem/internal/note"
)

func TestAddNoteTitleOrFirstWords(t *testing.T) {
	cfg := config.Config{NotesDir: t.TempDir(), AttachmentsDir: t.TempDir()}
	for _, c := range []struct{ title, body, slug, wantTitle string }{
		{"C++", "a language", "c", "C++"},
		{"  I/O \n", "input and output", "io", "I/O"},
		{"", "kafka rebalance blocks all partitions #kb", "kafka-rebalance-blocks-all", ""},
		{"+++", "only punctuation", "note", "+++"},
		{" \t ", "blank title falls back", "blank-title-falls-back", ""},
	} {
		n, err := addNote(cfg, c.title, c.body, nil, nil, nil, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		if n.Slug != c.slug || n.Title != c.wantTitle {
			t.Errorf("%q: slug %q title %q, want %q %q", c.title, n.Slug, n.Title, c.slug, c.wantTitle)
		}
	}
	if _, err := addNote(cfg, "CAS", "  ", nil, nil, nil, "", nil); err == nil {
		t.Error("empty body accepted")
	}
}

func TestAddNoteCreatesNotesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fresh", "notes")
	if _, err := addNote(config.Config{NotesDir: dir}, "", "first note on a new machine", nil, nil, nil, "", nil); err != nil {
		t.Fatal(err)
	}
	if notes, _ := note.List(dir); len(notes) != 1 {
		t.Errorf("%d notes", len(notes))
	}
}
