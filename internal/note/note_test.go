package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A dated note must survive a write/read cycle with its day intact, and must
// serialise identically regardless of the writer's timezone — otherwise sync
// sees a difference that is not one and re-pushes the note forever.
func TestDateRoundTripsAcrossZones(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	day := time.Date(2026, 9, 21, 0, 0, 0, 0, berlin)

	fm := BuildFrontmatter("", []string{"x"}, nil, nil, &day)
	if want := "date: 2026-09-20T22:00:00Z\n"; !contains(fm, want) {
		t.Fatalf("frontmatter = %q, want it to contain %q", fm, want)
	}

	utc := day.UTC()
	if BuildFrontmatter("", []string{"x"}, nil, nil, &utc) != fm {
		t.Error("same instant in a different zone produced different bytes")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, Filename(time.Date(2026, 9, 21, 16, 49, 52, 0, time.Local), "dated"))
	if err := os.WriteFile(path, []byte(fm+"body\n"), 0600); err != nil {
		t.Fatal(err)
	}
	n, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := n.DisplayTime().In(berlin).Format("2006-01-02"); got != "2026-09-21" {
		t.Errorf("display day = %s, want 2026-09-21", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

func TestSetBodyKeepsFrontmatter(t *testing.T) {
	day := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), Filename(time.Date(2026, 10, 27, 8, 0, 0, 0, time.Local), "finding"))
	if err := WriteRaw(path, BuildFrontmatter("I/O", []string{"kb"}, []string{"agentdeck"}, []string{"a/b.png"}, &day)+"old body\n"); err != nil {
		t.Fatal(err)
	}
	if err := SetBody(path, "new body #extra @elsewhere\n\n"); err != nil {
		t.Fatal(err)
	}
	n, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if n.Date == nil || !n.Date.Equal(day) {
		t.Errorf("date = %v, want %v", n.Date, day)
	}
	if got := strings.Join(n.Tags, ","); got != "kb,extra" {
		t.Errorf("tags = %s", got)
	}
	if got := strings.Join(n.Sources, ","); got != "agentdeck,elsewhere" {
		t.Errorf("sources = %s", got)
	}
	if got := strings.Join(n.Attachments, ","); got != "a/b.png" {
		t.Errorf("attachments = %s", got)
	}
	if n.Title != "I/O" {
		t.Errorf("title = %q", n.Title)
	}
	raw, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(raw), "---\nnew body #extra @elsewhere\n") {
		t.Errorf("file = %q", raw)
	}
}

func TestTitleRoundTrips(t *testing.T) {
	dir := t.TempDir()
	for i, title := range []string{
		"CAS", "C++", ".NET", "I/O", "Größe", "Kafka: rebalance", "#1 rule", "C# vs F#", "a #b",
		"- dash", "[x]", "{y}", `say "hi"`, "it's", "'quoted'", "back\\slash", "true", "No", "null",
		"42", "1.5", "1_000", "ends:", "50% off", "@home", "*star", "&anchor", "!tag", "|pipe", ">gt",
		"`tick`", "?q", ",comma", "~", "tab\there",
	} {
		path := FilePath(dir, time.Date(2026, 10, 28, 10, 30, i, 0, time.Local), "t")
		if err := WriteRaw(path, BuildFrontmatter(title, []string{"glossary"}, nil, nil, nil)+"meaning\n"); err != nil {
			t.Fatal(err)
		}
		n, err := Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		want := CleanTitle(title)
		if n.Title != want || n.DisplayTitle() != want || n.Body != "meaning" || strings.Join(n.Tags, ",") != "glossary" {
			raw, _ := os.ReadFile(path)
			t.Errorf("%q read back as title %q body %q tags %v from\n%s", title, n.Title, n.Body, n.Tags, raw)
		}
	}
}

func TestTitleWrittenPlainWhereYAMLReadsItBack(t *testing.T) {
	for title, want := range map[string]string{
		"CAS":              "title: CAS\n",
		"C++":              "title: C++\n",
		"Größe":            "title: Größe\n",
		"I/O":              "title: I/O\n",
		"Kafka: rebalance": "title: \"Kafka: rebalance\"\n",
		"true":             "title: \"true\"\n",
		"42":               "title: \"42\"\n",
		"0x1F":             "title: \"0x1F\"\n",
		"12:30":            "title: \"12:30\"\n",
		"2026-10-28":       "title: \"2026-10-28\"\n",
		"1:1 meeting":      "title: 1:1 meeting\n",
		"#1 rule":          "title: \"#1 rule\"\n",
		"  two\n lines ":   "title: two lines\n",
	} {
		if got := BuildFrontmatter(title, nil, nil, nil, nil); got != "---\n"+want+"tags: []\n---\n" {
			t.Errorf("%q written as %q", title, got)
		}
	}
	if got := BuildFrontmatter("   ", nil, nil, nil, nil); strings.Contains(got, "title") {
		t.Errorf("blank title written: %q", got)
	}
}

func TestTitleReadFromQuotedYAML(t *testing.T) {
	dir := t.TempDir()
	for i, c := range []struct{ line, want string }{
		{`title: 'it''s: here'`, "it's: here"},
		{`title: "tab\there"`, "tab here"},
		{`title:   spaced   `, "spaced"},
		{`title:`, ""},
	} {
		path := FilePath(dir, time.Date(2026, 10, 28, 10, 30, i, 0, time.Local), "t")
		if err := WriteRaw(path, "---\n"+c.line+"\ntags: []\n---\nbody\n"); err != nil {
			t.Fatal(err)
		}
		n, _ := Parse(path)
		if n.Title != c.want {
			t.Errorf("%s read as %q, want %q", c.line, n.Title, c.want)
		}
	}
}

func TestNoteWithoutTitleShowsItsSlug(t *testing.T) {
	path := FilePath(t.TempDir(), time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local), "kafka-rebalance")
	if err := WriteRaw(path, "---\ntags: [kb]\n---\nbody\n"); err != nil {
		t.Fatal(err)
	}
	n, _ := Parse(path)
	if n.Title != "" || n.DisplayTitle() != "kafka-rebalance" {
		t.Errorf("title %q display %q", n.Title, n.DisplayTitle())
	}
	if err := SetDate(path, nil); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); strings.Contains(string(raw), "title") {
		t.Errorf("rewrite added a title:\n%s", raw)
	}
}

func TestTitleKeptBySetDateAndUpdateAttachments(t *testing.T) {
	path := FilePath(t.TempDir(), time.Date(2026, 10, 28, 10, 30, 0, 0, time.Local), "c")
	if err := WriteRaw(path, BuildFrontmatter("C++", nil, nil, nil, nil)+"body\n"); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := SetDate(path, &day); err != nil {
		t.Fatal(err)
	}
	if err := UpdateAttachments(path, []string{"a.png"}); err != nil {
		t.Fatal(err)
	}
	if n, _ := Parse(path); n.Title != "C++" || n.Date == nil || len(n.Attachments) != 1 {
		t.Errorf("title %q date %v attachments %v", n.Title, n.Date, n.Attachments)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	for _, c := range []struct{ in, fm, body string }{
		{"---\ntags: []\n---\nbody\n", "---\ntags: []\n---\n", "body\n"},
		{"---\ntags: []\n---\nbody", "---\ntags: []\n---\n", "body"},
		{"---\ntags: []\n---\n", "---\ntags: []\n---\n", ""},
		{"---\ntags: []\n---", "---\ntags: []\n---", ""},
		{"---\n---\nbody\n", "---\n---\n", "body\n"},
		{"---\ntags: []\nunclosed\n", "", "---\ntags: []\nunclosed\n"},
		{"no frontmatter\n", "", "no frontmatter\n"},
		{"---\na: b\n---\ntext\n---\nmore\n", "---\na: b\n---\n", "text\n---\nmore\n"},
	} {
		if fm, body := SplitFrontmatter(c.in); fm != c.fm || body != c.body {
			t.Errorf("SplitFrontmatter(%q) = %q, %q; want %q, %q", c.in, fm, body, c.fm, c.body)
		}
	}
}
