package tui

import (
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestReviewModelPanelsStayWithinWidth(t *testing.T) {
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), true)
	model.data.ConfigPath = "/very/long/config/path/for/testing/dotbot-go.toml"
	model.data.BaseDir = "/very/long/base/dir/for/testing"
	model.data.Risks = []output.RiskItem{
		{Kind: "replace protected target", Path: "/very/long/path/that/should/wrap/cleanly/without/breaking/the/panel"},
	}
	model.data.Entries = []output.Entry{
		{
			Stage:    "link",
			Target:   "/very/long/target/path/that/needs/to/wrap/without/breaking/the/card/border/example.toml",
			Source:   "/very/long/source/path/that/also/needs/to/wrap/without/breaking/the/card/border/example.toml",
			Decision: "replace existing path with new symlink",
			Status:   output.StatusReplaced,
			Message:  "protected target, confirmation required",
		},
	}
	model.data.Summary = output.Summary{Created: 1, Linked: 1, Deleted: 1}
	model.width = 84
	model.height = 24

	assertRenderedWithinWidth(t, model.renderOverviewPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderRiskPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderSummaryPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderEntryCard(1, model.data.Entries[0], clamp(model.bodyWidth()-4, 52, model.bodyWidth())), clamp(model.bodyWidth()-4, 52, model.bodyWidth()))
}

func TestReviewModelPanelsStayWithinWidthWithCJK(t *testing.T) {
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), true)
	model.data.ConfigPath = "/仓库/非常长的配置路径/用于测试/dotbot-go.toml"
	model.data.BaseDir = "/仓库/基础目录"
	model.data.Risks = []output.RiskItem{
		{Kind: "replace protected target", Path: "/用户/桌面/中文路径/配置目录"},
	}
	model.data.Entries = []output.Entry{
		{
			Stage:    "link",
			Target:   "/用户/配置/非常长的中文路径/需要正确换行/示例.toml",
			Source:   "/仓库/源文件/非常长的中文路径/需要正确换行/示例.toml",
			Decision: "create symlink",
			Status:   output.StatusLinked,
			Message:  "中文路径也不应该把边框撑坏",
		},
	}
	model.width = 84
	model.height = 24

	assertRenderedWithinWidth(t, model.renderOverviewPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderRiskPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderEntryCard(1, model.data.Entries[0], clamp(model.bodyWidth()-4, 52, model.bodyWidth())), clamp(model.bodyWidth()-4, 52, model.bodyWidth()))
}

func TestWrapTextRespectsDisplayWidthForCJK(t *testing.T) {
	t.Parallel()

	lines := wrapText("这是一个非常长的中文路径用于验证宽度换行是否正确", 10)
	for _, line := range lines {
		if got := displayWidth(line); got > 10 {
			t.Fatalf("displayWidth(%q) = %d, want <= 10", line, got)
		}
	}
}

func TestReviewOverviewTableWrapsLongValuesIntoSeparateRows(t *testing.T) {
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), true)
	model.width = 84
	model.height = 24

	view := model.renderOverviewTable([]tableRow{
		{Label: "config file", Value: "/very/long/config/path/for/testing/dotbot-go.toml"},
	}, contentWidth(model.styles.panel, model.bodyWidth()))

	if !strings.Contains(view, "config file") {
		t.Fatalf("renderOverviewTable() = %q, want label row", view)
	}
	if strings.Contains(view, "config file\n") {
		t.Fatalf("renderOverviewTable() = %q, should keep one-line rows for bubbles table", view)
	}
	assertRenderedWithinWidth(t, view, contentWidth(model.styles.panel, model.bodyWidth()))
}
