package main

import (
	"fmt"

	"mem-cli/internal/config"
	"mem-cli/internal/index"

	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:               "index",
	Short:             "Rebuild the note index",
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		if _, err := index.Rebuild(cfg.NotesDir); err != nil {
			return err
		}
		fmt.Println("index rebuilt")
		return nil
	},
}

func init() {
	root.AddCommand(indexCmd)
}
