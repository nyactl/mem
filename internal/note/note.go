package note

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const tsFormat = "20060102T150405"

type Note struct {
	Path        string
	Slug        string
	Created     time.Time
	Tags        []string
	Source      string
	Attachments []string
	Body        string
}

func (n Note) CreatedStr() string {
	return n.Created.Format("2006-01-02")
}

// Slugify converts a title to a filename slug.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// Filename returns the note filename for the given timestamp and slug.
func Filename(ts time.Time, slug string) string {
	return ts.Format(tsFormat) + "-" + slug + ".md"
}

// Create writes a new note file with frontmatter and returns its path.
func Create(dir string, ts time.Time, slug string, tags []string, source string, attachments []string) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create notes dir: %w", err)
	}
	path := filepath.Join(dir, Filename(ts, slug))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("note %q already exists — use: mem edit %s", slug, slug)
	}
	content := buildFrontmatter(tags, source, attachments) + "\n"
	return path, os.WriteFile(path, []byte(content), 0600)
}

// Parse reads and parses a note file.
func Parse(path string) (Note, error) {
	ts, slug, err := parseFilename(filepath.Base(path))
	if err != nil {
		return Note{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Note{}, err
	}
	tags, source, attachments, body := parseFrontmatter(string(data))
	return Note{
		Path:        path,
		Slug:        slug,
		Created:     ts,
		Tags:        tags,
		Source:      source,
		Attachments: attachments,
		Body:        body,
	}, nil
}

// List returns all notes in dir sorted newest first.
func List(dir string) ([]Note, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read notes dir: %w", err)
	}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		n, err := Parse(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Created.After(notes[j].Created)
	})
	return notes, nil
}

// FindBySlug finds a note by slug, ignoring the timestamp prefix.
func FindBySlug(dir, slug string) (Note, error) {
	slug = strings.TrimSuffix(slug, ".md")
	notes, err := List(dir)
	if err != nil {
		return Note{}, err
	}
	for _, n := range notes {
		if n.Slug == slug {
			return n, nil
		}
	}
	return Note{}, fmt.Errorf("note %q not found", slug)
}

// UpdateAttachments appends new attachment paths to a note's frontmatter.
func UpdateAttachments(path string, newAttachments []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tags, source, existing, body := parseFrontmatter(string(data))
	merged := append(existing, newAttachments...)
	content := buildFrontmatter(tags, source, merged) + body + "\n"
	return os.WriteFile(path, []byte(content), 0600)
}

// SourcesFromNotes returns all unique non-empty source values across all notes.
func SourcesFromNotes(dir string) ([]string, error) {
	notes, err := List(dir)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var out []string
	for _, n := range notes {
		if n.Source != "" {
			if _, ok := seen[n.Source]; !ok {
				seen[n.Source] = struct{}{}
				out = append(out, n.Source)
			}
		}
	}
	return out, nil
}

// TagsFromNotes returns all unique tags across all notes with their counts.
func TagsFromNotes(dir string) (map[string]int, error) {
	notes, err := List(dir)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, n := range notes {
		for _, t := range n.Tags {
			counts[t]++
		}
	}
	return counts, nil
}

func parseFilename(base string) (time.Time, string, error) {
	base = strings.TrimSuffix(base, ".md")
	if len(base) < 16 || base[15] != '-' {
		return time.Time{}, "", fmt.Errorf("invalid note filename: %q", base)
	}
	ts, err := time.ParseInLocation(tsFormat, base[:15], time.Local)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse timestamp in %q: %w", base, err)
	}
	return ts, base[16:], nil
}

func buildFrontmatter(tags []string, source string, attachments []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	if len(tags) > 0 {
		sb.WriteString("tags: [" + strings.Join(tags, ", ") + "]\n")
	} else {
		sb.WriteString("tags: []\n")
	}
	if source != "" {
		sb.WriteString("source: " + source + "\n")
	}
	if len(attachments) > 0 {
		sb.WriteString("attachments: [" + strings.Join(attachments, ", ") + "]\n")
	}
	sb.WriteString("---\n")
	return sb.String()
}

func parseFrontmatter(content string) (tags []string, source string, attachments []string, body string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFM, first, afterFM := false, true, false
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if first {
			first = false
			if line == "---" {
				inFM = true
				continue
			}
			bodyLines = append(bodyLines, line)
			afterFM = true
			continue
		}
		if inFM && line == "---" {
			inFM = false
			afterFM = true
			continue
		}
		if inFM {
			if strings.HasPrefix(line, "tags:") {
				tags = parseInlineList(line)
			} else if strings.HasPrefix(line, "source:") {
				source = strings.TrimSpace(strings.TrimPrefix(line, "source:"))
			} else if strings.HasPrefix(line, "attachments:") {
				attachments = parseInlineList(line)
			}
		} else if afterFM {
			bodyLines = append(bodyLines, line)
		}
	}
	body = strings.Join(bodyLines, "\n")
	return
}

func parseInlineList(line string) []string {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start < 0 || end <= start {
		return nil
	}
	inner := strings.TrimSpace(line[start+1 : end])
	if inner == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(inner, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
