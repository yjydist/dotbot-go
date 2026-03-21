package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

type confirmModel struct {
	risks     []output.RiskItem
	styles    styles
	width     int
	height    int
	viewport  viewport.Model
	confirmed bool
}

func newConfirmModel(risks []output.RiskItem, noColor bool) confirmModel {
	m := confirmModel{
		risks:  risks,
		styles: newStyles(noColor),
		width:  88,
		height: 24,
	}
	m.viewport = viewport.New(m.width, m.viewportHeight())
	m.viewport.SetContent(m.renderContent())
	return m
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = clamp(msg.Width-8, 56, 92)
		m.height = max(msg.Height-4, 12)
		m.viewport.Width = m.width
		m.viewport.Height = m.viewportHeight()
		m.viewport.SetContent(m.renderContent())
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			m.confirmed = true
			return m, tea.Quit
		case "esc", "q", "n", "ctrl+c":
			return m, tea.Quit
		case "j":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k":
			m.viewport.ScrollUp(1)
			return m, nil
		case "g", "home":
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			m.viewport.GotoBottom()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m confirmModel) View() string {
	lines := []string{
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.styles.riskBadge.Render("DANGER"),
			" ",
			m.styles.title.Render("高风险操作确认"),
		),
		m.viewport.View(),
		m.styles.muted.Render(
			fmt.Sprintf("%s/%s 滚动  •  %s/%s 顶部底部  •  %s 确认  •  %s 取消",
				m.styles.key.Render("j"),
				m.styles.key.Render("k"),
				m.styles.key.Render("g"),
				m.styles.key.Render("G"),
				m.styles.key.Render("y"),
				m.styles.key.Render("Esc"),
			),
		),
	}
	return renderSized(m.styles.doc, m.width+m.styles.doc.GetHorizontalFrameSize(), strings.Join(lines, "\n\n"))
}

func (m confirmModel) renderContent() string {
	lines := []string{
		renderSized(m.styles.confirmPanel, m.width, strings.Join([]string{
			"本次执行命中了高风险操作。",
			fmt.Sprintf("风险项数量: %d", len(m.risks)),
			"输入 y 才会继续执行覆盖或清理动作, 取消则本次执行直接终止。",
		}, "\n")),
	}
	for _, risk := range m.risks {
		lines = append(lines, renderSized(m.styles.confirmRisk, m.width, wrapBullet(fmt.Sprintf("%s: %s", risk.Kind, risk.Path), contentWidth(m.styles.confirmRisk, m.width))))
	}
	return strings.Join(lines, "\n\n")
}

func (m confirmModel) viewportHeight() int {
	return max(m.height-6, 6)
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
