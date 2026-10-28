// Package render shows a note in a terminal: markdown through glamour, mermaid diagrams drawn
// with box characters by mermaid-ascii, sized to the width they are given.
package render

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"mem/internal/note"

	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	mermaid "github.com/AlexanderGrooff/mermaid-ascii/pkg/render"
	"github.com/charmbracelet/glamour"
	"github.com/mattn/go-runewidth"
)

// codeMargin is what glamour keeps around a code block: the document's margin and the block's own
// on the left, the document's on the right. A wider line is wrapped, which breaks a drawing.
const codeMargin = 6

// drawTimeout bounds one diagram; the layout searches paths and a pathological graph must not hang
// a preview.
const drawTimeout = 2 * time.Second

// Note renders n for a terminal width columns wide. A preview, a glance to recognise the note by,
// gets a line naming each diagram instead of its drawing, which is slow to make and in a narrow
// pane taller than the pane.
func Note(n note.Note, width int, preview bool) (string, error) {
	var meta []string
	meta = append(meta, n.DisplayTime().Format("2006-01-02 15:04"))
	if len(n.Tags) > 0 {
		meta = append(meta, "#"+strings.Join(n.Tags, " #"))
	}
	if len(n.Sources) > 0 {
		meta = append(meta, "@"+strings.Join(n.Sources, " @"))
	}
	md := "# " + n.DisplayTitle() + "\n\n" + strings.Join(meta, " · ") + "\n\n" + Diagrams(n.Body, width-codeMargin, preview)
	return Markdown(md, width)
}

// Markdown renders md with the style named by GLAMOUR_STYLE, dark if unset. The style is never
// detected from the terminal: in an fzf preview the output is not one.
func Markdown(md string, width int) (string, error) {
	style := os.Getenv("GLAMOUR_STYLE")
	if style == "" {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStylePath(style), glamour.WithWordWrap(width))
	if err != nil {
		return "", err
	}
	return r.Render(md)
}

// Diagrams replaces each mermaid block in md by a code block holding its drawing, at most width
// columns wide. A diagram that cannot be drawn or does not fit stays as source, with a hint. With
// named, a block becomes one line naming its kind instead.
func Diagrams(md string, width int, named bool) string {
	lines := strings.Split(md, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		fence, ok := mermaidFence(lines[i])
		if !ok {
			out = append(out, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && strings.TrimSpace(lines[end]) != fence {
			end++
		}
		if end == len(lines) {
			out = append(out, lines[i:]...)
			break
		}
		src := strings.Join(lines[i+1:end], "\n")
		if named {
			out = append(out, "*▸ "+kind(src)+" · drawn in the full view*")
		} else if drawing, ok := Fit(src, width); ok {
			out = append(out, "```", drawing, "```")
		} else {
			out = append(out, lines[i:end+1]...)
			out = append(out, "", "*Diagram shown as source; the web UI draws it.*")
		}
		i = end
	}
	return strings.Join(out, "\n")
}

// kind names the diagram src declares on its first line.
func kind(src string) string {
	for _, l := range strings.Split(src, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		switch word := strings.Fields(t)[0]; word {
		case "graph", "flowchart":
			return "flowchart"
		case "sequenceDiagram":
			return "sequence diagram"
		case "erDiagram":
			return "ER diagram"
		default:
			return word + " diagram"
		}
	}
	return "diagram"
}

var reFence = regexp.MustCompile("^ {0,3}(```+|~~~+)\\s*mermaid\\s*$")

func mermaidFence(line string) (string, bool) {
	m := reFence.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// Fit draws src at most width columns wide: as written, then, for a left-to-right graph that is too
// wide, top to bottom without its subgraphs, which mermaid-ascii draws through the nodes in that
// layout.
func Fit(src string, width int) (string, bool) {
	if s, ok := draw(src, width); ok {
		return s, true
	}
	if flipped, ok := topDown(src); ok {
		return draw(flipped, width)
	}
	return "", false
}

var reHeader = regexp.MustCompile(`^(\s*(?:graph|flowchart)\s+)(LR|RL)\b`)

// topDown turns a left-to-right graph top to bottom and drops its subgraphs, keeping what is inside
// them. It reports false for anything else.
func topDown(src string) (string, bool) {
	lines := strings.Split(src, "\n")
	header := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		if reHeader.MatchString(l) {
			header = i
		}
		break
	}
	if header < 0 {
		return "", false
	}
	out := []string{reHeader.ReplaceAllString(lines[header], "${1}TD")}
	depth := 0
	for _, l := range lines[header+1:] {
		t := strings.TrimSpace(l)
		switch {
		case t == "subgraph" || strings.HasPrefix(t, "subgraph "):
			depth++
		case depth > 0 && t == "end":
			depth--
		case depth > 0 && strings.HasPrefix(t, "direction "):
		default:
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n"), true
}

// spacings are tried in order, each as {between nodes across, between nodes down, inside boxes}.
// mermaid-ascii panics on some graphs at some spacings: an edge back to an earlier node needs 2
// rows between nodes left to right, and 2 columns top to bottom. Its own fitting (MaxWidth) goes
// down to 1 and hits that, so it stays off.
var spacings = [][3]int{{2, 1, 0}, {2, 2, 0}}

// draw renders src compactly and reports whether it worked and fits width.
func draw(src string, width int) (string, bool) {
	for _, sp := range spacings {
		if s, err := drawOnce(src, sp); err == nil && s != "" {
			return s, Width(s) <= width
		}
	}
	return "", false
}

func drawOnce(src string, sp [3]int) (string, error) {
	type result struct {
		s   string
		err error
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("mermaid-ascii: %v", r)}
			}
		}()
		cfg := diagram.DefaultConfig()
		cfg.PaddingBetweenX, cfg.PaddingBetweenY, cfg.BoxBorderPadding = sp[0], sp[1], sp[2]
		s, err := mermaid.RenderDiagram(src, cfg)
		done <- result{strings.TrimRight(s, "\n"), err}
	}()
	select {
	case r := <-done:
		return r.s, r.err
	case <-time.After(drawTimeout):
		return "", fmt.Errorf("mermaid-ascii: no drawing after %s", drawTimeout)
	}
}

// Width is the widest line of s in terminal columns.
func Width(s string) int {
	w := 0
	for _, l := range strings.Split(s, "\n") {
		w = max(w, runewidth.StringWidth(strings.TrimRight(l, " ")))
	}
	return w
}
