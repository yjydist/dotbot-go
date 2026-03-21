package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

func sampleDryRunReviewData() output.ReviewData {
	return output.ReviewData{
		Mode:        output.ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Create: 1, Link: 2, Clean: 1},
		Risks:       []output.RiskItem{{Kind: "replace protected target", Path: "/tmp/a"}},
		VerboseLines: []string{
			"link: create=true relink=false force=false relative=false ignore_missing=false",
			"create: mode=0755",
			"clean: force=false recursive=true",
		},
		Entries: []output.Entry{
			{Stage: "link", Target: "/tmp/a", Source: "/repo/a", Decision: "replace", Status: output.StatusReplaced, Message: "protected target, confirmation required"},
		},
		Summary: output.Summary{Replaced: 1},
	}
}

func assertRenderedWithinWidth(t *testing.T, rendered string, maxWidth int) {
	t.Helper()

	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > maxWidth {
			t.Fatalf("line width = %d, want <= %d, line=%q", lipgloss.Width(line), maxWidth, line)
		}
	}
}
