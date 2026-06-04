package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"mem-cli/internal/note"
)

type Index struct {
	DirMtime   int64          `json:"dir_mtime"`
	Tags       []string       `json:"tags"`
	From       []string       `json:"from"`
	TotalNotes int            `json:"total_notes"`
	TagCounts  map[string]int `json:"tag_counts"`
	FromCounts map[string]int `json:"from_counts"`
}

func indexPath(notesDir string) string {
	return filepath.Join(notesDir, ".mem-index.json")
}

// Load returns a fresh index, rebuilding from notes if the cached index is
// missing or the notes directory has been modified since last build.
func Load(notesDir string) (Index, error) {
	info, err := os.Stat(notesDir)
	if os.IsNotExist(err) {
		return Index{}, nil
	}
	if err != nil {
		return Index{}, err
	}
	dirMtime := info.ModTime().UnixNano()

	data, err := os.ReadFile(indexPath(notesDir))
	if err == nil {
		var idx Index
		if json.Unmarshal(data, &idx) == nil && idx.DirMtime == dirMtime {
			return idx, nil
		}
	}

	return Rebuild(notesDir)
}

// Rebuild scans all notes and writes a fresh index to disk.
func Rebuild(notesDir string) (Index, error) {
	info, err := os.Stat(notesDir)
	if os.IsNotExist(err) {
		return Index{}, nil
	}
	if err != nil {
		return Index{}, err
	}

	notes, err := note.List(notesDir)
	if err != nil {
		return Index{}, err
	}

	tagCounts := make(map[string]int)
	fromCounts := make(map[string]int)

	for _, n := range notes {
		for _, t := range n.Tags {
			tagCounts[t]++
		}
		for _, f := range n.From {
			fromCounts[f]++
		}
	}

	var tags []string
	for t := range tagCounts {
		tags = append(tags, t)
	}
	var froms []string
	for f := range fromCounts {
		froms = append(froms, f)
	}

	sort.Strings(tags)
	sort.Strings(froms)

	idx := Index{
		DirMtime:   info.ModTime().UnixNano(),
		Tags:       tags,
		From:       froms,
		TotalNotes: len(notes),
		TagCounts:  tagCounts,
		FromCounts: fromCounts,
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return idx, err
	}
	_ = os.WriteFile(indexPath(notesDir), data, 0600)
	return idx, nil
}
