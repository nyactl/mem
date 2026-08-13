package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Init initialises a git repo in dir. No-op if already initialised.
func Init(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	cmd := exec.Command("git", "-C", dir, "init", "--quiet")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w: %s", err, out)
	}
	return nil
}

// IsRepo reports whether dir is inside a git repository.
func IsRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--git-dir")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// EnsureAuthor sets a local git identity if none is configured. Used in
// environments (containers, CI) that have no global git config.
func EnsureAuthor(dir string) {
	out, err := exec.Command("git", "-C", dir, "config", "user.name").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return
	}
	_ = exec.Command("git", "-C", dir, "config", "user.name", "mem").Run()
	_ = exec.Command("git", "-C", dir, "config", "user.email", "mem@localhost").Run()
}

// Commit stages all changes in dir and creates a commit with message.
// Returns nil (no error) when there is nothing to commit.
func Commit(dir, message string) error {
	add := exec.Command("git", "-C", dir, "add", "-A")
	if out, err := add.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, out)
	}
	// exit 0 means nothing staged → nothing to commit
	diff := exec.Command("git", "-C", dir, "diff", "--cached", "--quiet")
	if diff.Run() == nil {
		return nil
	}
	commit := exec.Command("git", "-C", dir, "commit", "--quiet", "-m", message)
	if out, err := commit.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, out)
	}
	return nil
}

// Log returns the last n commit messages in dir.
func Log(dir string, n int) ([]string, error) {
	out, err := exec.Command(
		"git", "-C", dir, "log",
		fmt.Sprintf("--%d", n),
		"--pretty=format:%h %s",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}
