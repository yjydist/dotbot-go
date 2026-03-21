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
	minBodyWidth  = 12
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
		m.width = max(msg.Width, 1)
		m.height = max(msg.Height, 1)
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
	outerWidth := m.bodyWidth() + m.styles.doc.GetHorizontalFrameSize()
	docWidth := contentWidth(m.styles.doc, outerWidth)

	parts := []string{
		m.renderTitle(docWidth),
		m.styles.subtitle.Render(strings.Join(wrapByDelimiter(m.headerSummary(), docWidth, "  •  "), "\n")),
		m.viewport.View(),
		m.renderFooter(docWidth),
	}
	return renderSized(m.styles.doc, outerWidth, strings.Join(parts, "\n\n"))
}

func (m reviewModel) bodyWidth() int {
	return clamp(m.width-8, minBodyWidth, maxBodyWidth)
}

func (m reviewModel) viewportHeight() int {
	return max(m.height-8, 1)
}

func (m reviewModel) headerSummary() string {
	switch m.data.Mode {
	case output.ReviewModeCheck:
		return fmt.Sprintf("校验预览  •  风险 %d 项", len(m.data.Risks))
	default:
		return fmt.Sprintf("计划动作 %d 项  •  风险 %d 项", len(m.data.Entries), len(m.data.Risks))
	}
}

func (m reviewModel) renderTitle(width int) string {
	badgeText := reviewTitle(m.data.Mode)
	titleText := filepath.Base(m.data.ConfigPath)
	badge := m.styles.modeBadge.Render(badgeText)
	if displayWidth(badgeText)+1+displayWidth(titleText) <= width {
		return lipgloss.JoinHorizontal(lipgloss.Top, badge, " ", m.styles.title.Render(titleText))
	}

	remaining := width - displayWidth(badgeText) - 1
	if remaining >= 1 {
		lines := wrapText(titleText, remaining)
		if len(lines) > 0 {
			rendered := []string{lipgloss.JoinHorizontal(lipgloss.Top, badge, " ", m.styles.title.Render(lines[0]))}
			for _, line := range lines[1:] {
				rendered = append(rendered, m.styles.title.Render(line))
			}
			return strings.Join(rendered, "\n")
		}
	}

	titleLines := wrapText(titleText, width)
	rendered := []string{badge}
	for _, line := range titleLines {
		rendered = append(rendered, m.styles.title.Render(line))
	}
	return strings.Join(rendered, "\n")
}

func (m reviewModel) renderFooter(width int) string {
	segments := []string{m.styles.key.Render("q") + " 退出"}
	if m.data.Mode == output.ReviewModeDryRun {
		segments = []string{
			m.styles.key.Render("j/k") + " 滚动",
			m.styles.key.Render("g/G") + " 顶部底部",
			m.styles.key.Render("q") + " 退出",
		}
	}
	return m.styles.muted.Render(joinWrappedSegments(segments, "  •  ", width))
}

func joinWrappedSegments(segments []string, delimiter string, width int) string {
	if len(segments) == 0 {
		return ""
	}
	lines := []string{segments[0]}
	current := segments[0]
	for _, segment := range segments[1:] {
		candidate := current + delimiter + segment
		if lipgloss.Width(candidate) <= width {
			current = candidate
			lines[len(lines)-1] = current
			continue
		}
		lines = append(lines, segment)
		current = segment
	}
	return strings.Join(lines, "\n")
}
