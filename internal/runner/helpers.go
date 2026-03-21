package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

// resolveProtectedTargetAllowance 统一处理 protected target 的交互确认和非交互 override.
func resolveProtectedTargetAllowance(stdin io.Reader, stdout io.Writer, opts Options, links []config.LinkConfig, protectedTargets []string) (bool, []string, error) {
	riskyTargets := collectProtectedTargets(links, protectedTargets)
	if len(riskyTargets) == 0 || opts.DryRun || opts.Check || opts.AllowProtectedTarget {
		return opts.AllowProtectedTarget, riskyTargets, nil
	}
	if !interactiveTerminal(stdin, stdout) {
		return false, nil, fmt.Errorf("runtime error: protected target requires confirmation or --allow-protected-target: %s", riskyTargets[0])
	}
	return true, riskyTargets, nil
}

// resolveRiskyCleanAllowance 统一处理 risky clean root 的交互确认和非交互 override.
func resolveRiskyCleanAllowance(stdin io.Reader, stdout io.Writer, opts Options, roots, protectedRoots []string) (bool, []string, error) {
	riskyRoots := collectRiskyCleanRoots(roots, protectedRoots)
	if len(riskyRoots) == 0 || opts.DryRun || opts.Check || opts.AllowRiskyClean {
		return opts.AllowRiskyClean, riskyRoots, nil
	}
	if !interactiveTerminal(stdin, stdout) {
		return false, nil, fmt.Errorf("runtime error: risky clean requires confirmation or --allow-risky-clean: %s", riskyRoots[0])
	}
	return true, riskyRoots, nil
}

// collectProtectedTargets 从 force=true 的 link 中提取需要确认的危险目标.
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

// collectRiskyCleanRoots 只负责发现风险, 不负责确认交互.
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

// collectRiskItems 用于 dry-run/check 审阅界面, 它不会考虑 override 是否已显式放行.
func collectRiskItems(opts Options, protectedTargets, riskyCleanRoots []string) []output.RiskItem {
	items := make([]output.RiskItem, 0, len(protectedTargets)+len(riskyCleanRoots))
	for _, target := range protectedTargets {
		items = append(items, output.RiskItem{
			Kind:    "replace protected target",
			Path:    target,
			Allowed: opts.AllowProtectedTarget,
		})
	}
	for _, root := range riskyCleanRoots {
		items = append(items, output.RiskItem{
			Kind:    "risky clean root",
			Path:    root,
			Allowed: opts.AllowRiskyClean,
		})
	}
	return items
}

// collectConfirmRiskItems 只保留当前仍然需要用户确认的风险项.
func collectConfirmRiskItems(opts Options, protectedTargets, riskyCleanRoots []string) []output.RiskItem {
	items := make([]output.RiskItem, 0, len(protectedTargets)+len(riskyCleanRoots))
	if !opts.AllowProtectedTarget {
		for _, target := range protectedTargets {
			items = append(items, output.RiskItem{Kind: "replace protected target", Path: target})
		}
	}
	if !opts.AllowRiskyClean {
		for _, root := range riskyCleanRoots {
			items = append(items, output.RiskItem{Kind: "risky clean root", Path: root})
		}
	}
	return items
}

// collectReviewEntries 会保留各阶段原有顺序, 方便 dry-run 审阅.
func collectReviewEntries(groups ...[]output.Entry) []output.Entry {
	var entries []output.Entry
	for _, group := range groups {
		entries = append(entries, group...)
	}
	return entries
}

// shouldUseReviewUI 决定当前是否进入 Bubble Tea 审阅界面.
func shouldUseReviewUI(opts Options, stdin io.Reader, stdout io.Writer) bool {
	if opts.OutputMode == output.ModeQuiet {
		return false
	}
	if !opts.DryRun && !opts.Check {
		return false
	}
	return interactiveTerminal(stdin, stdout)
}

// buildVerboseLines 生成“实际生效配置”的简要摘要, 供 verbose 输出和审阅界面复用.
func buildVerboseLines(cfg config.Config) []string {
	linkSummary := buildLinkVerboseSummary(cfg.Links, cfg.Default.Link)

	return []string{
		"link: " + linkSummary,
		fmt.Sprintf("create: mode=%#o",
			cfg.Create.Mode,
		),
		fmt.Sprintf("clean: force=%t recursive=%t",
			cfg.Clean.Force,
			cfg.Clean.Recursive,
		),
	}
}

// buildLinkVerboseSummary 尝试给出“本次 link 阶段实际会怎么执行”的摘要.
// 如果所有 link 的生效布尔值一致, 就直接展示具体值; 否则明确标注为 mixed.
func buildLinkVerboseSummary(links []config.LinkConfig, defaults config.LinkDefaults) string {
	if len(links) == 0 {
		return fmt.Sprintf(
			"create=%t relink=%t force=%t relative=%t ignore_missing=%t",
			defaults.Create,
			defaults.Relink,
			defaults.Force,
			defaults.Relative,
			defaults.IgnoreMissing,
		)
	}

	first := links[0]
	same := true
	for _, link := range links[1:] {
		if link.Create != first.Create ||
			link.Relink != first.Relink ||
			link.Force != first.Force ||
			link.Relative != first.Relative ||
			link.IgnoreMissing != first.IgnoreMissing {
			same = false
			break
		}
	}
	if !same {
		return "mixed per-link values"
	}

	return fmt.Sprintf(
		"create=%t relink=%t force=%t relative=%t ignore_missing=%t",
		first.Create,
		first.Relink,
		first.Force,
		first.Relative,
		first.IgnoreMissing,
	)
}

// buildVerboseReport 是普通执行模式下的前置 verbose 文本块.
func buildVerboseReport(cfg config.Config) []string {
	lines := []string{
		fmt.Sprintf("config: %s", cfg.Path),
		fmt.Sprintf("base dir: %s", cfg.BaseDir),
	}
	lines = append(lines, buildVerboseLines(cfg)...)
	lines = append(lines, fmt.Sprintf("stages: create=%d link=%d clean=%d", len(cfg.Create.Paths), len(cfg.Links), len(cfg.Clean.Paths)))
	return lines
}

// isInteractive 需要同时满足 stdin 和 stdout 都是终端设备.
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
