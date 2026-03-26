package fscheck

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func CheckWritableParent(path string) error {
	parent, err := nearestExistingDirectory(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("resolve parent for %s: %w", path, err)
	}
	if err := unix.Access(parent, unix.W_OK|unix.X_OK); err != nil {
		return fmt.Errorf("parent directory is not writable: %s: %w", parent, err)
	}
	return nil
}

func nearestExistingDirectory(path string) (string, error) {
	current := filepath.Clean(path)
	for {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("path is not a directory: %s", current)
			}
			return current, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		next := filepath.Dir(current)
		if next == current {
			return "", fmt.Errorf("path does not have an existing parent: %s", path)
		}
		current = next
	}
}
