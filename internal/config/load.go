package config

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load 是配置层唯一公开入口.
// 它负责三件事:
// 1. 固化外部环境依赖, 例如工作目录和 HOME
// 2. 严格解码并拒绝未知字段
// 3. 把原始 TOML 归一化为执行层直接消费的 Config
//
// 设计上刻意不把这些逻辑散落到 runner 或各执行包里,
// 这样 create/link/clean 都可以假设拿到的是“已校验, 已解析路径, 已合并默认值”的配置.
func Load(opts LoadOptions) (*Config, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("config path is required")
	}
	if opts.WorkingDir == "" {
		return nil, fmt.Errorf("working directory is required")
	}
	if opts.HomeDir == "" {
		return nil, fmt.Errorf("home directory is required")
	}

	configPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("config path: %w", err)
	}
	workingDir, err := filepath.Abs(opts.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("working directory: %w", err)
	}

	var raw rawConfig
	meta, err := toml.DecodeFile(configPath, &raw)
	if err != nil {
		return nil, configError("config", fmt.Sprintf("decode config: %v", err))
	}
	if err := validateUndecoded(meta); err != nil {
		return nil, err
	}

	cfg, err := buildConfig(raw, meta, configPath, workingDir, opts.HomeDir)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
