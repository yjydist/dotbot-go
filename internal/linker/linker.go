package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/fscheck"
	"github.com/yjydist/dotbot-go/internal/output"
	"github.com/yjydist/dotbot-go/internal/policy"
)

// Result 汇总 link 阶段的动作统计和输出条目.
type Result struct {
	Linked   int
	Replaced int
	Skipped  int
	Entries  []output.Entry
}

// ApplyOptions 控制 link 阶段的 dry-run 和高风险覆盖边界.
// protected/allow 这组字段由 runner 提前计算好, linker 只负责执行最后一道防线.
type ApplyOptions struct {
	DryRun               bool
	Check                bool
	ProtectedTargets     []string
	AllowProtectedTarget bool
}

// Apply 逐个执行 [[link]] 配置, 并保留与输入顺序一致的日志条目.
// 这里保持“逐项执行, 失败即停”的策略, 因为 link 的副作用本身就和输入顺序强相关.
func Apply(links []config.LinkConfig, opts ApplyOptions) (Result, error) {
	result := Result{}
	for i, link := range links {
		entry, changed, skipped, err := applyOne(link, opts)
		result.Entries = append(result.Entries, entry)
		if err != nil {
			return result, fmt.Errorf("runtime error: [[link]][%d]: %w", i+1, err)
		}
		if skipped {
			result.Skipped++
			continue
		}
		if changed.replaced {
			result.Replaced++
		} else if changed.linked {
			result.Linked++
		}
	}
	return result, nil
}

type change struct {
	linked   bool
	replaced bool
}

// applyOne 实现单个 link 的完整决策流程:
// 校验 source, 处理 create/relink/force, 最终落到创建或替换 symlink.
//
// 这个函数分支很多, 但核心判断顺序是固定的:
// 1. source 是否可用
// 2. target 父目录是否满足 create 语义
// 3. target 当前是缺失 / symlink / 普通文件目录
// 4. 再决定是跳过, 创建, relink, 还是 force 覆盖
func applyOne(link config.LinkConfig, opts ApplyOptions) (output.Entry, change, bool, error) {
	entry := output.Entry{Stage: "link", Target: link.Target, Source: link.Source}
	if _, err := os.Stat(link.Source); err != nil {
		if os.IsNotExist(err) && link.IgnoreMissing {
			entry.Decision = string(output.StatusSkipped)
			entry.Status = output.StatusSkipped
			entry.Message = "source missing, ignore_missing=true"
			return entry, change{}, true, nil
		}
		if os.IsNotExist(err) {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = "source does not exist"
			return entry, change{}, false, fmt.Errorf("source does not exist: %s", link.Source)
		}
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = err.Error()
		return entry, change{}, false, fmt.Errorf("stat source %s: %w", link.Source, err)
	}

	if link.Create {
		parent := filepath.Dir(link.Target)
		if !opts.DryRun {
			if err := os.MkdirAll(parent, 0o777); err != nil {
				entry.Decision = string(output.StatusFailed)
				entry.Status = output.StatusFailed
				entry.Message = err.Error()
				return entry, change{}, false, fmt.Errorf("create parent directory %s: %w", parent, err)
			}
		} else if _, err := os.Stat(parent); err != nil && !os.IsNotExist(err) {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("stat target parent %s: %w", parent, err)
		}
	} else {
		if _, err := os.Stat(filepath.Dir(link.Target)); err != nil {
			if os.IsNotExist(err) {
				entry.Decision = string(output.StatusFailed)
				entry.Status = output.StatusFailed
				entry.Message = "target parent directory does not exist"
				return entry, change{}, false, fmt.Errorf("target parent directory does not exist: %s", filepath.Dir(link.Target))
			}
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("stat target parent %s: %w", filepath.Dir(link.Target), err)
		}
	}

	// 这里先记住“target 是否缺失”, 后面 relative=true 还会复用 err.
	// 如果直接继续使用同一个 err, 很容易把 lstat 的语义覆盖掉.
	info, err := os.Lstat(link.Target)
	targetMissing := os.IsNotExist(err)
	if err != nil && !targetMissing {
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = err.Error()
		return entry, change{}, false, fmt.Errorf("lstat target %s: %w", link.Target, err)
	}

	linkPath := link.Source
	if link.Relative {
		linkPath, err = filepath.Rel(filepath.Dir(link.Target), link.Source)
		if err != nil {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("build relative path: %w", err)
		}
	}

	if targetMissing {
		entry.Decision = "linked"
		entry.Status = output.StatusLinked
		if opts.Check {
			if err := fscheck.CheckWritableParent(link.Target); err != nil {
				entry.Decision = string(output.StatusFailed)
				entry.Status = output.StatusFailed
				entry.Message = err.Error()
				return entry, change{}, false, err
			}
		}
		if opts.DryRun || opts.Check {
			entry.Decision = "create symlink"
			return entry, change{linked: true}, false, nil
		}
		if err := os.Symlink(linkPath, link.Target); err != nil {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("create symlink %s -> %s: %w", link.Target, linkPath, err)
		}
		return entry, change{linked: true}, false, nil
	}

	if info.Mode()&os.ModeSymlink != 0 {
		targetPath, err := os.Readlink(link.Target)
		if err == nil && targetPath == linkPath {
			entry.Decision = string(output.StatusSkipped)
			entry.Status = output.StatusSkipped
			entry.Message = "symlink already matches"
			return entry, change{}, false, nil
		}
		if !link.Relink && !link.Force {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = "target already exists as symlink and relink=false"
			return entry, change{}, false, fmt.Errorf("target already exists as symlink and relink=false: %s", link.Target)
		}
		entry.Decision = "replaced"
		entry.Status = output.StatusReplaced
		if opts.Check {
			if err := fscheck.CheckWritableParent(link.Target); err != nil {
				entry.Decision = string(output.StatusFailed)
				entry.Status = output.StatusFailed
				entry.Message = err.Error()
				return entry, change{}, false, err
			}
		}
		if opts.DryRun || opts.Check {
			entry.Decision = "replace"
			if policy.IsProtectedTarget(link.Target, opts.ProtectedTargets) {
				entry.Message = "protected target, confirmation required"
			} else if link.Force {
				entry.Message = "force=true"
			}
			return entry, change{replaced: true}, false, nil
		}
		// 受保护 symlink 和受保护目录/文件一样, 都必须经过同一套确认护栏.
		if policy.IsProtectedTarget(link.Target, opts.ProtectedTargets) && !opts.AllowProtectedTarget {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = "protected target requires confirmation"
			return entry, change{}, false, fmt.Errorf("protected target requires confirmation or --allow-protected-target: %s", link.Target)
		}
		if err := os.Remove(link.Target); err != nil {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("remove existing symlink %s: %w", link.Target, err)
		}
		if err := os.Symlink(linkPath, link.Target); err != nil {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, fmt.Errorf("create symlink %s -> %s: %w", link.Target, linkPath, err)
		}
		return entry, change{replaced: true}, false, nil
	}

	if !link.Force {
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = "target exists and force=false"
		return entry, change{}, false, fmt.Errorf("target exists and force=false: %s", link.Target)
	}
	entry.Decision = "replaced"
	entry.Status = output.StatusReplaced
	entry.Message = "force=true"
	if opts.Check {
		if err := fscheck.CheckWritableParent(link.Target); err != nil {
			entry.Decision = string(output.StatusFailed)
			entry.Status = output.StatusFailed
			entry.Message = err.Error()
			return entry, change{}, false, err
		}
	}
	if opts.DryRun || opts.Check {
		entry.Decision = "replace"
		if policy.IsProtectedTarget(link.Target, opts.ProtectedTargets) {
			entry.Message = "protected target, confirmation required"
		}
		return entry, change{replaced: true}, false, nil
	}
	if policy.IsProtectedTarget(link.Target, opts.ProtectedTargets) && !opts.AllowProtectedTarget {
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = "protected target requires confirmation"
		return entry, change{}, false, fmt.Errorf("protected target requires confirmation or --allow-protected-target: %s", link.Target)
	}
	if err := os.RemoveAll(link.Target); err != nil {
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = err.Error()
		return entry, change{}, false, fmt.Errorf("remove existing target %s: %w", link.Target, err)
	}
	if err := os.Symlink(linkPath, link.Target); err != nil {
		entry.Decision = string(output.StatusFailed)
		entry.Status = output.StatusFailed
		entry.Message = err.Error()
		return entry, change{}, false, fmt.Errorf("create symlink %s -> %s: %w", link.Target, linkPath, err)
	}
	return entry, change{replaced: true}, false, nil
}
