package runner

import (
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

func TestBuildConfigGroupsUsesEffectiveCreateAndCleanConfig(t *testing.T) {
	// verbose 区块必须展示最终生效值, 不是 default 原始值.
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

	groups := buildConfigGroups(cfg)
	got := strings.Join([]string{
		output.RenderConfigGroup(groups[0]),
		output.RenderConfigGroup(groups[1]),
		output.RenderConfigGroup(groups[2]),
	}, "\n")
	if !strings.Contains(got, "create: mode=0755") {
		t.Fatalf("buildConfigGroups() = %q, want effective create mode", got)
	}
	if !strings.Contains(got, "clean: force=true recursive=true") {
		t.Fatalf("buildConfigGroups() = %q, want effective clean config", got)
	}
	if !strings.Contains(got, "link: create=true relink=false force=false relative=false ignore_missing=true") {
		t.Fatalf("buildConfigGroups() = %q, want effective link config", got)
	}
}

func TestBuildConfigGroupsMarksMixedLinkValues(t *testing.T) {
	// 当不同 [[link]] 的生效布尔值不一致时, 摘要要明确标成 mixed.
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

	groups := buildConfigGroups(cfg)
	if got := output.RenderConfigGroup(groups[0]); got != "link: mixed per-link values" {
		t.Fatalf("buildConfigGroups() first group = %q, want mixed summary", got)
	}
	joined := strings.Join([]string{output.RenderConfigGroup(groups[3]), output.RenderConfigGroup(groups[4])}, "\n")
	if !strings.Contains(joined, "link[1]: target=") || !strings.Contains(joined, "link[2]: target=") {
		t.Fatalf("buildConfigGroups() = %q, want per-link details for mixed values", joined)
	}
}
