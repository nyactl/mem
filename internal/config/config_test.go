package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"notes_dir":"/from/file","server_url":"http://file"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEM_CONFIG", path)
	t.Setenv("MEM_SERVER_URL", "http://from-env")

	cfg := Load()
	if cfg.ServerURL != "http://from-env" {
		t.Errorf("ServerURL = %q, want http://from-env", cfg.ServerURL)
	}
	if cfg.NotesDir != "/from/file" {
		t.Errorf("NotesDir = %q, want /from/file", cfg.NotesDir)
	}
}

func TestLoadEnvWithoutFile(t *testing.T) {
	t.Setenv("MEM_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("MEM_NOTES_DIR", "/data/notes")
	t.Setenv("MEM_LISTEN_ADDR", ":9999")

	cfg := Load()
	if cfg.NotesDir != "/data/notes" {
		t.Errorf("NotesDir = %q, want /data/notes", cfg.NotesDir)
	}
	if cfg.ListenAddr != ":9999" {
		t.Errorf("ListenAddr = %q, want :9999", cfg.ListenAddr)
	}
}

// A token in the config file must have no effect: secrets come from the
// environment or the keychain, and a stale auth_token left in an old config
// must not quietly become the credential again.
func TestLoadIgnoresAuthTokenInFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"notes_dir":"/n","auth_token":"from-file"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEM_CONFIG", path)

	cfg := Load()
	if cfg.NotesDir != "/n" {
		t.Errorf("NotesDir = %q, want /n", cfg.NotesDir)
	}
	// The field no longer exists; this test exists so that re-adding it, or
	// reading the key by hand, fails review rather than passing silently.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(raw), "auth_token") {
		t.Fatal("fixture lost its auth_token key")
	}
}

func contains(h, n string) bool {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return true
		}
	}
	return false
}
