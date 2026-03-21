package runner

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yjydist/dotbot-go/internal/cleaner"
	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/creator"
	"github.com/yjydist/dotbot-go/internal/linker"
	"github.com/yjydist/dotbot-go/internal/output"
)

const (
	exitSuccess = 0
	exitRuntime = 1
	exitConfig  = 2
)

type Options struct {
	ConfigPath           string
	Check                bool
	DryRun               bool
	OutputMode           output.Mode
	NoColor              bool
	AllowProtectedTarget bool
	AllowRiskyClean      bool
}

// Run 是 dotbot-go 的主执行入口.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, os.Stdin, stdout, stderr)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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
		fmt.Fprintf(stdout, "defaults: link(create=%t relink=%t force=%t relative=%t ignore_missing=%t) create(mode=%#o) clean(force=%t recursive=%t)\n",
			cfg.Default.Link.Create,
			cfg.Default.Link.Relink,
			cfg.Default.Link.Force,
			cfg.Default.Link.Relative,
			cfg.Default.Link.IgnoreMissing,
			cfg.Default.Create.Mode,
			cfg.Default.Clean.Force,
			cfg.Default.Clean.Recursive,
		)
		fmt.Fprintf(stdout, "stages: create=%d link=%d clean=%d\n", len(cfg.Create.Paths), len(cfg.Links), len(cfg.Clean.Paths))
	}
	outOpts := output.Options{
		Mode:        opts.OutputMode,
		DryRun:      opts.DryRun,
		EnableColor: output.ColorEnabled(stdout, opts.NoColor),
	}

	dryRun := opts.DryRun || opts.Check
	createResult, err := creator.Apply(cfg.Create.Paths, cfg.Create.Mode, dryRun)
	if !opts.Check {
		output.WriteEntries(stdout, outOpts, createResult.Entries)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	protectedTargets := []string{cfg.BaseDir, workingDir, homeDir}
	allowProtectedTarget, err := resolveProtectedTargetAllowance(stdin, stdout, opts, cfg.Links, protectedTargets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	linkResult, err := linker.Apply(cfg.Links, linker.ApplyOptions{
		DryRun:               dryRun,
		ProtectedTargets:     protectedTargets,
		AllowProtectedTarget: allowProtectedTarget,
	})
	if !opts.Check {
		output.WriteEntries(stdout, outOpts, linkResult.Entries)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	protectedRoots := []string{cfg.BaseDir, workingDir, homeDir}
	allowRiskyClean, err := resolveRiskyCleanAllowance(stdin, stdout, opts, cfg.Clean.Paths, protectedRoots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	cleanResult, err := cleaner.Apply(*cfg, cleaner.ApplyOptions{
		DryRun:          dryRun,
		ProtectedRoots:  protectedRoots,
		AllowRiskyClean: allowRiskyClean,
	})
	if !opts.Check {
		output.WriteEntries(stdout, outOpts, cleanResult.Entries)
	}
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
	if opts.Check {
		if opts.OutputMode != output.ModeQuiet {
			fmt.Fprintln(stdout, "check ok")
		}
		return exitSuccess
	}
	output.WriteSummary(stdout, outOpts, summary)
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
	fmt.Fprintln(w, "      --check           仅校验配置和关键运行前条件")
	fmt.Fprintln(w, "      --dry-run         仅展示计划动作, 不修改文件系统")
	fmt.Fprintln(w, "      --verbose         输出配置路径, 默认值摘要, 阶段统计")
	fmt.Fprintln(w, "      --quiet           仅输出失败信息, 不输出成功和摘要")
	fmt.Fprintln(w, "      --no-color        关闭彩色输出")
	fmt.Fprintln(w, "      --allow-protected-target  允许覆盖受保护目标, 高风险")
	fmt.Fprintln(w, "      --allow-risky-clean       允许高风险 clean 根路径, 高风险")
	fmt.Fprintln(w, "  -h, --help            显示帮助")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Path rules:")
	fmt.Fprintln(w, "  source 相对路径基于配置文件目录解析")
	fmt.Fprintln(w, "  target 相对路径基于当前工作目录解析")
	fmt.Fprintln(w, "  source 和 target 都支持 ~ 展开")
}

func resolveProtectedTargetAllowance(stdin io.Reader, stdout io.Writer, opts Options, links []config.LinkConfig, protectedTargets []string) (bool, error) {
	riskyTargets := collectProtectedTargets(links, protectedTargets)
	if len(riskyTargets) == 0 || opts.DryRun || opts.Check || opts.AllowProtectedTarget {
		return opts.AllowProtectedTarget, nil
	}
	if !isInteractive(stdin, stdout) {
		return false, fmt.Errorf("runtime error: protected target requires confirmation or --allow-protected-target: %s", riskyTargets[0])
	}
	if err := confirmTargets(stdin, stdout, "REPLACE", riskyTargets, "protected target"); err != nil {
		return false, err
	}
	return true, nil
}

func resolveRiskyCleanAllowance(stdin io.Reader, stdout io.Writer, opts Options, roots, protectedRoots []string) (bool, error) {
	riskyRoots := collectRiskyCleanRoots(roots, protectedRoots)
	if len(riskyRoots) == 0 || opts.DryRun || opts.Check || opts.AllowRiskyClean {
		return opts.AllowRiskyClean, nil
	}
	if !isInteractive(stdin, stdout) {
		return false, fmt.Errorf("runtime error: risky clean requires confirmation or --allow-risky-clean: %s", riskyRoots[0])
	}
	if err := confirmTargets(stdin, stdout, "CLEAN", riskyRoots, "risky clean"); err != nil {
		return false, err
	}
	return true, nil
}

func collectProtectedTargets(links []config.LinkConfig, protectedTargets []string) []string {
	seen := map[string]struct{}{}
	var risky []string
	for _, link := range links {
		if !link.Force {
			continue
		}
		if !linkerProtectedTarget(link.Target, protectedTargets) {
			continue
		}
		target := filepath.Clean(link.Target)
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		risky = append(risky, target)
	}
	return risky
}

func collectRiskyCleanRoots(roots, protectedRoots []string) []string {
	seen := map[string]struct{}{}
	var risky []string
	for _, root := range roots {
		info, err := os.Lstat(root)
		if err != nil {
			continue
		}
		if cleanerRiskyRoot(root, info, protectedRoots) == "" {
			continue
		}
		cleanedRoot := filepath.Clean(root)
		if _, ok := seen[cleanedRoot]; ok {
			continue
		}
		seen[cleanedRoot] = struct{}{}
		risky = append(risky, cleanedRoot)
	}
	return risky
}

func confirmTargets(stdin io.Reader, stdout io.Writer, action string, targets []string, reason string) error {
	reader := bufio.NewReader(stdin)
	for _, target := range targets {
		expected := fmt.Sprintf("CONFIRM %s %s", action, target)
		fmt.Fprintf(stdout, "risk: %s: %s\n", reason, target)
		fmt.Fprintf(stdout, "type %q to continue: ", expected)
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("runtime error: confirmation input: %w", err)
		}
		if strings.TrimSpace(line) != expected {
			return fmt.Errorf("runtime error: confirmation rejected: %s", target)
		}
	}
	return nil
}

func isInteractive(stdin io.Reader, stdout io.Writer) bool {
	return isTerminal(stdin) && isTerminal(stdout)
}

func isTerminal(v any) bool {
	file, ok := v.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func linkerProtectedTarget(target string, protectedTargets []string) bool {
	cleanedTarget := filepath.Clean(target)
	if cleanedTarget == string(filepath.Separator) {
		return true
	}
	for _, path := range protectedTargets {
		if path == "" {
			continue
		}
		if cleanedTarget == filepath.Clean(path) {
			return true
		}
	}
	return false
}

func cleanerRiskyRoot(root string, info os.FileInfo, protectedRoots []string) string {
	if info.Mode()&os.ModeSymlink != 0 {
		return "clean root is symlink"
	}
	cleanedRoot := filepath.Clean(root)
	if cleanedRoot == string(filepath.Separator) {
		return "clean root is protected"
	}
	for _, path := range protectedRoots {
		if path == "" {
			continue
		}
		if cleanedRoot == filepath.Clean(path) {
			return "clean root is protected"
		}
	}
	return ""
}
