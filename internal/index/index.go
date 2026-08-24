package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"mem/internal/note"
)

type Index struct {
	DirMtime int64    `json:"dir_mtime"`
	Tags     []string `json:"tags"`
	Sources  []string `json:"sources"`
}

func indexPath(notesDir string) string {
	return filepath.Join(notesDir, ".mem-index.json")
}

// Load returns a fresh index, rebuilding from notes if the cached index is
// missing or the notes directory has been modified since last build.
func Load(notesDir string) (Index, error) {
	info, err := os.Stat(notesDir)
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
	if err != nil {
		return Index{}, err
	}

	notes, err := note.List(notesDir)
	if err != nil {
		return Index{}, err
	}

	tagSeen := make(map[string]struct{})
	srcSeen := make(map[string]struct{})
	var tags, sources []string

	for _, n := range notes {
		for _, t := range n.Tags {
			if _, ok := tagSeen[t]; !ok {
				tagSeen[t] = struct{}{}
				tags = append(tags, t)
			}
		}
		for _, s := range n.Sources {
			if _, ok := srcSeen[s]; !ok {
				srcSeen[s] = struct{}{}
				sources = append(sources, s)
			}
		}
	}

	sort.Strings(tags)
	sort.Strings(sources)

	idx := Index{
		DirMtime: info.ModTime().UnixNano(),
		Tags:     tags,
		Sources:  sources,
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return idx, err
	}
	_ = os.WriteFile(indexPath(notesDir), data, 0600)
	return idx, nil
}
