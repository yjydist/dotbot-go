package linker

import (
	"os"
	"path/filepath"
	"testing"

	"dotbot-go/internal/config"
)

func TestApplyCreatesSymlink(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(baseDir, "target.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source}}, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Linked, 1; got != want {
		t.Fatalf("Result.Linked = %d, want %d", got, want)
	}
	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if got, want := linkTarget, source; got != want {
		t.Fatalf("Readlink() = %q, want %q", got, want)
	}
}

func TestApplyForceReplacesFile(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(baseDir, "target.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source, Force: true}}, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Replaced, 1; got != want {
		t.Fatalf("Result.Replaced = %d, want %d", got, want)
	}
	if _, err := os.Readlink(target); err != nil {
		t.Fatalf("target is not symlink: %v", err)
	}
}

func TestApplyIgnoreMissingSkips(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	missing := filepath.Join(baseDir, "missing.txt")
	target := filepath.Join(baseDir, "target.txt")

	result, err := Apply([]config.LinkConfig{{Target: target, Source: missing, IgnoreMissing: true}}, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Skipped, 1; got != want {
		t.Fatalf("Result.Skipped = %d, want %d", got, want)
	}
}
