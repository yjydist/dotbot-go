package tui

import (
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
