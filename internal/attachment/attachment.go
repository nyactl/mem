package attachment

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const tsFormat = "20060102T150405"

// Store copies src into dir, prefixed with the note's timestamp, and returns
// the absolute destination path.
func Store(src string, noteTimestamp time.Time, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create attachments dir: %w", err)
	}
	name := noteTimestamp.Format(tsFormat) + "-" + filepath.Base(src)
	dst := filepath.Join(dir, name)
	if err := copyFile(src, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
