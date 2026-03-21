package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestReviewModelViewDryRunIncludesRiskAndCards(t *testing.T) {
	t.Parallel()

	model := newReviewModel(sampleDryRunReviewData(), false)
	model.width = 120
	model.height = 80
	model.viewport.Width = model.bodyWidth()
	model.viewport.Height = model.viewportHeight()
	model.viewport.SetContent(model.renderContent())

	view := model.View()
	if !strings.Contains(view, "DRY-RUN") {
		t.Fatalf("View() = %q, want review title", view)
	}
	if !strings.Contains(view, "replace protected target: /tmp/a") {
		t.Fatalf("View() = %q, want risk item", view)
	}
	if !strings.Contains(view, "计划动作") {
		t.Fatalf("View() = %q, want plan section", view)
	}
	if !strings.Contains(view, "config file") || !strings.Contains(view, "/repo/dotbot-go.toml") {
		t.Fatalf("View() = %q, want config path", view)
	}
	if !strings.Contains(view, "base dir") || !strings.Contains(view, "/repo") {
		t.Fatalf("View() = %q, want base dir", view)
	}
	if !strings.Contains(view, "字段") || !strings.Contains(view, "值") {
		t.Fatalf("View() = %q, want overview table header", view)
	}
	if strings.Contains(view, "default") {
		t.Fatalf("View() = %q, should not expose default terminology", view)
	}
	if !strings.Contains(view, "link.create") || !strings.Contains(view, "true") {
		t.Fatalf("View() = %q, want expanded link config row", view)
	}
	if !strings.Contains(view, "link.ignore_missing") {
		t.Fatalf("View() = %q, want expanded ignore_missing config", view)
	}
	if !strings.Contains(view, "target: /tmp/a") {
		t.Fatalf("View() = %q, want target field", view)
	}
	if !strings.Contains(view, "action: replace") {
		t.Fatalf("View() = %q, want action field", view)
	}
}

func TestReviewModelViewCheckOmitsPlanTable(t *testing.T) {
	t.Parallel()

	model := newReviewModel(output.ReviewData{
		Mode:        output.ReviewModeCheck,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Create: 0, Link: 1, Clean: 0},
		VerboseLines: []string{
			"link: create=false relink=false force=false relative=false ignore_missing=false",
			"create: mode=0777",
		},
		Result: "check ok",
	}, true)
	model.width = 120
	model.height = 80
	model.viewport.Width = model.bodyWidth()
	model.viewport.Height = model.viewportHeight()
	model.viewport.SetContent(model.renderContent())

	view := model.View()
	if !strings.Contains(view, "CHECK OK") {
		t.Fatalf("View() = %q, want check result", view)
	}
	if strings.Contains(view, "计划动作") {
		t.Fatalf("View() = %q, check should not include plan section", view)
	}
	if !strings.Contains(view, "config file") || !strings.Contains(view, "/repo/dotbot-go.toml") {
		t.Fatalf("View() = %q, want config path", view)
	}
	if !strings.Contains(view, "base dir") || !strings.Contains(view, "/repo") {
		t.Fatalf("View() = %q, want base dir", view)
	}
	if !strings.Contains(view, "link.create") || !strings.Contains(view, "false") {
		t.Fatalf("View() = %q, want active link config", view)
	}
	if strings.Contains(view, "create: mode=0777") {
		t.Fatalf("View() = %q, should not show inactive create config", view)
	}
}

func TestReviewModelSupportsNavigationKeys(t *testing.T) {
	t.Parallel()

	var entries []output.Entry
	for i := 0; i < 24; i++ {
		entries = append(entries, output.Entry{
			Stage:    "link",
			Target:   fmt.Sprintf("/tmp/target-%02d", i),
			Source:   fmt.Sprintf("/repo/source-%02d", i),
			Decision: "create symlink",
			Status:   output.StatusLinked,
		})
	}
	model := newReviewModel(output.ReviewData{
		Mode:        output.ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Link: len(entries)},
		Entries:     entries,
	}, true)
	model.viewport.Height = 4
	model.viewport.SetContent(model.renderContent())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	afterDown := updated.(reviewModel)
	if afterDown.viewport.YOffset == 0 {
		t.Fatal("YOffset = 0, want scroll down after j")
	}

	updated, _ = afterDown.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	afterBottom := updated.(reviewModel)
	if !afterBottom.viewport.AtBottom() {
		t.Fatal("viewport not at bottom after G")
	}

	updated, _ = afterBottom.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	afterTop := updated.(reviewModel)
	if !afterTop.viewport.AtTop() {
		t.Fatal("viewport not at top after g")
	}
}

func TestReviewModelKeepsMixedLinkSummaryAsSingleOverviewValue(t *testing.T) {
	t.Parallel()

	model := newReviewModel(output.ReviewData{
		Mode:        output.ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Link: 2},
		VerboseLines: []string{
			"link: mixed per-link values",
		},
	}, true)
	model.width = 120
	model.height = 40
	model.viewport.Width = model.bodyWidth()
	model.viewport.Height = model.viewportHeight()
	model.viewport.SetContent(model.renderContent())

	view := model.View()
	if !strings.Contains(view, "mixed per-link values") {
		t.Fatalf("View() = %q, want mixed link summary", view)
	}
	if strings.Contains(view, "link   per-link") || strings.Contains(view, "link   values") {
		t.Fatalf("View() = %q, should keep mixed summary as one row", view)
	}
}
