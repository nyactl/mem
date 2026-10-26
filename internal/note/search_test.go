package note

import (
	"strings"
	"testing"
	"time"
)

func day(s string) *time.Time {
	t, _ := ParseDateTime(s)
	return t
}

func TestSearch(t *testing.T) {
	notes := []Note{
		{Slug: "parser-port", Created: *day("2026-08-20"), Tags: []string{"kb", "ai"}, Sources: []string{"topic-ab12"},
			Body: "Ported the parser.\nThe new grammar needs Go 1.23.\nunrelated line"},
		{Slug: "kafka-rebalance", Created: *day("2026-07-01"), Tags: []string{"kb"}, Body: "Rebalance blocks all partitions for 2 min."},
		{Slug: "kafka-rebalance", Created: *day("2026-09-01"), Tags: []string{"kb"}, Sources: []string{"supersedes-20260701t000000-kafka-rebalance"},
			Body: "Cooperative rebalance: about 30 s."},
		{Slug: "private", Created: *day("2026-08-21"), Tags: []string{"diary"}, Body: "The parser of my thoughts."},
	}
	slugs := func(ms []Match) string {
		var s []string
		for _, m := range ms {
			s = append(s, m.Slug)
		}
		return strings.Join(s, " ")
	}
	for _, c := range []struct {
		name string
		q    Query
		want string
	}{
		{"all terms, any case", Query{Terms: []string{"PARSER", "grammar"}}, "parser-port"},
		{"terms in sources and tags count", Query{Terms: []string{"topic-ab12"}}, "parser-port"},
		{"tag keeps others out", Query{Terms: []string{"parser"}, Tags: []string{"kb"}}, "parser-port"},
		{"every tag", Query{Tags: []string{"kb", "ai"}}, "parser-port"},
		{"tags ignore case", Query{Tags: []string{"KB"}, Terms: []string{"cooperative"}}, "kafka-rebalance"},
		{"superseded left out, the same slug kept", Query{Terms: []string{"rebalance"}}, "kafka-rebalance"},
		{"superseded with all", Query{Terms: []string{"rebalance"}, All: true}, "kafka-rebalance kafka-rebalance"},
		{"since", Query{Tags: []string{"kb"}, Since: day("2026-08-21"), All: true}, "kafka-rebalance"},
		{"until", Query{Tags: []string{"kb"}, Until: day("2026-08-20"), All: true}, "kafka-rebalance"},
		{"window", Query{Since: day("2026-08-01"), Until: day("2026-08-31")}, "parser-port private"},
		{"no match", Query{Terms: []string{"parser", "kafka"}}, ""},
	} {
		if got := slugs(Search(notes, c.q)); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	m := Search(notes, Query{Terms: []string{"parser", "grammar"}})[0]
	if strings.Join(m.Lines, "|") != "Ported the parser.|The new grammar needs Go 1.23." {
		t.Errorf("lines: %q", m.Lines)
	}
	m2 := Search(notes, Query{Terms: []string{"partitions"}, All: true})
	if len(m2) != 1 || strings.Join(m2[0].SupersededBy, " ") != "20260901T000000-kafka-rebalance" {
		t.Errorf("superseded by: %+v", m2)
	}
}

func TestSlugifySpellsOutLetters(t *testing.T) {
	for in, want := range map[string]string{
		"Flüge nach Köln":     "fluege-nach-koeln",
		"Straße & Größe":      "strasse-groesse",
		"Café Déjà vu, Ångst": "cafe-deja-vu-angst",
		"ÄÖÜ":                 "aeoeue",
		"plain-ascii 42":      "plain-ascii-42",
	} {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
