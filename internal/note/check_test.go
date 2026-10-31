package note

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// One note per problem, each found and named; a well-formed note, one with a title, a legacy
// source: line and a frontmatter-less note are not reported.
func TestCheckDirFindsEachProblem(t *testing.T) {
	dir := t.TempDir()
	att := writeFixture(t, t.TempDir(), "kept.png", "x")
	good := map[string]string{
		"20261028T100000-good.md":     "---\ntitle: \"C++: templates\"\ndate: 2026-10-28T09:00:00Z\ntags: [kb, go]\nsources: [supersedes-20261028t100001-old]\nattachments: [" + att + "]\n---\nbody\n",
		"20261028T100001-old.md":      "---\ntags: []\n---\nold\n",
		"20261028T100002-legacy.md":   "---\n# hand-written comment\ntags: [a]\nsource: Some Book\n\n---\nbody\n",
		"20261028T100003-plain.md":    "no frontmatter at all\n",
		"20261028T100004-dateonly.md": "---\ndate: 2026-10-28\ntags: []\n---\nb\n",
		"20261028T100005-flags.md":    "---\ntags: [edit--d, open---force, 1password]\n---\nb\n",
	}
	bad := map[string]string{
		"notes about kafka.md":          "---\ntags: []\n---\nb\n",
		"20261399T100000-badts.md":      "---\ntags: []\n---\nb\n",
		"20261028T110000-unclosed.md":   "---\ntags: [kb]\n\nthe body\nNote: date: is a word here\ndate: not a frontmatter line\n",
		"20261028T110001-unknown.md":    "---\ntags: []\nauthor: me\n---\nb\n",
		"20261028T110002-blocklist.md":  "---\ntags:\n  - kb\n---\nb\n",
		"20261028T110003-noslug.md":     "---\ntags: [Kafka Streams]\n---\nb\n",
		"20261028T110004-dup.md":        "---\ntags: [kb, kb]\n---\nb\n",
		"20261028T110005-badsrc.md":     "---\ntags: []\nsources: [Some Book]\n---\nb\n",
		"20261028T110006-baddate.md":    "---\ndate: last tuesday\ntags: []\n---\nb\n",
		"20261028T110007-supersedes.md": "---\ntags: []\nsources: [supersedes-20200101t000000-gone]\n---\nb\n",
		"20261028T110008-attach.md":     "---\ntags: []\nattachments: [/nonexistent/x.png]\n---\nb\n",
		"20261028T110009-srcnolist.md":  "---\ntags: []\nsources: a, b\n---\nb\n",
	}
	want := map[string]string{
		"notes about kafka.md":          "file name",
		"20261399T100000-badts.md":      "timestamp",
		"20261028T110000-unclosed.md":   "never closed",
		"20261028T110001-unknown.md":    `unknown key "author"`,
		"20261028T110002-blocklist.md":  "tags: is not an inline list",
		"20261028T110003-noslug.md":     `"Kafka Streams" is not a slug`,
		"20261028T110004-dup.md":        `"kb" is listed twice`,
		"20261028T110005-badsrc.md":     `sources: "Some Book" is not a slug`,
		"20261028T110006-baddate.md":    `"last tuesday" cannot be read`,
		"20261028T110007-supersedes.md": "supersedes 20200101t000000-gone, which does not exist",
		"20261028T110008-attach.md":     "attachment /nonexistent/x.png is missing",
		"20261028T110009-srcnolist.md":  "sources: is not an inline list",
	}
	for name, c := range good {
		writeFixture(t, dir, name, c)
	}
	for name, c := range bad {
		writeFixture(t, dir, name, c)
	}

	problems, err := CheckDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, p := range problems {
		got[filepath.Base(p.Path)] = append(got[filepath.Base(p.Path)], p.Msg)
	}
	for name, msg := range want {
		if !strings.Contains(strings.Join(got[name], "\n"), msg) {
			t.Errorf("%s: problems %q, want one with %q", name, got[name], msg)
		}
	}
	for name := range good {
		if len(got[name]) > 0 {
			t.Errorf("%s is fine, reported %q", name, got[name])
		}
	}
	if n := len(got["20261028T110000-unclosed.md"]); n != 1 {
		t.Errorf("unclosed note: %d problems %q, want only the unclosed one (the body is not frontmatter)", n, got["20261028T110000-unclosed.md"])
	}
}

// Fix repairs the safe cases, leaves every other byte alone and every other note untouched, and a
// second run changes nothing.
func TestFixRepairsOnlySafeCases(t *testing.T) {
	dir := t.TempDir()
	cases := []struct{ name, in, out string }{
		{"20261028T110000-unclosed.md", "---\ntitle: T\ntags: [kb]\nthe body\nsecond line\n", "---\ntitle: T\ntags: [kb]\n---\nthe body\nsecond line\n"},
		{"20261028T110001-tags.md", "---\ntags: [Kafka Streams, kb, KB, kafka-streams, set--e]\nsources: [x]\n---\nbody #kb\n", "---\ntags: [kafka-streams, kb, set--e]\nsources: [x]\n---\nbody #kb\n"},
		{"20261028T110002-both.md", "---\ntags: [Go, go]\nbody", "---\ntags: [go]\n---\nbody"},
	}
	untouched := map[string]string{
		"20261028T110003-unknown.md":   "---\ntags: []\nauthor: me\n---\nb\n",
		"20261028T110004-blocklist.md": "---\ntags:\n  - Kb\n---\nb\n",
		"20261028T110005-baddate.md":   "---\ndate: last tuesday\ntags: []\n---\nb\n",
		"20261028T110006-rule.md":      "---\njust a rule, not frontmatter\n",
		"20261028T110007-good.md":      "---\ntags: [kb]\n---\nb\n",
		"20261028T110008-badsrc.md":    "---\ntags: []\nsources: [Some Book]\n---\nb\n",
		"20261028T110009-flags.md":     "---\ntags: [edit--d, open---force]\n---\nb\n",
	}
	for _, c := range cases {
		writeFixture(t, dir, c.name, c.in)
	}
	for name, c := range untouched {
		writeFixture(t, dir, name, c)
	}

	for _, c := range cases {
		changes, err := Fix(filepath.Join(dir, c.name))
		if err != nil {
			t.Fatal(err)
		}
		if len(changes) == 0 {
			t.Errorf("%s: no changes", c.name)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, c.name)); string(got) != c.out {
			t.Errorf("%s fixed to\n%q\nwant\n%q", c.name, got, c.out)
		}
		if ps, _ := Check(filepath.Join(dir, c.name)); len(ps) > 0 {
			t.Errorf("%s still has %v", c.name, ps)
		}
	}
	for name, c := range untouched {
		if changes, err := Fix(filepath.Join(dir, name)); err != nil || len(changes) > 0 {
			t.Errorf("%s: changes %q, err %v", name, changes, err)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, name)); string(got) != c {
			t.Errorf("%s changed to %q", name, got)
		}
	}
	for _, c := range cases {
		if changes, _ := Fix(filepath.Join(dir, c.name)); len(changes) > 0 {
			t.Errorf("%s: second fix changed %q", c.name, changes)
		}
	}
}

// Every note mem writes itself passes the check.
func TestBuildFrontmatterPassesCheck(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	path := filepath.Join(dir, "20261028T100000-x.md")
	content := BuildFrontmatter("Yes: no", []string{"kb", "go-1"}, []string{"supersedes-20261028t100000-x"}, nil, &date) + "body\n"
	writeFixture(t, dir, "20261028T100000-x.md", content)
	if ps, err := Check(path); err != nil || len(ps) > 0 {
		t.Errorf("problems %v, err %v", ps, err)
	}
}

func TestLeftoverFrontmatter(t *testing.T) {
	for body, want := range map[string]bool{
		"---\ntags: [kb]\nbody":     true,
		"---\ntitle: x\n":           true,
		"---\nNote: a rule first\n": false,
		"--- \ntags: [kb]\n":        false,
		"text\n---\ntags: [kb]\n":   false,
		"---":                       false,
		"":                          false,
	} {
		if got := LeftoverFrontmatter(body); got != want {
			t.Errorf("LeftoverFrontmatter(%q) = %v", body, got)
		}
	}
}
