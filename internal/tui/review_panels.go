package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/yjydist/dotbot-go/internal/output"
)

// renderContent 只负责把各个 panel 拼成滚动区内容, 不处理页头和页脚.
func (m reviewModel) renderContent() string {
	sections := []string{
		m.renderOverviewPanel(),
		m.renderRiskPanel(),
	}

	switch m.data.Mode {
	case output.ReviewModeDryRun:
		sections = append(sections, m.renderEntrySection(), m.renderSummaryPanel())
	case output.ReviewModeCheck:
		sections = append(sections, m.renderCheckPanel())
	}

	return strings.Join(sections, "\n\n")
}

// renderOverviewPanel 展示配置定位信息和本次运行真正生效的配置字段.
func (m reviewModel) renderOverviewPanel() string {
	outerWidth := m.bodyWidth()
	innerWidth := contentWidth(m.styles.panel, outerWidth)
	rows := []tableRow{
		{Label: "config file", Value: m.data.ConfigPath},
		{Label: "base dir", Value: m.data.BaseDir},
		{Label: "create count", Value: fmt.Sprintf("%d", m.data.StageCounts.Create)},
		{Label: "link count", Value: fmt.Sprintf("%d", m.data.StageCounts.Link)},
		{Label: "clean count", Value: fmt.Sprintf("%d", m.data.StageCounts.Clean)},
	}
	rows = append(rows, m.effectiveConfigRows()...)

	lines := []string{
		m.styles.panelTitle.Render("概览"),
		m.renderOverviewTable(rows, innerWidth),
	}
	if outerWidth < 14 {
		return strings.Join(lines, "\n")
	}
	return renderSized(m.styles.panel, outerWidth, strings.Join(lines, "\n"))
}

func (m reviewModel) renderRiskPanel() string {
	outerWidth := m.bodyWidth()
	innerWidth := contentWidth(m.styles.panel, outerWidth)

	lines := []string{m.styles.panelTitle.Render("风险")}
	if len(m.data.Risks) == 0 {
		if innerWidth < 12 {
			lines = append(lines, "无风险项")
		} else {
			lines = append(lines, m.styles.safeBadge.Render("SAFE")+" 无风险项")
		}
	} else {
		if innerWidth < 12 {
			lines = append(lines, fmt.Sprintf("风险项: %d", len(m.data.Risks)))
		} else {
			lines = append(lines, m.styles.riskBadge.Render(fmt.Sprintf("%d 项风险", len(m.data.Risks))))
		}
		for _, risk := range m.data.Risks {
			label := fmt.Sprintf("%s: %s", risk.Kind, risk.Path)
			if risk.Allowed {
				label += " (已放行)"
			}
			lines = append(lines, wrapBullet(label, innerWidth))
		}
	}
	if outerWidth < 14 {
		return strings.Join(lines, "\n")
	}
	return renderSized(m.styles.panel, outerWidth, strings.Join(lines, "\n"))
}

func (m reviewModel) renderEntrySection() string {
	if len(m.data.Entries) == 0 {
		return renderSized(m.styles.panel, m.bodyWidth(), strings.Join([]string{
			m.styles.panelTitle.Render("计划动作"),
			m.styles.muted.Render("没有待展示的计划动作"),
		}, "\n"))
	}

	cards := make([]string, 0, len(m.data.Entries)+1)
	cards = append(cards, m.styles.panelTitle.Render("计划动作"))
	cardWidth := reviewCardWidth(m.bodyWidth())
	for i, entry := range m.data.Entries {
		cards = append(cards, m.renderEntryCard(i+1, entry, cardWidth))
	}
	return strings.Join(cards, "\n\n")
}

// renderEntryCard 是长路径场景下比纯表格更易读的计划动作展示形式.
func (m reviewModel) renderEntryCard(index int, entry output.Entry, outerWidth int) string {
	if outerWidth < 16 {
		return m.renderCompactEntryCard(index, entry, outerWidth)
	}
	innerWidth := contentWidth(m.styles.card, outerWidth)
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.styles.stageBadge.Render(strings.ToUpper(entry.Stage)),
		" ",
		reviewEntryBadgeStyle(m.styles, m.data.Mode, entry).Render(reviewEntryBadgeLabel(m.data.Mode, entry)),
		" ",
		m.styles.muted.Render(fmt.Sprintf("#%02d", index)),
	)

	lines := []string{
		header,
		wrapLabelValue("target", entry.Target, innerWidth, m.styles.fieldLabel),
	}
	if entry.Source != "" {
		lines = append(lines, wrapLabelValue("source", entry.Source, innerWidth, m.styles.fieldLabel))
	}
	if entry.Decision != "" {
		lines = append(lines, wrapLabelValue("action", entry.Decision, innerWidth, m.styles.fieldLabel))
	}
	if entry.Message != "" {
		lines = append(lines, wrapLabelValue("note", entry.Message, innerWidth, m.styles.fieldLabel))
	}

	return renderSized(m.styles.card, outerWidth, strings.Join(lines, "\n"))
}

func (m reviewModel) renderCompactEntryCard(index int, entry output.Entry, width int) string {
	header := fmt.Sprintf("#%02d %s %s", index, strings.ToUpper(entry.Stage), reviewEntryBadgeLabel(m.data.Mode, entry))
	lines := wrapText(header, max(width, 1))
	lines = append(lines, wrapLabelValue("target", entry.Target, width, m.styles.fieldLabel))
	if entry.Source != "" {
		lines = append(lines, wrapLabelValue("source", entry.Source, width, m.styles.fieldLabel))
	}
	if entry.Decision != "" {
		lines = append(lines, wrapLabelValue("action", entry.Decision, width, m.styles.fieldLabel))
	}
	if entry.Message != "" {
		lines = append(lines, wrapLabelValue("note", entry.Message, width, m.styles.fieldLabel))
	}
	return strings.Join(lines, "\n")
}

func (m reviewModel) renderSummaryPanel() string {
	outerWidth := m.bodyWidth()
	innerWidth := contentWidth(m.styles.panel, outerWidth)
	summaryLine := fmt.Sprintf(
		"created=%d  linked=%d  skipped=%d  replaced=%d  deleted=%d  failed=%d",
		m.data.Summary.Created,
		m.data.Summary.Linked,
		m.data.Summary.Skipped,
		m.data.Summary.Replaced,
		m.data.Summary.Deleted,
		m.data.Summary.Failed,
	)

	lines := []string{
		m.styles.panelTitle.Render("摘要"),
		strings.Join(wrapByDelimiter(summaryLine, innerWidth, "  "), "\n"),
	}
	if outerWidth < 14 {
		return strings.Join(lines, "\n")
	}
	return renderSized(m.styles.panel, outerWidth, strings.Join(lines, "\n"))
}

func (m reviewModel) renderCheckPanel() string {
	outerWidth := m.bodyWidth()
	lines := []string{
		m.styles.panelTitle.Render("结果"),
		statusStyle(m.styles, output.StatusCreated).Render(strings.ToUpper(m.data.Result)),
	}
	actionableRisks := 0
	allowedRisks := 0
	for _, risk := range m.data.Risks {
		if risk.Allowed {
			allowedRisks++
		} else {
			actionableRisks++
		}
	}
	switch {
	case actionableRisks > 0:
		lines = append(lines, m.styles.muted.Render("存在高风险项, 正式执行时仍需确认或显式 override"))
	case allowedRisks > 0:
		lines = append(lines, m.styles.muted.Render("存在高风险项, 但已由当前命令的 override 显式放行"))
	default:
		lines = append(lines, m.styles.muted.Render("配置和关键前置检查通过"))
	}
	if outerWidth < 14 {
		return strings.Join(lines, "\n")
	}
	return renderSized(m.styles.panel, outerWidth, strings.Join(lines, "\n"))
}

func (m reviewModel) effectiveConfigLines() []string {
	return output.ActiveVerboseLines(m.data.VerboseLines, m.data.StageCounts)
}

// effectiveConfigRows 会把 link/create/clean 的配置摘要展开成字段级行.
// 这样既便于表格展示, 也避免把一整串配置挤进同一个单元格.
func (m reviewModel) effectiveConfigRows() []tableRow {
	rows := make([]tableRow, 0, len(m.data.VerboseLines)*4)
	for _, line := range m.effectiveConfigLines() {
		group, value, ok := strings.Cut(line, ": ")
		if !ok {
			rows = append(rows, tableRow{Label: "config", Value: line})
			continue
		}
		if !strings.Contains(value, "=") {
			rows = append(rows, tableRow{Label: group, Value: value})
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			rows = append(rows, tableRow{Label: group, Value: "-"})
			continue
		}
		for _, field := range fields {
			name, fieldValue, ok := strings.Cut(field, "=")
			if !ok {
				rows = append(rows, tableRow{Label: group, Value: field})
				continue
			}
			rows = append(rows, tableRow{
				Label: group + "." + name,
				Value: fieldValue,
			})
		}
	}
	return rows
}

type tableRow struct {
	Label string
	Value string
}

// renderOverviewTable 用 bubbles/table 渲染概览区的字段表.
func (m reviewModel) renderOverviewTable(rows []tableRow, width int) string {
	if len(rows) == 0 {
		return ""
	}
	if width < 16 {
		return m.renderOverviewList(rows, width)
	}

	labelWidth := lipgloss.Width("字段")
	for _, row := range rows {
		if w := lipgloss.Width(row.Label); w > labelWidth {
			labelWidth = w
		}
	}
	maxLabelWidth := max(width/3, 1)
	if labelWidth > maxLabelWidth {
		labelWidth = maxLabelWidth
	}
	valueWidth := max(width-labelWidth-7, 1)

	columns := []table.Column{
		{Title: "字段", Width: labelWidth},
		{Title: "值", Width: valueWidth},
	}
	tableRows := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		valueLines := wrapByDelimiter(row.Value, valueWidth, " ")
		if len(valueLines) == 0 {
			valueLines = []string{"-"}
		}
		tableRows = append(tableRows, table.Row{row.Label, valueLines[0]})
		for _, line := range valueLines[1:] {
			tableRows = append(tableRows, table.Row{"", line})
		}
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(tableRows),
		table.WithHeight(len(tableRows)+1),
	)
	tbl.SetStyles(table.Styles{
		Header: m.styles.fieldLabel.
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle().Padding(0, 1),
	})
	tbl.Blur()
	tbl.SetWidth(width)
	return tbl.View()
}

func (m reviewModel) renderOverviewList(rows []tableRow, width int) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, wrapLabelValue(row.Label, row.Value, width, m.styles.fieldLabel))
	}
	return strings.Join(lines, "\n")
}

func reviewCardWidth(bodyWidth int) int {
	width := bodyWidth - 4
	if width < 12 {
		return bodyWidth
	}
	if width > bodyWidth {
		return bodyWidth
	}
	return width
}

func reviewEntryBadgeLabel(mode output.ReviewMode, entry output.Entry) string {
	if mode == output.ReviewModeDryRun {
		switch entry.Status {
		case output.StatusFailed:
			return "FAILED"
		case output.StatusSkipped:
			return "SKIPPED"
		case output.StatusInfo:
			return "INFO"
		default:
			return "PLANNED"
		}
	}
	return strings.ToUpper(string(entry.Status))
}

func reviewEntryBadgeStyle(s styles, mode output.ReviewMode, entry output.Entry) lipgloss.Style {
	if mode == output.ReviewModeDryRun {
		switch entry.Status {
		case output.StatusFailed:
			return s.statusError
		case output.StatusSkipped:
			return s.statusWarn
		default:
			return s.statusInfo
		}
	}
	return statusStyle(s, entry.Status)
}
