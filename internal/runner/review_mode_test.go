package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunOutputsPlan(t *testing.T) {
	// dry-run 的核心价值是把计划动作完整展示出来, 同时不落任何副作用.
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[create]",
		"paths = [\"~/.cache/zsh\"]",
		"",
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(dry-run) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dry-run:") {
		t.Fatalf("stdout = %q, want dry-run review header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "阶段") || !strings.Contains(stdout.String(), "目标") {
		t.Fatalf("stdout = %q, want plan table", stdout.String())
	}
	if !strings.Contains(stdout.String(), "summary:") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
}

func TestRunCheckValidatesWithoutApplying(t *testing.T) {
	// check 需要复用预检逻辑, 但不能真的修改文件系统.
	fixture := newRunnerFixture(t, true)
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "check:") {
		t.Fatalf("stdout = %q, want check header", stdout.String())
	}
	if !strings.Contains(stdout.String(), "result: check ok") {
		t.Fatalf("stdout = %q, want check result", stdout.String())
	}
	if strings.Contains(stdout.String(), "阶段 | 目标") {
		t.Fatalf("stdout = %q, check should not print action table", stdout.String())
	}
	if _, err := os.Lstat(filepath.Join(fixture.homeDir, ".gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("check should not create symlink, err=%v", err)
	}
}

func TestRunCheckFailsOnExistingTargetConflict(t *testing.T) {
	// check 也必须暴露 target 冲突, 不能因为“不执行”就错过真实失败路径.
	fixture := newRunnerFixture(t, true)
	if err := os.WriteFile(filepath.Join(fixture.homeDir, ".gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
		"create = true",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "target exists and force=false") {
		t.Fatalf("stderr = %q, want conflict error", stderr.String())
	}
	if !strings.Contains(stdout.String(), ".gitconfig") || !strings.Contains(stdout.String(), "failed") {
		t.Fatalf("stdout = %q, want failed entry detail before exit", stdout.String())
	}
	if strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, should not report check ok", stdout.String())
	}
}

func TestRunDryRunQuietKeepsFailureEntry(t *testing.T) {
	// quiet + dry-run 仍然要保留失败明细, 否则无法定位哪一项预检失败.
	fixture := newRunnerFixture(t, true)
	if err := os.WriteFile(filepath.Join(fixture.homeDir, ".gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run", "--quiet"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(dry-run --quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".gitconfig") || !strings.Contains(stdout.String(), "target exists and force=false") {
		t.Fatalf("stdout = %q, want failed entry detail in quiet dry-run", stdout.String())
	}
}

func TestRunCheckQuietKeepsFailureEntry(t *testing.T) {
	// quiet + check 和 quiet + dry-run 一样, 失败明细不能丢.
	fixture := newRunnerFixture(t, true)
	if err := os.WriteFile(filepath.Join(fixture.homeDir, ".gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check", "--quiet"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check --quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), ".gitconfig") || !strings.Contains(stdout.String(), "target exists and force=false") {
		t.Fatalf("stdout = %q, want failed entry detail in quiet check", stdout.String())
	}
}

func TestRunCheckFailsWhenCreatePathParentIsNotWritable(t *testing.T) {
	fixture := newRunnerFixture(t, false)
	lockedRoot := filepath.Join(fixture.baseDir, "locked")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})
	fixture.writeConfig(t,
		"[create]",
		"paths = ["+quote(filepath.Join(lockedRoot, "nested", "dir"))+"]",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parent directory is not writable") {
		t.Fatalf("stderr = %q, want writable parent error", stderr.String())
	}
	if strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, should not report check ok", stdout.String())
	}
	if !strings.Contains(stdout.String(), "failed") {
		t.Fatalf("stdout = %q, want failed create detail", stdout.String())
	}
}

func TestRunCheckFailsWhenLinkParentIsNotWritable(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	lockedRoot := filepath.Join(fixture.baseDir, "locked")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})
	fixture.writeConfig(t,
		"[[link]]",
		"target = "+quote(filepath.Join(lockedRoot, "target.txt")),
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parent directory is not writable") {
		t.Fatalf("stderr = %q, want writable parent error", stderr.String())
	}
	if strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, should not report check ok", stdout.String())
	}
}

func TestRunCheckSkipsMatchingSymlinkBeforePermissionPreflight(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	lockedRoot := filepath.Join(fixture.baseDir, "locked")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(lockedRoot, "target.txt")
	source := filepath.Join(fixture.baseDir, "git", "gitconfig")
	if err := os.Symlink(source, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})
	fixture.writeConfig(t,
		"[[link]]",
		"target = "+quote(target),
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, want check ok", stdout.String())
	}
	if strings.Contains(stderr.String(), "parent directory is not writable") {
		t.Fatalf("stderr = %q, should not report writable parent error", stderr.String())
	}
}

func TestRunCheckReportsConflictBeforePermissionError(t *testing.T) {
	fixture := newRunnerFixture(t, true)
	lockedRoot := filepath.Join(fixture.baseDir, "locked")
	if err := os.MkdirAll(lockedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(lockedRoot, "target.txt")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(lockedRoot, 0o755)
	})
	fixture.writeConfig(t,
		"[[link]]",
		"target = "+quote(target),
		"source = \"git/gitconfig\"",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "target exists and force=false") {
		t.Fatalf("stderr = %q, want target conflict", stderr.String())
	}
	if strings.Contains(stderr.String(), "parent directory is not writable") {
		t.Fatalf("stderr = %q, should not report writable parent error first", stderr.String())
	}
}

func TestRunCheckFailsWhenCleanDeleteParentIsNotWritable(t *testing.T) {
	fixture := newRunnerFixture(t, false)
	root := filepath.Join(fixture.baseDir, "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	deadLink := filepath.Join(root, "dead-link")
	if err := os.Symlink(filepath.Join(fixture.baseDir, "missing.txt"), deadLink); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(root, 0o755)
	})
	fixture.writeConfig(t,
		"[clean]",
		"paths = ["+quote(root)+"]",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parent directory is not writable") {
		t.Fatalf("stderr = %q, want writable parent error", stderr.String())
	}
	if strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, should not report check ok", stdout.String())
	}
}
