package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	// local store
	AttachmentBackend string `json:"attachment_backend"`
	NotesDir          string `json:"notes_dir"`
	AttachmentsDir    string `json:"attachments_dir"`

	// server mode (mem serve)
	ListenAddr string `json:"listen_addr"` // default ":4747"
	AuthToken  string `json:"auth_token"`  // bearer token; empty = no auth

	// client mode (mem sync)
	ServerURL string `json:"server_url"` // e.g. "http://192.168.1.10:4747"
}

// Load reads config from MEM_CONFIG env var path, then the default XDG path,
// filling in defaults for any missing fields.
func Load() Config {
	cfg := defaults()
	path := os.Getenv("MEM_CONFIG")
	if path == "" {
		path = configPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	// re-apply defaults for zero-value fields
	d := defaults()
	if cfg.NotesDir == "" {
		cfg.NotesDir = d.NotesDir
	}
	if cfg.AttachmentsDir == "" {
		cfg.AttachmentsDir = d.AttachmentsDir
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = d.ListenAddr
	}
	return cfg
}

func defaults() Config {
	home, _ := os.UserHomeDir()
	return Config{
		AttachmentBackend: "local",
		NotesDir:          filepath.Join(home, ".mem", "notes"),
		AttachmentsDir:    filepath.Join(home, ".mem", "attachments"),
		ListenAddr:        ":4747",
	}
}

func configPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "mem", "config.json")
}
