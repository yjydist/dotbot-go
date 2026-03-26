package runner

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

// parseFlags 只负责参数语义, 不做任何配置加载或文件系统检查.
// 这样 CLI 错误会在进入真正执行流程前就被截断, 不会和运行时错误混在一起.
func parseFlags(args []string, stdout, stderr io.Writer) (Options, bool, int, error) {
	opts := Options{}
	normalizedArgs := normalizeArgs(args)
	for _, arg := range normalizedArgs {
		// Go 的 flag 包会把 -h 视为特殊输入, 这里先手动拦截,
		// 让 --help / -help / -h 都走同一套帮助输出分支.
		if arg == "-h" || arg == "-help" {
			writeHelp(stdout)
			return Options{}, true, exitSuccess, nil
		}
	}

	fs := flag.NewFlagSet("dotbot-go", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.ConfigPath, "config", config.DefaultConfigName, "")
	fs.StringVar(&opts.ConfigPath, "c", config.DefaultConfigName, "")
	fs.BoolVar(&opts.Check, "check", false, "")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "")
	verbose := fs.Bool("verbose", false, "")
	quiet := fs.Bool("quiet", false, "")
	fs.BoolVar(&opts.NoColor, "no-color", false, "")
	fs.BoolVar(&opts.AllowProtectedTarget, "allow-protected-target", false, "")
	fs.BoolVar(&opts.AllowRiskyClean, "allow-risky-clean", false, "")
	showHelp := fs.Bool("help", false, "")
	fs.BoolVar(showHelp, "h", false, "")

	if err := fs.Parse(normalizedArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeHelp(stdout)
			return Options{}, true, exitSuccess, nil
		}
		return Options{}, true, exitConfig, fmt.Errorf("config error: parse flags: %w", err)
	}
	if *showHelp {
		writeHelp(stdout)
		return Options{}, true, exitSuccess, nil
	}
	if *verbose && *quiet {
		return Options{}, true, exitConfig, fmt.Errorf("config error: --verbose and --quiet cannot be used together")
	}
	if fs.NArg() != 0 {
		return Options{}, true, exitConfig, fmt.Errorf("config error: unexpected arguments: %v", fs.Args())
	}
	if *verbose {
		opts.OutputMode = output.ModeVerbose
	}
	if *quiet {
		opts.OutputMode = output.ModeQuiet
	}
	return opts, false, exitSuccess, nil
}

// normalizeArgs 把 --long 形式归一到 flag 包可识别的 -long 形式.
// 这样可以继续使用标准库 flag, 同时又不牺牲常见的 GNU 风格参数写法.
func normalizeArgs(args []string) []string {
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if len(arg) > 2 && strings.HasPrefix(arg, "--") {
			normalized = append(normalized, "-"+arg[2:])
			continue
		}
		normalized = append(normalized, arg)
	}
	return normalized
}

// writeHelp 只关心用户可见的 CLI 帮助文本, 与执行逻辑解耦.
// 这里刻意保持“帮助文案就是契约”的思路, 所以新参数和新行为都应该同步到这里.
func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "dotbot-go - 面向类 Unix 系统的声明式 dotfiles 管理工具")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dotbot-go [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintf(w, "  -c, --config <path>   配置文件路径, 默认: ./%s\n", config.DefaultConfigName)
	fmt.Fprintln(w, "      --check           仅校验配置和关键运行前条件")
	fmt.Fprintln(w, "      --dry-run         仅展示计划动作, 不修改文件系统")
	fmt.Fprintln(w, "      --verbose         输出配置路径, 生效配置摘要, 阶段统计")
	fmt.Fprintln(w, "      --quiet           仅输出失败信息, 不输出成功和摘要")
	fmt.Fprintln(w, "      --no-color        关闭彩色输出")
	fmt.Fprintln(w, "      --allow-protected-target  允许覆盖受保护目标, 高风险")
	fmt.Fprintln(w, "      --allow-risky-clean       允许高风险 clean 根路径, 高风险")
	fmt.Fprintln(w, "  -h, --help            显示帮助")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "交互模式:")
	fmt.Fprintln(w, "  交互终端中, --dry-run 和 --check 会自动进入审阅界面")
	fmt.Fprintln(w, "  非交互环境会回退为纯文本输出")
	fmt.Fprintln(w, "  --quiet 不进入审阅界面")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Path rules:")
	fmt.Fprintln(w, "  source 相对路径基于配置文件目录解析")
	fmt.Fprintln(w, "  target 相对路径基于当前工作目录解析")
	fmt.Fprintln(w, "  source 和 target 都支持 ~ 展开")
}
