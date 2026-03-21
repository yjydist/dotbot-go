package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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
	if got, want := run([]string{"--verbose", "--quiet"}, strings.NewReader(""), &stdout, &stderr), 2; got != want {
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
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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
	if got, want := run([]string{"--dry-run"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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
	if got, want := run([]string{"--quiet"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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

func TestRunCheckFailsOnExistingTargetConflict(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(homeDir, ".gitconfig"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
		"create = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--check"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(check) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "target exists and force=false") {
		t.Fatalf("stderr = %q, want conflict error", stderr.String())
	}
	if strings.Contains(stdout.String(), "check ok") {
		t.Fatalf("stdout = %q, should not report check ok", stdout.String())
	}
}

func TestRunQuietStillPrintsFailure(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"missing/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--quiet"}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run(quiet) = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[fail]") {
		t.Fatalf("stdout = %q, want failure line", stdout.String())
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
	if got, want := run([]string{"--verbose"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
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
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 2; got != want {
		t.Fatalf("Run() = %d, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "decode config") {
		t.Fatalf("stderr = %q, want decode config error", stderr.String())
	}
}

func TestRunRejectsProtectedTargetWithoutOverrideInNonInteractiveMode(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	source := filepath.Join(baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents := strings.Join([]string{
		"[[link]]",
		fmt.Sprintf("target = %q", baseDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--allow-protected-target") {
		t.Fatalf("stderr = %q, want protected target override error", stderr.String())
	}
}

func TestRunAllowsProtectedTargetWithOverride(t *testing.T) {
	baseDir := t.TempDir()
	parentDir := t.TempDir()
	configPath := filepath.Join(parentDir, "dotbot-go.toml")
	source := filepath.Join(parentDir, "source.txt")
	protectedTarget := filepath.Join(parentDir, "protected")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(protectedTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := strings.Join([]string{
		"[[link]]",
		fmt.Sprintf("target = %q", protectedTarget),
		fmt.Sprintf("source = %q", source),
		"force = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", configPath, "--allow-protected-target"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if _, err := os.Readlink(protectedTarget); err != nil {
		t.Fatalf("protected target is not symlink: %v", err)
	}
}

func TestRunDryRunMarksProtectedTargetConfirmation(t *testing.T) {
	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, "dotbot-go.toml")
	source := filepath.Join(baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents := strings.Join([]string{
		"[[link]]",
		fmt.Sprintf("target = %q", baseDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "protected target, confirmation required") {
		t.Fatalf("stdout = %q, want protected target confirmation hint", stdout.String())
	}
}

func TestRunAllowsRiskyCleanWithOverride(t *testing.T) {
	baseDir := t.TempDir()
	configDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	root := homeDir
	linkPath := filepath.Join(root, "dead-link")
	if err := os.Symlink(filepath.Join(configDir, "missing.txt"), linkPath); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "dotbot-go.toml")
	contents := strings.Join([]string{
		"[clean]",
		fmt.Sprintf("paths = [%q]", root),
		"force = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", configPath, "--allow-risky-clean"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("dead link still exists, err=%v", err)
	}
}
