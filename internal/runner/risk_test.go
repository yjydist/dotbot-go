package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestRunRejectsProtectedTargetWithoutOverrideInNonInteractiveMode(t *testing.T) {
	// 非交互环境命中受保护目标时, 必须要求显式 override.
	fixture := newRunnerFixture(t, false)
	source := filepath.Join(fixture.baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", fixture.baseDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run(nil, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--allow-protected-target") {
		t.Fatalf("stderr = %q, want protected target override error", stderr.String())
	}
}

func TestRunDryRunMarksProtectedTargetConfirmation(t *testing.T) {
	// dry-run 要提前暴露受保护目标风险, 方便用户在真正执行前预审.
	fixture := newRunnerFixture(t, false)
	source := filepath.Join(fixture.baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", fixture.baseDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "protected target, confirmation required") {
		t.Fatalf("stdout = %q, want protected target confirmation hint", stdout.String())
	}
	if !strings.Contains(stdout.String(), "replace protected target") {
		t.Fatalf("stdout = %q, want risk summary", stdout.String())
	}
}

func TestRunDryRunMarksProtectedRelinkConfirmation(t *testing.T) {
	// 受保护 symlink 的 relink 路径也必须在 dry-run 里标记成高风险.
	fixture := newRunnerFixture(t, false)
	oldSource := filepath.Join(fixture.baseDir, "old.txt")
	newSource := filepath.Join(fixture.baseDir, "new.txt")
	target := fixture.homeDir
	if err := os.WriteFile(oldSource, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSource, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, target); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", target),
		fmt.Sprintf("source = %q", newSource),
		"relink = true",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--dry-run", "--config", fixture.configPath}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stdout.String(), "replace protected target") {
		t.Fatalf("stdout = %q, want protected relink risk summary", stdout.String())
	}
	if !strings.Contains(stdout.String(), "protected target, confirmation required") {
		t.Fatalf("stdout = %q, want protected relink confirmation hint", stdout.String())
	}
}

func TestRunAllowsRiskyCleanWithOverride(t *testing.T) {
	// 显式 allow-risky-clean 后, 仓库外 dead target 的 risky clean 应允许继续.
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
	if err := os.WriteFile(configPath, []byte(strings.Join([]string{
		"[clean]",
		fmt.Sprintf("paths = [%q]", root),
		"force = true",
	}, "\n")), 0o644); err != nil {
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

func TestRunRejectsRiskyCleanWithoutOverrideInNonInteractiveMode(t *testing.T) {
	// 非交互环境命中 risky clean root 时, 必须阻止执行并保留原始 dead link.
	baseDir := t.TempDir()
	configDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	linkPath := filepath.Join(homeDir, "dead-link")
	if err := os.Symlink(filepath.Join(configDir, "missing.txt"), linkPath); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "dotbot-go.toml")
	if err := os.WriteFile(configPath, []byte(strings.Join([]string{
		"[clean]",
		fmt.Sprintf("paths = [%q]", homeDir),
		"force = true",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", configPath}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--allow-risky-clean") {
		t.Fatalf("stderr = %q, want risky clean override error", stderr.String())
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("dead link should remain after rejection, err=%v", err)
	}
}

func TestRunUsesConfirmUIForRiskyOperationsWhenInteractive(t *testing.T) {
	// 交互环境命中高风险覆盖时, runner 应交给 confirm UI 而不是直接执行.
	fixture := newRunnerFixture(t, false)
	source := filepath.Join(fixture.baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", fixture.homeDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	)

	called := false
	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		nil,
		func(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
			called = true
			if len(risks) != 1 {
				t.Fatalf("confirm risks = %d, want 1", len(risks))
			}
			if risks[0].Kind != "replace protected target" {
				t.Fatalf("risk kind = %q, want protected target", risks[0].Kind)
			}
			return fmt.Errorf("stop after confirm")
		},
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", fixture.configPath}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !called {
		t.Fatal("confirm UI not called")
	}
}

func TestRunUsesConfirmUIForProtectedRelinkWhenInteractive(t *testing.T) {
	// 受保护 symlink 的 relink 风险也必须进入同一套确认 UI.
	fixture := newRunnerFixture(t, false)
	oldSource := filepath.Join(fixture.baseDir, "old.txt")
	newSource := filepath.Join(fixture.baseDir, "new.txt")
	target := fixture.homeDir
	if err := os.WriteFile(oldSource, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newSource, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(oldSource, target); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", target),
		fmt.Sprintf("source = %q", newSource),
		"relink = true",
	)

	called := false
	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		nil,
		func(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
			called = true
			if len(risks) != 1 {
				t.Fatalf("confirm risks = %d, want 1", len(risks))
			}
			if risks[0].Kind != "replace protected target" {
				t.Fatalf("risk kind = %q, want protected target", risks[0].Kind)
			}
			if risks[0].Path != target {
				t.Fatalf("risk path = %q, want %q", risks[0].Path, target)
			}
			return fmt.Errorf("stop after confirm")
		},
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", fixture.configPath}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !called {
		t.Fatal("confirm UI not called for protected relink")
	}
}

func TestRunSkipsConfirmUIWhenOverrideProvided(t *testing.T) {
	// 已显式 override 后, 交互执行不应再重复弹确认 UI.
	fixture := newRunnerFixture(t, false)
	source := filepath.Join(fixture.baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", fixture.homeDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	)

	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		nil,
		func(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
			t.Fatalf("confirm UI should not be called, risks=%v", risks)
			return nil
		},
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", fixture.configPath, "--allow-protected-target"}, strings.NewReader(""), &stdout, &stderr), 0; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
}

func TestRunStopsWithoutSideEffectsWhenConfirmRejected(t *testing.T) {
	// 用户拒绝确认后, runner 必须终止且不能留下任何覆盖副作用.
	fixture := newRunnerFixture(t, false)
	source := filepath.Join(fixture.baseDir, "source.txt")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.homeDir, []byte("boom"), 0o644); err == nil {
		t.Fatal("expected write home dir to fail, fixture assumption broken")
	}
	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		nil,
		func(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
			return fmt.Errorf("runtime error: confirmation rejected")
		},
	)

	fixture.writeConfig(t,
		"[[link]]",
		fmt.Sprintf("target = %q", fixture.homeDir),
		fmt.Sprintf("source = %q", source),
		"force = true",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", fixture.configPath}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if !strings.Contains(stderr.String(), "confirmation rejected") {
		t.Fatalf("stderr = %q, want rejection error", stderr.String())
	}
	info, err := os.Stat(fixture.homeDir)
	if err != nil {
		t.Fatalf("Stat(homeDir) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("homeDir should remain directory after rejection")
	}
}

func TestRunLeavesDeadLinkWhenRiskyCleanConfirmRejected(t *testing.T) {
	// risky clean 被拒绝确认后, 原始 dead link 必须原样保留.
	baseDir := t.TempDir()
	configDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", homeDir)
	linkPath := filepath.Join(homeDir, "dead-link")
	if err := os.Symlink(filepath.Join(configDir, "missing.txt"), linkPath); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "dotbot-go.toml")
	if err := os.WriteFile(configPath, []byte(strings.Join([]string{
		"[clean]",
		fmt.Sprintf("paths = [%q]", homeDir),
		"force = true",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(baseDir)

	withRunnerHooks(t,
		func(io.Reader, io.Writer) bool { return true },
		nil,
		func(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
			return fmt.Errorf("runtime error: confirmation rejected")
		},
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if got, want := run([]string{"--config", configPath}, strings.NewReader(""), &stdout, &stderr), 1; got != want {
		t.Fatalf("Run() = %d, want %d, stderr=%q", got, want, stderr.String())
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("dead link should remain after rejection, err=%v", err)
	}
}
