package runner

import (
	"fmt"
	"io"
	"os"

	"github.com/yjydist/dotbot-go/internal/cleaner"
	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/creator"
	"github.com/yjydist/dotbot-go/internal/linker"
	"github.com/yjydist/dotbot-go/internal/output"
	"github.com/yjydist/dotbot-go/internal/tui"
)

const (
	exitSuccess = 0
	exitRuntime = 1
	exitConfig  = 2
)

// Options 是 runner 在解析 CLI 后使用的执行选项.
type Options struct {
	ConfigPath           string
	Check                bool
	DryRun               bool
	OutputMode           output.Mode
	NoColor              bool
	AllowProtectedTarget bool
	AllowRiskyClean      bool
}

var (
	interactiveTerminal = isInteractive
	runReviewUI         = tui.RunReview
	runConfirmUI        = tui.RunConfirm
)

// Run 是 dotbot-go 的主执行入口.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, os.Stdin, stdout, stderr)
}

// run 负责串起完整执行流程:
// 1. 解析参数和环境
// 2. 加载配置
// 3. 依次执行 create -> link -> clean
// 4. 根据 dry-run/check/normal 三种模式选择输出路径
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

	showReviewUI := shouldUseReviewUI(opts, stdin, stdout)
	verboseLines := buildVerboseLines(*cfg)
	outOpts := output.Options{
		Mode:        opts.OutputMode,
		DryRun:      opts.DryRun,
		EnableColor: output.ColorEnabled(stdout, opts.NoColor),
	}
	if opts.OutputMode == output.ModeVerbose && !showReviewUI && !opts.DryRun && !opts.Check {
		for _, line := range buildVerboseReport(*cfg) {
			fmt.Fprintln(stdout, line)
		}
	}

	// check 复用 dry-run 的执行路径, 但在输出层只展示校验结果.
	dryRun := opts.DryRun || opts.Check
	createResult, err := creator.Apply(cfg.Create.Paths, cfg.Create.Mode, dryRun)
	if !opts.Check && !opts.DryRun {
		output.WriteEntries(stdout, outOpts, createResult.Entries)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	protectedTargets := []string{cfg.BaseDir, workingDir, homeDir}
	allowProtectedTarget, riskyProtectedTargets, err := resolveProtectedTargetAllowance(stdin, stdout, opts, cfg.Links, protectedTargets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	protectedRoots := []string{cfg.BaseDir, workingDir, homeDir}
	allowRiskyClean, riskyCleanRoots, err := resolveRiskyCleanAllowance(stdin, stdout, opts, cfg.Clean.Paths, protectedRoots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	reviewRisks := collectRiskItems(riskyProtectedTargets, riskyCleanRoots)
	confirmRisks := collectConfirmRiskItems(opts, riskyProtectedTargets, riskyCleanRoots)
	if !opts.DryRun && !opts.Check && interactiveTerminal(stdin, stdout) && len(confirmRisks) > 0 {
		if err := runConfirmUI(stdin, stdout, opts.NoColor, confirmRisks); err != nil {
			fmt.Fprintln(stderr, err)
			return exitRuntime
		}
	}
	linkResult, err := linker.Apply(cfg.Links, linker.ApplyOptions{
		DryRun:               dryRun,
		ProtectedTargets:     protectedTargets,
		AllowProtectedTarget: allowProtectedTarget,
	})
	if !opts.Check && !opts.DryRun {
		output.WriteEntries(stdout, outOpts, linkResult.Entries)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitRuntime
	}
	cleanResult, err := cleaner.Apply(*cfg, cleaner.ApplyOptions{
		DryRun:          dryRun,
		ProtectedRoots:  protectedRoots,
		AllowRiskyClean: allowRiskyClean,
	})
	if !opts.Check && !opts.DryRun {
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

	// reviewData 是 dry-run / check 两种审阅视图共享的数据快照.
	reviewData := output.ReviewData{
		ConfigPath:   cfg.Path,
		BaseDir:      cfg.BaseDir,
		StageCounts:  output.StageCounts{Create: len(cfg.Create.Paths), Link: len(cfg.Links), Clean: len(cfg.Clean.Paths)},
		Entries:      collectReviewEntries(createResult.Entries, linkResult.Entries, cleanResult.Entries),
		Risks:        reviewRisks,
		Summary:      summary,
		VerboseLines: verboseLines,
	}

	if opts.Check {
		reviewData.Mode = output.ReviewModeCheck
		reviewData.Result = "check ok"
		if opts.OutputMode != output.ModeQuiet {
			if showReviewUI {
				if err := runReviewUI(stdin, stdout, opts.NoColor, reviewData); err != nil {
					fmt.Fprintln(stderr, err)
					return exitRuntime
				}
			} else {
				output.WriteReviewText(stdout, outOpts, reviewData)
			}
		}
		return exitSuccess
	}

	if opts.DryRun {
		reviewData.Mode = output.ReviewModeDryRun
		if opts.OutputMode != output.ModeQuiet {
			if showReviewUI {
				if err := runReviewUI(stdin, stdout, opts.NoColor, reviewData); err != nil {
					fmt.Fprintln(stderr, err)
					return exitRuntime
				}
			} else {
				output.WriteReviewText(stdout, outOpts, reviewData)
			}
		}
		return exitSuccess
	}
	output.WriteSummary(stdout, outOpts, summary)
	return exitSuccess
}
