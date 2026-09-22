package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"notes_dir":"/from/file","auth_token":"file-token"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEM_CONFIG", path)
	t.Setenv("MEM_AUTH_TOKEN", "env-token")

	cfg := Load()
	if cfg.AuthToken != "env-token" {
		t.Errorf("AuthToken = %q, want env-token", cfg.AuthToken)
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
