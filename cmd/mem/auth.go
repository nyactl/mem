package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"mem/internal/auth"
	"mem/internal/config"

	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage the token mem sends to the server",
	Long: `The token is read from ` + auth.EnvVar + ` if set, otherwise from the system
keychain. It is never written to a configuration file.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store the server's token in the system keychain",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		fmt.Fprint(os.Stderr, "Token for the mem server: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		token := strings.TrimSpace(scanner.Text())
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Verify before storing, so a typo surfaces now rather than at the
		// next sync. Without a server configured there is nothing to ask.
		cfg := config.Load()
		if cfg.ServerURL != "" {
			switch code, err := probe(cfg.ServerURL, token); {
			case err != nil:
				fmt.Fprintf(os.Stderr, "warning: could not reach %s (%v) — storing anyway\n", cfg.ServerURL, err)
			case code == http.StatusUnauthorized || code == http.StatusForbidden:
				return fmt.Errorf("%s rejected this token (%d) — not stored", cfg.ServerURL, code)
			case code != http.StatusOK:
				fmt.Fprintf(os.Stderr, "warning: %s answered %d — storing anyway\n", cfg.ServerURL, code)
			}
		}

		if err := auth.Save(token); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "token stored in the system keychain")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the stored token from the system keychain",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		switch err := auth.Delete(); {
		case errors.Is(err, keyring.ErrNotFound):
			fmt.Fprintln(os.Stderr, "no token stored")
		case err != nil:
			return err
		default:
			fmt.Fprintln(os.Stderr, "token removed")
		}
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report where the token comes from and whether the server accepts it",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg := config.Load()
		token, src := auth.Token()

		fmt.Printf("%-8s %s\n", "token", src)
		if cfg.ServerURL == "" {
			fmt.Printf("%-8s not set — add server_url to the config file\n", "server")
			return nil
		}
		fmt.Printf("%-8s %s\n", "server", cfg.ServerURL)

		code, err := probe(cfg.ServerURL, token)
		switch {
		case err != nil:
			fmt.Printf("%-8s unreachable: %v\n", "check", err)
		case code == http.StatusOK:
			fmt.Printf("%-8s ok\n", "check")
		case code == http.StatusUnauthorized || code == http.StatusForbidden:
			if token == "" {
				fmt.Printf("%-8s rejected (%d) — run: mem auth login (or set %s)\n", "check", code, auth.EnvVar)
			} else {
				fmt.Printf("%-8s rejected (%d) — the token does not match the server's\n", "check", code)
			}
		default:
			fmt.Printf("%-8s server answered %d\n", "check", code)
		}
		return nil
	},
}

// probe asks the server for the note index and reports the status code only,
// which is all that is needed to tell a good token from a bad one.
func probe(base, token string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/api/notes", nil)
	if err != nil {
		return 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	root.AddCommand(authCmd)
}
