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

// confirmModel 和 reviewModel 的区别在于:
// review 是只读审阅, confirm 是有副作用前的最后一道交互确认.
// 因此这里会保留滚动能力, 但操作键尽量收敛, 避免误触.
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
		// 和 reviewModel 一样, confirm UI 也必须跟随真实终端收缩,
		// 否则用户在最需要看清风险的时候反而会被布局裁切.
		m.width = clamp(msg.Width-8, 12, 92)
		m.height = max(msg.Height-4, 1)
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
	outerWidth := m.width + m.styles.doc.GetHorizontalFrameSize()
	docWidth := contentWidth(m.styles.doc, outerWidth)
	footerText := fmt.Sprintf("%s/%s 滚动  •  %s/%s 顶部底部  •  %s 确认  •  %s 取消",
		m.styles.key.Render("j"),
		m.styles.key.Render("k"),
		m.styles.key.Render("g"),
		m.styles.key.Render("G"),
		m.styles.key.Render("y"),
		m.styles.key.Render("Esc"),
	)
	lines := []string{
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.styles.riskBadge.Render("DANGER"),
			" ",
			m.styles.title.Render("高风险操作确认"),
		),
		m.viewport.View(),
		m.styles.muted.Render(strings.Join(wrapByDelimiter(footerText, docWidth, "  •  "), "\n")),
	}
	return renderSized(m.styles.doc, outerWidth, strings.Join(lines, "\n\n"))
}

func (m confirmModel) renderContent() string {
	// 风险摘要和风险列表都进入 viewport, 这样小终端里也能完整滚动查看.
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
	return max(m.height-6, 1)
}

func RunConfirm(stdin io.Reader, stdout io.Writer, noColor bool, risks []output.RiskItem) error {
	// RunConfirm 的返回值只有两类:
	// - UI 本身出错
	// - 用户明确拒绝确认
	// runner 只需要根据 error 决定是否中止执行, 不关心更细的 UI 状态.
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
