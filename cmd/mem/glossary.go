package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

// glossaryTag marks a glossary entry: its title is the term, its body the meaning.
const glossaryTag = "glossary"

var glossaryCmd = &cobra.Command{
	Use:     "g [<term>]",
	Aliases: []string{"glossary"},
	Short:   "Look up a term in the glossary",
	Long: `Looks up the glossary entries for a term, case-insensitively: exact matches first, then
the terms starting with it, one line each with its meaning and date. Without a term, lists
every entry.

  mem g cas
  mem g
  mem g add CAS "compare-and-swap, an atomic CPU instruction for lock-free updates"

A glossary entry is a note tagged glossary whose title is the term and whose body is the
meaning; a second meaning of a term is a second entry.`,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		notes, err := note.List(config.Load().NotesDir)
		if err != nil {
			return err
		}
		term := strings.Join(args, " ")
		entries := glossaryLookup(notes, term)
		if len(entries) == 0 {
			if term == "" {
				fmt.Fprintln(os.Stderr, "the glossary is empty")
			} else {
				fmt.Fprintf(os.Stderr, "no glossary entry for %s\n", term)
			}
			return nil
		}
		fmt.Print(glossaryLines(entries))
		return nil
	},
}

var glossaryAddCmd = &cobra.Command{
	Use:   "add <term> <meaning>",
	Short: "Add a term and its meaning to the glossary",
	Long: `Adds a glossary entry: a note titled with the term, tagged glossary, the meaning as its body.
The same as mem new --title <term> -t glossary <meaning>. Words after the term are the meaning.`,
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		term := note.CleanTitle(args[0])
		if term == "" {
			return fmt.Errorf("the term cannot be empty")
		}
		n, err := addNote(config.Load(), term, strings.Join(args[1:], " "), []string{glossaryTag}, nil, nil, "", nil)
		if err != nil {
			return err
		}
		fmt.Printf("glossary: %s\n", n.DisplayTitle())
		return nil
	},
}

// glossaryLookup returns the glossary entries for term: titles equal to it, then titles starting
// with it, both ignoring case and each sorted by title and date. An empty term returns them all.
// Superseded entries are left out.
func glossaryLookup(notes []note.Note, term string) []note.Note {
	term = strings.ToLower(note.CleanTitle(term))
	var exact, prefix []note.Note
	for _, m := range note.Search(notes, note.Query{Tags: []string{glossaryTag}}) {
		title := strings.ToLower(m.DisplayTitle())
		switch {
		case title == term:
			exact = append(exact, m.Note)
		case strings.HasPrefix(title, term):
			prefix = append(prefix, m.Note)
		}
	}
	sortEntries(exact)
	sortEntries(prefix)
	return append(exact, prefix...)
}

func sortEntries(entries []note.Note) {
	sort.SliceStable(entries, func(i, j int) bool {
		ti, tj := strings.ToLower(entries[i].DisplayTitle()), strings.ToLower(entries[j].DisplayTitle())
		if ti != tj {
			return ti < tj
		}
		return entries[i].DisplayTime().Before(entries[j].DisplayTime())
	})
}

// glossaryLines prints each entry on one line: the term padded to the widest one, the meaning
// with its line breaks folded, the date.
func glossaryLines(entries []note.Note) string {
	width := 0
	for _, e := range entries {
		width = max(width, runewidth.StringWidth(e.DisplayTitle()))
	}
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(runewidth.FillRight(e.DisplayTitle(), width) + "  " + strings.Join(strings.Fields(e.Body), " ") +
			"  (" + e.DisplayTime().Format("2006-01-02") + ")\n")
	}
	return sb.String()
}

func init() {
	glossaryCmd.AddCommand(glossaryAddCmd)
	root.AddCommand(glossaryCmd)
}
