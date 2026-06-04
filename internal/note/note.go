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
	From        []string
	Attachments []string
}

func (n Note) CreatedStr() string {
	return n.Created.Format("2006-01-02")
}

// IsUnnamed returns true when the note has no slug (timestamp-only filename).
func (n Note) IsUnnamed() bool {
	return n.Slug == ""
}

// Identifier returns the slug for named notes, or the bare timestamp string for unnamed notes.
func (n Note) Identifier() string {
	if n.Slug != "" {
		return n.Slug
	}
	return n.Created.Format(tsFormat)
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
// If slug is empty, returns a timestamp-only filename.
func Filename(ts time.Time, slug string) string {
	if slug == "" {
		return ts.Format(tsFormat) + ".md"
	}
	return ts.Format(tsFormat) + "-" + slug + ".md"
}

// NewTemp creates an empty body-only temp file and returns its path.
func NewTemp() (string, error) {
	f, err := os.CreateTemp("", "mem-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()
	return f.Name(), nil
}

// BeginEdit extracts the body from path into a temp file for editing.
// Does NOT modify the original file.
func BeginEdit(path string) (tmpPath string, tags, from, attachments []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, nil, nil, err
	}
	tags, from, attachments, body := parseFrontmatter(string(data))

	f, err := os.CreateTemp("", "mem-*.md")
	if err != nil {
		return "", nil, nil, nil, fmt.Errorf("create temp file: %w", err)
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, nil, nil, err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", nil, nil, nil, err
	}
	return f.Name(), tags, from, attachments, nil
}

// CommitNew reads the temp file, finalizes the note, and writes it to dir.
// Returns "" if the body is empty and no metadata was provided (abandoned).
// Deletes the temp file.
func CommitNew(dir string, ts time.Time, slug, tmpPath string, extraTags, extraFrom, extraAttachments []string) (string, error) {
	defer os.Remove(tmpPath)

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}
	body := string(data)

	if strings.TrimSpace(body) == "" && len(extraTags) == 0 && len(extraFrom) == 0 && len(extraAttachments) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create notes dir: %w", err)
	}

	tags := dedup(extraTags, extractInlineTags(body))
	from := dedup(extraFrom, extractInlineFrom(body))
	attachments := dedup(extraAttachments)

	// Derive slug from heading if not provided.
	if slug == "" {
		slug = extractHeading(body)
	}

	path := filepath.Join(dir, Filename(ts, slug))
	content := buildFrontmatter(tags, from, attachments) + body
	if err := writeAtomic(path, content); err != nil {
		return "", err
	}
	return path, nil
}

// CommitEdit reads the temp file, finalizes the note, and writes it back atomically.
// Always saves (never abandons). Deletes the temp file.
func CommitEdit(originalPath, tmpPath string, preservedTags, preservedFrom, preservedAttachments []string) (string, error) {
	defer os.Remove(tmpPath)

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return originalPath, err
	}
	body := string(data)

	tags := dedup(preservedTags, extractInlineTags(body))
	from := dedup(preservedFrom, extractInlineFrom(body))
	attachments := dedup(preservedAttachments)

	content := buildFrontmatter(tags, from, attachments) + body
	if err := writeAtomic(originalPath, content); err != nil {
		return originalPath, err
	}

	return originalPath, nil
}

// Body returns the note body with frontmatter stripped.
func Body(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	_, _, _, body := parseFrontmatter(string(data))
	return body, nil
}

// UpdateAttachments appends new attachment paths to a note's frontmatter.
func UpdateAttachments(path string, newAttachments []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tags, from, existing, body := parseFrontmatter(string(data))
	merged := dedup(existing, newAttachments)
	content := buildFrontmatter(tags, from, merged) + body
	return writeAtomic(path, content)
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

// FromFromNotes returns all unique from values across all notes with their counts.
func FromFromNotes(dir string) (map[string]int, error) {
	notes, err := List(dir)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, n := range notes {
		for _, f := range n.From {
			counts[f]++
		}
	}
	return counts, nil
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
	tags, from, attachments, _ := parseFrontmatter(string(data))
	return Note{
		Path:        path,
		Slug:        slug,
		Created:     ts,
		Tags:        tags,
		From:        from,
		Attachments: attachments,
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
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.HasPrefix(e.Name(), ".") {
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

// FindByIdentifier finds a note by slug first, then by bare timestamp prefix.
func FindByIdentifier(dir, id string) (Note, error) {
	id = strings.TrimSuffix(id, ".md")
	notes, err := List(dir)
	if err != nil {
		return Note{}, err
	}
	// Try slug match first.
	for _, n := range notes {
		if n.Slug == id {
			return n, nil
		}
	}
	// Try timestamp prefix match.
	for _, n := range notes {
		if strings.HasPrefix(n.Created.Format(tsFormat), id) {
			return n, nil
		}
	}
	return Note{}, fmt.Errorf("note %q not found", id)
}

// FindBySlug is an alias for FindByIdentifier.
func FindBySlug(dir, slug string) (Note, error) {
	return FindByIdentifier(dir, slug)
}

// RenameSlug renames a note file and rewrites [[oldSlug]] → [[newSlug]] in all notes.
// Returns the list of modified file paths. Errors if newSlug already exists.
func RenameSlug(dir, oldSlug, newSlug string) ([]string, error) {
	old, err := FindByIdentifier(dir, oldSlug)
	if err != nil {
		return nil, err
	}

	ts, _, _ := parseFilename(filepath.Base(old.Path))
	newPath := filepath.Join(dir, Filename(ts, newSlug))
	if _, err := os.Stat(newPath); err == nil {
		return nil, fmt.Errorf("note %q already exists", newSlug)
	}

	// Rename the file.
	if err := os.Rename(old.Path, newPath); err != nil {
		return nil, fmt.Errorf("rename: %w", err)
	}

	// Rewrite [[oldSlug]] → [[newSlug]] in all notes.
	notes, err := List(dir)
	if err != nil {
		return nil, err
	}

	oldRef := "[[" + oldSlug + "]]"
	newRef := "[[" + newSlug + "]]"
	var modified []string

	for _, n := range notes {
		data, err := os.ReadFile(n.Path)
		if err != nil {
			continue
		}
		content := string(data)
		if !strings.Contains(content, oldRef) {
			continue
		}
		newContent := strings.ReplaceAll(content, oldRef, newRef)
		if err := writeAtomic(n.Path, newContent); err != nil {
			return modified, err
		}
		modified = append(modified, n.Path)
	}

	return modified, nil
}

var reInlineTag = regexp.MustCompile(`(?:^|[^#\w])#([a-z][a-z0-9-]*)`)

var reInlineFrom = regexp.MustCompile(`@([a-z][a-z0-9-]*)`)

var reHeading = regexp.MustCompile(`(?m)^#\s+(.+)$`)

var reFencedCode = regexp.MustCompile("(?s)```.*?```")
var reIndentedCode = regexp.MustCompile(`(?m)^(    |\t).+$`)
var reInlineCode = regexp.MustCompile("`[^`]+`")
var reURL = regexp.MustCompile(`https?://\S+`)

// stripCodeAndURLs removes code blocks and URLs from body before tag/from extraction.
func stripCodeAndURLs(body string) string {
	s := reFencedCode.ReplaceAllString(body, "")
	s = reIndentedCode.ReplaceAllString(s, "")
	s = reInlineCode.ReplaceAllString(s, "")
	s = reURL.ReplaceAllString(s, "")
	return s
}

func extractInlineTags(body string) []string {
	safe := stripCodeAndURLs(body)
	matches := reInlineTag.FindAllStringSubmatch(safe, -1)
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

func extractInlineFrom(body string) []string {
	safe := stripCodeAndURLs(body)
	matches := reInlineFrom.FindAllStringSubmatch(safe, -1)
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

func dedup(slices ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, sl := range slices {
		for _, v := range sl {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}

func parseFilename(base string) (time.Time, string, error) {
	base = strings.TrimSuffix(base, ".md")
	if len(base) < 15 {
		return time.Time{}, "", fmt.Errorf("invalid note filename: %q", base)
	}
	ts, err := time.ParseInLocation(tsFormat, base[:15], time.Local)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse timestamp in %q: %w", base, err)
	}
	// Slug follows after the '-' separator; if none, slug is empty.
	if len(base) > 15 && base[15] == '-' {
		return ts, base[16:], nil
	}
	if len(base) == 15 {
		return ts, "", nil
	}
	return time.Time{}, "", fmt.Errorf("invalid note filename: %q", base)
}

func buildFrontmatter(tags []string, from []string, attachments []string) string {
	if len(tags) == 0 && len(from) == 0 && len(attachments) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	if len(tags) > 0 {
		sb.WriteString("tags: [" + strings.Join(tags, ", ") + "]\n")
	}
	if len(from) > 0 {
		sb.WriteString("from: [" + strings.Join(from, ", ") + "]\n")
	}
	if len(attachments) > 0 {
		sb.WriteString("attachments: [" + strings.Join(attachments, ", ") + "]\n")
	}
	sb.WriteString("---\n")
	return sb.String()
}

func parseFrontmatter(content string) (tags []string, from []string, attachments []string, body string) {
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
			} else if strings.HasPrefix(line, "from:") {
				from = parseInlineList(line)
			} else if strings.HasPrefix(line, "sources:") {
				// backwards compat
				from = parseInlineList(line)
			} else if strings.HasPrefix(line, "source:") {
				// backwards compat: single source
				s := Slugify(strings.TrimSpace(strings.TrimPrefix(line, "source:")))
				if s != "" {
					from = []string{s}
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

func writeAtomic(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
