package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var (
	searchTags  []string
	searchSince string
	searchUntil string
	searchJSON  bool
	searchAll   bool
)

// maxShownLines is how many matching lines a note shows in the plain output.
const maxShownLines = 3

var searchCmd = &cobra.Command{
	Use:   "search [<term>...]",
	Short: "Find notes holding all terms, optionally by tag and date",
	Long: `Finds the notes holding every term (case-insensitive) in their body, slug, tags or
sources, newest first, with the lines that match.

  mem search kafka rebalance
  mem search restore --tag kb --since 2026-08-01 --until 2026-09-15
  mem search --tag kb --since 2026-10-01 --json

--tag (repeatable) keeps notes carrying every given tag; no other note is shown.
--since and --until take a day or a datetime and compare the note's date (its date:
field, else when it was created); --until includes the whole day it names.
A note superseded by a newer one (a source supersedes-<slug>, see mem new --supersedes)
is left out unless --all.`,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && len(searchTags) == 0 && searchSince == "" && searchUntil == "" {
			return fmt.Errorf("give a term, a --tag or a date range")
		}
		q := note.Query{Terms: args, Tags: searchTags, All: searchAll}
		var err error
		if q.Since, err = searchDate("--since", searchSince, false); err != nil {
			return err
		}
		if q.Until, err = searchDate("--until", searchUntil, true); err != nil {
			return err
		}
		notes, err := note.List(config.Load().NotesDir)
		if err != nil {
			return err
		}
		matches := note.Search(notes, q)
		if searchJSON {
			return printSearchJSON(matches)
		}
		if len(matches) == 0 {
			fmt.Fprintln(os.Stderr, "no matches found")
			return nil
		}
		for _, m := range matches {
			line := m.DisplayTime().Format("2006-01-02") + "  " + m.Slug
			if len(m.Tags) > 0 {
				line += "  [" + strings.Join(m.Tags, ", ") + "]"
			}
			if len(m.SupersededBy) > 0 {
				line += "  superseded by " + strings.Join(m.SupersededBy, ", ")
			}
			fmt.Println(line)
			for i, l := range m.Lines {
				if i == maxShownLines {
					fmt.Printf("    ... %d more\n", len(m.Lines)-i)
					break
				}
				fmt.Println("    " + l)
			}
		}
		return nil
	},
}

// searchDate reads a --since or --until value; a bare day as --until means up to its end.
func searchDate(flag, v string, endOfDay bool) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, ok := note.ParseDateTime(v)
	if !ok {
		return nil, fmt.Errorf("%s %q: unrecognised datetime format", flag, v)
	}
	if endOfDay && len(v) == len("2006-01-02") {
		next := t.AddDate(0, 0, 1)
		t = &next
	}
	return t, nil
}

type searchResult struct {
	ID           string   `json:"id"`
	Slug         string   `json:"slug"`
	Path         string   `json:"path"`
	Date         string   `json:"date"`
	Tags         []string `json:"tags"`
	Sources      []string `json:"sources"`
	Lines        []string `json:"lines"`
	SupersededBy []string `json:"superseded_by,omitempty"`
	Body         string   `json:"body"`
}

func printSearchJSON(matches []note.Match) error {
	out := make([]searchResult, 0, len(matches))
	for _, m := range matches {
		out = append(out, searchResult{
			ID: m.ID(), Slug: m.Slug, Path: m.Path, Date: m.DisplayTime().Format(time.RFC3339),
			Tags: nonNil(m.Tags), Sources: nonNil(m.Sources), Lines: nonNil(m.Lines), SupersededBy: m.SupersededBy,
			Body: strings.TrimSpace(m.Body),
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func init() {
	searchCmd.Flags().StringArrayVarP(&searchTags, "tag", "t", nil, "only notes with this tag, repeatable")
	searchCmd.Flags().StringVar(&searchSince, "since", "", "only notes dated on or after, e.g. 2026-08-01")
	searchCmd.Flags().StringVar(&searchUntil, "until", "", "only notes dated up to, e.g. 2026-09-15 (the whole day)")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "print the matches as JSON")
	searchCmd.Flags().BoolVar(&searchAll, "all", false, "include notes a newer note supersedes")
	searchCmd.RegisterFlagCompletionFunc("tag", tagCompleter)
	root.AddCommand(searchCmd)
}
