package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

// WriteReviewText 是非交互环境下的审阅输出回退实现.
// 它和 TUI 共享同一个 ReviewData, 但布局上只追求稳定和可重定向, 不追求复杂交互.
func WriteReviewText(w io.Writer, opts Options, data ReviewData) {
	if opts.Mode == ModeQuiet {
		return
	}

	fmt.Fprintf(w, "%s:\n", data.Mode)
	fmt.Fprintf(w, "  config: %s\n", data.ConfigPath)
	fmt.Fprintf(w, "  base dir: %s\n", data.BaseDir)
	fmt.Fprintf(w, "  stages: create=%d link=%d clean=%d\n", data.StageCounts.Create, data.StageCounts.Link, data.StageCounts.Clean)
	if len(data.Risks) == 0 {
		fmt.Fprintln(w, "  risks: none")
	} else {
		fmt.Fprintf(w, "  risks: %d\n", len(data.Risks))
		for _, risk := range data.Risks {
			suffix := ""
			if risk.Allowed {
				suffix = " (已通过当前命令放行)"
			}
			fmt.Fprintf(w, "    - %s: %s%s\n", risk.Kind, risk.Path, suffix)
		}
	}
	if opts.Mode == ModeVerbose {
		for _, group := range ActiveConfigGroups(data.ConfigGroups, data.StageCounts) {
			fmt.Fprintf(w, "  %s\n", RenderConfigGroup(group))
		}
	}

	switch data.Mode {
	case ReviewModeDryRun:
		if len(data.Entries) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, RenderEntryTable(data.Entries))
		}
		WriteSummary(w, opts, data.Summary)
	case ReviewModeCheck:
		if data.Result != "" {
			fmt.Fprintf(w, "  result: %s\n", data.Result)
		}
	}
}

// RenderEntryTable 用纯文本表格展示 dry-run 的计划动作.
func RenderEntryTable(entries []Entry) string {
	headers := []string{"阶段", "目标", "来源", "动作", "备注"}
	rows := make([][]string, 0, len(entries))
	widths := []int{
		displayWidth(headers[0]),
		displayWidth(headers[1]),
		displayWidth(headers[2]),
		displayWidth(headers[3]),
		displayWidth(headers[4]),
	}

	for _, entry := range entries {
		row := []string{
			entry.Stage,
			entry.Target,
			entry.Source,
			entry.Decision,
			entry.Message,
		}
		if row[2] == "" {
			row[2] = "-"
		}
		if row[4] == "" {
			row[4] = "-"
		}
		rows = append(rows, row)
		for i, cell := range row {
			if w := displayWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	lines := make([]string, 0, len(rows)+2)
	lines = append(lines, renderTableRow(headers, widths))
	lines = append(lines, renderDivider(widths))
	for _, row := range rows {
		lines = append(lines, renderTableRow(row, widths))
	}
	return strings.Join(lines, "\n")
}

func renderTableRow(cells []string, widths []int) string {
	parts := make([]string, 0, len(cells))
	for i, cell := range cells {
		parts = append(parts, padDisplay(cell, widths[i]))
	}
	return strings.Join(parts, " | ")
}

func renderDivider(widths []int) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		parts = append(parts, strings.Repeat("-", width))
	}
	return strings.Join(parts, "-+-")
}

func padDisplay(value string, width int) string {
	current := displayWidth(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}

// ActiveConfigGroups 过滤掉本次运行不会参与执行的阶段配置摘要.
// 这样 verbose 审阅文本不会把“根本不会跑到的阶段配置”也展示出来, 避免误导排查.
func ActiveConfigGroups(groups []ConfigGroup, counts StageCounts) []ConfigGroup {
	filtered := make([]ConfigGroup, 0, len(groups))
	for _, group := range groups {
		switch scopeStage(group.Scope) {
		case "link":
			if counts.Link > 0 {
				filtered = append(filtered, group)
			}
		case "create":
			if counts.Create > 0 {
				filtered = append(filtered, group)
			}
		case "clean":
			if counts.Clean > 0 {
				filtered = append(filtered, group)
			}
		default:
			filtered = append(filtered, group)
		}
	}
	return filtered
}

func RenderConfigGroup(group ConfigGroup) string {
	parts := make([]string, 0, len(group.Fields))
	for _, field := range group.Fields {
		if field.Key == "" {
			parts = append(parts, field.Value)
			continue
		}
		parts = append(parts, field.Key+"="+field.Value)
	}
	if len(parts) == 0 {
		return group.Scope + ": -"
	}
	return group.Scope + ": " + strings.Join(parts, " ")
}

func scopeStage(scope string) string {
	if idx := strings.Index(scope, "["); idx >= 0 {
		return scope[:idx]
	}
	return scope
}
