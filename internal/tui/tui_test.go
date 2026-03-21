package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yjydist/dotbot-go/internal/output"
)

func TestReviewModelViewDryRunIncludesRiskAndTable(t *testing.T) {
	t.Parallel()

	model := newReviewModel(output.ReviewData{
		Mode:        output.ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Create: 1, Link: 2, Clean: 1},
		Risks:       []output.RiskItem{{Kind: "replace protected target", Path: "/tmp/a"}},
		Entries: []output.Entry{
			{Stage: "link", Target: "/tmp/a", Source: "/repo/a", Decision: "replace", Status: output.StatusReplaced, Message: "protected target, confirmation required"},
		},
		Summary: output.Summary{Replaced: 1},
	}, false)

	view := model.View()
	if !strings.Contains(view, "dry-run review") {
		t.Fatalf("View() = %q, want review title", view)
	}
	if !strings.Contains(view, "replace protected target: /tmp/a") {
		t.Fatalf("View() = %q, want risk item", view)
	}
	if !strings.Contains(view, "计划动作") {
		t.Fatalf("View() = %q, want plan section", view)
	}
	if !strings.Contains(view, "阶段") || !strings.Contains(view, "目标") {
		t.Fatalf("View() = %q, want table content", view)
	}
}

func TestReviewModelViewCheckOmitsPlanTable(t *testing.T) {
	t.Parallel()

	model := newReviewModel(output.ReviewData{
		Mode:        output.ReviewModeCheck,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: output.StageCounts{Create: 0, Link: 1, Clean: 0},
		Result:      "check ok",
	}, true)

	view := model.View()
	if !strings.Contains(view, "check ok") {
		t.Fatalf("View() = %q, want check result", view)
	}
	if strings.Contains(view, "计划动作") {
		t.Fatalf("View() = %q, check should not include plan section", view)
	}
}

func TestConfirmModelAcceptsEnter(t *testing.T) {
	t.Parallel()

	model := newConfirmModel([]output.RiskItem{{Kind: "risky clean root", Path: "/tmp/a"}}, true)
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update() cmd = nil, want quit command")
	}
	result := updated.(confirmModel)
	if !result.confirmed {
		t.Fatal("Update() confirmed = false, want true")
	}
}

func TestConfirmModelRejectsEscape(t *testing.T) {
	t.Parallel()

	model := newConfirmModel([]output.RiskItem{{Kind: "risky clean root", Path: "/tmp/a"}}, true)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := updated.(confirmModel)
	if result.confirmed {
		t.Fatal("Update() confirmed = true, want false")
	}
	if !strings.Contains(result.View(), "detected risky operations") {
		t.Fatalf("View() = %q, want confirmation title", result.View())
	}
}
