package main

import (
	"fmt"
	"os"
	"os/exec"

	"mem-cli/internal/config"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:               "search <query>",
	Short:             "Full-text search notes via ripgrep",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		rg, err := exec.LookPath("rg")
		if err != nil {
			return fmt.Errorf("ripgrep (rg) not found in PATH")
		}

		rgArgs := []string{
			"--color=always",
			"--heading",
			"--line-number",
		}
		rgArgs = append(rgArgs, args...)
		rgArgs = append(rgArgs, cfg.NotesDir)

		c := exec.Command(rg, rgArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
				fmt.Fprintln(os.Stderr, "no matches found")
				return nil
			}
			return err
		}
		return nil
	},
}

func init() {
	root.AddCommand(searchCmd)
}
