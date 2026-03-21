package output

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type ReviewMode string

const (
	ReviewModeDryRun ReviewMode = "dry-run"
	ReviewModeCheck  ReviewMode = "check"
)

type StageCounts struct {
	Create int
	Link   int
	Clean  int
}

type RiskItem struct {
	Kind string
	Path string
}

type ReviewData struct {
	Mode         ReviewMode
	ConfigPath   string
	BaseDir      string
	StageCounts  StageCounts
	Entries      []Entry
	Risks        []RiskItem
	Summary      Summary
	Result       string
	VerboseLines []string
}

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
			fmt.Fprintf(w, "    - %s: %s\n", risk.Kind, risk.Path)
		}
	}
	if opts.Mode == ModeVerbose {
		for _, line := range data.VerboseLines {
			fmt.Fprintf(w, "  %s\n", line)
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
	return utf8.RuneCountInString(value)
}
