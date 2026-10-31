package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// doctor lists each problem and fails; --fix shows what it changed and fails only for what is
// left; a second --fix changes nothing; a clean or missing notes dir passes.
func TestDoctor(t *testing.T) {
	dir := t.TempDir()
	for name, c := range map[string]string{
		"20261028T100000-unclosed.md": "---\ntags: [Kb, kb]\nbody\n",
		"20261028T100001-unknown.md":  "---\ntags: []\nauthor: me\n---\nb\n",
		"20261028T100002-good.md":     "---\ntags: [kb]\n---\nb\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(c), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	if err := doctor(dir, false, &out); err == nil || !strings.Contains(err.Error(), "3 problem") {
		t.Errorf("doctor: %v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "good") || strings.Count(out.String(), "\n") != 3 {
		t.Errorf("doctor output:\n%s", out.String())
	}

	out.Reset()
	err := doctor(dir, true, &out)
	if err == nil || !strings.Contains(err.Error(), "1 problem") {
		t.Errorf("doctor --fix: %v\n%s", err, out.String())
	}
	for _, want := range []string{"fixed 20261028T100000-unclosed.md: closed the frontmatter", "tags: [Kb, kb] → tags: [kb]", `20261028T100001-unknown.md: unknown key "author"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("doctor --fix output lacks %q:\n%s", want, out.String())
		}
	}

	out.Reset()
	_ = doctor(dir, true, &out)
	if strings.Contains(out.String(), "fixed") {
		t.Errorf("second --fix changed:\n%s", out.String())
	}

	if err := os.Remove(filepath.Join(dir, "20261028T100001-unknown.md")); err != nil {
		t.Fatal(err)
	}
	if err := doctor(dir, false, &out); err != nil {
		t.Errorf("clean dir: %v", err)
	}
	if err := doctor(filepath.Join(dir, "missing"), false, &out); err != nil {
		t.Errorf("missing dir: %v", err)
	}
}
