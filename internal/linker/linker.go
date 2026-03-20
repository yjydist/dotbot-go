package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"dotbot-go/internal/config"
	"dotbot-go/internal/output"
)

type Result struct {
	Linked   int
	Replaced int
	Skipped  int
	Entries  []output.Entry
}

func Apply(links []config.LinkConfig, dryRun bool) (Result, error) {
	result := Result{}
	for i, link := range links {
		entry, changed, skipped, err := applyOne(link, dryRun)
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

func applyOne(link config.LinkConfig, dryRun bool) (output.Entry, change, bool, error) {
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
		if !dryRun {
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

	info, err := os.Lstat(link.Target)
	if err != nil && !os.IsNotExist(err) {
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

	if os.IsNotExist(err) {
		entry.Decision = "create symlink"
		entry.Status = output.StatusLinked
		if dryRun {
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
		entry.Decision = "replace"
		entry.Status = output.StatusReplaced
		if dryRun {
			if link.Force {
				entry.Message = "force=true"
			}
			return entry, change{replaced: true}, false, nil
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
	entry.Decision = "replace"
	entry.Status = output.StatusReplaced
	entry.Message = "force=true"
	if dryRun {
		return entry, change{replaced: true}, false, nil
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
