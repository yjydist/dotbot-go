package creator

import (
	"fmt"
	"os"

	"github.com/yjydist/dotbot-go/internal/output"
)

// Result 汇总 create 阶段的动作统计和输出条目.
type Result struct {
	Created int
	Entries []output.Entry
}

// Apply 按 [create].paths 的声明创建目录, 并同步产出执行日志条目.
func Apply(paths []string, mode os.FileMode, dryRun bool) (Result, error) {
	result := Result{}
	for _, path := range paths {
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err == nil {
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

		if dryRun {
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
