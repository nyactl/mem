package main

import (
	"fmt"

	"mem/internal/config"
	"mem/internal/server"

	"github.com/spf13/cobra"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the mem server (web UI + REST sync API)",
	Long: `serve starts an HTTP server that:

  · hosts the web UI at http://<addr>/
  · exposes a REST API at /api/notes for sync clients
  · commits every write to git in the notes directory

The notes directory is initialised as a git repository on first run.

Configuration (config.json or MEM_CONFIG env; MEM_* variables override it):
  listen_addr  MEM_LISTEN_ADDR  server address         (default ":4747")
  auth_token   MEM_AUTH_TOKEN   bearer token for auth  (default: none — no auth)
  notes_dir    MEM_NOTES_DIR    notes directory        (default ~/.mem/notes)

Run behind a reverse proxy with TLS for remote access.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		addr := serveAddr
		if addr == "" {
			addr = cfg.ListenAddr
		}
		fmt.Printf("mem serve  %s\n", addr)
		fmt.Printf("notes dir  %s\n", cfg.NotesDir)
		if cfg.AuthToken != "" {
			fmt.Println("auth       bearer token (set)")
		} else {
			fmt.Println("auth       none (set auth_token in config to enable)")
		}
		fmt.Println()

		ctx := cmd.Context()
		srv := server.New(cfg.NotesDir, cfg.AuthToken)
		return srv.StartContext(ctx, addr)
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveAddr, "addr", "a", "", "listen address (overrides config listen_addr)")
	root.AddCommand(serveCmd)
}
