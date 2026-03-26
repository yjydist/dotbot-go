package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/yjydist/dotbot-go/internal/output"
)

func reviewTitle(mode output.ReviewMode) string {
	switch mode {
	case output.ReviewModeCheck:
		return "CHECK"
	default:
		return "DRY-RUN"
	}
}

// statusStyle 统一把执行状态映射到 TUI badge 样式.
// review 和 confirm 都依赖这层映射, 避免各处手写颜色判断.
func statusStyle(s styles, status output.Status) lipgloss.Style {
	switch status {
	case output.StatusSkipped, output.StatusReplaced:
		return s.statusWarn
	case output.StatusFailed:
		return s.statusError
	case output.StatusInfo:
		return s.statusInfo
	default:
		return s.statusOk
	}
}

// renderSized 先把内容补齐到目标宽度, 再交给 lipgloss 画边框.
// 这样比直接依赖 Style.Width 的自动布局更稳定, 不容易把边框撑坏.
func renderSized(style lipgloss.Style, outerWidth int, content string) string {
	return style.Render(padBlock(content, contentWidth(style, outerWidth)))
}

func contentWidth(style lipgloss.Style, outerWidth int) int {
	return max(outerWidth-style.GetHorizontalFrameSize(), 1)
}

// wrapBullet 用来把风险项或摘要项渲染成可换行的项目符号列表.
func wrapBullet(value string, width int) string {
	lines := wrapText(value, max(width-2, 1))
	if len(lines) == 0 {
		return "- "
	}
	result := []string{"- " + lines[0]}
	for _, line := range lines[1:] {
		result = append(result, "  "+line)
	}
	return strings.Join(result, "\n")
}

// wrapLabelValue 用于 target/source/action 这类“标签: 值”结构.
// 它会尽量把标签留在首行, 宽度太窄时再整体降级成普通换行文本.
func wrapLabelValue(label, value string, width int, labelStyle lipgloss.Style) string {
	prefix := label + ": "
	if displayWidth(prefix) >= width {
		lines := wrapText(prefix+value, width)
		if len(lines) == 0 {
			return labelStyle.Render(prefix + "-")
		}
		rendered := []string{labelStyle.Render(lines[0])}
		rendered = append(rendered, lines[1:]...)
		return strings.Join(rendered, "\n")
	}
	lines := wrapText(value, max(width-displayWidth(prefix), 1))
	if len(lines) == 0 {
		lines = []string{"-"}
	}
	rendered := []string{labelStyle.Render(prefix) + lines[0]}
	indent := strings.Repeat(" ", displayWidth(prefix))
	for _, line := range lines[1:] {
		rendered = append(rendered, indent+line)
	}
	return strings.Join(rendered, "\n")
}

// wrapText 按终端 cell width 切行, 而不是按 rune 数切行.
// 这样中文路径和双宽字符不会把 panel/card 的右边框撑坏.
func wrapText(value string, width int) []string {
	if width <= 0 || value == "" {
		return []string{value}
	}
	var lines []string
	for _, rawLine := range strings.Split(value, "\n") {
		runes := []rune(rawLine)
		if len(runes) == 0 {
			lines = append(lines, "")
			continue
		}
		start := 0
		for start < len(runes) {
			lineWidth := 0
			end := start
			for end < len(runes) {
				runeWidth := runewidth.RuneWidth(runes[end])
				if runeWidth == 0 {
					runeWidth = 1
				}
				if end > start && lineWidth+runeWidth > width {
					break
				}
				lineWidth += runeWidth
				end++
				if lineWidth >= width {
					break
				}
			}
			lines = append(lines, string(runes[start:end]))
			start = end
		}
	}
	return lines
}

// padBlock 会把多行内容逐行补齐到目标宽度.
// renderSized 依赖它来确保交给 lipgloss 画边框前, 内容块本身已经是稳定宽度.
func padBlock(value string, width int) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			lines[i] = line + strings.Repeat(" ", width-lineWidth)
		}
	}
	return strings.Join(lines, "\n")
}

// wrapByDelimiter 优先按给定分隔符换行, 分隔失败时再退回逐字符切分.
func wrapByDelimiter(value string, width int, delimiter string) []string {
	if width <= 0 || value == "" {
		return []string{value}
	}

	parts := strings.Split(value, delimiter)
	if len(parts) == 1 {
		return wrapText(value, width)
	}

	var lines []string
	current := parts[0]
	for _, part := range parts[1:] {
		candidate := current + delimiter + part
		if displayWidth(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = part
	}
	lines = append(lines, current)

	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, wrapText(line, width)...)
	}
	return wrapped
}

func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}

// max/clamp 只服务于布局计算, 统一收口到这里方便之后调 TUI 宽度策略.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(value, lower, upper int) int {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}
