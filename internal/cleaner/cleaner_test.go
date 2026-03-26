package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestApplyRemovesDeadLinkWithinBase(t *testing.T) {
	// 默认 clean 允许删除仓库内 dead target 的失效链接.
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

	result, err := Apply([]string{root}, baseDir, false, false, ApplyOptions{})
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
	// force=false 时, 仓库外 dead target 必须被保守跳过.
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

	result, err := Apply([]string{root}, baseDir, false, false, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Skipped, 1; got != want {
		t.Fatalf("Result.Skipped = %d, want %d", got, want)
	}
}

func TestApplyRejectsSymlinkRoot(t *testing.T) {
	// symlink 作为 clean root 属于高风险扫描根, 默认直接失败.
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

	result, err := Apply([]string{rootLink}, baseDir, false, false, ApplyOptions{})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
}

func TestApplyForceRemovesDeadLinkOutsideBase(t *testing.T) {
	// clean.force=true 只放宽“仓库外 dead target”这一条边界.
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

	result, err := Apply([]string{root}, baseDir, true, false, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Deleted, 1; got != want {
		t.Fatalf("Result.Deleted = %d, want %d", got, want)
	}
}

func TestApplyDryRunMarksRiskyCleanConfirmation(t *testing.T) {
	// dry-run 要把高风险 clean root 明确标记成需要确认.
	t.Parallel()

	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]string{root}, baseDir, false, false, ApplyOptions{DryRun: true, ProtectedRoots: []string{root}})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Entries[0].Message, "risky clean, confirmation required"; got != want {
		t.Fatalf("Result.Entries[0].Message = %q, want %q", got, want)
	}
}

func TestApplyAllowsSymlinkRootWithOverride(t *testing.T) {
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
	deadTarget := filepath.Join(baseDir, "missing.txt")
	deadLink := filepath.Join(realRoot, "dead-link")
	if err := os.Symlink(deadTarget, deadLink); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]string{rootLink}, baseDir, false, false, ApplyOptions{AllowRiskyClean: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Deleted, 1; got != want {
		t.Fatalf("Result.Deleted = %d, want %d", got, want)
	}
	if _, err := os.Lstat(deadLink); !os.IsNotExist(err) {
		t.Fatalf("dead link still exists, err = %v", err)
	}
}

func TestApplyCheckFailsWhenDeadLinkParentIsNotWritable(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	root := filepath.Join(baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	deadLink := filepath.Join(root, "dead-link")
	if err := os.Symlink(filepath.Join(baseDir, "missing.txt"), deadLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(root, 0o755)
	})

	result, err := Apply([]string{root}, baseDir, false, false, ApplyOptions{Check: true})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 2; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[1].Status, output.StatusFailed; got != want {
		t.Fatalf("Result.Entries[1].Status = %q, want %q", got, want)
	}
}
