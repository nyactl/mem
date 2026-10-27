package render

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"mem/internal/note"
)

const wideGraph = `flowchart LR
    subgraph once["Once: indexing"]
        D["Docs"] --> C["Cut into chunks"] --> E1["Embedding model"] --> V[("Vector database")]
    end
    subgraph per["Per question"]
        Q["Question"] --> E2["Embedding"] --> S["Search nearest chunks"]
        S --> R["Reranker"] --> P["Question + top chunks"] --> M["Big model"] --> A["Answer with sources"]
    end
    V --> S`

const loopGraph = `flowchart LR
    A["Text"] --> B["Split into tokens"]
    B --> C["Model calculates<br/>with its parameters"]
    C --> D["Most likely<br/>next token"]
    D --> E["Append to text"]
    E -->|"not done"| C
    E -->|"done"| F["Answer"]`

func TestFitKeepsALayoutThatFits(t *testing.T) {
	s, ok := Fit("graph LR\nA --> B", 80)
	if !ok {
		t.Fatal("small graph not drawn")
	}
	if strings.Count(s, "\n") > 4 {
		t.Errorf("small LR graph drawn top to bottom:\n%s", s)
	}
}

func TestFitFlipsWideGraphs(t *testing.T) {
	for name, src := range map[string]string{"subgraphs": wideGraph, "loop": loopGraph} {
		t.Run(name, func(t *testing.T) {
			if _, ok := draw(src, 70); ok {
				t.Fatal("fixture fits left to right; it no longer tests the flip")
			}
			s, ok := Fit(src, 70)
			if !ok {
				t.Fatal("not drawn within 70 columns")
			}
			if w := Width(s); w > 70 {
				t.Errorf("width %d > 70", w)
			}
			for _, label := range []string{"Docs", "Answer with sources", "Answer"} {
				if name == "subgraphs" && !strings.Contains(s, label) {
					t.Errorf("label %q lost:\n%s", label, s)
				}
			}
		})
	}
}

func TestFitGivesUpWhenNothingFits(t *testing.T) {
	if _, ok := Fit(loopGraph, 8); ok {
		t.Error("drawn into 8 columns")
	}
}

func TestTopDown(t *testing.T) {
	got, ok := topDown("%% c\nflowchart LR\n  subgraph a[\"A\"]\n    direction LR\n    x --> y\n  end\n  y --> z")
	if !ok {
		t.Fatal("not flipped")
	}
	want := "flowchart TD\n    x --> y\n  y --> z"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	for _, src := range []string{"graph TD\nA-->B", "sequenceDiagram\nA->>B: hi", "pie\n\"a\": 1"} {
		if _, ok := topDown(src); ok {
			t.Errorf("flipped %q", src)
		}
	}
}

func TestDiagramsReplacesOnlyMermaidBlocks(t *testing.T) {
	md := "intro\n\n```go\nx := 1\n```\n\n```mermaid\ngraph LR\nA --> B\n```\n\nafter"
	got := Diagrams(md, 80)
	if !strings.Contains(got, "```go\nx := 1\n```") {
		t.Errorf("code block changed:\n%s", got)
	}
	if strings.Contains(got, "```mermaid") || !strings.Contains(got, "┌") {
		t.Errorf("diagram not drawn:\n%s", got)
	}
	if !strings.HasPrefix(got, "intro\n") || !strings.HasSuffix(got, "\nafter") {
		t.Errorf("surrounding text changed:\n%s", got)
	}
}

func TestDiagramsKeepsSourceItCannotDraw(t *testing.T) {
	for name, md := range map[string]string{
		"too wide":    "```mermaid\n" + loopGraph + "\n```",
		"unknown":     "```mermaid\nnot a diagram ((( \n```",
		"unclosed":    "text\n```mermaid\ngraph LR\nA --> B",
		"tilde fence": "~~~mermaid\n" + loopGraph + "\n~~~",
	} {
		t.Run(name, func(t *testing.T) {
			got := Diagrams(md, 8)
			if name == "unclosed" {
				if got != md {
					t.Errorf("unclosed block changed:\n%s", got)
				}
				return
			}
			if !strings.Contains(got, "mermaid") || !strings.Contains(got, "web UI") {
				t.Errorf("source or hint missing:\n%s", got)
			}
		})
	}
}

func TestNoteFitsItsWidth(t *testing.T) {
	n := note.Note{
		Slug:    "llm-embeddings",
		Created: time.Date(2026, 10, 27, 8, 30, 0, 0, time.UTC),
		Tags:    []string{"ai", "kb"},
		Sources: []string{"agentdeck"},
		Body:    "Some text that is long enough to need wrapping in a narrow preview pane, more than once.\n\n```mermaid\n" + wideGraph + "\n```\n\n```sh\nmem ls --tag kb\n```",
	}
	for _, width := range []int{60, 73, 120} {
		out, err := Note(n, width)
		if err != nil {
			t.Fatal(err)
		}
		plain := stripANSI(out)
		if w := Width(plain); w > width {
			t.Errorf("width %d: output is %d columns:\n%s", width, w, plain)
		}
		for _, want := range []string{"llm-embeddings", "2026-10-27 08:30", "#ai #kb", "@agentdeck", "Vector database", "mem ls --tag kb"} {
			if !strings.Contains(plain, want) {
				t.Errorf("width %d: %q missing:\n%s", width, want, plain)
			}
		}
	}
}

var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string { return reANSI.ReplaceAllString(s, "") }

func TestFitSurvivesSpacingsThatPanic(t *testing.T) {
	// The loop's edge back to an earlier node panics mermaid-ascii at one row between nodes.
	if _, err := drawOnce(loopGraph, spacings[0]); err == nil {
		t.Fatal("fixture no longer panics at the first spacing; it no longer tests the fallback")
	}
	s, ok := Fit(loopGraph, 200)
	if !ok {
		t.Fatal("not drawn")
	}
	if strings.Count(s, "\n") > 10 {
		t.Errorf("drawn top to bottom although it fits left to right:\n%s", s)
	}
}

// A diagram nearly as wide as the terminal does not fit: glamour wraps a code block that reaches
// its margins, which breaks the drawing.
func TestNoteLeavesRoomForCodeMargins(t *testing.T) {
	src := "graph LR\nA[\"first step\"] --> B[\"second step\"]"
	lr, ok := Fit(src, 1000)
	if !ok {
		t.Fatal("not drawn")
	}
	width := Width(lr) + 4
	out, err := Note(note.Note{Slug: "s", Body: "```mermaid\n" + src + "\n```"}, width)
	if err != nil {
		t.Fatal(err)
	}
	plain := stripANSI(out)
	if !strings.Contains(plain, "second step") {
		t.Errorf("drawing wrapped in a terminal of %d:\n%s", width, plain)
	}
}
