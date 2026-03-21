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

const (
	defaultWidth  = 100
	defaultHeight = 28
)

type styles struct {
	doc       lipgloss.Style
	header    lipgloss.Style
	subtitle  lipgloss.Style
	section   lipgloss.Style
	content   lipgloss.Style
	footer    lipgloss.Style
	risk      lipgloss.Style
	success   lipgloss.Style
	muted     lipgloss.Style
	key       lipgloss.Style
	tableWrap lipgloss.Style
}

func newStyles(noColor bool) styles {
	base := styles{
		doc:       lipgloss.NewStyle().Padding(1, 2),
		header:    lipgloss.NewStyle().Bold(true),
		subtitle:  lipgloss.NewStyle().Bold(true),
		section:   lipgloss.NewStyle().Bold(true).Underline(true),
		content:   lipgloss.NewStyle(),
		footer:    lipgloss.NewStyle().Italic(true),
		risk:      lipgloss.NewStyle().Bold(true),
		success:   lipgloss.NewStyle().Bold(true),
		muted:     lipgloss.NewStyle().Faint(true),
		key:       lipgloss.NewStyle().Bold(true),
		tableWrap: lipgloss.NewStyle(),
	}
	if noColor {
		return base
	}
	base.header = base.header.Foreground(lipgloss.Color("212"))
	base.subtitle = base.subtitle.Foreground(lipgloss.Color("69"))
	base.section = base.section.Foreground(lipgloss.Color("39"))
	base.footer = base.footer.Foreground(lipgloss.Color("244"))
	base.risk = base.risk.Foreground(lipgloss.Color("203"))
	base.success = base.success.Foreground(lipgloss.Color("42"))
	base.muted = base.muted.Foreground(lipgloss.Color("246"))
	base.key = base.key.Foreground(lipgloss.Color("220"))
	base.tableWrap = base.tableWrap.BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	return base
}

type reviewModel struct {
	data     output.ReviewData
	styles   styles
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

func newReviewModel(data output.ReviewData, noColor bool) reviewModel {
	m := reviewModel{
		data:   data,
		styles: newStyles(noColor),
		width:  defaultWidth,
		height: defaultHeight,
	}
	m.viewport = viewport.New(defaultWidth-6, defaultHeight-10)
	m.viewport.SetContent(m.renderContent())
	m.ready = true
	return m
}

func (m reviewModel) Init() tea.Cmd {
	return nil
}

func (m reviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(msg.Width, 60)
		m.height = max(msg.Height, 16)
		m.viewport.Width = max(m.width-6, 40)
		m.viewport.Height = max(m.height-10, 6)
		m.viewport.SetContent(m.renderContent())
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m reviewModel) View() string {
	var footer string
	switch m.data.Mode {
	case output.ReviewModeDryRun:
		footer = fmt.Sprintf("%s/%s 或方向键滚动, %s 退出", m.styles.key.Render("j"), m.styles.key.Render("k"), m.styles.key.Render("q"))
	case output.ReviewModeCheck:
		footer = fmt.Sprintf("%s 退出", m.styles.key.Render("q"))
	}

	parts := []string{
		m.styles.header.Render(reviewTitle(m.data.Mode)),
		m.styles.subtitle.Render(fmt.Sprintf("config: %s", m.data.ConfigPath)),
		m.styles.subtitle.Render(fmt.Sprintf("base dir: %s", m.data.BaseDir)),
		m.viewport.View(),
		m.styles.footer.Render(footer),
	}
	return m.styles.doc.Width(max(m.width-2, 58)).Render(strings.Join(parts, "\n\n"))
}

func (m reviewModel) renderContent() string {
	lines := []string{
		m.styles.section.Render("概览"),
		fmt.Sprintf("阶段数量: create=%d link=%d clean=%d", m.data.StageCounts.Create, m.data.StageCounts.Link, m.data.StageCounts.Clean),
	}
	if len(m.data.VerboseLines) > 0 {
		lines = append(lines, m.data.VerboseLines...)
	}

	lines = append(lines, "", m.styles.section.Render("风险"))
	if len(m.data.Risks) == 0 {
		lines = append(lines, m.styles.success.Render("无风险项"))
	} else {
		for _, risk := range m.data.Risks {
			lines = append(lines, m.styles.risk.Render(fmt.Sprintf("- %s: %s", risk.Kind, risk.Path)))
		}
	}

	switch m.data.Mode {
	case output.ReviewModeDryRun:
		lines = append(lines, "", m.styles.section.Render("计划动作"))
		table := output.RenderEntryTable(m.data.Entries)
		lines = append(lines, m.styles.tableWrap.Width(max(m.viewport.Width, 40)).Render(table))
		lines = append(lines, "", m.styles.section.Render("摘要"))
		lines = append(lines, fmt.Sprintf("created=%d linked=%d skipped=%d replaced=%d deleted=%d failed=%d",
			m.data.Summary.Created,
			m.data.Summary.Linked,
			m.data.Summary.Skipped,
			m.data.Summary.Replaced,
			m.data.Summary.Deleted,
			m.data.Summary.Failed,
		))
	case output.ReviewModeCheck:
		lines = append(lines, "", m.styles.section.Render("结果"))
		lines = append(lines, m.styles.success.Render(m.data.Result))
	}

	return strings.Join(lines, "\n")
}

type confirmModel struct {
	risks     []output.RiskItem
	styles    styles
	confirmed bool
}

func newConfirmModel(risks []output.RiskItem, noColor bool) confirmModel {
	return confirmModel{
		risks:  risks,
		styles: newStyles(noColor),
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
		m.styles.header.Render("detected risky operations"),
		"",
	}
	for _, risk := range m.risks {
		lines = append(lines, m.styles.risk.Render(fmt.Sprintf("- %s: %s", risk.Kind, risk.Path)))
	}
	lines = append(lines,
		"",
		m.styles.footer.Render(fmt.Sprintf("%s 确认, %s 取消", m.styles.key.Render("Enter"), m.styles.key.Render("Esc"))),
	)
	return m.styles.doc.Render(strings.Join(lines, "\n"))
}

func RunReview(stdin io.Reader, stdout io.Writer, noColor bool, data output.ReviewData) error {
	model := newReviewModel(data, noColor)
	_, err := tea.NewProgram(
		model,
		tea.WithInput(stdin),
		tea.WithOutput(stdout),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return fmt.Errorf("runtime error: review ui: %w", err)
	}
	return nil
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

func reviewTitle(mode output.ReviewMode) string {
	switch mode {
	case output.ReviewModeCheck:
		return "check review"
	default:
		return "dry-run review"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
