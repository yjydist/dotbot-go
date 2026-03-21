package linker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

func TestApplyCreatesSymlink(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(baseDir, "target.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source}}, ApplyOptions{})
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

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source, Force: true}}, ApplyOptions{})
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

	result, err := Apply([]config.LinkConfig{{Target: target, Source: missing, IgnoreMissing: true}}, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Skipped, 1; got != want {
		t.Fatalf("Result.Skipped = %d, want %d", got, want)
	}
}

func TestApplyDryRunDetectsExistingTargetConflictWithCreate(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	source := filepath.Join(baseDir, "source.txt")
	targetDir := filepath.Join(baseDir, "nested")
	target := filepath.Join(targetDir, "target.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source, Create: true}}, ApplyOptions{DryRun: true})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Status, output.StatusFailed; got != want {
		t.Fatalf("Result.Entries[0].Status = %q, want %q", got, want)
	}
}

func TestApplyForceRejectsProtectedTarget(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(baseDir, "target-dir")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source, Force: true}}, ApplyOptions{
		ProtectedTargets: []string{target},
	})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Status, output.StatusFailed; got != want {
		t.Fatalf("Result.Entries[0].Status = %q, want %q", got, want)
	}
	if _, statErr := os.Stat(target); statErr != nil {
		t.Fatalf("Stat(%q) error = %v, want protected target kept", target, statErr)
	}
}
