package note

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Problem is one thing wrong with a note file.
type Problem struct {
	Path string
	Msg  string
}

func (p Problem) String() string { return filepath.Base(p.Path) + ": " + p.Msg }

var knownKeys = map[string]bool{"title": true, "date": true, "tags": true, "sources": true, "source": true, "attachments": true}

var (
	reFilename = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}-[a-z0-9][a-z0-9-]*\.md$`)
	reSlug     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	reKey      = regexp.MustCompile(`^([A-Za-z0-9_-]+):`)
	reList     = regexp.MustCompile(`^\[[^\[\]]*\]$`)
)

// IsSlug reports whether s can stand as a tag or source: lower-case letters, digits and hyphens,
// starting with a letter or digit. Repeated hyphens are allowed; notes have tags like edit--d.
func IsSlug(s string) bool { return reSlug.MatchString(s) }

// Check lists what is wrong with one note file on its own: its name, and a frontmatter the reader
// would misread or partly ignore.
func Check(path string) ([]Problem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ps []Problem
	add := func(format string, a ...any) { ps = append(ps, Problem{path, fmt.Sprintf(format, a...)}) }

	base := filepath.Base(path)
	if !reFilename.MatchString(base) {
		add("file name is not <YYYYMMDDTHHMMSS>-<slug>.md")
	} else if _, err := time.ParseInLocation(tsFormat, base[:15], time.Local); err != nil {
		add("file name has no valid timestamp")
	}

	lines, closed := frontmatterLines(string(data))
	if lines == nil && !closed {
		return ps, nil
	}
	if !closed {
		add("frontmatter is opened with --- but never closed, so the whole note reads as frontmatter")
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := reKey.FindStringSubmatch(line)
		if !closed && (m == nil || !knownKeys[m[1]]) {
			break // the body begins here
		}
		if m == nil {
			add("frontmatter line %q is not a key", line)
			continue
		}
		key, val := m[1], strings.TrimSpace(line[len(m[0]):])
		if !knownKeys[key] {
			add("unknown key %q", key)
			continue
		}
		switch key {
		case "tags", "sources", "attachments":
			if !reList.MatchString(val) {
				add("%s: is not an inline list like [a, b]", key)
				continue
			}
			if key == "attachments" {
				continue
			}
			seen := map[string]bool{}
			for _, v := range parseInlineList(val) {
				if !IsSlug(v) {
					add("%s: %q is not a slug", key, v)
				}
				if seen[v] {
					add("%s: %q is listed twice", key, v)
				}
				seen[v] = true
			}
		case "date":
			if parseDateTime(val) == nil {
				add("date: %q cannot be read", val)
			}
		}
	}
	return ps, nil
}

// CheckDir checks every .md file in dir, and also what one note says about others: a superseded
// note that does not exist, an attachment that is missing.
func CheckDir(dir string) ([]Problem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ps []Problem
	ids := map[string]bool{}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		p, err := Check(path)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p...)
		if n, err := Parse(path); err == nil {
			ids[strings.ToLower(n.ID())] = true
			notes = append(notes, n)
		}
	}
	for _, n := range notes {
		for _, s := range n.Sources {
			if old, ok := strings.CutPrefix(s, SupersedesPrefix); ok && !ids[old] {
				ps = append(ps, Problem{n.Path, fmt.Sprintf("supersedes %s, which does not exist", old)})
			}
		}
		for _, a := range n.Attachments {
			if _, err := os.Stat(a); err != nil {
				ps = append(ps, Problem{n.Path, fmt.Sprintf("attachment %s is missing", a)})
			}
		}
	}
	return ps, nil
}

// Fix repairs what is safe to repair in a note: it closes an unclosed frontmatter after its last
// leading known key, and turns tags that are not slugs into slugs and drops duplicates. Every other line stays as it
// is. It returns a description of each change; none means the file was not touched.
func Fix(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return nil, nil
	}
	lines := strings.Split(content, "\n")
	var changes []string

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		last := 0
		for i := 1; i < len(lines); i++ {
			m := reKey.FindStringSubmatch(lines[i])
			if m == nil || !knownKeys[m[1]] {
				break
			}
			last = i
		}
		if last == 0 {
			return nil, nil
		}
		lines = append(lines[:last+1], append([]string{"---"}, lines[last+1:]...)...)
		end = last + 1
		changes = append(changes, fmt.Sprintf("closed the frontmatter after line %d", last+1))
	}

	for i := 1; i < end; i++ {
		val, ok := strings.CutPrefix(lines[i], "tags:")
		if !ok || !reList.MatchString(strings.TrimSpace(val)) {
			continue
		}
		var tags []string
		seen := map[string]bool{}
		for _, t := range parseInlineList(val) {
			s := t
			if !IsSlug(t) {
				s = Slugify(t)
			}
			if s != "" && !seen[s] {
				seen[s] = true
				tags = append(tags, s)
			}
		}
		fixed := "tags: [" + strings.Join(tags, ", ") + "]"
		if fixed != lines[i] {
			changes = append(changes, fmt.Sprintf("%s → %s", lines[i], fixed))
			lines[i] = fixed
		}
	}

	if len(changes) == 0 {
		return nil, nil
	}
	return changes, WriteRaw(path, strings.Join(lines, "\n"))
}

// LeftoverFrontmatter reports whether a body begins with what looks like a frontmatter: a ---
// line followed by a known key. A note whose frontmatter was never closed is sent that way.
func LeftoverFrontmatter(body string) bool {
	first, rest, ok := strings.Cut(body, "\n")
	if !ok || strings.TrimRight(first, "\r") != "---" {
		return false
	}
	m := reKey.FindStringSubmatch(rest)
	return m != nil && knownKeys[m[1]]
}

// frontmatterLines returns the lines between the opening and closing ---, and whether it was
// closed. A file that does not start with --- has no frontmatter: nil, false.
func frontmatterLines(content string) ([]string, bool) {
	if !strings.HasPrefix(content, "---\n") && content != "---" {
		return nil, false
	}
	lines := strings.Split(strings.TrimPrefix(content, "---\n"), "\n")
	for i, l := range lines {
		if l == "---" {
			return lines[:i], true
		}
	}
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines, false
}
