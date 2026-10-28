package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"mem/internal/index"
	"mem/internal/note"
)

// supersedeNote edits a copy of n's body and saves it as a new note that supersedes n, with n's
// tags, sources and attachments. n itself is not touched. It returns the new note, or a zero Note
// when the body was left unchanged or emptied.
func supersedeNote(notesDir string, n note.Note) (note.Note, error) {
	body, err := editBody(n.Body)
	if err != nil {
		return note.Note{}, err
	}
	if body == "" || body == strings.TrimRight(n.Body, "\n \t") {
		fmt.Fprintln(os.Stderr, "unchanged, nothing superseded")
		return note.Note{}, nil
	}

	var sources []string
	for _, s := range n.Sources {
		if !strings.HasPrefix(s, note.SupersedesPrefix) {
			sources = append(sources, s)
		}
	}
	sources = append(sources, note.SupersedesPrefix+strings.ToLower(n.ID()))
	tags := note.MergeTags(n.Tags, note.ExtractInlineTags(body))
	sources = note.MergeSources(sources, note.ExtractInlineSources(body))

	path := uniquePath(notesDir, time.Now(), n.Slug)
	if err := note.WriteRaw(path, note.BuildFrontmatter(n.Title, tags, sources, n.Attachments, nil)+body+"\n"); err != nil {
		return note.Note{}, err
	}
	_, _ = index.Rebuild(notesDir)
	nn, err := note.Parse(path)
	if err != nil {
		return note.Note{}, err
	}
	fmt.Fprintf(os.Stderr, "%s supersedes %s\n", nn.ID(), n.ID())
	return nn, nil
}
