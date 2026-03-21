package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// buildConfig 把原始 TOML 结构转换成执行层直接使用的 Config.
// 这里会统一处理 default 归并, 路径解析和重复 target 校验.
func buildConfig(raw rawConfig, meta toml.MetaData, configPath, workingDir, homeDir string) (*Config, error) {
	baseDir := filepath.Dir(configPath)

	defaultCreateMode, err := resolveOptionalMode(raw.Default.Create.Mode, 0o777)
	if err != nil {
		return nil, configError("[default.create].mode", err.Error())
	}

	createMode, err := resolveMode(raw.Create.Mode, raw.Default.Create.Mode, 0o777)
	if err != nil {
		return nil, configError("[create].mode", err.Error())
	}
	cleanForce := boolValue(raw.Clean.Force, raw.Default.Clean.Force, false)
	cleanRecursive := boolValue(raw.Clean.Recursive, raw.Default.Clean.Recursive, false)

	createPaths, err := resolvePaths(raw.Create.Paths, homeDir, workingDir)
	if err != nil {
		return nil, configError("[create].paths", err.Error())
	}
	cleanPaths, err := resolvePaths(raw.Clean.Paths, homeDir, workingDir)
	if err != nil {
		return nil, configError("[clean].paths", err.Error())
	}

	if meta.IsDefined("create") && raw.Create.Paths == nil {
		return nil, configError("[create].paths", "required field is missing")
	}
	if meta.IsDefined("clean") && raw.Clean.Paths == nil {
		return nil, configError("[clean].paths", "required field is missing")
	}

	cfg := &Config{
		Path:    configPath,
		BaseDir: baseDir,
		Default: DefaultConfig{
			Link: LinkDefaults{
				Create:        boolValue(nil, raw.Default.Link.Create, false),
				Relink:        boolValue(nil, raw.Default.Link.Relink, false),
				Force:         boolValue(nil, raw.Default.Link.Force, false),
				Relative:      boolValue(nil, raw.Default.Link.Relative, false),
				IgnoreMissing: boolValue(nil, raw.Default.Link.IgnoreMissing, false),
			},
			Create: CreateDefaults{Mode: defaultCreateMode},
			Clean: CleanDefaults{
				Force:     boolValue(nil, raw.Default.Clean.Force, false),
				Recursive: boolValue(nil, raw.Default.Clean.Recursive, false),
			},
		},
		Create: CreateConfig{Paths: createPaths, Mode: createMode},
		Clean:  CleanConfig{Paths: cleanPaths, Force: cleanForce, Recursive: cleanRecursive},
	}

	targets := make(map[string]int)
	for i, link := range raw.Links {
		cfgLink, err := resolveLink(link, cfg.Default.Link, baseDir, workingDir, homeDir)
		if err != nil {
			return nil, configError(fmt.Sprintf("[[link]][%d]", i+1), err.Error())
		}
		if prev, ok := targets[cfgLink.Target]; ok {
			return nil, configError(
				fmt.Sprintf("[[link]][%d].target", i+1),
				fmt.Sprintf("duplicate target path: %s; already defined at [[link]][%d]", cfgLink.Target, prev),
			)
		}
		targets[cfgLink.Target] = i + 1
		cfg.Links = append(cfg.Links, cfgLink)
	}

	return cfg, nil
}

// resolveLink 会把单个 raw link 配置解析成可执行的 LinkConfig.
func resolveLink(raw rawLinkConfig, defaults LinkDefaults, baseDir, workingDir, homeDir string) (LinkConfig, error) {
	if strings.TrimSpace(raw.Target) == "" {
		return LinkConfig{}, fmt.Errorf("target: required field is missing")
	}
	if strings.TrimSpace(raw.Source) == "" {
		return LinkConfig{}, fmt.Errorf("source: required field is missing")
	}

	target, err := resolvePath(raw.Target, homeDir, workingDir)
	if err != nil {
		return LinkConfig{}, fmt.Errorf("target: %w", err)
	}
	source, err := resolveSource(raw.Source, homeDir, baseDir)
	if err != nil {
		return LinkConfig{}, fmt.Errorf("source: %w", err)
	}

	return LinkConfig{
		Target:        target,
		Source:        source,
		Create:        boolValue(raw.Create, boolPtr(defaults.Create), false),
		Relink:        boolValue(raw.Relink, boolPtr(defaults.Relink), false),
		Force:         boolValue(raw.Force, boolPtr(defaults.Force), false),
		Relative:      boolValue(raw.Relative, boolPtr(defaults.Relative), false),
		IgnoreMissing: boolValue(raw.IgnoreMissing, boolPtr(defaults.IgnoreMissing), false),
	}, nil
}
