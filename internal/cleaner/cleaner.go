package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yjydist/dotbot-go/internal/config"
	"github.com/yjydist/dotbot-go/internal/output"
)

type Result struct {
	Deleted int
	Skipped int
	Entries []output.Entry
}

func Apply(cfg config.Config, dryRun bool) (Result, error) {
	result := Result{}
	for _, root := range cfg.Clean.Paths {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				result.Skipped++
				result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusSkipped), Status: output.StatusSkipped, Message: "path missing"})
				continue
			}
			result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()})
			return result, fmt.Errorf("runtime error: [clean].paths: stat %s: %w", root, err)
		}
		if !info.IsDir() {
			result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: "path is not a directory"})
			return result, fmt.Errorf("runtime error: [clean].paths: path is not a directory: %s", root)
		}
		result.Entries = append(result.Entries, output.Entry{Stage: "clean", Target: root, Decision: "scan dead symlinks", Status: output.StatusInfo})

		if cfg.Clean.Recursive {
			err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if path == root {
					return nil
				}
				entry, deleted, skipped, err := maybeRemoveDeadLink(path, cfg.BaseDir, cfg.Clean.Force, dryRun)
				result.Deleted += deleted
				result.Skipped += skipped
				if entry != nil {
					result.Entries = append(result.Entries, *entry)
				}
				return err
			})
		} else {
			entries, readErr := os.ReadDir(root)
			if readErr != nil {
				return result, fmt.Errorf("runtime error: [clean].paths: read %s: %w", root, readErr)
			}
			for _, entry := range entries {
				out, deleted, skipped, err := maybeRemoveDeadLink(filepath.Join(root, entry.Name()), cfg.BaseDir, cfg.Clean.Force, dryRun)
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
			return result, fmt.Errorf("runtime error: [clean].paths: walk %s: %w", root, err)
		}
	}
	return result, nil
}

func maybeRemoveDeadLink(path, baseDir string, force, dryRun bool) (entry *output.Entry, deleted, skipped int, err error) {
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
		out := output.Entry{Stage: "clean", Target: path, Decision: string(output.StatusSkipped), Status: output.StatusSkipped, Message: "outside base and force=false"}
		return &out, 0, 1, nil
	}
	decision := output.Entry{Stage: "clean", Target: path, Decision: "deleted", Status: output.StatusDeleted}
	if dryRun {
		decision.Decision = "delete dead symlink"
		return &decision, 1, 0, nil
	}
	if err := os.Remove(path); err != nil {
		out := output.Entry{Stage: "clean", Target: path, Decision: string(output.StatusFailed), Status: output.StatusFailed, Message: err.Error()}
		return &out, 0, 0, err
	}
	return &decision, 1, 0, nil
}

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
