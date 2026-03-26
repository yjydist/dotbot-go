package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// buildConfig 把“原始 TOML + 外部环境”压缩成运行期 Config.
//
// 这里是配置层最核心的归一化步骤:
// - 把 default.* 合并到具体阶段配置
// - 统一完成 ~ 和相对路径解析
// - 把 string/bool 指针转换成运行期最终值
// - 提前拦截 duplicate target 这类执行期很难优雅处理的问题
//
// runner 之后看到的 Config 不再保留“字段是显式写的还是从 default 继承的”这类信息,
// 因为执行层只关心最终应该怎么做.
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
		// duplicate target 在配置阶段报错更直接,
		// 否则执行阶段会遇到“后一个 link 覆盖前一个 link”这类不透明行为.
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

// resolveLink 负责把单个 [[link]] 的“原始声明”压缩成执行层最终值.
// 相比 create/clean, link 的维度最多, 所以也最适合在这里一次性完成:
// - 必填字段校验
// - source/target 路径基准差异
// - default.link 布尔开关归并
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
