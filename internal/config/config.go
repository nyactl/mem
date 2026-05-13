package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	TagBackend        string `json:"tag_backend"`
	AttachmentBackend string `json:"attachment_backend"`
	NotesDir          string `json:"notes_dir"`
	AttachmentsDir    string `json:"attachments_dir"`
}

// Load reads config from XDG_CONFIG_HOME/mem/config.json, filling in defaults.
func Load() Config {
	cfg := defaults()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	if cfg.TagBackend == "" {
		cfg.TagBackend = defaults().TagBackend
	}
	if cfg.NotesDir == "" {
		cfg.NotesDir = defaults().NotesDir
	}
	if cfg.AttachmentsDir == "" {
		cfg.AttachmentsDir = defaults().AttachmentsDir
	}
	return cfg
}

func defaults() Config {
	home, _ := os.UserHomeDir()
	return Config{
		TagBackend:        "todoist-cli labels",
		AttachmentBackend: "local",
		NotesDir:          filepath.Join(home, ".mem", "notes"),
		AttachmentsDir:    filepath.Join(home, ".mem", "attachments"),
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
