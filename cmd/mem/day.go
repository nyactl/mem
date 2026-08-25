package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"mem/internal/config"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var dayCmd = &cobra.Command{
	Use:   "day [YYYY-MM-DD]",
	Short: "List all notes from a given day (default today)",
	Long: `Prints every note from the given day in chronological order — oldest first,
as a timeline. Useful for reviewing what happened during a day.

  mem day            # today
  mem day 2026-08-14 # specific date`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		var target time.Time
		if len(args) == 0 {
			target = time.Now()
		} else {
			var err error
			target, err = time.ParseInLocation("2006-01-02", args[0], time.Local)
			if err != nil {
				return fmt.Errorf("date must be YYYY-MM-DD, got %q", args[0])
			}
		}
		dateStr := target.Format("2006-01-02")

		notes, err := note.List(cfg.NotesDir)
		if err != nil {
			return err
		}

		// filter by DisplayTime so date: overrides are respected
		var day []note.Note
		for _, n := range notes {
			if n.DisplayTime().Format("2006-01-02") == dateStr {
				day = append(day, n)
			}
		}

		if len(day) == 0 {
			fmt.Fprintf(os.Stderr, "no notes for %s\n", dateStr)
			return nil
		}

		// sort chronologically; use Created as tiebreaker for same DisplayTime
		sort.Slice(day, func(i, j int) bool {
			di, dj := day[i].DisplayTime(), day[j].DisplayTime()
			if di.Equal(dj) {
				return day[i].Created.Before(day[j].Created)
			}
			return di.Before(dj)
		})

		fmt.Printf("── %s ─────────────────────────────────────\n", dateStr)
		for _, n := range day {
			dt := n.DisplayTime()
			var timeStr string
			if n.Date != nil && dt.Hour() == 0 && dt.Minute() == 0 && dt.Second() == 0 {
				timeStr = "--:--   "
			} else {
				timeStr = dt.Format("15:04:05")
			}
			tags := ""
			if len(n.Tags) > 0 {
				tags = "  #" + joinStr(n.Tags, " #")
			}
			firstLine := firstNonEmpty(n.Body)
			fmt.Printf("%s  %-32s %s%s\n", timeStr, firstLine, n.Slug, tags)
		}
		return nil
	},
}

func joinStr(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// firstNonEmpty returns the first non-blank line of body text, trimmed.
func firstNonEmpty(body string) string {
	for _, line := range splitLines(body) {
		if t := trimLine(line); t != "" && t != "---" {
			if len(t) > 32 {
				t = t[:29] + "…"
			}
			return t
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimLine(s string) string {
	// strip leading # markdown heading markers and spaces
	i := 0
	for i < len(s) && (s[i] == '#' || s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func init() {
	root.AddCommand(dayCmd)
}
