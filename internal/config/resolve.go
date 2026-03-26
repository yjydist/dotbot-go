package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// validateUndecoded 用来把 TOML 未消费掉的字段直接转成配置错误.
// 这里故意不做“忽略未知字段”的兼容策略, 因为 dotfiles 配置一旦写错,
// 静默忽略通常比直接报错更危险.
func validateUndecoded(meta toml.MetaData) error {
	undecoded := meta.Undecoded()
	if len(undecoded) == 0 {
		return nil
	}
	parts := make([]string, 0, len(undecoded))
	for _, key := range undecoded {
		parts = append(parts, key.String())
	}
	sort.Strings(parts)
	return configError("config", fmt.Sprintf("unknown field or section: %s", strings.Join(parts, ", ")))
}

// resolvePaths 用统一规则解析 [create].paths 和 [clean].paths.
// 两者都以当前工作目录为相对路径基准, 与 [[link]].source 明确区分.
func resolvePaths(paths []string, homeDir, workingDir string) ([]string, error) {
	resolved := make([]string, 0, len(paths))
	for i, path := range paths {
		value, err := resolvePath(path, homeDir, workingDir)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		resolved = append(resolved, value)
	}
	return resolved, nil
}

// resolveSource 和 resolvePath 的区别在于相对路径基准不同:
// source 相对配置文件目录, 其他路径相对当前工作目录.
func resolveSource(path, homeDir, baseDir string) (string, error) {
	expanded, err := expandHome(path, homeDir)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(baseDir, expanded)), nil
}

func resolvePath(path, homeDir, baseDir string) (string, error) {
	expanded, err := expandHome(path, homeDir)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(baseDir, expanded)), nil
}

// expandHome 只支持 ~ 和 ~/... 这两种 home 展开, 不做环境变量替换.
func expandHome(path, homeDir string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("empty path")
	}
	if path == "~" {
		return homeDir, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}
	return path, nil
}

// resolveMode 体现显式值 > fallback > 硬编码默认值 的优先级.
// 这里把优先级逻辑放在配置层, 可以避免执行阶段再反复判断“到底该看哪一层”.
func resolveMode(explicit, fallback string, hardDefault os.FileMode) (os.FileMode, error) {
	if explicit != "" {
		return parseMode(explicit)
	}
	if fallback != "" {
		return parseMode(fallback)
	}
	return hardDefault, nil
}

func resolveOptionalMode(value string, hardDefault os.FileMode) (os.FileMode, error) {
	if value == "" {
		return hardDefault, nil
	}
	return parseMode(value)
}

// parseMode 只接受八进制字符串, 例如 0755.
func parseMode(value string) (os.FileMode, error) {
	if strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("empty mode")
	}
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid mode %q", value)
	}
	return os.FileMode(parsed), nil
}

// boolValue 统一处理 bool 字段的三层优先级归并.
// raw 结构用 *bool 的原因也在这里: nil 才能表达“用户根本没写”.
func boolValue(explicit, fallback *bool, hardDefault bool) bool {
	if explicit != nil {
		return *explicit
	}
	if fallback != nil {
		return *fallback
	}
	return hardDefault
}

func boolPtr(v bool) *bool {
	return &v
}

func configError(path, reason string) error {
	return fmt.Errorf("config error: %s: %s", path, reason)
}
