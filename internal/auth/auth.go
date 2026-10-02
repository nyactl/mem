// Package auth resolves the bearer token mem presents to a mem serve instance.
// The token is never stored in a file: it comes from the environment, for
// containers and headless hosts, or from the system keychain.
package auth

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "mem"
	keyringUser    = "auth-token"
	EnvVar         = "MEM_AUTH_TOKEN"
)

// Source names where a token came from, so `mem auth status` can report it
// without printing the token itself.
type Source string

const (
	SourceEnv     Source = "MEM_AUTH_TOKEN"
	SourceKeyring Source = "system keychain"
	SourceNone    Source = "not configured"
)

// Token returns the token and where it came from. An empty token is not an
// error: a server started without auth_token accepts unauthenticated requests.
func Token() (string, Source) {
	if t := strings.TrimSpace(os.Getenv(EnvVar)); t != "" {
		return t, SourceEnv
	}
	if t, err := keyring.Get(keyringService, keyringUser); err == nil {
		if t = strings.TrimSpace(t); t != "" {
			return t, SourceKeyring
		}
	}
	return "", SourceNone
}

// Save stores the token in the system keychain.
func Save(token string) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("token cannot be empty")
	}
	if err := keyring.Set(keyringService, keyringUser, strings.TrimSpace(token)); err != nil {
		return fmt.Errorf("store token in keychain: %w", err)
	}
	return nil
}

// Delete removes the stored token.
func Delete() error {
	if runtime.GOOS == "darwin" {
		// go-keyring's delete silently fails for unsigned binaries on macOS;
		// the security CLI works, scoped to this exact service and account.
		out, err := exec.Command("security", "delete-generic-password",
			"-s", keyringService, "-a", keyringUser).CombinedOutput()
		if err != nil {
			if strings.Contains(string(out), "could not be found") {
				return keyring.ErrNotFound
			}
			return fmt.Errorf("delete token: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	return keyring.Delete(keyringService, keyringUser)
}
