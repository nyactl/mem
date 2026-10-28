package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"mem/internal/config"
	"mem/internal/note"
)

// entry writes a glossary entry dated day, a second apart from the others so their files differ.
func entry(t *testing.T, dir, term, meaning string, day int, tags ...string) note.Note {
	t.Helper()
	d := time.Date(2026, 10, day, 9, 0, 0, 0, time.Local)
	path := uniquePath(dir, d, note.Slugify(term))
	if tags == nil {
		tags = []string{glossaryTag}
	}
	if err := note.WriteRaw(path, note.BuildFrontmatter(term, tags, nil, nil, nil)+meaning+"\n"); err != nil {
		t.Fatal(err)
	}
	n, err := note.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func titles(entries []note.Note) string {
	var out []string
	for _, e := range entries {
		out = append(out, e.DisplayTitle()+"="+e.Body)
	}
	return strings.Join(out, " | ")
}

func TestGlossaryLookupExactBeforePrefix(t *testing.T) {
	dir := t.TempDir()
	entry(t, dir, "CASE", "a switch arm", 1)
	entry(t, dir, "CAS", "compare-and-swap", 2)
	entry(t, dir, "cas", "Central Authentication Service", 3)
	entry(t, dir, "Cascade", "a waterfall", 4)
	entry(t, dir, "ACAS", "not a prefix match", 5)
	entry(t, dir, "CAS", "not a glossary entry", 6, "kb")
	notes, _ := note.List(dir)

	want := "CAS=compare-and-swap | cas=Central Authentication Service | Cascade=a waterfall | CASE=a switch arm"
	for _, term := range []string{"cas", "CAS", " Cas "} {
		if got := titles(glossaryLookup(notes, term)); got != want {
			t.Errorf("%q:\n got %s\nwant %s", term, got, want)
		}
	}
	if got := titles(glossaryLookup(notes, "casc")); got != "Cascade=a waterfall" {
		t.Errorf("casc: %s", got)
	}
	if got := glossaryLookup(notes, "xyz"); len(got) != 0 {
		t.Errorf("xyz: %s", titles(got))
	}
	if got := glossaryLookup(notes, ""); len(got) != 5 || got[0].DisplayTitle() != "ACAS" {
		t.Errorf("all: %s", titles(got))
	}
}

func TestGlossaryLookupMatchesTitleNotSlug(t *testing.T) {
	dir := t.TempDir()
	entry(t, dir, "C++", "a language", 1)
	entry(t, dir, "C#", "another", 2)
	entry(t, dir, ".NET", "a runtime", 3)
	notes, _ := note.List(dir)
	if got := titles(glossaryLookup(notes, "c++")); got != "C++=a language" {
		t.Errorf("c++: %s", got)
	}
	if got := titles(glossaryLookup(notes, ".net")); got != ".NET=a runtime" {
		t.Errorf(".net: %s", got)
	}
	if got := titles(glossaryLookup(notes, "net")); got != "" {
		t.Errorf("net matched the slug: %s", got)
	}
}

// An entry from before titles were kept has only its slug, which is then its term.
func TestGlossaryLookupOlderEntryBySlug(t *testing.T) {
	dir := t.TempDir()
	path := note.FilePath(dir, time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local), "cas")
	if err := note.WriteRaw(path, "---\ntags: [glossary]\n---\ncompare-and-swap\n"); err != nil {
		t.Fatal(err)
	}
	notes, _ := note.List(dir)
	if got := titles(glossaryLookup(notes, "CAS")); got != "cas=compare-and-swap" {
		t.Errorf("got %s", got)
	}
}

func TestGlossaryLookupLeavesOutSuperseded(t *testing.T) {
	dir := t.TempDir()
	old := entry(t, dir, "CAS", "wrong", 1)
	path := uniquePath(dir, time.Date(2026, 10, 2, 9, 0, 0, 0, time.Local), "cas")
	fm := note.BuildFrontmatter("CAS", []string{glossaryTag}, []string{note.SupersedesPrefix + strings.ToLower(old.ID())}, nil, nil)
	if err := note.WriteRaw(path, fm+"right\n"); err != nil {
		t.Fatal(err)
	}
	notes, _ := note.List(dir)
	if got := titles(glossaryLookup(notes, "cas")); got != "CAS=right" {
		t.Errorf("got %s", got)
	}
}

func TestGlossaryLines(t *testing.T) {
	dir := t.TempDir()
	a := entry(t, dir, "CAS", "compare-and-swap,\n  an atomic\n\nCPU instruction", 2)
	b := entry(t, dir, "Größe", "size", 3)
	got := glossaryLines([]note.Note{a, b})
	want := "CAS    compare-and-swap, an atomic CPU instruction  (2026-10-02)\n" +
		"Größe  size  (2026-10-03)\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestAddNoteWritesGlossaryEntry(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{NotesDir: dir, AttachmentsDir: t.TempDir()}
	n, err := addNote(cfg, "CAS", "compare-and-swap, an atomic CPU instruction for lock-free updates", []string{glossaryTag}, nil, nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(n.Path)
	want := "---\ntitle: CAS\ntags: [glossary]\n---\ncompare-and-swap, an atomic CPU instruction for lock-free updates\n"
	if string(raw) != want {
		t.Errorf("file\n%s\nwant\n%s", raw, want)
	}
	if n.Slug != "cas" || n.Title != "CAS" {
		t.Errorf("slug %q title %q", n.Slug, n.Title)
	}
	again, err := addNote(cfg, "CAS", "Central Authentication Service", []string{glossaryTag}, nil, nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	notes, _ := note.List(dir)
	if again.Path == n.Path || len(glossaryLookup(notes, "cas")) != 2 {
		t.Errorf("second meaning: %s, %d entries", again.Path, len(glossaryLookup(notes, "cas")))
	}
}
