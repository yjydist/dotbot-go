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
