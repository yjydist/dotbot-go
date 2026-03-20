# TODO

## 使用规则

- 本文件在编码阶段用于持续记录当前进度
- 每次开始一轮实现前先更新状态
- 每完成一个任务后立即更新状态
- 任务粒度保持中等, 一般一个功能拆成 3 到 7 个可执行项
- 当前清单只记录实现阶段任务, 不再重复设计阶段已定稿内容

## 当前阶段

- [x] 第一轮实现: 配置加载与校验骨架

## 第一轮实现任务

- [x] 初始化 Go 项目骨架
  - [x] 初始化 Go module
  - [x] 创建 `cmd/dotbot-go`
  - [x] 创建 `internal/config`
  - [x] 创建 `internal/output`
  - [x] 创建 `internal/runner`

- [x] 实现 TOML 配置模型
  - [x] 定义顶层配置结构体
  - [x] 定义 `[[link]]` 配置结构体
  - [x] 定义 `[create]` 配置结构体
  - [x] 定义 `[clean]` 配置结构体
  - [x] 定义 `[default.*]` 配置结构体

- [x] 实现配置解析与严格校验
  - [x] 接入 TOML 解析
  - [x] 校验必填字段
  - [x] 校验未知字段和未知 section
  - [x] 校验重复 `target`
  - [x] 校验空字符串和非法类型

- [x] 实现默认值归并
  - [x] 落实 `显式配置 > default > 硬编码默认值`
  - [x] 应用到 `link`
  - [x] 应用到 `create`
  - [x] 应用到 `clean`

- [x] 实现路径解析基础能力
  - [x] 处理 `~` 展开
  - [x] 处理 `source` 相对配置文件目录解析
  - [x] 处理 `source` 绝对路径
  - [x] 处理 `target` 相对当前工作目录解析

- [x] 实现最小 CLI 入口
  - [x] 支持 `-c, --config`
  - [x] 支持 `--dry-run`
  - [x] 支持 `--verbose`
  - [x] 支持 `--quiet`
  - [x] 支持 `--no-color`
  - [x] 定义 `0/1/2` 退出码行为

- [x] 为第一轮补齐核心测试
  - [x] 配置解析测试
  - [x] 配置校验测试
  - [x] 默认值合并测试
  - [x] 路径展开测试
  - [x] CLI 退出码测试

## 第二轮实现任务

- [x] 实现 `create`
  - [x] 创建单层目录
  - [x] 创建多层目录
  - [x] 处理 `mode`

- [x] 实现 `link`
  - [x] 创建 symlink
  - [x] 实现 `create`
  - [x] 实现 `relink`
  - [x] 实现 `force`
  - [x] 实现 `relative`
  - [x] 实现 `ignore_missing`

- [x] 实现 `clean`
  - [x] 检测失效链接
  - [x] 删除失效链接
  - [x] 实现 `recursive`
  - [x] 实现 `force`

- [x] 为核心行为补齐测试
  - [x] `create` 行为测试
  - [x] `link` 基础行为测试
  - [x] `link` 冲突行为测试
  - [x] `clean` 行为测试

## 第三轮实现任务

- [x] 实现 dry-run 输出
  - [x] create 输出
  - [x] link 输出
  - [x] clean 输出
  - [x] skip 原因输出
  - [x] 摘要输出

- [x] 实现普通执行输出
  - [x] `created`
  - [x] `linked`
  - [x] `replaced`
  - [x] `deleted`
  - [x] `failed`

- [x] 补齐输出相关测试
  - [x] dry-run 输出测试
  - [x] 普通执行输出测试
  - [x] `--verbose` / `--quiet` 行为测试

## 文档同步任务

- [x] 在第一轮实现后同步更新 `README.md`
- [x] 在第一轮实现后同步更新 `DESIGN.md`
- [x] 在第二轮实现后补充真实配置示例
- [x] 在第三轮实现后补充真实 dry-run 输出示例
- [x] 增加设计约束文档
  - [x] 为什么不用 YAML
  - [x] 为什么不用任务列表
  - [x] 为什么不支持 shell
  - [x] 为什么不支持 plugin

## 后续再评估

- [ ] 是否支持 Windows 特殊行为适配
- [ ] 是否需要 `check` 子命令做纯校验
- [ ] 是否需要 `fmt` 子命令整理 TOML 配置
