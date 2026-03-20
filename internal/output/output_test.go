package output

import (
	"strings"
	"testing"
)

func TestFormatEntry(t *testing.T) {
	t.Parallel()

	got := FormatEntry(true, Entry{
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
