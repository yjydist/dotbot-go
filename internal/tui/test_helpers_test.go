package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

// sampleDryRunReviewData 提供一份最小但足够复杂的审阅样例,
// 让各类 TUI 测试都能复用同一组输入事实.
func sampleDryRunReviewData() output.ReviewData {
	return output.ReviewData{
		Mode:        output.ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Create: 1, Link: 2, Clean: 1},
		Risks:       []output.RiskItem{{Kind: "replace protected target", Path: "/tmp/a"}},
		ConfigGroups: []output.ConfigGroup{
			{Scope: "link", Fields: []output.ConfigField{{Key: "create", Value: "true"}, {Key: "relink", Value: "false"}, {Key: "force", Value: "false"}, {Key: "relative", Value: "false"}, {Key: "ignore_missing", Value: "false"}}},
			{Scope: "create", Fields: []output.ConfigField{{Key: "mode", Value: "0755"}}},
			{Scope: "clean", Fields: []output.ConfigField{{Key: "force", Value: "false"}, {Key: "recursive", Value: "true"}}},
		},
		Entries: []output.Entry{
			{Stage: "link", Target: "/tmp/a", Source: "/repo/a", Decision: "replace", Status: output.StatusReplaced, Message: "protected target, confirmation required"},
		},
		Summary: output.Summary{Replaced: 1},
	}
}

// assertRenderedWithinWidth 用来保证渲染结果不会超过给定宽度.
// TUI 的很多回归都是“功能没错, 但布局溢出了”, 所以这类断言非常关键.
func assertRenderedWithinWidth(t *testing.T, rendered string, maxWidth int) {
	t.Helper()

	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > maxWidth {
			t.Fatalf("line width = %d, want <= %d, line=%q", lipgloss.Width(line), maxWidth, line)
		}
	}
}
