package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yjydist/dotbot-go/internal/fscheck"
	"github.com/yjydist/dotbot-go/internal/output"
	"github.com/yjydist/dotbot-go/internal/policy"
)

// Result 汇总 clean 阶段的动作统计和输出条目.
type Result struct {
	Deleted int
	Skipped int
	Entries []output.Entry
}

// ApplyOptions 控制 clean 阶段的 dry-run 和高风险 clean 根路径豁免.
type ApplyOptions struct {
	DryRun          bool
	Check           bool
	ProtectedRoots  []string
	AllowRiskyClean bool
}

// Apply 负责遍历 clean.paths, 识别 dead symlink, 并按保守边界决定是否删除.
//
// clean 的设计重点不是“尽可能多删”, 而是“默认保守, 只删用户能解释清楚的失效链接”.
// 所以这里先判断 risky root, 再决定扫描根, 最后才进入逐项删除判断.
func Apply(paths []string, baseDir string, force, recursive bool, opts ApplyOptions) (Result, error) {
	result := Result{}
	for _, root := range paths {
		scanRoot := root
		info, err := os.Lstat(root)
		if err != nil {
			if os.IsNotExist(err) {
				result.Skipped++
				result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusSkipped), Status: output.StatusSkipped, Message: "path missing"})
				continue
			}
			result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
			return result, fmt.Errorf("runtime error: [clean].paths: stat %s: %w", root, err)
		}
		riskyReason := policy.RiskyCleanRootReason(root, info, opts.ProtectedRoots)
		if info.Mode()&os.ModeSymlink != 0 {
			// clean root 允许是 symlink, 但它属于高风险扫描根.
			// 一旦允许继续执行, 真正扫描的仍然应该是解析后的实体目录.
			resolvedRoot, resolveErr := filepath.EvalSymlinks(root)
			if resolveErr != nil {
				result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: resolveErr.Error()})
				return result, fmt.Errorf("runtime error: [clean].paths: resolve %s: %w", root, resolveErr)
			}
			scanRoot = resolvedRoot
			info, err = os.Stat(scanRoot)
			if err != nil {
				result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
				return result, fmt.Errorf("runtime error: [clean].paths: stat %s: %w", scanRoot, err)
			}
		}
		if riskyReason != "" && !opts.AllowRiskyClean && !opts.DryRun && !opts.Check {
			result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: "risky clean requires confirmation"})
			return result, fmt.Errorf("runtime error: [clean].paths: risky clean requires confirmation or --allow-risky-clean: %s", root)
		}
		if !info.IsDir() {
			result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: "path is not a directory"})
			return result, fmt.Errorf("runtime error: [clean].paths: path is not a directory: %s", root)
		}
		scanEntry := output.Entry{Stage: "clean", Target: root, Decision: "scan dead symlinks", Status: output.StatusInfo}
		if riskyReason != "" && !opts.AllowRiskyClean {
			scanEntry.Message = "risky clean, confirmation required"
		}
		result.Entries = append(result.Entries, scanEntry)

		if recursive {
			err = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if path == scanRoot {
					return nil
				}
				entry, deleted, skipped, err := maybeRemoveDeadLink(path, baseDir, force, opts.DryRun, opts.Check)
				result.Deleted += deleted
				result.Skipped += skipped
				if entry != nil {
					result.Entries = append(result.Entries, *entry)
				}
				return err
			})
		} else {
			entries, readErr := os.ReadDir(scanRoot)
			if readErr != nil {
				return result, fmt.Errorf("runtime error: [clean].paths: read %s: %w", scanRoot, readErr)
			}
			for _, entry := range entries {
				out, deleted, skipped, err := maybeRemoveDeadLink(filepath.Join(scanRoot, entry.Name()), baseDir, force, opts.DryRun, opts.Check)
				result.Deleted += deleted
				result.Skipped += skipped
				if out != nil {
					result.Entries = append(result.Entries, *out)
				}
				if err != nil {
					return result, err
				}
			}
		}
		if err != nil {
			return result, fmt.Errorf("runtime error: [clean].paths: walk %s: %w", scanRoot, err)
		}
	}
	return result, nil
}

// maybeRemoveDeadLink 只处理“当前路径本身是 dead symlink”的情况.
// 非 symlink 或目标仍然存在的路径都会被静默跳过.
// force=false 时还会额外检查 dead target 是否仍位于仓库 baseDir 内,
// 这是 clean 默认保守边界的真正执行点.
func maybeRemoveDeadLink(path, baseDir string, force, dryRun, check bool) (entry *output.Entry, deleted, skipped int, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, 0, nil
		}
		return nil, 0, 0, err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return nil, 0, 0, nil
	}
	target, err := os.Readlink(path)
	if err != nil {
		return nil, 0, 0, err
	}
	resolved := target
	if !filepath.IsAbs(target) {
		resolved = filepath.Join(filepath.Dir(path), target)
	}
	resolved = filepath.Clean(resolved)
	if _, err := os.Stat(resolved); err == nil {
		return nil, 0, 0, nil
	} else if !os.IsNotExist(err) {
		return nil, 0, 0, err
	}
	if !force && !isWithinBase(resolved, baseDir) {
		out := output.Entry{Stage: "clean", Target: path, Decision: string(output.StatusSkipped), Status: output.StatusSkipped, Message: "target outside base"}
		return &out, 0, 1, nil
	}
	decision := output.Entry{Stage: "clean", Target: path, Decision: "deleted", Status: output.StatusDeleted}
	if check {
		if err := fscheck.CheckWritableParent(path); err != nil {
			out := output.Entry{Stage: "clean", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()}
			return &out, 0, 0, err
		}
	}
	if dryRun || check {
		decision.Decision = "delete dead symlink"
		return &decision, 1, 0, nil
	}
	if err := os.Remove(path); err != nil {
		out := output.Entry{Stage: "clean", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()}
		return &out, 0, 0, err
	}
	return &decision, 1, 0, nil
}

// isWithinBase 用 filepath.Rel 判断路径是否仍位于仓库 baseDir 内.
func isWithinBase(path, baseDir string) bool {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}
	if rel == "." || rel == "" {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
