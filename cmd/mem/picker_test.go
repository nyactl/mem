package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mem/internal/note"
)

func TestSupersedeNote(t *testing.T) {
	dir := t.TempDir()
	n := dated(t, dir)
	before, _ := os.ReadFile(n.Path)
	useEditor(t, `printf 'corrected line\n' > "$1"`)

	nn, err := supersedeNote(dir, n)
	if err != nil {
		t.Fatal(err)
	}
	if nn.Path == "" || nn.Path == n.Path || nn.Slug != n.Slug {
		t.Fatalf("new note %+v", nn)
	}
	if after, _ := os.ReadFile(n.Path); string(after) != string(before) {
		t.Errorf("old note changed:\n%s", after)
	}
	if nn.Body != "corrected line" || nn.Date != nil {
		t.Errorf("body %q date %v", nn.Body, nn.Date)
	}
	want := "agentdeck,supersedes-" + strings.ToLower(n.ID())
	if got := strings.Join(nn.Sources, ","); got != want {
		t.Errorf("sources = %s, want %s", got, want)
	}
	if strings.Join(nn.Tags, ",") != "kb,work" || strings.Join(nn.Attachments, ",") != "x.png" {
		t.Errorf("tags %v attachments %v", nn.Tags, nn.Attachments)
	}
	all, _ := note.List(dir)
	if m := note.Search(all, note.Query{}); len(m) != 1 || m[0].Path != nn.Path {
		t.Errorf("search shows %d notes, want only the new one", len(m))
	}
}

func TestSupersedeNoteUnchangedOrEmptied(t *testing.T) {
	for name, script := range map[string]string{"unchanged": `true`, "emptied": `: > "$1"`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			n := dated(t, dir)
			useEditor(t, script)
			nn, err := supersedeNote(dir, n)
			if err != nil || nn.Path != "" {
				t.Fatalf("new note %q err %v", nn.Path, err)
			}
			if all, _ := note.List(dir); len(all) != 1 {
				t.Errorf("%d notes, want 1", len(all))
			}
		})
	}
}

func TestReplaceLine(t *testing.T) {
	dir := t.TempDir()
	a := note.Note{Path: filepath.Join(dir, "a.md"), Slug: "a", Created: time.Unix(0, 0)}
	b := note.Note{Path: filepath.Join(dir, "b.md"), Slug: "b", Created: time.Unix(0, 0)}
	c := note.Note{Path: filepath.Join(dir, "c.md"), Slug: "c", Created: time.Unix(0, 0), Tags: []string{"t"}}
	list := filepath.Join(dir, "list")
	if err := os.WriteFile(list, []byte(pickerLine(a)+"\n"+pickerLine(b)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := replaceLine(list, a.Path, &c); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(list); string(got) != pickerLine(c)+"\n"+pickerLine(b)+"\n" {
		t.Errorf("after replace:\n%s", got)
	}
	if err := replaceLine(list, b.Path, nil); err != nil {
		t.Fatal(err)
	}
	if err := replaceLine(list, c.Path, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(list); string(got) != "" {
		t.Errorf("after removing all: %q", got)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("/a b/it's"); got != `'/a b/it'\''s'` {
		t.Errorf("got %s", got)
	}
}
