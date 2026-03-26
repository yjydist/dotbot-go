package tui

import (
	"strings"
	"testing"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestReviewModelPanelsStayWithinWidth(t *testing.T) {
	// 常规宽度下, overview/risk/summary/card 都不应突破各自外框宽度.
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
	assertRenderedWithinWidth(t, model.renderEntryCard(1, model.data.Entries[0], reviewCardWidth(model.bodyWidth())), reviewCardWidth(model.bodyWidth()))
}

func TestReviewModelPanelsStayWithinWidthWithCJK(t *testing.T) {
	// 中文路径是最容易把宽度计算打崩的输入, 这条用例专门卡住它.
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
	assertRenderedWithinWidth(t, model.renderEntryCard(1, model.data.Entries[0], reviewCardWidth(model.bodyWidth())), reviewCardWidth(model.bodyWidth()))
}

func TestWrapTextRespectsDisplayWidthForCJK(t *testing.T) {
	// wrapText 必须按终端 cell width 切行, 否则中文宽度会低估.
	t.Parallel()

	lines := wrapText("这是一个非常长的中文路径用于验证宽度换行是否正确", 10)
	for _, line := range lines {
		if got := displayWidth(line); got > 10 {
			t.Fatalf("displayWidth(%q) = %d, want <= 10", line, got)
		}
	}
}

func TestReviewOverviewTableWrapsLongValuesIntoSeparateRows(t *testing.T) {
	// overview 表格在长值场景下需要拆行, 不能把多行内容硬塞进单个 cell.
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

func TestReviewPanelsStayWithinVeryNarrowWidth(t *testing.T) {
	// 很窄的终端里也必须优先保证“不溢出”, 再谈样式.
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), true)
	model.data.ConfigPath = "/very/long/config/path/for/testing/dotbot-go.toml"
	model.data.BaseDir = "/very/long/base/dir/for/testing"
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
	model.width = 50
	model.height = 12

	assertRenderedWithinWidth(t, model.renderOverviewPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderEntrySection(), model.bodyWidth())
}

func TestReviewPanelsStayWithinUltraNarrowWidth(t *testing.T) {
	// 极窄终端是 TUI 最容易回归的边界, 需要专门有一条极限用例.
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), true)
	model.data.ConfigPath = "/x"
	model.data.BaseDir = "/y"
	model.data.Risks = []output.RiskItem{
		{Kind: "replace protected target", Path: "/z"},
	}
	model.data.Entries = []output.Entry{
		{
			Stage:    "link",
			Target:   "/very/long/target/path",
			Source:   "/very/long/source/path",
			Decision: "create symlink",
			Status:   output.StatusLinked,
		},
	}
	model.width = 20
	model.height = 10

	assertRenderedWithinWidth(t, model.renderOverviewPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderRiskPanel(), model.bodyWidth())
	assertRenderedWithinWidth(t, model.renderEntryCard(1, model.data.Entries[0], reviewCardWidth(model.bodyWidth())), reviewCardWidth(model.bodyWidth()))
}
