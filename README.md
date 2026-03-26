# dotbot-go

`dotbot-go` 是一个使用 Go 实现的 dotfiles 安装工具.

它受到 [Dotbot](https://github.com/anishathalye/dotbot) 启发, 但不会完整复刻 Dotbot 的全部能力. `dotbot-go` 有意保持更小的范围, 只专注于 dotfiles 的安装和同步, 不实现 plugin 系统, 也不提供 shell 执行能力.

如果你要阅读源码, 可以先看 [`docs/code-reading-guide.md`](docs/code-reading-guide.md).

## 项目简介

管理 dotfiles 的目标通常很简单:

```sh
git clone <your-dotfiles-repo> ~/.dotfiles
cd ~/.dotfiles
./install
```

`dotbot-go` 关注的就是这一步.

它负责把版本管理中的配置文件, 按照一份清晰的声明式配置, 安装到目标系统中. 当前聚焦的能力只有 4 个:

- `link`
- `create`
- `clean`
- `default`

它不替代 Git, 也不试图解决系统配置管理的所有问题.

## 项目边界

`dotbot-go` 当前的设计边界非常明确:

- 配置格式使用 TOML
- 配置模型是声明式的
- 只支持符号链接, 不支持 hardlink
- 第一版目标平台是 macOS 和 Linux

明确不做的内容:

- plugin 系统
- shell 指令
- glob 批量匹配
- 环境变量展开
- 回滚承诺

如果用户显式设置 `force = true`, 就表示用户自己承担覆盖风险.

## 快速开始

最小示例:

```toml
[[link]]
target = "~/.gitconfig"
source = "./git/gitconfig"
```

更完整的示例:

```toml
[default.link]
create = true
relink = true
relative = true

[default.create]
mode = "0755"

[default.clean]
force = false
recursive = false

[create]
paths = [
  "~/.cache/zsh",
  "~/.local/share/nvim",
]
mode = "0700"

[clean]
paths = [
  "~",
  "~/.config",
]
recursive = true

[[link]]
target = "~/.gitconfig"
source = "./git/gitconfig"

[[link]]
target = "~/.zshrc"
source = "../shared/zsh/zshrc"

[[link]]
target = "~/.config/nvim"
source = "/Users/example/dotfiles/nvim"
create = true
relative = false

[[link]]
target = "./local-test/tmux.conf"
source = "../../tmux/tmux.conf"
force = true

[[link]]
target = "~/.config/ghostty/config"
source = "./ghostty/config"
ignore_missing = true
```

运行方式:

```sh
dotbot-go -c dotbot-go.toml
dotbot-go --dry-run -c dotbot-go.toml
```

当前支持的 CLI 参数:

- `-c`, `--config`
- `--check`
- `--dry-run`
- `--verbose`
- `--quiet`
- `--no-color`
- `--allow-protected-target`
- `--allow-risky-clean`

其中:

- 不传 `-c` 时, 默认读取当前工作目录下的 `dotbot-go.toml`
- `--check` 只校验配置和关键运行前条件, 不修改文件系统
- `--verbose` 和 `--quiet` 互斥
- 交互终端里, `--dry-run` 和 `--check` 会自动进入审阅界面; 非交互环境回退为纯文本输出
- `--verbose` 会在文本回退中额外展示生效配置摘要; 如果 `link` 的生效值彼此不一致, 还会补充逐项摘要
- 终端环境下默认允许彩色输出, `--no-color` 可关闭
- 命中受保护目标时, 交互环境会进入风险确认界面, 非交互环境需显式传入 `--allow-protected-target`
- 命中高风险 clean 根路径时, 交互环境会进入风险确认界面, 非交互环境需显式传入 `--allow-risky-clean`

## 推荐目录结构

```text
dotfiles/
├── install
├── dotbot-go.toml
├── git/
│   └── gitconfig
├── shell/
│   ├── zshrc
│   └── aliases
├── nvim/
│   └── init.lua
└── tmux/
    └── tmux.conf
```

这种结构的优点是:

- 按工具或场景分组, 容易维护
- 仓库路径和目标路径的映射更清楚
- 后续增加配置时不容易混乱

## 配置格式

`dotbot-go` 的配置文件是声明式的.

它描述的是"最终希望系统处于什么状态", 而不是"按什么顺序执行哪些任务". 内部如果需要执行顺序, 由工具自己固定为:

1. `create`
2. `link`
3. `clean`

当前支持的顶层 section:

- `[default.link]`
- `[default.create]`
- `[default.clean]`
- `[create]`
- `[clean]`
- `[[link]]`

## 路径规则

- `source` 可以使用相对路径或绝对路径
- `source` 为相对路径时, 按标准相对路径语义相对于配置文件目录解析
- `source` 可以写成 `./foo/bar`, `../foo/bar`, `../../foo/bar`
- `target` 可以使用相对路径, 绝对路径, 或 `~`
- `target` 为相对路径时, 相对于当前工作目录解析
- `target` 和 `source` 都支持 `~` 展开
- 不支持环境变量展开

## 内置指令

### `[[link]]`

用于把文件或目录链接到目标位置.

字段:

- `target`: 必填
- `source`: 必填
- `create`: 选填, 默认 `false`
- `relink`: 选填, 默认 `false`
- `force`: 选填, 默认 `false`
- `relative`: 选填, 默认 `false`
- `ignore_missing`: 选填, 默认 `false`

规则:

- 只支持 symlink
- `relink` 只处理已存在的符号链接
- `force` 优先级高于 `relink`
- `force = true` 表示用户接受覆盖风险
- `force = true` 可覆盖普通文件, 目录, 或符号链接
- 命中受保护目标时, 交互环境会进入风险确认界面, 非交互环境需显式传入 `--allow-protected-target`
- `ignore_missing = true` 时, 缺失 `source` 会被跳过而不是报错

### `[create]`

用于创建目录.

字段:

- `paths`: 必填, 但允许为空数组
- `mode`: 选填, 默认 `0777`

### `[clean]`

用于清理失效链接.

字段:

- `paths`: 必填, 但允许为空数组
- `force`: 选填, 默认 `false`
- `recursive`: 选填, 默认 `false`

规则:

- 默认只清理 dead target 解析后仍位于仓库基准目录内的失效链接
- `force = true` 时, 允许清理 dead target 位于仓库基准目录外的失效链接
- `clean.paths` 如果命中高风险根路径, 交互环境会进入风险确认界面, 非交互环境需显式传入 `--allow-risky-clean`

### `[default.*]`

用于为特定能力提供默认值.

支持的分组:

- `[default.link]`: `create`, `relink`, `force`, `relative`, `ignore_missing`
- `[default.create]`: `mode`
- `[default.clean]`: `force`, `recursive`

优先级固定为:

```text
显式配置 > default > 项目内置默认值
```

## 校验与错误处理

在真正修改文件系统之前, 应先完成配置校验和关键运行前检查.

例如:

- 必填字段缺失时直接失败
- 未知字段或未知 section 直接失败
- 重复 `target` 直接失败
- `source` 不存在时按 `ignore_missing` 决定跳过或失败
- `[create].paths` 和 `[clean].paths` 可以为空数组, 表示当前没有对应操作

运行时错误的策略是:

- 失败即停
- 不承诺回滚
- `force = true` 的覆盖风险由用户承担
- 真正高风险的覆盖和清理行为会先列出全部风险项, 再在交互环境进入确认界面, 或在非交互环境要求显式 override 参数

如果你只想做纯校验, 可以使用:

```sh
dotbot-go --check -c dotbot-go.toml
```

## Dry Run 与 Check

`dotbot-go` 会把 `--dry-run` 和 `--check` 作为核心审阅能力.

交互终端中:

- `--dry-run` 会自动进入 Bubble Tea 审阅界面, 展示包含 `config file` 绝对路径, `base dir`, 生效配置表格, 风险区, 卡片式计划列表和摘要的概览
- `--check` 会自动进入摘要型审阅界面, 展示包含 `config file` 绝对路径, `base dir`, 生效配置表格, 风险区和最终结论的概览
- 文本回退下, `--verbose` 会额外输出生效配置摘要

非交互环境中:

- `--dry-run` 回退为纯文本表格输出
- `--check` 回退为纯文本摘要输出
- 所有输出都适合重定向, 管道和 CI 日志采集

普通执行示例:

```text
[ok] create  /Users/example/.cache/zsh               created
[ok] link    /Users/example/.gitconfig <- /repo/git/gitconfig linked
[info] clean   /Users/example                        scan dead symlinks
summary: created=1 linked=1 skipped=0 replaced=0 deleted=0 failed=0
```

`--dry-run` 的非交互文本回退示例:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=1 link=1 clean=1
  risks: none

阶段   | 目标                         | 来源                 | 动作             | 备注
-----+----------------------------+--------------------+----------------+---
create | /Users/example/.cache/zsh   | -                  | create         | -
link   | /Users/example/.gitconfig   | /repo/git/gitconfig | create symlink | -
clean  | /Users/example              | -                  | scan dead symlinks | -
summary: created=1 linked=1 skipped=0 replaced=0 deleted=0 failed=0
```

`--check` 的非交互文本回退示例:

```text
check:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=1 link=1 clean=1
  risks: none
  result: check ok
```

如果你要重复做 TUI 手工验收, 可以直接使用:

```sh
scripts/tui-check.sh commands
```

它会自动准备固定夹具, 并打印 `dry-run` / `check` / `risky` / 非交互回退这几组可直接执行的命令.

## 与 Dotbot 的关系

`dotbot-go` 受到 Dotbot 启发, 但不是 Dotbot 的逐功能重写.

它会继承 Dotbot 最值得保留的部分:

- 只做 bootstrap
- 配置模型保持简单
- 与 Git 协作
- 强调幂等执行

同时, 它会主动缩小范围:

- 不实现 plugin 系统
- 不提供 shell 执行能力
- 不以生态扩展为核心目标
- 更专注于 dotfiles 安装场景本身

## 路线图

当前优先级更高的方向包括:

- 实现稳定的核心 directives
- 明确配置语义
- 做好 dry-run 输出
- 做好错误提示和配置校验
- 做好跨平台路径和链接处理

后续如果需要额外初始化逻辑, 推荐通过仓库中的独立脚本配合使用, 而不是把这些行为内置到 `dotbot-go`.

当前不在优先范围内的方向包括:

- plugin 系统
- shell 指令
- 通用任务编排能力
- 复杂的外部扩展机制

如果你想了解这些取舍背后的原因, 可以继续阅读 `CONSTRAINTS.md`.

## License

MIT
