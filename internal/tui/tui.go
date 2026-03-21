package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

const (
	defaultWidth  = 100
	defaultHeight = 28
	minBodyWidth  = 56
	maxBodyWidth  = 128
)

type styles struct {
	doc           lipgloss.Style
	title         lipgloss.Style
	subtitle      lipgloss.Style
	panel         lipgloss.Style
	panelTitle    lipgloss.Style
	muted         lipgloss.Style
	key           lipgloss.Style
	stageBadge    lipgloss.Style
	modeBadge     lipgloss.Style
	riskBadge     lipgloss.Style
	safeBadge     lipgloss.Style
	statusOk      lipgloss.Style
	statusWarn    lipgloss.Style
	statusError   lipgloss.Style
	statusInfo    lipgloss.Style
	card          lipgloss.Style
	fieldLabel    lipgloss.Style
	confirmPanel  lipgloss.Style
	confirmRisk   lipgloss.Style
	summaryMetric lipgloss.Style
}

func newStyles(noColor bool) styles {
	base := styles{
		doc:           lipgloss.NewStyle().Padding(1, 2),
		title:         lipgloss.NewStyle().Bold(true),
		subtitle:      lipgloss.NewStyle().Faint(true),
		panel:         lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1),
		panelTitle:    lipgloss.NewStyle().Bold(true),
		muted:         lipgloss.NewStyle().Faint(true),
		key:           lipgloss.NewStyle().Bold(true),
		stageBadge:    lipgloss.NewStyle().Bold(true).Padding(0, 1),
		modeBadge:     lipgloss.NewStyle().Bold(true).Padding(0, 1),
		riskBadge:     lipgloss.NewStyle().Bold(true).Padding(0, 1),
		safeBadge:     lipgloss.NewStyle().Bold(true).Padding(0, 1),
		statusOk:      lipgloss.NewStyle().Bold(true).Padding(0, 1),
		statusWarn:    lipgloss.NewStyle().Bold(true).Padding(0, 1),
		statusError:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
		statusInfo:    lipgloss.NewStyle().Bold(true).Padding(0, 1),
		card:          lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1),
		fieldLabel:    lipgloss.NewStyle().Bold(true),
		confirmPanel:  lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).Padding(0, 1),
		confirmRisk:   lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).Padding(0, 1),
		summaryMetric: lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Padding(0, 1),
	}
	if noColor {
		return base
	}
	base.title = base.title.Foreground(lipgloss.Color("212"))
	base.subtitle = base.subtitle.Foreground(lipgloss.Color("246"))
	base.panel = base.panel.BorderForeground(lipgloss.Color("240"))
	base.panelTitle = base.panelTitle.Foreground(lipgloss.Color("39"))
	base.key = base.key.Foreground(lipgloss.Color("220"))
	base.stageBadge = base.stageBadge.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	base.modeBadge = base.modeBadge.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63"))
	base.riskBadge = base.riskBadge.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("160"))
	base.safeBadge = base.safeBadge.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("34"))
	base.statusOk = base.statusOk.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("35"))
	base.statusWarn = base.statusWarn.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("172"))
	base.statusError = base.statusError.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("160"))
	base.statusInfo = base.statusInfo.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("31"))
	base.card = base.card.BorderForeground(lipgloss.Color("238"))
	base.fieldLabel = base.fieldLabel.Foreground(lipgloss.Color("69"))
	base.confirmPanel = base.confirmPanel.BorderForeground(lipgloss.Color("203"))
	base.confirmRisk = base.confirmRisk.BorderForeground(lipgloss.Color("203"))
	base.summaryMetric = base.summaryMetric.BorderForeground(lipgloss.Color("241"))
	return base
}

type reviewModel struct {
	data     output.ReviewData
	styles   styles
	viewport viewport.Model
	width    int
	height   int
}

// newReviewModel 构造 dry-run / check 共用的只读审阅界面.
func newReviewModel(data output.ReviewData, noColor bool) reviewModel {
	m := reviewModel{
		data:   data,
		styles: newStyles(noColor),
		width:  defaultWidth,
		height: defaultHeight,
	}
	m.viewport = viewport.New(m.bodyWidth(), m.viewportHeight())
	m.viewport.SetContent(m.renderContent())
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
		m.viewport.Width = m.bodyWidth()
		m.viewport.Height = m.viewportHeight()
		m.viewport.SetContent(m.renderContent())
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "g", "home":
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			m.viewport.GotoBottom()
			return m, nil
		case "j":
			m.viewport.ScrollDown(1)
			return m, nil
		case "k":
			m.viewport.ScrollUp(1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m reviewModel) View() string {
	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.styles.modeBadge.Render(reviewTitle(m.data.Mode)),
		" ",
		m.styles.title.Render(filepath.Base(m.data.ConfigPath)),
	)

	footer := m.styles.muted.Render(fmt.Sprintf("%s 退出", m.styles.key.Render("q")))
	if m.data.Mode == output.ReviewModeDryRun {
		footer = m.styles.muted.Render(
			fmt.Sprintf(
				"%s/%s 滚动  •  %s/%s 顶部底部  •  %s 退出",
				m.styles.key.Render("j"),
				m.styles.key.Render("k"),
				m.styles.key.Render("g"),
				m.styles.key.Render("G"),
				m.styles.key.Render("q"),
			),
		)
	}

	parts := []string{
		titleLine,
		m.styles.subtitle.Render(m.headerSummary()),
		m.viewport.View(),
		footer,
	}
	return renderSized(m.styles.doc, m.bodyWidth()+m.styles.doc.GetHorizontalFrameSize(), strings.Join(parts, "\n\n"))
}

func (m reviewModel) bodyWidth() int {
	return clamp(m.width-8, minBodyWidth, maxBodyWidth)
}

func (m reviewModel) viewportHeight() int {
	return max(m.height-8, 6)
}

func (m reviewModel) headerSummary() string {
	switch m.data.Mode {
	case output.ReviewModeCheck:
		return fmt.Sprintf("校验预览  •  风险 %d 项", len(m.data.Risks))
	default:
		return fmt.Sprintf("计划动作 %d 项  •  风险 %d 项", len(m.data.Entries), len(m.data.Risks))
	}
}
