package runner

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"dotbot-go/internal/cleaner"
	"dotbot-go/internal/config"
	"dotbot-go/internal/creator"
	"dotbot-go/internal/linker"
	"dotbot-go/internal/output"
)

const (
	exitSuccess = 0
	exitRuntime = 1
	exitConfig  = 2
)

type Options struct {
	ConfigPath string
	DryRun     bool
	OutputMode output.Mode
	NoColor    bool
}

// Run 是 dotbot-go 的主执行入口.
func Run(args []string, stdout, stderr io.Writer) int {
	opts, shouldExit, exitCode, err := parseFlags(args, stdout, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitCode
	}
	if shouldExit {
		return exitCode
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, fmt.Errorf("runtime error: get working directory: %w", err))
		return exitRuntime
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(stderr, fmt.Errorf("runtime error: get home directory: %w", err))
		return exitRuntime
	}

	cfg, err := config.Load(config.LoadOptions{
		Path:       opts.ConfigPath,
		WorkingDir: workingDir,
		HomeDir:    homeDir,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitConfig
	}

	if opts.OutputMode == output.ModeVerbose {
		fmt.Fprintf(stdout, "config: %s\n", cfg.Path)
		fmt.Fprintf(stdout, "base dir: %s\n", cfg.BaseDir)
		fmt.Fprintf(stdout, "stages: create=%d link=%d clean=%d\n", len(cfg.Create.Paths), len(cfg.Links), len(cfg.Clean.Paths))
	}

	createResult, err := creator.Apply(cfg.Create.Paths, cfg.Create.Mode, opts.DryRun)
	output.WriteEntries(stdout, opts.OutputMode, opts.DryRun, createResult.Entries)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	linkResult, err := linker.Apply(cfg.Links, opts.DryRun)
	output.WriteEntries(stdout, opts.OutputMode, opts.DryRun, linkResult.Entries)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	cleanResult, err := cleaner.Apply(*cfg, opts.DryRun)
	output.WriteEntries(stdout, opts.OutputMode, opts.DryRun, cleanResult.Entries)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}

	summary := output.Summary{}
	for _, entry := range createResult.Entries {
		summary.Add(entry.Status)
	}
	for _, entry := range linkResult.Entries {
		summary.Add(entry.Status)
	}
	for _, entry := range cleanResult.Entries {
		summary.Add(entry.Status)
	}
	output.WriteSummary(stdout, opts.OutputMode, summary)
	return exitSuccess
}

func parseFlags(args []string, stdout, stderr io.Writer) (Options, bool, int, error) {
	opts := Options{}
	normalizedArgs := normalizeArgs(args)
	for _, arg := range normalizedArgs {
		if arg == "-h" || arg == "-help" {
			writeHelp(stdout)
			return Options{}, true, exitSuccess, nil
		}
	}

	fs := flag.NewFlagSet("dotbot-go", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.ConfigPath, "config", config.DefaultConfigName, "")
	fs.StringVar(&opts.ConfigPath, "c", config.DefaultConfigName, "")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "")
	verbose := fs.Bool("verbose", false, "")
	quiet := fs.Bool("quiet", false, "")
	fs.BoolVar(&opts.NoColor, "no-color", false, "")
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

func writeHelp(w io.Writer) {
	fmt.Fprintln(w, "dotbot-go - 面向类 Unix 系统的声明式 dotfiles 管理工具")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  dotbot-go [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintf(w, "  -c, --config <path>   配置文件路径, 默认: ./%s\n", config.DefaultConfigName)
	fmt.Fprintln(w, "      --dry-run         仅展示计划动作, 不修改文件系统")
	fmt.Fprintln(w, "      --verbose         输出更详细的信息")
	fmt.Fprintln(w, "      --quiet           仅输出失败信息")
	fmt.Fprintln(w, "      --no-color        关闭彩色输出")
	fmt.Fprintln(w, "  -h, --help            显示帮助")
}
