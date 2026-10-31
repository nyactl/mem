package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mem/internal/config"
	"mem/internal/index"
	"mem/internal/note"

	"github.com/spf13/cobra"
)

var doctorFix bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Find notes with a malformed name or frontmatter",
	Long: `Lists notes with problems, one per line, and exits non-zero when it finds any:
a file name that is not <YYYYMMDDTHHMMSS>-<slug>.md, a frontmatter that is never
closed, unknown keys, tags or sources that are not an inline list of slugs, a
date that cannot be read, a superseded note or an attachment that does not exist.

mem sync does not push a note with a problem in its own file.

--fix repairs only what is safe: it closes an unclosed frontmatter after its
known keys, and turns tags into slugs without duplicates. Everything else is
left to you.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return doctor(config.Load().NotesDir, doctorFix, os.Stdout)
	},
}

func doctor(dir string, fix bool, out io.Writer) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	if fix {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		fixed := false
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			changes, err := note.Fix(filepath.Join(dir, e.Name()))
			if err != nil {
				return err
			}
			for _, c := range changes {
				fmt.Fprintf(out, "fixed %s: %s\n", e.Name(), c)
			}
			fixed = fixed || len(changes) > 0
		}
		if fixed {
			_, _ = index.Rebuild(dir)
		}
	}
	problems, err := note.CheckDir(dir)
	if err != nil {
		return err
	}
	for _, p := range problems {
		fmt.Fprintln(out, p)
	}
	if len(problems) > 0 {
		return fmt.Errorf("%d problem(s) found", len(problems))
	}
	return nil
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "repair the safe cases and show each change")
	root.AddCommand(doctorCmd)
}
