package creator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestApplyCreatesDirectories(t *testing.T) {
	// 基线用例: 不存在的目录会被正常创建, 包括多层目录.
	t.Parallel()

	baseDir := t.TempDir()
	paths := []string{
		filepath.Join(baseDir, "one"),
		filepath.Join(baseDir, "nested", "two"),
	}

	result, err := Apply(paths, 0o755, false, false)
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

func TestApplyDryRunReportsCreateWithoutFilesystemChange(t *testing.T) {
	// dry-run 只能记录计划, 不能真的在文件系统里创建目录.
	t.Parallel()

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "nested", "dir")

	result, err := Apply([]string{target}, 0o755, true, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Created, 1; got != want {
		t.Fatalf("Result.Created = %d, want %d", got, want)
	}
	if got, want := len(result.Entries), 1; got != want {
		t.Fatalf("len(Result.Entries) = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Decision, "create"; got != want {
		t.Fatalf("Result.Entries[0].Decision = %q, want %q", got, want)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) err = %v, want not exist after dry-run", target, err)
	}
}

func TestApplySkipsExistingDirectory(t *testing.T) {
	// create 对已存在目录的语义是 skip, 而不是报错或重复创建.
	t.Parallel()

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "existing")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]string{target}, 0o755, false, false)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if got, want := result.Created, 0; got != want {
		t.Fatalf("Result.Created = %d, want %d", got, want)
	}
	if got, want := result.Entries[0].Status, output.StatusSkipped; got != want {
		t.Fatalf("Result.Entries[0].Status = %q, want %q", got, want)
	}
}

func TestApplyFailsWhenPathExistsAsFile(t *testing.T) {
	// 已存在普通文件时必须失败, 防止 create 把文件路径误当目录处理.
	t.Parallel()

	baseDir := t.TempDir()
	target := filepath.Join(baseDir, "file")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply([]string{target}, 0o755, false, false)
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

func TestApplyCheckFailsWhenParentIsNotWritable(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	lockedRoot := filepath.Join(baseDir, "locked")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})

	target := filepath.Join(lockedRoot, "nested", "dir")
	result, err := Apply([]string{target}, 0o755, false, true)
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
