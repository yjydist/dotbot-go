package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yjydist/dotbot-go/internal/output"
)

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
	if !strings.Contains(result.View(), "高风险操作确认") {
		t.Fatalf("View() = %q, want confirmation title", result.View())
	}
}

func TestConfirmModelSupportsScrolling(t *testing.T) {
	t.Parallel()

	var risks []output.RiskItem
	for i := 0; i < 20; i++ {
		risks = append(risks, output.RiskItem{Kind: "replace protected target", Path: "/tmp/path"})
	}
	model := newConfirmModel(risks, true)
	model.height = 12
	model.viewport.Height = model.viewportHeight()
	model.viewport.SetContent(model.renderContent())

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	afterDown := updated.(confirmModel)
	if afterDown.viewport.YOffset == 0 {
		t.Fatal("YOffset = 0, want scroll down after j")
	}
}
