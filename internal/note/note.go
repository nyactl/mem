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
	Sources     []string
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
func Create(dir string, ts time.Time, slug string, tags []string, sources []string, attachments []string) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create notes dir: %w", err)
	}
	path := filepath.Join(dir, Filename(ts, slug))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("note %q already exists — use: mem edit %s", slug, slug)
	}
	content := buildFrontmatter(tags, sources, attachments) + "\n"
	return path, os.WriteFile(path, []byte(content), 0600)
}

// CreateDraft writes a placeholder note (no title yet) and returns its path.
// Call FinalizeNote after the editor closes to derive slug and inline metadata.
func CreateDraft(dir string, ts time.Time) (string, error) {
	return Create(dir, ts, "draft", nil, nil, nil)
}

// FinalizeNote reads a note after editing, extracts inline #tags and @sources
// from the body, merges them with frontmatter, and renames the file if the
// first # Heading provides a better slug than the current one.
// Returns the (possibly new) path.
func FinalizeNote(path string, extraTags []string, extraSources []string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return path, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		_ = os.Remove(path)
		return "", nil
	}

	tags, sources, attachments, body := parseFrontmatter(string(data))

	tags = mergeTags(tags, extraTags)
	tags = mergeTags(tags, extractInlineTags(body))
	sources = mergeSources(sources, extraSources)
	sources = mergeSources(sources, extractInlineSources(body))

	newSlug := extractHeading(body)
	_, currentSlug, _ := parseFilename(filepath.Base(path))

	targetSlug := currentSlug
	if newSlug != "" {
		targetSlug = newSlug
	}

	newContent := buildFrontmatter(tags, sources, attachments) + body
	if err := os.WriteFile(path, []byte(newContent), 0600); err != nil {
		return path, err
	}

	if targetSlug != currentSlug && targetSlug != "" {
		ts, _, _ := parseFilename(filepath.Base(path))
		newPath := filepath.Join(filepath.Dir(path), Filename(ts, targetSlug))
		if err := os.Rename(path, newPath); err != nil {
			return path, err
		}
		return newPath, nil
	}
	return path, nil
}

var reInlineTag = regexp.MustCompile(`(?:^|[^#\w])#([a-z][a-z0-9-]*)`)

var reInlineSource = regexp.MustCompile(`@([a-z][a-z0-9-]*)`)

var reHeading = regexp.MustCompile(`(?m)^#\s+(.+)$`)

func extractInlineTags(body string) []string {
	matches := reInlineTag.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		t := strings.TrimSpace(m[1])
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

func extractInlineSources(body string) []string {
	matches := reInlineSource.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		s := m[1]
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func extractHeading(body string) string {
	m := reHeading.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return Slugify(m[1])
}

func mergeTags(existing, extra []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(existing)+len(extra))
	for _, t := range append(existing, extra...) {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

func mergeSources(existing, extra []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(existing)+len(extra))
	for _, s := range append(existing, extra...) {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
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
	tags, sources, attachments, body := parseFrontmatter(string(data))
	return Note{
		Path:        path,
		Slug:        slug,
		Created:     ts,
		Tags:        tags,
		Sources:     sources,
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
	tags, sources, existing, body := parseFrontmatter(string(data))
	merged := append(existing, newAttachments...)
	content := buildFrontmatter(tags, sources, merged) + body + "\n"
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
		for _, s := range n.Sources {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				out = append(out, s)
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

func buildFrontmatter(tags []string, sources []string, attachments []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	if len(tags) > 0 {
		sb.WriteString("tags: [" + strings.Join(tags, ", ") + "]\n")
	} else {
		sb.WriteString("tags: []\n")
	}
	if len(sources) > 0 {
		sb.WriteString("sources: [" + strings.Join(sources, ", ") + "]\n")
	}
	if len(attachments) > 0 {
		sb.WriteString("attachments: [" + strings.Join(attachments, ", ") + "]\n")
	}
	sb.WriteString("---\n")
	return sb.String()
}

func parseFrontmatter(content string) (tags []string, sources []string, attachments []string, body string) {
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
			} else if strings.HasPrefix(line, "sources:") {
				sources = parseInlineList(line)
			} else if strings.HasPrefix(line, "source:") {
				// backwards compat: migrate single source to list
				s := Slugify(strings.TrimSpace(strings.TrimPrefix(line, "source:")))
				if s != "" {
					sources = []string{s}
				}
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
