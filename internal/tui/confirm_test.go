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
