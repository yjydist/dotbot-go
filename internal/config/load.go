package config

import (
	"fmt"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load 负责完成配置文件读取, 严格校验和运行期归一化.
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
