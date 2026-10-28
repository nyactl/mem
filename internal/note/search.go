package note

import (
	"strings"
	"time"
)

// SupersedesPrefix marks a source naming an older note this one replaces by its ID, lowercased
// like every source: "supersedes-20260810t090000-kafka-rebalance". IDs, unlike slugs, are unique.
// A source syncs like any other, so the older note itself is never touched.
const SupersedesPrefix = "supersedes-"

// Query selects notes: every term must appear (case-insensitive) in the title, slug, tags, sources
// or body, every tag must be on the note, and its display time must fall within Since and Until.
type Query struct {
	Terms []string
	Tags  []string
	Since *time.Time
	Until *time.Time
	// All keeps notes a newer note supersedes.
	All bool
}

// Match is a note a query selected, with the body lines holding a term.
type Match struct {
	Note
	Lines        []string
	SupersededBy []string
}

// Search returns the notes a query selects, in the order given. Notes without the query's tags are
// skipped before their body is looked at.
func Search(notes []Note, q Query) []Match {
	supersededBy := map[string][]string{}
	for _, n := range notes {
		for _, s := range n.Sources {
			if old, ok := strings.CutPrefix(s, SupersedesPrefix); ok && old != "" {
				supersededBy[old] = append(supersededBy[old], n.ID())
			}
		}
	}
	terms := make([]string, 0, len(q.Terms))
	for _, t := range q.Terms {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			terms = append(terms, t)
		}
	}
	var out []Match
	for _, n := range notes {
		if !hasTags(n, q.Tags) || !within(n.DisplayTime(), q.Since, q.Until) {
			continue
		}
		key := strings.ToLower(n.ID())
		if len(supersededBy[key]) > 0 && !q.All {
			continue
		}
		text := strings.ToLower(n.Title + "\n" + n.Slug + "\n" + strings.Join(n.Tags, " ") + "\n" + strings.Join(n.Sources, " ") + "\n" + n.Body)
		found := true
		for _, t := range terms {
			found = found && strings.Contains(text, t)
		}
		if !found {
			continue
		}
		m := Match{Note: n, SupersededBy: supersededBy[key]}
		for _, l := range strings.Split(n.Body, "\n") {
			low := strings.ToLower(l)
			for _, t := range terms {
				if strings.Contains(low, t) {
					m.Lines = append(m.Lines, strings.TrimSpace(l))
					break
				}
			}
		}
		out = append(out, m)
	}
	return out
}

func hasTags(n Note, want []string) bool {
	for _, w := range want {
		found := false
		for _, t := range n.Tags {
			found = found || strings.EqualFold(t, w)
		}
		if !found {
			return false
		}
	}
	return true
}

func within(t time.Time, since, until *time.Time) bool {
	return (since == nil || !t.Before(*since)) && (until == nil || t.Before(*until))
}

// ParseDateTime reads a datetime in the formats a note's date: field accepts, in local time.
func ParseDateTime(s string) (*time.Time, bool) {
	t := parseDateTime(s)
	return t, t != nil
}
