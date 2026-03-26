package config

import "os"

const DefaultConfigName = "dotbot-go.toml"

// LoadOptions 描述配置解析时依赖的外部环境.
type LoadOptions struct {
	Path       string
	WorkingDir string
	HomeDir    string
}

// Config 是运行阶段使用的最终配置模型.
// 约定是: 只要拿到 Config, 就可以直接执行, 不再需要关心 TOML 解码细节.
// 这也是 config 包与 runner/create/link/clean 的边界.
type Config struct {
	Path    string
	BaseDir string
	Default DefaultConfig
	Create  CreateConfig
	Clean   CleanConfig
	Links   []LinkConfig
}

// DefaultConfig 保存 default.* 归并后的默认值.
type DefaultConfig struct {
	Link   LinkDefaults
	Create CreateDefaults
	Clean  CleanDefaults
}

// LinkDefaults 是 [[link]] 缺失字段的默认值集合.
type LinkDefaults struct {
	Create        bool
	Relink        bool
	Force         bool
	Relative      bool
	IgnoreMissing bool
}

// CreateDefaults 是 [create] 缺失字段的默认值集合.
type CreateDefaults struct {
	Mode os.FileMode
}

// CleanDefaults 是 [clean] 缺失字段的默认值集合.
type CleanDefaults struct {
	Force     bool
	Recursive bool
}

// CreateConfig 是 [create] 在运行期的规范化结果.
type CreateConfig struct {
	Paths []string
	Mode  os.FileMode
}

// CleanConfig 是 [clean] 在运行期的规范化结果.
type CleanConfig struct {
	Paths     []string
	Force     bool
	Recursive bool
}

// LinkConfig 是单个 [[link]] 在运行期的规范化结果.
type LinkConfig struct {
	Target        string
	Source        string
	Create        bool
	Relink        bool
	Force         bool
	Relative      bool
	IgnoreMissing bool
}

// rawConfig 及其子结构只服务于 TOML 解码阶段.
// 这层保留指针和原始字符串, 方便后续显式区分"未填写"和"填写了零值".
// 一旦进入 buildConfig, 这些原始结构就不应该再泄漏到执行层.
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
