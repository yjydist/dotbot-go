# DESIGN

## 目标

本文档用于固定 `dotbot-go` 当前阶段的核心设计决策.

当前目标不是把 Dotbot 逐功能复刻到 Go, 而是定义一个边界清晰, 配置稳定, 易于实现和维护的 dotfiles 管理工具.

## 核心边界

- 配置格式使用 TOML
- 配置模型是声明式的
- 用户描述目标状态, 不编排任务顺序
- 第一阶段只支持 `link`, `create`, `clean`, `default`
- 明确不支持 `plugin`
- 明确不支持 `shell`
- 明确不支持 `glob`

## 决策总表

| 主题 | 当前定稿 |
|---|---|
| 配置格式 | TOML |
| 配置模型 | 声明式, 不暴露任务顺序 |
| 默认配置文件名 | `dotbot-go.toml` |
| 默认执行方式 | `dotbot-go [flags]` |
| 顶层能力 | `link`, `create`, `clean`, `default` |
| 明确不支持 | `plugin`, `shell`, `glob` |
| 默认执行阶段 | `create -> link -> clean` |
| `default` 优先级 | 显式配置 > `default` > 硬编码默认值 |
| `link` 类型 | 仅支持 `symlink` |
| `[[link]].source` | 必填 |
| `source` 路径范围 | 允许相对路径和绝对路径 |
| `source` 相对路径基准 | 配置文件所在目录, 支持 `./`, `../`, `../../` |
| `target` 路径范围 | 允许相对路径, 也允许绝对路径和 `~` |
| 路径展开 | 仅支持 `~`, 不支持环境变量展开 |
| 空数组语义 | 合法但无操作 |
| `force` 语义 | 允许覆盖已有普通文件, 目录, 或符号链接 |
| `force` 风险归属 | 用户显式设置 `force = true` 即自行承担风险 |
| 失败策略 | 失败即停, 不承诺回滚 |
| Dry Run | 必须展示阶段, 动作, 目标, 决策, 结果 |
| 日志粒度 | 默认, `--verbose`, `--quiet` |
| CLI 参数 | `-c/--config`, `--dry-run`, `--verbose`, `--quiet`, `--no-color`, `-h/--help` |
| 纯校验能力 | `--check` |
| 退出码 | `0` 成功, `1` 运行时错误, `2` 配置错误 |
| 第一版平台 | macOS 和 Linux |

## 测试设计总表

| 测试类别 | 目标 | 关键用例 |
|---|---|---|
| 配置解析测试 | 保证 TOML 能正确解码到内部配置结构 | 顶层 section 正常解析, `[[link]]` 多项解析, 空数组解析 |
| 配置校验测试 | 保证非法配置在执行前失败 | 缺少 `source/target`, 未知字段, 重复 `target`, 非法空字符串 |
| 默认值合并测试 | 保证 `显式配置 > default > 硬编码默认值` | `default.link.create` 生效, 显式字段覆盖 default, default 缺失时落到硬编码默认值 |
| 路径展开测试 | 保证 `~` 与相对路径语义稳定 | `target` 的 `~` 展开, `source` 的 `./`, `../`, `../../` 解析, 绝对 `source` 保持不变 |
| `create` 行为测试 | 保证目录创建语义稳定 | 创建单层目录, 多层目录, 空 `paths`, `mode` 解析 |
| `link` 基础行为测试 | 保证 symlink 创建行为正确 | 新建 symlink, `relative=true`, `create=true`, `ignore_missing=true` |
| `link` 冲突行为测试 | 保证 `relink` / `force` / 冲突规则正确 | 已有 symlink + `relink`, 已有普通文件 + `force`, 未设置 `force` 时报错 |
| `clean` 行为测试 | 保证死链清理规则稳定 | 清理死链, `recursive=true`, `force=false` 下跳过仓库外目标, 空 `paths` |
| dry-run 输出测试 | 保证 dry-run 可信且稳定 | create/link/clean 单行输出, `skip` 原因, 摘要输出 |
| 普通输出测试 | 保证真实执行输出与 dry-run 风格一致 | `created`, `linked`, `replaced`, `deleted`, `failed` 状态输出 |
| CLI 行为测试 | 保证参数和退出码语义稳定 | 默认配置查找, `--verbose`/`--quiet` 互斥, `--help`, `0/1/2` 退出码 |
| 平台差异测试 | 保证第一版平台边界清晰 | macOS/Linux 路径与 symlink 行为, Windows 暂不纳入主测试矩阵 |

### 测试分层建议

- 单元测试: 配置解析, 默认值合并, 路径解析, 输出格式
- 集成测试: 在临时目录中执行 `create/link/clean` 的真实文件系统行为
- 平台测试: 先聚焦 macOS 和 Linux, Windows 暂不作为第一版门禁

### 第一版必须具备的测试

- 配置解析测试
- 配置校验测试
- 默认值合并测试
- 路径展开测试
- `link` 基础行为测试
- `link` 冲突行为测试
- `clean` 行为测试
- dry-run 输出测试
- CLI 退出码测试

### 第一版可以后补的测试

- 更细粒度的输出快照测试
- 更完整的 `mode` 平台差异测试
- 更复杂的类 Unix 平台兼容性矩阵

## 配置示例总表

### 完整合法配置示例

```toml
[default.link]
create = true
relink = true
relative = true
ignore_missing = false

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

这个示例覆盖了当前已定稿的主要语义:

- `default` 为缺失字段提供默认值
- `source` 支持相对路径和绝对路径
- `source` 的相对路径允许 `./`, `../`, `../../`
- `target` 支持 `~`, 绝对路径, 以及相对路径
- `force`, `relink`, `relative`, `ignore_missing` 可按项覆盖默认值
- `[create].paths` 和 `[clean].paths` 都是显式数组

### 最小合法配置示例

```toml
[[link]]
target = "~/.gitconfig"
source = "git/gitconfig"
```

### 空操作合法配置示例

```toml
[create]
paths = []

[clean]
paths = []
```

该配置合法, 但不会执行任何 create 或 clean 动作.

### 非法配置示例

#### 1. 缺少 `source`

```toml
[[link]]
target = "~/.gitconfig"
```

原因:

- `source` 是必填字段

#### 2. 缺少 `target`

```toml
[[link]]
source = "git/gitconfig"
```

原因:

- `target` 是必填字段

#### 3. 未知字段

```toml
[[link]]
target = "~/.gitconfig"
source = "git/gitconfig"
backup = true
```

原因:

- 当前版本不支持 `backup`
- 未知字段直接报错

#### 4. 重复 `target`

```toml
[[link]]
target = "~/.gitconfig"
source = "git/gitconfig"

[[link]]
target = "~/.gitconfig"
source = "git/other-gitconfig"
```

原因:

- 不允许多个 `[[link]]` 指向同一个 `target`

#### 5. 非法空字符串

```toml
[[link]]
target = ""
source = "git/gitconfig"
```

原因:

- 空字符串视为无效路径值

#### 6. 未知 section

```toml
[shell]
commands = ["echo hello"]
```

原因:

- 当前版本不支持 `shell`
- 未知 section 直接报错

#### 7. `create.paths` 类型错误

```toml
[create]
paths = "~/.cache/zsh"
```

原因:

- `paths` 必须是字符串数组, 不能是单个字符串

#### 8. `source` 路径虽然可以越出配置目录, 但推荐使用常见可读写法

```toml
[[link]]
target = "~/.zshrc"
source = "../shared/zshrc"
```

说明:

- 当前实现允许任意普通相对路径字符串
- 文档层面仍推荐使用 `./`, `../`, `../../` 这类更易读的路径写法

## P0 定稿

### 1. 配置文件名规范

- 默认配置文件名: `dotbot-go.toml`
- CLI 允许通过 `-c` 或 `--config` 指定其他路径
- 如果未显式指定配置文件, 则优先在当前工作目录查找 `dotbot-go.toml`

理由:

- 名称直接表达工具归属, 比 `install.toml` 更清晰
- 允许自定义路径, 可以兼容不同仓库布局

### 2. 内部执行阶段

虽然配置模型是声明式的, 但执行器内部仍然采用固定阶段顺序:

1. `create`
2. `link`
3. `clean`

约束:

- 该顺序是工具内部实现细节, 不是用户配置能力
- 不提供用户自定义阶段顺序
- `default` 不是执行阶段, 而是在配置装载后先完成归并

理由:

- `create` 先执行, 可以保证父目录准备完成
- `link` 在中间执行, 是核心动作
- `clean` 放在最后执行, 可避免过早删除仍可能被后续修复的失效链接

### 3. 仓库根目录语义

- 仓库根目录定义为配置文件所在目录
- `[[link]].source` 允许使用相对路径或绝对路径
- `[[link]].source` 为相对路径时, 按标准相对路径语义相对于配置文件目录解析
- CLI 当前工作目录不影响 `source` 的相对路径解释
- 如果用户通过 `-c path/to/dotbot-go.toml` 指定配置文件, 则仓库根目录为 `path/to`

理由:

- 配置和仓库内容天然放在一起时, 以配置文件目录为基准的相对路径语义最直观
- 同时允许绝对路径, 可以保留更大的表达自由度
- 避免当前工作目录变化导致行为漂移

### 4. 路径展开规则

当前阶段固定以下规则:

- 支持 `~` 展开到当前用户 Home 目录
- `target` 中的相对路径按当前工作目录解析
- `source` 中允许绝对路径, 相对路径按配置文件目录解析, 可使用 `./`, `../`, `../../` 等写法
- Windows 平台暂不做额外语法兼容设计, 先以通用路径清洗和 `filepath` 规则为基础

补充约束:

- `target` 推荐始终写成绝对路径或以 `~` 开头的路径
- `source` 可以写相对路径或绝对路径
- `source` 的相对路径不限制层级, 只要按标准路径规则可解析即可
- README 中的所有示例优先使用 `~`

理由:

- `~` 是 dotfiles 场景最核心的展开需求
- 只保留 `~` 这一种显式锚点, 可以让路径语义更稳定更易懂
- `source` 和 `target` 分开定义解析基准后, 语义更稳定

### 5. 冲突处理规则

#### `relink` 和 `force`

- `force` 优先级高于 `relink`
- `relink` 仅处理目标已存在且目标本身是符号链接的情况
- `force` 可处理目标已存在且为普通文件, 目录, 或符号链接的情况
- 即使 `force = true`, 也不能覆盖受保护目标, 包括 `/`, Home 根目录, 当前工作目录根, 配置文件基准目录

#### 目标已存在但类型不匹配

- 如果目标已存在且类型不匹配, 且未设置 `force`, 则直接报错
- 如果目标已存在且类型不匹配, 且设置了 `force`, 则直接重建

补充约束:

- 任何具有删除语义的动作都必须在 dry-run 中明确展示
- 任何覆盖行为都必须在普通输出中明确标记为 `replace`

### 6. 默认值合并规则

- `default` 只为缺失字段提供默认值
- 单个配置项中的显式字段优先级高于 `default`
- 不做深度合并
- 第一版只支持扁平字段覆盖

最终优先级规则固定为:

1. 配置项显式字段
2. `default` 中对应字段
3. 项目内置硬编码默认值

示例:

```toml
[default.link]
create = true
relative = true

[[link]]
target = "~/.gitconfig"
source = "git/gitconfig"
relative = false
```

在上面的例子中:

- `create` 最终为 `true`
- `relative` 最终为 `false`

理由:

- 规则简单, 实现简单, 可解释性强
- 避免深度合并带来的隐藏行为

## P0 Schema 定稿

### 1. 顶层结构

配置文件固定支持以下顶层 section:

- `[default.link]`
- `[default.create]`
- `[default.clean]`
- `[create]`
- `[clean]`
- `[[link]]`

约束:

- 不支持其他顶层 section
- 未知字段直接报错
- 未知 section 直接报错

### 2. `[[link]]` 字段定稿

#### 必填字段

- `target: string`
- `source: string`

#### 选填字段

- `create: bool`
- `relink: bool`
- `force: bool`
- `relative: bool`
- `ignore_missing: bool`

#### 字段规则

- `target` 必填, 且不能为空字符串
- `source` 必填, 且不能为空字符串
- 布尔字段未显式指定时, 按 `default.link` 补值, 再按项目硬编码默认值补齐

#### 硬编码默认值

- `create = false`
- `relink = false`
- `force = false`
- `relative = false`
- `ignore_missing = false`

### 3. `[create]` 字段定稿

#### 必填字段

- `paths: []string`

#### 选填字段

- `mode: string`

#### 字段规则

- `paths` 必填, 但允许为空数组
- `paths` 中每个元素必须是非空字符串
- `mode` 使用字符串表示, 例如 `"0700"`

#### 硬编码默认值

- `mode = "0777"`

### 4. `[clean]` 字段定稿

#### 必填字段

- `paths: []string`

#### 选填字段

- `force: bool`
- `recursive: bool`

#### 字段规则

- `paths` 必填, 但允许为空数组
- `paths` 中每个元素必须是非空字符串

#### 硬编码默认值

- `force = false`
- `recursive = false`

### 5. `[default.*]` 字段覆盖范围定稿

#### `[default.link]`

允许字段:

- `create`
- `relink`
- `force`
- `relative`
- `ignore_missing`

不允许字段:

- `target`
- `source`

#### `[default.create]`

允许字段:

- `mode`

#### `[default.clean]`

允许字段:

- `force`
- `recursive`

### 6. 字段类型与校验基线

- 所有路径字段使用字符串
- 所有布尔字段必须是 TOML 布尔值
- 所有数组字段必须是同构字符串数组
- 空字符串视为无效值

## P0 错误模型与校验定稿

### 1. 配置校验规则

#### 必填字段缺失

- `[[link]]` 缺少 `target` 时直接报错
- `[create]` 缺少 `paths` 时直接报错
- `[clean]` 缺少 `paths` 时直接报错

#### 非法字段

- 任何未定义字段都直接报错
- 任何未定义 section 都直接报错
- 不做“忽略未知字段”的宽松模式

#### 非法枚举值

- 当前版本不支持 `type` 字段, 出现即报错

#### 重复目标路径

- 不允许多个 `[[link]]` 指向同一个 `target`
- 一旦发现重复目标路径, 在配置校验阶段直接报错

#### 空数组

- `[create].paths = []` 合法, 表示当前不需要创建任何目录
- `[clean].paths = []` 合法, 表示当前不需要清理任何目录
- `[[link]]` 为空数组同样合法, 表示当前没有链接项

#### 空字符串

- 路径字段出现空字符串视为配置错误

### 2. 运行前校验规则

#### `source` 不存在

- 当 `ignore_missing = false` 时, 直接报错
- 当 `ignore_missing = true` 时, 跳过该项并输出 `skip`

#### `target` 父目录不存在

- 当 `create = true` 时, 在 `create/link` 阶段负责补齐父目录
- 当 `create = false` 时, 直接报错

#### `clean.paths` 不存在

- 不报错
- 输出 `skip`, 并标记原因是目录不存在

#### `clean.paths` 为符号链接

- 直接报错
- 不跟随目录符号链接作为清理根路径

#### 权限不足

- 直接报错
- 错误信息中应尽量带上目标路径和实际操作类型

#### `source` 为绝对路径

- 合法
- 不做“必须位于仓库内”的限制
- dry-run 和普通输出中都按最终解析后的路径展示

### 3. 错误输出格式

错误输出必须尽量结构化, 至少包含以下信息:

- 错误类别
- 配置段名称
- 字段路径
- 相关路径
- 人类可读原因

推荐格式:

```text
config error: [[link]][2].target: duplicate target path: ~/.gitconfig
runtime error: [[link]][1].source: source does not exist: /repo/git/gitconfig
```

约束:

- 配置错误和运行时错误要明确区分
- 报错中尽量使用配置视角字段名, 不只返回底层系统错误
- 如果存在底层系统错误, 作为补充信息附在原因后面

### 4. 退出策略

- 配置错误: 在真正执行前终止
- 运行时错误: 默认失败即停止, 不继续执行后续动作
- dry-run 中如发现配置错误, 也应直接失败

理由:

- dotfiles 安装更适合保守策略
- 失败即停比“尽量继续”更容易理解
- `force = true` 表示用户明确接受覆盖风险, 工具不承诺恢复执行前状态
- `clean` 默认只删除 dead target 仍位于仓库基准目录内的失效链接
- `clean.force` 不会放宽仓库边界

## P0 Dry Run 与可观察性定稿

### 1. dry-run 输出格式

dry-run 必须按内部固定阶段输出, 即:

1. `create`
2. `link`
3. `clean`

每一条输出至少包含:

- 动作阶段
- 动作类型
- 目标路径
- 关键决策
- 最终状态

交互终端中, `--dry-run` 默认进入审阅界面, 展示:

- 配置路径与 base dir
- 阶段统计
- 风险列表
- 更适合长路径阅读的卡片式计划列表
- 摘要统计

非交互环境回退为纯文本表格, 推荐格式:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=1 link=2 clean=1
  risks: none

阶段   | 目标               | 来源              | 动作             | 备注
-----+------------------+-----------------+----------------+---
create | ~/.cache/zsh      | -               | create         | -
link   | ~/.gitconfig      | git/gitconfig   | create symlink | -
link   | ~/.zshrc          | shell/zshrc     | skipped        | source missing, ignore_missing=true
clean  | ~/.config/old-link | -               | delete dead symlink | -
summary: created=1 linked=1 skipped=1 replaced=0 deleted=1 failed=0
```

具体要求:

- `create` 要展示将创建哪些目录
- `link` 要展示 `target <- source`
- `clean` 要展示将删除哪些死链
- 如果某项被跳过, 必须显示 `skip` 和原因
- 如果某项会触发覆盖, 备份或删除, 必须明确写出

### 2. 普通执行输出格式

普通执行输出保持与 dry-run 尽量一致, 但状态改为实际结果.

推荐状态集合:

- `created`
- `linked`
- `skipped`
- `replaced`
- `deleted`
- `failed`

推荐文本格式:

```text
[ok]   create  ~/.cache/zsh                         created
[ok]   link    ~/.gitconfig <- git/gitconfig       linked
[ok]   link    ~/.zshrc <- shell/zshrc             replaced
[skip] clean   ~/.cache/unused-link                path missing
[fail] link    ~/.tmux.conf <- tmux/tmux.conf      target exists and force=false
```

具体要求:

- 覆盖行为显示为 `replaced`
- 删除死链显示为 `deleted`
- 失败时显示为 `failed`, 并附人类可读原因

### 3. 日志粒度

当前阶段固定三档日志粒度:

#### 默认输出

- 输出每个实际动作的单行结果
- 输出 `skip` 项
- 不输出内部调试细节

#### `--verbose`

- 普通执行模式下保持原有额外文本信息
- 审阅界面展示当前实际生效配置, 文本回退在 `--verbose` 下额外展示生效配置摘要
- 不再要求在 `--dry-run` / `--check` 前单独打印一段前置文本块

#### `--quiet`

- 仅输出失败信息
- 成功和跳过默认不输出
- 若全部成功, 保持静默并以退出码表示结果

### 4. 摘要输出

在默认和 verbose 模式下, 执行结束后输出摘要:

```text
summary: created=2 linked=4 skipped=1 replaced=1 deleted=0 failed=0
```

约束:

- dry-run 也输出摘要
- 摘要字段顺序固定
- 没有发生的状态也显示为 `0`

### 5. 颜色策略

- 默认允许彩色输出
- 支持 `--no-color` 关闭颜色
- 仅在终端环境下启用颜色
- 颜色只做辅助, 不承载唯一语义
- 即使去掉颜色, 文本本身也必须可读

### 6. dry-run 输出示例

#### 示例 1: 非交互常规计划输出

配置片段:

```toml
[create]
paths = ["~/.cache/zsh"]

[[link]]
target = "~/.gitconfig"
source = "./git/gitconfig"

[clean]
paths = ["~"]
```

期望 dry-run 文本回退:

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

#### 示例 1b: 交互终端审阅界面

期望行为:

- 自动进入审阅界面
- 默认展示 `config file` 绝对路径, `base dir`, 阶段统计, 实际生效配置表格, 风险列表和卡片式计划列表
- `q` / `esc` 退出界面

#### 示例 1c: verbose 输出

期望 verbose 额外内容:

```text
link: create=true relink=true force=false relative=true ignore_missing=false
create: mode=0755
clean: force=false recursive=false
```

#### 示例 2: 跳过缺失 source

配置片段:

```toml
[[link]]
target = "~/.config/ghostty/config"
source = "./ghostty/config"
ignore_missing = true
```

期望 dry-run 文本回退:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=0 link=1 clean=0
  risks: none

阶段 | 目标                                   | 来源                  | 动作    | 备注
----+--------------------------------------+---------------------+-------+------------------------------------
link | /Users/example/.config/ghostty/config | /repo/ghostty/config | skipped | source missing, ignore_missing=true
summary: created=0 linked=0 skipped=1 replaced=0 deleted=0 failed=0
```

#### 示例 3: force 覆盖已有目标

配置片段:

```toml
[[link]]
target = "~/.tmux.conf"
source = "./tmux/tmux.conf"
force = true
```

期望 dry-run 文本回退:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=0 link=1 clean=0
  risks: none

阶段 | 目标                    | 来源                | 动作    | 备注
----+-----------------------+-------------------+-------+----------
link | /Users/example/.tmux.conf | /repo/tmux/tmux.conf | replace | force=true
summary: created=0 linked=0 skipped=0 replaced=1 deleted=0 failed=0
```

#### 示例 4: clean 路径不存在

配置片段:

```toml
[clean]
paths = ["~/.cache/nonexistent"]
```

期望 dry-run 文本回退:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=0 link=0 clean=1
  risks: none

阶段 | 目标                              | 来源 | 动作    | 备注
----+---------------------------------+----+-------+-------------
clean | /Users/example/.cache/nonexistent | -  | skipped | path missing
summary: created=0 linked=0 skipped=1 replaced=0 deleted=0 failed=0
```

#### 示例 5: 配置错误

配置片段:

```toml
[[link]]
target = "~/.gitconfig"
```

期望结果:

```text
config error: [[link]][1].source: required field is missing
```

## P1 风险确认机制定稿

本节定义当前 `force` 与高风险清理行为的最终语义.

目标不是取消风险, 而是把风险显式暴露给用户, 并在真正危险的场景下要求二次确认.

### 1. 设计目标

- 保留 `link.force` 和 `clean.force` 的实际能力
- 默认执行仍保持保守
- 真正高风险的文件系统操作必须显式确认
- 交互环境和非交互环境采用不同确认方式
- 不能只靠文档声明“风险自负”, 必须让工具在执行前明确暴露风险

### 2. 风险分级

#### 普通风险

普通风险指用户在 dotfiles 场景中经常需要, 但仍具有覆盖或删除语义的操作:

- 使用 `link.force = true` 覆盖普通文件
- 使用 `link.force = true` 覆盖普通目录
- 使用 `clean.force = true` 清理 dead target 位于仓库外的失效链接

约束:

- dry-run 必须明确显示覆盖或删除动作
- 默认输出必须明确标记 `replace` 或 `delete`
- 普通风险不要求二次确认

#### 危险风险

危险风险指一旦配置错误就可能造成大面积破坏的操作:

- 覆盖 `/`
- 覆盖用户 Home 根目录
- 覆盖当前工作目录根
- 覆盖配置文件基准目录
- 把 `clean.paths` 设为目录符号链接并以其作为扫描根
- 清理范围明显越过用户预期边界的大范围根路径

约束:

- 危险风险默认不得直接静默执行
- 交互环境必须要求二次确认
- 非交互环境必须要求显式 override 参数

### 3. `link.force` 最终语义

- `force = false` 时, 目标已存在且无法安全复用时直接报错
- `force = true` 时, 允许覆盖普通文件, 目录, 或符号链接
- 当目标命中危险风险集合时, 不再直接执行, 而是进入风险确认流程

风险确认通过后:

- 允许继续执行覆盖
- 覆盖前仍需在输出中明确标记将要替换的目标

### 4. `clean.force` 最终语义

- 默认 `clean` 只清理 dead target 解析后仍位于仓库基准目录内的失效链接
- `clean.force = true` 时, 允许清理 dead target 位于仓库基准目录外的失效链接
- `clean.paths` 即使是目录符号链接, 也只在确认后才允许作为高风险 clean 根路径使用
- 如果 `clean.paths` 命中危险风险集合, 进入风险确认流程

补充约束:

- `clean.force` 只放宽“dead target 是否位于仓库内”的限制
- `clean.force` 不自动放宽危险扫描根路径限制

### 5. 交互确认流程

仅当 stdout/stderr/stdin 处于 TTY 交互环境时启用交互确认.

确认规则:

- 工具在执行前进入 Bubble Tea 确认界面
- 界面一次性展示全部风险项摘要
- 输出必须包含动作类型和目标路径
- 用户只需做一次确认

建议交互内容:

```text
detected risky operations:
- replace protected target: /absolute/target
- risky clean root: /absolute/path

Enter 确认, Esc 取消
```

如果用户未确认:

- 本次执行直接终止
- 返回运行时错误退出码

### 6. 非交互 override 规则

非交互环境不得进入等待用户输入的状态.

因此:

- 如果命中危险风险且当前不是 TTY, 直接报错
- 用户必须显式传入更强的 override 参数才允许继续

建议参数:

- `--allow-protected-target`
- `--allow-risky-clean`

约束:

- 这些参数只用于放行危险风险
- 不改变普通风险的默认语义
- 必须在 help 和文档中明确标注为高风险参数

### 7. dry-run 与输出要求

dry-run 必须在危险风险场景下输出足够明确的提示.

交互终端中:

- `--dry-run` 自动进入审阅界面
- `--check` 自动进入摘要审阅界面
- `--quiet` 不进入审阅界面

非交互环境推荐示例:

```text
dry-run:
  config: /repo/dotbot-go.toml
  base dir: /repo
  stages: create=0 link=1 clean=1
  risks: 2
    - replace protected target: /Users/example
    - risky clean root: /Users/example

阶段 | 目标           | 来源      | 动作               | 备注
----+--------------+---------+------------------+--------------------------------------
link | /Users/example | /repo/file | replace          | protected target, confirmation required
clean | /Users/example | -         | scan dead symlinks | risky clean, confirmation required
```

普通执行中, 如果进入确认流程, 应直接进入确认界面, 不再使用裸 `y/N` 提问.

### 8. 为什么采用该方案

- 保留 `force` 的表达能力, 不把配置字段做成空壳
- 把“风险由用户承担”从抽象声明变成真实确认动作
- 兼顾交互使用和脚本使用
- 避免把一次误配置静默升级成灾难性删除

### 9. 实现顺序

实现按以下顺序推进:

1. 在运行时识别交互环境与非交互环境
2. 抽象危险目标与危险清理根的判定函数
3. 为 `link.force` 接入风险确认流程
4. 为 `clean.force` 恢复仓库外 dead target 清理能力
5. 增加交互确认与非交互 override 的测试
6. 同步 README, help 文案, dry-run 示例

约束:

- 配置错误时不输出 dry-run 动作列表
- 配置错误直接返回配置错误退出码

## P1 CLI 设计定稿

### 1. 最小参数集合

第一版 CLI 固定支持以下参数:

- `-c, --config <path>`: 指定配置文件路径
- `--check`: 只校验配置和关键运行前条件, 不修改文件系统
- `--dry-run`: 只展示计划动作, 不修改文件系统
- `--verbose`: 输出更详细的信息
- `--quiet`: 仅输出失败信息
- `--no-color`: 关闭彩色输出
- `-h, --help`: 显示帮助信息

约束:

- `--verbose` 和 `--quiet` 互斥
- 未传 `-c` 时, 默认在当前工作目录查找 `dotbot-go.toml`
- 若默认配置文件不存在, 直接报配置错误

### 2. 命令行为

当前阶段只设计单命令执行模型, 不引入子命令.

也就是说, 用户使用方式固定为:

```sh
dotbot-go [flags]
```

不设计如下子命令:

- `apply`
- `check`
- `fmt`
- `plan`

理由:

- 第一版范围应尽量收敛
- `--dry-run` 已能覆盖最核心的预览需求
- 子命令会明显增加 CLI 复杂度和文档负担

### 3. 退出码

当前阶段固定退出码如下:

- `0`: 成功
- `1`: 运行时错误
- `2`: 配置错误

补充约束:

- `--help` 正常输出后返回 `0`
- dry-run 只要成功完成分析, 返回 `0`
- dry-run 遇到配置错误, 返回 `2`

### 4. 帮助文案

帮助文案至少包含以下信息:

- 工具一句话简介
- 默认配置文件名
- `--dry-run` 的含义
- `--verbose` / `--quiet` 的区别
- 配置文件路径解析的基本规则

建议帮助概要:

```text
dotbot-go - 面向类 Unix 系统的声明式 dotfiles 管理工具

Usage:
  dotbot-go [flags]

Flags:
  -c, --config <path>   配置文件路径, 默认: ./dotbot-go.toml
      --check           仅校验配置和关键运行前条件
      --dry-run         仅展示计划动作, 不修改文件系统
      --verbose         输出配置路径, 生效配置摘要, 阶段统计
      --quiet           仅输出失败信息, 不输出成功和摘要
      --no-color        关闭彩色输出
  -h, --help            显示帮助

Path rules:
  source 相对路径基于配置文件目录解析
  target 相对路径基于当前工作目录解析
  source 和 target 都支持 ~ 展开
```

### 5. 第一版平台声明

第一版 CLI 面向类 Unix 系统:

- macOS
- Linux

当前不把 Windows 作为第一版目标平台.

约束:

- README 中明确写出平台范围
- CLI 帮助文案不承诺 Windows 完整支持

## 当前建议的最小配置结构

```toml
[default.link]
create = true
relink = true
relative = true

[create]
paths = [
  "~/.cache/zsh",
  "~/.local/share/nvim",
]

[clean]
paths = [
  "~",
]

[[link]]
target = "~/.gitconfig"
source = "git/gitconfig"

[[link]]
target = "~/.zshrc"
source = "shell/zshrc"
```

## 下一步

在以上设计冻结后, 下一阶段优先做这些事:

- 定义 Go 配置结构体
- 明确字段类型和默认值
- 实现配置解析和校验
- 设计 dry-run 和普通输出格式
