package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
	"github.com/yjydist/dotbot-go/internal/policy"
)

// resolveProtectedTargetAllowance 统一处理 protected target 的交互确认和非交互 override.
// 返回值里的 bool 表示“执行阶段是否已经被允许覆盖这些危险目标”.
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

// collectProtectedTargets 从 link 阶段里提取需要确认的危险目标.
// 这里既要覆盖 force 替换普通文件/目录, 也要覆盖 relink 替换现有 symlink.
// 后者需要额外 lstat 一次, 因为只有“目标当前确实是 symlink”时才会走 relink 语义.
func collectProtectedTargets(links []config.LinkConfig, protectedTargets []string) []string {
	seen := map[string]struct{}{}
	var risky []string
	for _, link := range links {
		if !link.Force && !link.Relink {
			continue
		}
		if !policy.IsProtectedTarget(link.Target, protectedTargets) {
			continue
		}
		if link.Relink && !link.Force {
			info, err := os.Lstat(link.Target)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				continue
			}
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
		if policy.RiskyCleanRootReason(root, info, protectedRoots) == "" {
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

// collectRiskItems 用于 dry-run/check 审阅界面.
// 即使 override 已显式放行, 这里也会保留风险项, 只是通过 Allowed 标注当前命令已经接管了风险.
// 这样审阅界面反映的是“操作本身仍然危险”, 而不是“命令还能不能继续执行”.
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
// 这份列表直接决定正式执行时是否弹确认 UI.
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

// writeReviewFailure 在 dry-run/check 提前失败时保留已收集到的逐项明细.
// 这样用户即使没进入审阅界面, 也能看到失败前的动作和具体失败目标.
func writeReviewFailure(stdout io.Writer, outOpts output.Options, groups ...[]output.Entry) {
	entries := collectReviewEntries(groups...)
	if len(entries) == 0 {
		return
	}
	output.WriteEntries(stdout, outOpts, entries)

	summary := output.Summary{}
	for _, entry := range entries {
		summary.Add(entry.Status)
	}
	output.WriteSummary(stdout, outOpts, summary)
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

func buildConfigGroups(cfg config.Config) []output.ConfigGroup {
	groups := []output.ConfigGroup{
		buildLinkConfigSummaryGroup(cfg.Links, cfg.Default.Link),
		{Scope: "create", Fields: []output.ConfigField{{Key: "mode", Value: fmt.Sprintf("%#o", cfg.Create.Mode)}}},
		{Scope: "clean", Fields: []output.ConfigField{{Key: "force", Value: fmt.Sprintf("%t", cfg.Clean.Force)}, {Key: "recursive", Value: fmt.Sprintf("%t", cfg.Clean.Recursive)}}},
	}
	return append(groups, buildLinkConfigDetailGroups(cfg.Links)...)
}

func buildLinkConfigSummaryGroup(links []config.LinkConfig, defaults config.LinkDefaults) output.ConfigGroup {
	if len(links) == 0 {
		return output.ConfigGroup{Scope: "link", Fields: []output.ConfigField{{Key: "create", Value: fmt.Sprintf("%t", defaults.Create)}, {Key: "relink", Value: fmt.Sprintf("%t", defaults.Relink)}, {Key: "force", Value: fmt.Sprintf("%t", defaults.Force)}, {Key: "relative", Value: fmt.Sprintf("%t", defaults.Relative)}, {Key: "ignore_missing", Value: fmt.Sprintf("%t", defaults.IgnoreMissing)}}}
	}

	first := links[0]
	if hasMixedLinkValues(links) {
		return output.ConfigGroup{Scope: "link", Fields: []output.ConfigField{{Value: "mixed per-link values"}}}
	}

	return output.ConfigGroup{Scope: "link", Fields: []output.ConfigField{{Key: "create", Value: fmt.Sprintf("%t", first.Create)}, {Key: "relink", Value: fmt.Sprintf("%t", first.Relink)}, {Key: "force", Value: fmt.Sprintf("%t", first.Force)}, {Key: "relative", Value: fmt.Sprintf("%t", first.Relative)}, {Key: "ignore_missing", Value: fmt.Sprintf("%t", first.IgnoreMissing)}}}
}

func buildLinkConfigDetailGroups(links []config.LinkConfig) []output.ConfigGroup {
	if len(links) <= 1 || !hasMixedLinkValues(links) {
		return nil
	}

	groups := make([]output.ConfigGroup, 0, len(links))
	for index, link := range links {
		groups = append(groups, output.ConfigGroup{Scope: fmt.Sprintf("link[%d]", index+1), Fields: []output.ConfigField{{Key: "target", Value: link.Target}, {Key: "create", Value: fmt.Sprintf("%t", link.Create)}, {Key: "relink", Value: fmt.Sprintf("%t", link.Relink)}, {Key: "force", Value: fmt.Sprintf("%t", link.Force)}, {Key: "relative", Value: fmt.Sprintf("%t", link.Relative)}, {Key: "ignore_missing", Value: fmt.Sprintf("%t", link.IgnoreMissing)}}})
	}
	return groups
}

func hasMixedLinkValues(links []config.LinkConfig) bool {
	if len(links) <= 1 {
		return false
	}

	first := links[0]
	for _, link := range links[1:] {
		if link.Create != first.Create ||
			link.Relink != first.Relink ||
			link.Force != first.Force ||
			link.Relative != first.Relative ||
			link.IgnoreMissing != first.IgnoreMissing {
			return true
		}
	}
	return false
}

// buildVerboseReport 是普通执行模式下的前置 verbose 文本块.
func buildVerboseReport(cfg config.Config) []string {
	lines := []string{
		fmt.Sprintf("config: %s", cfg.Path),
		fmt.Sprintf("base dir: %s", cfg.BaseDir),
	}
	for _, group := range buildConfigGroups(cfg) {
		lines = append(lines, output.RenderConfigGroup(group))
	}
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
