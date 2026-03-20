package creator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyCreatesDirectories(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	paths := []string{
		filepath.Join(baseDir, "one"),
		filepath.Join(baseDir, "nested", "two"),
	}

	result, err := Apply(paths, 0o755, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Created, 2; got != want {
		t.Fatalf("Result.Created = %d, want %d", got, want)
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", path)
		}
	}
}
