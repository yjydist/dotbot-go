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

func TestRunUsesConfirmUIForRiskyOperationsWhenInteractive(t *testing.T) {
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

func TestRunSkipsConfirmUIWhenOverrideProvided(t *testing.T) {
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
