package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunOutputsPlan(t *testing.T) {
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
