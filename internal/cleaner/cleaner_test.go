package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yjydist/dotbot-go/internal/config"
)

func TestApplyRemovesDeadLinkWithinBase(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	deadTarget := filepath.Join(baseDir, "missing.txt")
	linkPath := filepath.Join(root, "dead-link")
	if err := os.Symlink(deadTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(config.Config{
		BaseDir: baseDir,
		Clean:   config.CleanConfig{Paths: []string{root}},
	}, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Deleted, 1; got != want {
		t.Fatalf("Result.Deleted = %d, want %d", got, want)
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link still exists, err = %v", err)
	}
}

func TestApplySkipsDeadLinkOutsideBaseWithoutForce(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	root := filepath.Join(baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	deadTarget := filepath.Join(outsideDir, "missing.txt")
	linkPath := filepath.Join(root, "dead-link")
	if err := os.Symlink(deadTarget, linkPath); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(config.Config{
		BaseDir: baseDir,
		Clean:   config.CleanConfig{Paths: []string{root}},
	}, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Skipped, 1; got != want {
		t.Fatalf("Result.Skipped = %d, want %d", got, want)
	}
}

func TestApplyRejectsSymlinkRoot(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	realRoot := filepath.Join(baseDir, "real-root")
	rootLink := filepath.Join(baseDir, "root-link")
	if err := os.MkdirAll(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(config.Config{
		BaseDir: baseDir,
		Clean:   config.CleanConfig{Paths: []string{rootLink}},
	}, false)
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
}
