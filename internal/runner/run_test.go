package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--help"}, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(help) = %d, want %d", got, want)
	}
	if !strings.Contains(stdout.String(), "dotbot-go") {
		t.Fatalf("stdout = %q, want help output", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRejectsVerboseQuietTogether(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--verbose", "--quiet"}, &stdout, &stderr), 2; got != want {
		t.Fatalf("Run(verbose+quiet) = %d, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "cannot be used together") {
		t.Fatalf("stderr = %q, want mutual exclusion error", stderr.String())
	}
}

func TestRunLoadsDefaultConfig(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run(nil, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[ok]") {
		t.Fatalf("stdout = %q, want operation output", stdout.String())
	}
	if !strings.Contains(stdout.String(), "summary:") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
}

func TestRunDryRunOutputsPlan(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[create]",
		"paths = [\"~/.cache/zsh\"]",
		"",
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--dry-run"}, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(dry-run) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("stdout = %q, want dry-run output", stdout.String())
	}
	if !strings.Contains(stdout.String(), "summary:") {
		t.Fatalf("stdout = %q, want summary output", stdout.String())
	}
}

func TestRunQuietSuppressesSuccessOutput(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--quiet"}, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunCheckValidatesWithoutApplying(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--check"}, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, want check ok", stdout.String())
	}
	if strings.Contains(stdout.String(), "[ok]") || strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("stdout = %q, check should not print action lines", stdout.String())
	}
	if _, err := os.Lstat(filepath.Join(homeDir, ".gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("check should not create symlink, err=%v", err)
	}
}

func TestRunVerboseShowsConfigDetails(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	if err := os.MkdirAll(filepath.Join(baseDir, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(baseDir, "git", "gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatal(err)
	}
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run([]string{"--verbose"}, &stdout, &stderr), 0; got != want {
		t.Fatalf("Run(verbose) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config:") {
		t.Fatalf("stdout = %q, want config details", stdout.String())
	}
	if !strings.Contains(stdout.String(), "defaults:") {
		t.Fatalf("stdout = %q, want defaults summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "stages:") {
		t.Fatalf("stdout = %q, want stage summary", stdout.String())
	}
}

func TestRunMissingConfigReturnsConfigError(t *testing.T) {
	baseDir := t.TempDir()
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := Run(nil, &stdout, &stderr), 2; got != want {
		t.Fatalf("Run() = %d, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "decode config") {
		t.Fatalf("stderr = %q, want decode config error", stderr.String())
	}
}
