package main

import (
	"fmt"
	"sort"
	"strings"

	"ref-cli/internal/backend"
	"ref-cli/internal/config"
	"ref-cli/internal/note"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:               "tags",
	Short:             "List tags with note counts",
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		counts, err := note.TagsFromNotes(cfg.NotesDir)
		if err != nil {
			return err
		}

		// Merge backend tags (count 0 for tags with no notes yet).
		if backendTags, err := backend.Tags(cfg.TagBackend); err == nil {
			for _, t := range backendTags {
				if _, ok := counts[t]; !ok {
					counts[t] = 0
				}
			}
		}

		if len(counts) == 0 {
			fmt.Println("no tags found")
			return nil
		}

		type entry struct {
			name  string
			count int
		}
		entries := make([]entry, 0, len(counts))
		for name, count := range counts {
			entries = append(entries, entry{name, count})
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].count != entries[j].count {
				return entries[i].count > entries[j].count
			}
			return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
		})

		for _, e := range entries {
			fmt.Printf("%-30s %d\n", e.name, e.count)
		}
		return nil
	},
}

func init() {
	root.AddCommand(tagsCmd)
}
