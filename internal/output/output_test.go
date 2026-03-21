package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatEntry(t *testing.T) {
	t.Parallel()

	got := FormatEntry(Options{DryRun: true}, Entry{
		Stage:    "link",
		Target:   "~/.gitconfig",
		Source:   "./git/gitconfig",
		Decision: "create symlink",
		Status:   StatusLinked,
	})
	if !strings.Contains(got, "[dry-run]") {
		t.Fatalf("FormatEntry() = %q, want dry-run prefix", got)
	}
	if !strings.Contains(got, "~/.gitconfig <- ./git/gitconfig") {
		t.Fatalf("FormatEntry() = %q, want target/source", got)
	}
	if !strings.Contains(got, "create symlink") {
		t.Fatalf("FormatEntry() = %q, want decision", got)
	}
}

func TestFormatEntryWithColor(t *testing.T) {
	t.Parallel()

	got := FormatEntry(Options{EnableColor: true}, Entry{Stage: "create", Target: "~/.cache/zsh", Decision: "created", Status: StatusCreated})
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("FormatEntry() = %q, want ANSI color code", got)
	}
}

func TestWriteEntriesQuietOnlyPrintsFailure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteEntries(&buf, Options{Mode: ModeQuiet}, []Entry{
		{Stage: "create", Target: "/tmp/a", Decision: "created", Status: StatusCreated},
		{Stage: "link", Target: "/tmp/b", Decision: "failed", Status: StatusFailed, Message: "boom"},
	})
	got := buf.String()
	if strings.Contains(got, "created") {
		t.Fatalf("WriteEntries() = %q, should not include success entry", got)
	}
	if !strings.Contains(got, "[fail]") {
		t.Fatalf("WriteEntries() = %q, want failure entry", got)
	}
}

func TestFormatEntryNoColor(t *testing.T) {
	t.Parallel()

	got := FormatEntry(Options{EnableColor: false}, Entry{Stage: "create", Target: "~/.cache/zsh", Decision: "created", Status: StatusCreated})
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("FormatEntry() = %q, should not include ANSI color code", got)
	}
}

func TestRenderEntryTable(t *testing.T) {
	t.Parallel()

	got := RenderEntryTable([]Entry{
		{Stage: "create", Target: "/tmp/a", Decision: "create", Status: StatusCreated},
		{Stage: "link", Target: "/tmp/b", Source: "/repo/b", Decision: "create symlink", Status: StatusLinked, Message: "protected target, confirmation required"},
	})
	if !strings.Contains(got, "阶段") || !strings.Contains(got, "目标") {
		t.Fatalf("RenderEntryTable() = %q, want table header", got)
	}
	if !strings.Contains(got, "/repo/b") {
		t.Fatalf("RenderEntryTable() = %q, want source column", got)
	}
	if !strings.Contains(got, "protected target, confirmation required") {
		t.Fatalf("RenderEntryTable() = %q, want message column", got)
	}
}

func TestWriteReviewTextDryRun(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{DryRun: true}, ReviewData{
		Mode:        ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: StageCounts{Create: 1, Link: 1, Clean: 0},
		Risks:       []RiskItem{{Kind: "replace protected target", Path: "/tmp/a"}},
		Entries: []Entry{
			{Stage: "link", Target: "/tmp/a", Source: "/repo/a", Decision: "replace", Status: StatusReplaced, Message: "protected target, confirmation required"},
		},
		Summary: Summary{Replaced: 1},
	})

	got := buf.String()
	if !strings.Contains(got, "dry-run:") {
		t.Fatalf("WriteReviewText() = %q, want mode header", got)
	}
	if !strings.Contains(got, "risks: 1") {
		t.Fatalf("WriteReviewText() = %q, want risks summary", got)
	}
	if !strings.Contains(got, "阶段") || !strings.Contains(got, "目标") {
		t.Fatalf("WriteReviewText() = %q, want table output", got)
	}
	if !strings.Contains(got, "summary: created=0 linked=0 skipped=0 replaced=1 deleted=0 failed=0") {
		t.Fatalf("WriteReviewText() = %q, want summary", got)
	}
}

func TestWriteReviewTextCheck(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{}, ReviewData{
		Mode:        ReviewModeCheck,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: StageCounts{Create: 0, Link: 1, Clean: 1},
		Result:      "check ok",
	})

	got := buf.String()
	if !strings.Contains(got, "check:") {
		t.Fatalf("WriteReviewText() = %q, want check header", got)
	}
	if !strings.Contains(got, "result: check ok") {
		t.Fatalf("WriteReviewText() = %q, want check result", got)
	}
	if strings.Contains(got, "阶段 | 目标") {
		t.Fatalf("WriteReviewText() = %q, check should not print action table", got)
	}
}
