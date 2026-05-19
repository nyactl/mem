package main

import (
	"fmt"
	"sort"
	"strings"

	"mem-cli/internal/config"
	"mem-cli/internal/note"

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
