package tui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

type confirmModel struct {
	risks     []output.RiskItem
	styles    styles
	width     int
	confirmed bool
}

func newConfirmModel(risks []output.RiskItem, noColor bool) confirmModel {
	return confirmModel{
		risks:  risks,
		styles: newStyles(noColor),
		width:  88,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = clamp(msg.Width-8, 56, 92)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "y":
			m.confirmed = true
			return m, tea.Quit
		case "esc", "q", "n", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	lines := []string{
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.styles.riskBadge.Render("DANGER"),
			" ",
			m.styles.title.Render("高风险操作确认"),
		),
		"",
		renderSized(m.styles.confirmPanel, m.width, strings.Join([]string{
			"本次执行命中了高风险操作。",
			fmt.Sprintf("风险项数量: %d", len(m.risks)),
			"确认后将继续执行覆盖或清理动作, 取消则本次执行直接终止。",
		}, "\n")),
	}
	for _, risk := range m.risks {
		lines = append(lines, renderSized(m.styles.confirmRisk, m.width, wrapBullet(fmt.Sprintf("%s: %s", risk.Kind, risk.Path), contentWidth(m.styles.confirmRisk, m.width))))
	}
	lines = append(lines, "", m.styles.muted.Render(
		fmt.Sprintf("%s 继续执行  •  %s 取消", m.styles.key.Render("Enter"), m.styles.key.Render("Esc")),
	))
	return renderSized(m.styles.doc, m.width+m.styles.doc.GetHorizontalFrameSize(), strings.Join(lines, "\n"))
}

func RunConfirm(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
	model := newConfirmModel(risks, noColor)
	finalModel, err := tea.NewProgram(
		model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return fmt.Errorf("runtime error: confirmation ui: %w", err)
	}
	result, ok := finalModel.(confirmModel)
	if !ok {
		return fmt.Errorf("runtime error: confirmation ui: unexpected result")
	}
	if !result.confirmed {
		return fmt.Errorf("runtime error: confirmation rejected")
	}
	return nil
}
