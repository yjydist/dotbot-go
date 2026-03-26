package linker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

func TestApplyCreatesSymlink(t *testing.T) {
	// 基线用例: 缺失 target 时会创建 symlink.
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
	// force=true 时允许用新 symlink 覆盖已有普通文件.
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
	// ignore_missing=true 会把缺失 source 从失败降级成 skip.
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
	// dry-run 也必须暴露 target 冲突, 不能因为不写文件就静默通过.
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
	// 受保护目标即使 force=true 也不能绕过 runner 的确认模型.
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

func TestApplyDryRunMarksProtectedTargetConfirmation(t *testing.T) {
	// dry-run 要把危险覆盖明确标成“需要确认”, 方便用户预审.
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
		DryRun:           true,
		ProtectedTargets: []string{target},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Entries[0].Message, "protected target, confirmation required"; got != want {
		t.Fatalf("Result.Entries[0].Message = %q, want %q", got, want)
	}
}

func TestApplyRelinkReplacesExistingSymlink(t *testing.T) {
	// relink=true 的核心语义是替换现有 symlink 指向, 不需要 force.
	t.Parallel()

	baseDir := t.TempDir()
	oldSource := filepath.Join(baseDir, "old.txt")
	newSource := filepath.Join(baseDir, "new.txt")
	target := filepath.Join(baseDir, "target.txt")
	if err := os.WriteFile(oldSource, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSource, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, target); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: newSource, Relink: true}}, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Replaced, 1; got != want {
		t.Fatalf("Result.Replaced = %d, want %d", got, want)
	}
	gotTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if got, want := gotTarget, newSource; got != want {
		t.Fatalf("Readlink() = %q, want %q", got, want)
	}
}

func TestApplyRelativeCreatesRelativeSymlink(t *testing.T) {
	// relative=true 时实际落盘的 link target 必须是相对路径, 不是绝对 source.
	t.Parallel()

	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "repo", "git")
	targetDir := filepath.Join(baseDir, "home", ".config")
	source := filepath.Join(sourceDir, "gitconfig")
	target := filepath.Join(targetDir, "gitconfig")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source, Relative: true}}, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Linked, 1; got != want {
		t.Fatalf("Result.Linked = %d, want %d", got, want)
	}
	gotTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	wantTarget, err := filepath.Rel(filepath.Dir(target), source)
	if err != nil {
		t.Fatalf("filepath.Rel() error = %v", err)
	}
	if got, want := gotTarget, wantTarget; got != want {
		t.Fatalf("Readlink() = %q, want %q", got, want)
	}
}

func TestApplyRelinkRejectsProtectedSymlinkWithoutOverride(t *testing.T) {
	// 受保护 symlink 在 relink 路径下也必须经过确认护栏.
	t.Parallel()

	baseDir := t.TempDir()
	oldSource := filepath.Join(baseDir, "old.txt")
	newSource := filepath.Join(baseDir, "new.txt")
	target := filepath.Join(baseDir, "target.txt")
	if err := os.WriteFile(oldSource, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSource, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, target); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]config.LinkConfig{{Target: target, Source: newSource, Relink: true}}, ApplyOptions{
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
	gotTarget, readErr := os.Readlink(target)
	if readErr != nil {
		t.Fatalf("Readlink() error = %v", readErr)
	}
	if got, want := gotTarget, oldSource; got != want {
		t.Fatalf("Readlink() = %q, want %q", got, want)
	}
}

func TestApplyForceAllowsProtectedTargetWithOverride(t *testing.T) {
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
		ProtectedTargets:     []string{target},
		AllowProtectedTarget: true,
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Replaced, 1; got != want {
		t.Fatalf("Result.Replaced = %d, want %d", got, want)
	}
	gotTarget, readErr := os.Readlink(target)
	if readErr != nil {
		t.Fatalf("Readlink() error = %v", readErr)
	}
	if got, want := gotTarget, source; got != want {
		t.Fatalf("Readlink() = %q, want %q", got, want)
	}
}

func TestApplyCheckFailsWhenTargetParentIsNotWritable(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	lockedRoot := filepath.Join(baseDir, "locked")
	source := filepath.Join(baseDir, "source.txt")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})

	target := filepath.Join(lockedRoot, "target.txt")
	result, err := Apply([]config.LinkConfig{{Target: target, Source: source}}, ApplyOptions{Check: true})
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

func TestApplyCheckSkipsMatchingSymlinkEvenWhenParentIsReadOnly(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	lockedRoot := filepath.Join(baseDir, "locked")
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(lockedRoot, "target.txt")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source}}, ApplyOptions{Check: true})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Skipped, 0; got != want {
		t.Fatalf("Result.Skipped = %d, want %d because matching symlink does not increment skipped counter here", got, want)
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Status, output.StatusSkipped; got != want {
		t.Fatalf("Result.Entries[0].Status = %q, want %q", got, want)
	}
	if got, want := result.Entries[0].Message, "symlink already matches"; got != want {
		t.Fatalf("Result.Entries[0].Message = %q, want %q", got, want)
	}
}

func TestApplyCheckReportsConflictBeforeParentPermissionError(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	lockedRoot := filepath.Join(baseDir, "locked")
	source := filepath.Join(baseDir, "source.txt")
	target := filepath.Join(lockedRoot, "target.txt")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})

	result, err := Apply([]config.LinkConfig{{Target: target, Source: source}}, ApplyOptions{Check: true})
	if err == nil {
		t.Fatal("Apply() error = nil, want error")
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Message, "target exists and force=false"; got != want {
		t.Fatalf("Result.Entries[0].Message = %q, want %q", got, want)
	}
}
