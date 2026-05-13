package backend

import (
	"os/exec"
	"strings"
)

// Tags calls the configured command and returns tag names.
// Handles both "name" and "id\tname" output formats.
// Returns nil (no error) if the backend is unavailable.
func Tags(command string) ([]string, error) {
	if command == "" {
		return nil, nil
	}
	parts := strings.Fields(command)
	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		return nil, err
	}
	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// Take the last tab-separated field so both "name" and "id\tname" work.
		fields := strings.SplitN(line, "\t", 2)
		name := strings.TrimSpace(fields[len(fields)-1])
		if name != "" {
			tags = append(tags, name)
		}
	}
	return tags, nil
}
