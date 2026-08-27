package main

import (
	"fmt"
	"os"
	"sort"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var relatedCmd = &cobra.Command{
	Use:               "related [<slug>]",
	Short:             "Show notes sharing tags with a given note (fzf picker if no slug given)",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: slugCompleter,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		notes, err := note.List(cfg.NotesDir)
		if err != nil {
			return err
		}

		var target note.Note
		if len(args) == 0 {
			target, err = pickNote(notes)
			if err != nil || target.Path == "" {
				return err
			}
		} else {
			target, err = note.FindBySlug(cfg.NotesDir, args[0])
			if err != nil {
				return err
			}
		}

		if len(target.Tags) == 0 {
			fmt.Fprintf(os.Stderr, "%s has no tags — nothing to relate on\n", target.Slug)
			return nil
		}

		tagSet := make(map[string]struct{}, len(target.Tags))
		for _, t := range target.Tags {
			tagSet[t] = struct{}{}
		}

		type scored struct {
			n     note.Note
			score int
		}
		var candidates []scored
		for _, n := range notes {
			if n.Slug == target.Slug {
				continue
			}
			shared := 0
			for _, t := range n.Tags {
				if _, ok := tagSet[t]; ok {
					shared++
				}
			}
			if shared > 0 {
				candidates = append(candidates, scored{n, shared})
			}
		}

		if len(candidates) == 0 {
			fmt.Fprintf(os.Stderr, "no related notes found for %s\n", target.Slug)
			return nil
		}

		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].score != candidates[j].score {
				return candidates[i].score > candidates[j].score
			}
			return candidates[i].n.DisplayTime().After(candidates[j].n.DisplayTime())
		})

		related := make([]note.Note, len(candidates))
		for i, c := range candidates {
			related[i] = c.n
		}

		n, err := pickNote(related)
		if err != nil || n.Path == "" {
			return err
		}
		return viewFile(n.Path)
	},
}

func init() {
	root.AddCommand(relatedCmd)
}
