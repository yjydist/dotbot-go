package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndResolvesPaths(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	workingDir := filepath.Join(baseDir, "work")
	homeDir := filepath.Join(baseDir, "home")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(baseDir, DefaultConfigName)
	contents := strings.Join([]string{
		"[default.link]",
		"create = true",
		"relative = true",
		"",
		"[default.create]",
		"mode = \"0755\"",
		"",
		"[create]",
		"paths = [\"~/.cache/zsh\", \"./tmp\"]",
		"",
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"./git/gitconfig\"",
		"",
		"[[link]]",
		"target = \"./relative-target\"",
		"source = \"../shared/file\"",
		"ignore_missing = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{Path: configPath, WorkingDir: workingDir, HomeDir: homeDir})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Create.Mode.Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("Create.Mode = %v, want %v", got, want)
	}
	if got, want := cfg.Create.Paths[0], filepath.Join(homeDir, ".cache/zsh"); got != want {
		t.Fatalf("Create.Paths[0] = %q, want %q", got, want)
	}
	if got, want := cfg.Create.Paths[1], filepath.Join(workingDir, "tmp"); got != want {
		t.Fatalf("Create.Paths[1] = %q, want %q", got, want)
	}
	if got, want := cfg.Links[0].Target, filepath.Join(homeDir, ".gitconfig"); got != want {
		t.Fatalf("Links[0].Target = %q, want %q", got, want)
	}
	if got, want := cfg.Links[0].Source, filepath.Join(baseDir, "git/gitconfig"); got != want {
		t.Fatalf("Links[0].Source = %q, want %q", got, want)
	}
	if !cfg.Links[0].Create {
		t.Fatal("Links[0].Create = false, want true")
	}
	if !cfg.Links[0].Relative {
		t.Fatal("Links[0].Relative = false, want true")
	}
	if !cfg.Links[1].IgnoreMissing {
		t.Fatal("Links[1].IgnoreMissing = false, want true")
	}
	if got, want := cfg.Links[1].Target, filepath.Join(workingDir, "relative-target"); got != want {
		t.Fatalf("Links[1].Target = %q, want %q", got, want)
	}
	if got, want := cfg.Links[1].Source, filepath.Clean(filepath.Join(baseDir, "../shared/file")); got != want {
		t.Fatalf("Links[1].Source = %q, want %q", got, want)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, DefaultConfigName)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
		"backup = true",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Path: configPath, WorkingDir: baseDir, HomeDir: baseDir})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown field or section") {
		t.Fatalf("Load() error = %v, want unknown field error", err)
	}
}

func TestLoadRejectsDuplicateTargets(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, DefaultConfigName)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
		"",
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/other-gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Path: configPath, WorkingDir: baseDir, HomeDir: baseDir})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "duplicate target path") {
		t.Fatalf("Load() error = %v, want duplicate target error", err)
	}
}

func TestLoadRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, DefaultConfigName)
	contents := strings.Join([]string{
		"[[link]]",
		"target = \"~/.gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Path: configPath, WorkingDir: baseDir, HomeDir: baseDir})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "required field is missing") {
		t.Fatalf("Load() error = %v, want required field error", err)
	}
}

func TestLoadRejectsInvalidDefaultCreateMode(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	configPath := filepath.Join(baseDir, DefaultConfigName)
	contents := strings.Join([]string{
		"[default.create]",
		"mode = \"invalid\"",
		"",
		"[[link]]",
		"target = \"~/.gitconfig\"",
		"source = \"git/gitconfig\"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{Path: configPath, WorkingDir: baseDir, HomeDir: baseDir})
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "[default.create].mode") {
		t.Fatalf("Load() error = %v, want default.create.mode path", err)
	}
}
