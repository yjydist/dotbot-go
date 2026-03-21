package runner

import (
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/config"
)

func TestBuildVerboseLinesUsesEffectiveCreateAndCleanConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Default: config.DefaultConfig{
			Link:   config.LinkDefaults{Create: false, Relink: false, Force: false, Relative: false, IgnoreMissing: false},
			Create: config.CreateDefaults{Mode: 0o777},
			Clean:  config.CleanDefaults{Force: false, Recursive: false},
		},
		Create: config.CreateConfig{Mode: 0o755},
		Clean:  config.CleanConfig{Force: true, Recursive: true},
		Links: []config.LinkConfig{
			{Create: true, Relink: false, Force: false, Relative: false, IgnoreMissing: true},
		},
	}

	lines := buildVerboseLines(cfg)
	got := strings.Join(lines, "\n")
	if !strings.Contains(got, "create: mode=0755") {
		t.Fatalf("buildVerboseLines() = %q, want effective create mode", got)
	}
	if !strings.Contains(got, "clean: force=true recursive=true") {
		t.Fatalf("buildVerboseLines() = %q, want effective clean config", got)
	}
	if !strings.Contains(got, "link: create=true relink=false force=false relative=false ignore_missing=true") {
		t.Fatalf("buildVerboseLines() = %q, want effective link config", got)
	}
}

func TestBuildVerboseLinesMarksMixedLinkValues(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Default: config.DefaultConfig{
			Link: config.LinkDefaults{Create: false, Relink: false, Force: false, Relative: false, IgnoreMissing: false},
		},
		Links: []config.LinkConfig{
			{Create: true, Relink: false, Force: false, Relative: false, IgnoreMissing: false},
			{Create: false, Relink: false, Force: false, Relative: false, IgnoreMissing: false},
		},
	}

	lines := buildVerboseLines(cfg)
	if got := lines[0]; got != "link: mixed per-link values" {
		t.Fatalf("buildVerboseLines() first line = %q, want mixed summary", got)
	}
}
