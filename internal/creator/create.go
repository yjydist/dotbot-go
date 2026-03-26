package creator

import (
	"fmt"
	"os"

	"github.com/yjydist/dotbot-go/internal/fscheck"
	"github.com/yjydist/dotbot-go/internal/output"
)

// Result 汇总 create 阶段的动作统计和输出条目.
type Result struct {
	Created int
	Entries []output.Entry
}

// Apply 按 [create].paths 的声明创建目录, 并同步产出执行日志条目.
// create 阶段的语义相对单纯:
// - 已存在目录: 跳过
// - 已存在普通文件: 失败
// - 不存在: dry-run 只记录计划, 正式执行才 mkdir
func Apply(paths []string, mode os.FileMode, dryRun, check bool) (Result, error) {
	result := Result{}
	for _, path := range paths {
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err == nil {
			// create 只负责“确保目录存在”, 不负责把普通文件转换成目录.
			if !info.IsDir() {
				result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: "path exists and is not a directory"})
				return result, fmt.Errorf("runtime error: [create].paths: path exists and is not a directory: %s", path)
			}
			result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: string(output.StatusSkipped), Status: output.StatusSkipped, Message: "directory already exists"})
			continue
		}
		if !os.IsNotExist(err) {
			result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
			return result, fmt.Errorf("runtime error: [create].paths: stat %s: %w", path, err)
		}

		if check {
			if err := fscheck.CheckWritableParent(path); err != nil {
				result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
				return result, fmt.Errorf("runtime error: [create].paths: %w", err)
			}
		}

		if dryRun || check {
			// dry-run 里 Created 表示“计划创建”的数量, 不是文件系统已经发生变化.
			result.Created++
			result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: "create", Status: output.StatusCreated})
			continue
		}
		if err := os.MkdirAll(path, mode); err != nil {
			result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
			return result, fmt.Errorf("runtime error: [create].paths: mkdir %s: %w", path, err)
		}
		result.Created++
		result.Entries = append(result.Entries, output.Entry{Stage: "create", Target: path, Decision: "created", Status: output.StatusCreated})
	}
	return result, nil
}
