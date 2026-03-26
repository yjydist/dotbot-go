package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
)

// WriteEntries 输出 create/link/clean 每一条执行记录.
// quiet 模式只保留失败项, 其他模式都尽量保持“按执行顺序吐出条目”.
func WriteEntries(w io.Writer, opts Options, entries []Entry) {
	for _, entry := range entries {
		if opts.Mode == ModeQuiet && entry.Status != StatusFailed {
			continue
		}
		fmt.Fprintln(w, FormatEntry(opts, entry))
	}
}

// WriteSummary 输出最终汇总统计.
func WriteSummary(w io.Writer, opts Options, summary Summary) {
	if opts.Mode == ModeQuiet {
		return
	}
	fmt.Fprintf(w, "summary: created=%d linked=%d skipped=%d replaced=%d deleted=%d failed=%d\n", summary.Created, summary.Linked, summary.Skipped, summary.Replaced, summary.Deleted, summary.Failed)
}

// FormatEntry 负责把单条执行记录格式化成统一的终端输出.
// 这里是普通终端输出的唯一格式化入口, 所以 dry-run、失败优先级、颜色映射都集中在这里处理.
func FormatEntry(opts Options, entry Entry) string {
	prefix := "[ok]"
	if entry.Status == StatusFailed {
		prefix = "[fail]"
	} else if opts.DryRun {
		prefix = "[dry-run]"
	} else if entry.Status == StatusInfo {
		prefix = "[info]"
	} else if entry.Status == StatusSkipped {
		prefix = "[skip]"
	}
	prefix = colorize(prefix, entry.Status, opts)
	object := entry.Target
	if entry.Source != "" {
		object = fmt.Sprintf("%s <- %s", entry.Target, entry.Source)
	}
	parts := []string{prefix, pad(entry.Stage, 7), pad(object, 40)}
	if entry.Decision != "" {
		parts = append(parts, entry.Decision)
	}
	if entry.Message != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Message))
	}
	return strings.Join(parts, " ")
}

func pad(value string, width int) string {
	current := runewidth.StringWidth(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}

// ColorEnabled 只在真实终端上启用颜色, 避免污染重定向输出.
func ColorEnabled(w io.Writer, noColor bool) bool {
	if noColor {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// colorize 把状态映射成最小的一组 ANSI 前缀颜色.
func colorize(prefix string, status Status, opts Options) string {
	if !opts.EnableColor {
		return prefix
	}
	color := ""
	switch {
	case status == StatusFailed:
		color = "31"
	case opts.DryRun:
		color = "36"
	case status == StatusInfo:
		color = "34"
	case status == StatusSkipped:
		color = "33"
	default:
		color = "32"
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", color, prefix)
}
