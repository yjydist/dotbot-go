package policy

import (
	"os"
	"path/filepath"
)

func IsProtectedTarget(target string, protectedTargets []string) bool {
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

func RiskyCleanRootReason(root string, info os.FileInfo, protectedRoots []string) string {
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
