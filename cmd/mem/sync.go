package main

import (
	"fmt"
	"os"

	"mem/internal/config"
	"mem/internal/synclient"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync notes with the mem serve instance (pull then push)",
	Long: `Pulls new and changed notes from the server, then pushes local changes back.
Run pull or push individually to control direction.

Requires server_url in config (or MEM_CONFIG env var).`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg := config.Load()
		if cfg.ServerURL == "" {
			return fmt.Errorf("server_url is not set in config — add it to point at your mem serve instance")
		}
		c := synclient.New(cfg.ServerURL, cfg.AuthToken, cfg.NotesDir)

		fmt.Fprintf(os.Stderr, "pulling from %s …\n", cfg.ServerURL)
		created, updated, conflicts, err := c.Pull()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "pull: +%d new  ~%d updated  %d conflicts\n",
			created, updated, len(conflicts))
		for _, id := range conflicts {
			fmt.Fprintf(os.Stderr, "  conflict: %s (both sides changed — resolve manually)\n", id)
		}

		fmt.Fprintf(os.Stderr, "pushing …\n")
		pcreated, pupdated, pdeleted, pconflicts, err := c.Push()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "push: +%d new  ~%d updated  -%d deleted  %d conflicts\n",
			pcreated, pupdated, pdeleted, len(pconflicts))
		for _, id := range pconflicts {
			fmt.Fprintf(os.Stderr, "  conflict: %s (server modified concurrently — pull first)\n", id)
		}
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download new and changed notes from the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg := config.Load()
		if cfg.ServerURL == "" {
			return fmt.Errorf("server_url is not set in config — add it to point at your mem serve instance")
		}
		c := synclient.New(cfg.ServerURL, cfg.AuthToken, cfg.NotesDir)
		fmt.Fprintf(os.Stderr, "pulling from %s …\n", cfg.ServerURL)
		created, updated, conflicts, err := c.Pull()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "pull complete: +%d new  ~%d updated  %d conflicts\n",
			created, updated, len(conflicts))
		for _, id := range conflicts {
			fmt.Fprintf(os.Stderr, "  conflict: %s (both sides changed — resolve manually)\n", id)
		}
		return nil
	},
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local notes to the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg := config.Load()
		if cfg.ServerURL == "" {
			return fmt.Errorf("server_url is not set in config — add it to point at your mem serve instance")
		}
		c := synclient.New(cfg.ServerURL, cfg.AuthToken, cfg.NotesDir)
		fmt.Fprintf(os.Stderr, "pushing to %s …\n", cfg.ServerURL)
		created, updated, deleted, conflicts, err := c.Push()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "push complete: +%d new  ~%d updated  -%d deleted  %d conflicts\n",
			created, updated, deleted, len(conflicts))
		for _, id := range conflicts {
			fmt.Fprintf(os.Stderr, "  conflict: %s (server modified concurrently — pull first)\n", id)
		}
		return nil
	},
}

func init() {
	syncCmd.AddCommand(syncPullCmd, syncPushCmd)
	root.AddCommand(syncCmd)
}
