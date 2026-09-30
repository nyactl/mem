package note

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A dated note must survive a write/read cycle with its day intact, and must
// serialise identically regardless of the writer's timezone — otherwise sync
// sees a difference that is not one and re-pushes the note forever.
func TestDateRoundTripsAcrossZones(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	day := time.Date(2026, 9, 21, 0, 0, 0, 0, berlin)

	fm := BuildFrontmatter([]string{"x"}, nil, nil, &day)
	if want := "date: 2026-09-20T22:00:00Z\n"; !contains(fm, want) {
		t.Fatalf("frontmatter = %q, want it to contain %q", fm, want)
	}

	utc := day.UTC()
	if BuildFrontmatter([]string{"x"}, nil, nil, &utc) != fm {
		t.Error("same instant in a different zone produced different bytes")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, Filename(time.Date(2026, 9, 21, 16, 49, 52, 0, time.Local), "dated"))
	if err := os.WriteFile(path, []byte(fm+"body\n"), 0600); err != nil {
		t.Fatal(err)
	}
	n, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := n.DisplayTime().In(berlin).Format("2006-01-02"); got != "2026-09-21" {
		t.Errorf("display day = %s, want 2026-09-21", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
