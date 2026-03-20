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

const DefaultConfigName = "dotbot-go.toml"

type LoadOptions struct {
	Path       string
	WorkingDir string
	HomeDir    string
}

type Config struct {
	Path    string
	BaseDir string
	Default DefaultConfig
	Create  CreateConfig
	Clean   CleanConfig
	Links   []LinkConfig
}

type DefaultConfig struct {
	Link   LinkDefaults
	Create CreateDefaults
	Clean  CleanDefaults
}

type LinkDefaults struct {
	Create        bool
	Relink        bool
	Force         bool
	Relative      bool
	IgnoreMissing bool
}

type CreateDefaults struct {
	Mode os.FileMode
}

type CleanDefaults struct {
	Force     bool
	Recursive bool
}

type CreateConfig struct {
	Paths []string
	Mode  os.FileMode
}

type CleanConfig struct {
	Paths     []string
	Force     bool
	Recursive bool
}

type LinkConfig struct {
	Target        string
	Source        string
	Create        bool
	Relink        bool
	Force         bool
	Relative      bool
	IgnoreMissing bool
}

type rawConfig struct {
	Default rawDefaultConfig `toml:"default"`
	Create  rawCreateConfig  `toml:"create"`
	Clean   rawCleanConfig   `toml:"clean"`
	Links   []rawLinkConfig  `toml:"link"`
}

type rawDefaultConfig struct {
	Link   rawLinkDefaults   `toml:"link"`
	Create rawCreateDefaults `toml:"create"`
	Clean  rawCleanDefaults  `toml:"clean"`
}

type rawLinkDefaults struct {
	Create        *bool `toml:"create"`
	Relink        *bool `toml:"relink"`
	Force         *bool `toml:"force"`
	Relative      *bool `toml:"relative"`
	IgnoreMissing *bool `toml:"ignore_missing"`
}

type rawCreateDefaults struct {
	Mode string `toml:"mode"`
}

type rawCleanDefaults struct {
	Force     *bool `toml:"force"`
	Recursive *bool `toml:"recursive"`
}

type rawCreateConfig struct {
	Paths []string `toml:"paths"`
	Mode  string   `toml:"mode"`
}

type rawCleanConfig struct {
	Paths     []string `toml:"paths"`
	Force     *bool    `toml:"force"`
	Recursive *bool    `toml:"recursive"`
}

type rawLinkConfig struct {
	Target        string `toml:"target"`
	Source        string `toml:"source"`
	Create        *bool  `toml:"create"`
	Relink        *bool  `toml:"relink"`
	Force         *bool  `toml:"force"`
	Relative      *bool  `toml:"relative"`
	IgnoreMissing *bool  `toml:"ignore_missing"`
}

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
		return nil, fmt.Errorf("decode config: %w", err)
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

func buildConfig(raw rawConfig, meta toml.MetaData, configPath, workingDir, homeDir string) (*Config, error) {
	baseDir := filepath.Dir(configPath)

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
			Create: CreateDefaults{Mode: mustMode(raw.Default.Create.Mode, 0o777)},
			Clean:  CleanDefaults{Force: boolValue(nil, raw.Default.Clean.Force, false), Recursive: boolValue(nil, raw.Default.Clean.Recursive, false)},
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
			return nil, configError(fmt.Sprintf("[[link]][%d].target", i+1), fmt.Sprintf("duplicate target path: %s; already defined at [[link]][%d]", cfgLink.Target, prev))
		}
		targets[cfgLink.Target] = i + 1
		cfg.Links = append(cfg.Links, cfgLink)
	}

	return cfg, nil
}

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

func resolveMode(explicit, fallback string, hardDefault os.FileMode) (os.FileMode, error) {
	if explicit != "" {
		return parseMode(explicit)
	}
	if fallback != "" {
		return parseMode(fallback)
	}
	return hardDefault, nil
}

func mustMode(value string, hardDefault os.FileMode) os.FileMode {
	mode, err := resolveMode("", value, hardDefault)
	if err != nil {
		return hardDefault
	}
	return mode
}

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
