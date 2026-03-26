package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFormatEntry(t *testing.T) {
	// 普通成功输出的字段顺序和基本对齐是所有文本日志的基线.
	t.Parallel()

	got := FormatEntry(Options{DryRun: true}, Entry{
		Stage:    "link",
		Target:   "~/.gitconfig",
		Source:   "./git/gitconfig",
		Decision: "create symlink",
		Status:   StatusLinked,
	})
	if !strings.Contains(got, "[dry-run]") {
		t.Fatalf("FormatEntry() = %q, want dry-run prefix", got)
	}
	if !strings.Contains(got, "~/.gitconfig <- ./git/gitconfig") {
		t.Fatalf("FormatEntry() = %q, want target/source", got)
	}
	if !strings.Contains(got, "create symlink") {
		t.Fatalf("FormatEntry() = %q, want decision", got)
	}
}

func TestFormatEntryDryRunFailureUsesFailPrefix(t *testing.T) {
	// 失败优先级必须高于 dry-run, 否则预览失败会被伪装成普通计划项.
	t.Parallel()

	got := FormatEntry(Options{DryRun: true}, Entry{
		Stage:    "link",
		Target:   "~/.gitconfig",
		Decision: "failed",
		Status:   StatusFailed,
		Message:  "target exists and force=false",
	})
	if !strings.Contains(got, "[fail]") {
		t.Fatalf("FormatEntry() = %q, want fail prefix even in dry-run", got)
	}
	if strings.Contains(got, "[dry-run]") {
		t.Fatalf("FormatEntry() = %q, should not use dry-run prefix for failure", got)
	}
}

func TestFormatEntryKeepsCJKColumnsAligned(t *testing.T) {
	// 中文路径下的列宽计算要按终端 cell width, 不能只按 rune 数.
	t.Parallel()

	got := FormatEntry(Options{}, Entry{
		Stage:    "link",
		Target:   "/用户/配置",
		Source:   "/仓库/源文件",
		Decision: "create symlink",
		Status:   StatusLinked,
	})
	if !strings.Contains(got, "/用户/配置 <- /仓库/源文件") {
		t.Fatalf("FormatEntry() = %q, want CJK target/source text", got)
	}
	if !strings.Contains(got, "create symlink") {
		t.Fatalf("FormatEntry() = %q, want decision after padded columns", got)
	}
}

func TestFormatEntryWithColor(t *testing.T) {
	// 启用颜色时, 前缀应该带 ANSI 包装而不影响内容本身.
	t.Parallel()

	got := FormatEntry(Options{EnableColor: true}, Entry{Stage: "create", Target: "~/.cache/zsh", Decision: "created", Status: StatusCreated})
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("FormatEntry() = %q, want ANSI color code", got)
	}
}

func TestWriteEntriesQuietOnlyPrintsFailure(t *testing.T) {
	// quiet 模式只能保留失败项, 这是终端输出层最强的降噪约束.
	t.Parallel()

	var buf bytes.Buffer
	WriteEntries(&buf, Options{Mode: ModeQuiet}, []Entry{
		{Stage: "create", Target: "/tmp/a", Decision: "created", Status: StatusCreated},
		{Stage: "link", Target: "/tmp/b", Decision: "failed", Status: StatusFailed, Message: "boom"},
	})
	got := buf.String()
	if strings.Contains(got, "created") {
		t.Fatalf("WriteEntries() = %q, should not include success entry", got)
	}
	if !strings.Contains(got, "[fail]") {
		t.Fatalf("WriteEntries() = %q, want failure entry", got)
	}
}

func TestFormatEntryNoColor(t *testing.T) {
	// no-color 必须彻底关闭 ANSI 前缀, 方便重定向和日志采集.
	t.Parallel()

	got := FormatEntry(Options{EnableColor: false}, Entry{Stage: "create", Target: "~/.cache/zsh", Decision: "created", Status: StatusCreated})
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("FormatEntry() = %q, should not include ANSI color code", got)
	}
}

func TestRenderEntryTable(t *testing.T) {
	// 非交互审阅文本需要稳定表格布局, 这条用例锁住表头和内容基本形态.
	t.Parallel()

	got := RenderEntryTable([]Entry{
		{Stage: "create", Target: "/tmp/a", Decision: "create", Status: StatusCreated},
		{Stage: "link", Target: "/tmp/b", Source: "/repo/b", Decision: "create symlink", Status: StatusLinked, Message: "protected target, confirmation required"},
	})
	if !strings.Contains(got, "阶段") || !strings.Contains(got, "目标") {
		t.Fatalf("RenderEntryTable() = %q, want table header", got)
	}
	if !strings.Contains(got, "/repo/b") {
		t.Fatalf("RenderEntryTable() = %q, want source column", got)
	}
	if !strings.Contains(got, "protected target, confirmation required") {
		t.Fatalf("RenderEntryTable() = %q, want message column", got)
	}
}

func TestWriteReviewTextDryRun(t *testing.T) {
	// 非交互 dry-run 回退文本需要包含概览, 风险, 明细和摘要.
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{DryRun: true}, ReviewData{
		Mode:        ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: StageCounts{Create: 1, Link: 1, Clean: 0},
		Risks:       []RiskItem{{Kind: "replace protected target", Path: "/tmp/a"}},
		Entries: []Entry{
			{Stage: "link", Target: "/tmp/a", Source: "/repo/a", Decision: "replace", Status: StatusReplaced, Message: "protected target, confirmation required"},
		},
		Summary: Summary{Replaced: 1},
	})

	got := buf.String()
	if !strings.Contains(got, "dry-run:") {
		t.Fatalf("WriteReviewText() = %q, want mode header", got)
	}
	if !strings.Contains(got, "risks: 1") {
		t.Fatalf("WriteReviewText() = %q, want risks summary", got)
	}
	if !strings.Contains(got, "阶段") || !strings.Contains(got, "目标") {
		t.Fatalf("WriteReviewText() = %q, want table output", got)
	}
	if !strings.Contains(got, "summary: created=0 linked=0 skipped=0 replaced=1 deleted=0 failed=0") {
		t.Fatalf("WriteReviewText() = %q, want summary", got)
	}
}

func TestWriteReviewTextCheck(t *testing.T) {
	// check 模式只展示摘要, 不应该退化成完整计划动作列表.
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{}, ReviewData{
		Mode:        ReviewModeCheck,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: StageCounts{Create: 0, Link: 1, Clean: 1},
		Result:      "check ok",
	})

	got := buf.String()
	if !strings.Contains(got, "check:") {
		t.Fatalf("WriteReviewText() = %q, want check header", got)
	}
	if !strings.Contains(got, "result: check ok") {
		t.Fatalf("WriteReviewText() = %q, want check result", got)
	}
	if strings.Contains(got, "阶段 | 目标") {
		t.Fatalf("WriteReviewText() = %q, check should not print action table", got)
	}
}

func TestWriteReviewTextShowsAllowedRiskState(t *testing.T) {
	// 已放行风险不能被隐藏, 但也不能继续写成“仍需确认”.
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{}, ReviewData{
		Mode:  ReviewModeCheck,
		Risks: []RiskItem{{Kind: "replace protected target", Path: "/tmp/a", Allowed: true}},
	})

	if got := buf.String(); !strings.Contains(got, "已通过当前命令放行") {
		t.Fatalf("WriteReviewText() = %q, want allowed risk hint", got)
	}
}

func TestWriteReviewTextVerboseFiltersInactiveStages(t *testing.T) {
	// verbose 审阅文本只显示本次真正参与执行的阶段配置摘要.
	t.Parallel()

	var buf bytes.Buffer
	WriteReviewText(&buf, Options{Mode: ModeVerbose}, ReviewData{
		Mode:        ReviewModeDryRun,
		ConfigPath:  "/repo/dotbot-go.toml",
		BaseDir:     "/repo",
		StageCounts: StageCounts{Link: 1},
		ConfigGroups: []ConfigGroup{
			{Scope: "link", Fields: []ConfigField{{Key: "create", Value: "false"}, {Key: "relink", Value: "false"}, {Key: "force", Value: "false"}, {Key: "relative", Value: "false"}, {Key: "ignore_missing", Value: "false"}}},
			{Scope: "link[1]", Fields: []ConfigField{{Key: "target", Value: "/tmp/a"}, {Key: "create", Value: "false"}, {Key: "relink", Value: "false"}, {Key: "force", Value: "false"}, {Key: "relative", Value: "false"}, {Key: "ignore_missing", Value: "false"}}},
			{Scope: "create", Fields: []ConfigField{{Key: "mode", Value: "0777"}}},
			{Scope: "clean", Fields: []ConfigField{{Key: "force", Value: "true"}, {Key: "recursive", Value: "true"}}},
		},
	})

	got := buf.String()
	if !strings.Contains(got, "link: create=false") {
		t.Fatalf("WriteReviewText() = %q, want active link summary", got)
	}
	if !strings.Contains(got, "link[1]: target=/tmp/a") {
		t.Fatalf("WriteReviewText() = %q, want active per-link summary", got)
	}
	if strings.Contains(got, "create: mode=0777") || strings.Contains(got, "clean: force=true") {
		t.Fatalf("WriteReviewText() = %q, should not include inactive stage summaries", got)
	}
}

func TestDisplayWidthTreatsCJKAsTerminalCells(t *testing.T) {
	// CJK 宽度计算是文本输出和 TUI 共同依赖的基础能力.
	t.Parallel()

	if got, want := displayWidth("阶段"), 4; got != want {
		t.Fatalf("displayWidth() = %d, want %d", got, want)
	}
}
