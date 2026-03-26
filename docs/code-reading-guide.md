# 代码阅读导图

本文档不是设计文档的重复版本, 而是给后续阅读源码时的一条实际路径.

如果你的目标是尽快理解项目, 建议按下面的顺序读:

1. 先看入口和主流程
2. 再看配置如何被归一化
3. 再看 create / link / clean 三个执行包
4. 最后看输出层和 TUI

这样读会比“按文件夹随便点开”快很多.

## 一张总图

项目的主链路可以压缩成:

```text
cmd/dotbot-go/main.go
  -> internal/runner
    -> internal/config
    -> internal/creator
    -> internal/linker
    -> internal/cleaner
    -> internal/output / internal/tui
```

可以把它理解成 5 层:

- 入口层: 收 CLI 参数和标准流
- 配置层: 把 TOML 变成运行期 `Config`
- 执行编排层: 决定运行模式, 顺序, 风险确认和输出路径
- 执行层: `create` / `link` / `clean`
- 展示层: 终端文本输出和 TUI

## 推荐阅读顺序

### 1. 入口层

先看 [`cmd/dotbot-go/main.go`](../cmd/dotbot-go/main.go).

这个文件非常薄, 作用只有一个:

- 把 CLI 参数和标准流交给 `runner.Run`

它的价值不在业务逻辑, 而在告诉你:

- 真正的程序入口是 `internal/runner`

### 2. 执行编排层

然后看 [`internal/runner/run.go`](../internal/runner/run.go).

这是整个项目最重要的文件之一.

你在这里应该重点理解:

- 参数解析之后, 程序如何拿到 `workingDir` 和 `homeDir`
- 配置什么时候加载
- 为什么固定顺序是 `create -> link -> clean`
- `--dry-run` 和 `--check` 为什么共用一条执行路径
- 风险确认在进入 `link` / `clean` 前的哪个位置触发
- 为什么输出层同时支持文本和 TUI

如果你只想先把“大局”看懂, 读完这个文件就会知道:

- 数据从哪里来
- 会流到哪些包
- 哪些阶段可能失败
- 失败时为什么是“立即停止”

然后再看 [`internal/runner/cli.go`](../internal/runner/cli.go) 和 [`internal/runner/helpers.go`](../internal/runner/helpers.go).

它们分别负责:

- `cli.go`: 参数语义和 help 文案
- `helpers.go`: 风险发现, 交互判断, verbose 摘要, review 数据收集

这里建议重点注意 3 个 helper:

- `resolveProtectedTargetAllowance`
- `resolveRiskyCleanAllowance`
- `buildVerboseLines`

因为这三个函数直接决定:

- 哪些操作被视为高风险
- 什么时候要求确认
- 审阅界面里到底展示什么

### 3. 配置层

看完 runner 后, 再回到 [`internal/config/load.go`](../internal/config/load.go), [`internal/config/build.go`](../internal/config/build.go), [`internal/config/resolve.go`](../internal/config/resolve.go), [`internal/config/types.go`](../internal/config/types.go).

这里最关键的理解是:

- `raw*` 结构只服务于 TOML 解码
- `Config` 才是执行阶段使用的最终模型

建议按下面顺序读:

1. [`internal/config/types.go`](../internal/config/types.go)
2. [`internal/config/load.go`](../internal/config/load.go)
3. [`internal/config/build.go`](../internal/config/build.go)
4. [`internal/config/resolve.go`](../internal/config/resolve.go)

你应该在这里重点理解 4 件事:

- 为什么 `raw` 结构里很多字段是 `*bool`
- `default.*` 是怎么合并进最终配置的
- 为什么 `source` 和 `target` 的相对路径基准不同
- 为什么未知字段会直接报错

如果把配置层一句话说清楚, 就是:

- runner 拿到的不是“用户写了什么”, 而是“最终应该怎么执行”

### 4. 执行层

执行层分成 3 个包:

- [`internal/creator/create.go`](../internal/creator/create.go)
- [`internal/linker/linker.go`](../internal/linker/linker.go)
- [`internal/cleaner/cleaner.go`](../internal/cleaner/cleaner.go)

#### 4.1 `create`

`create` 最简单.

重点看:

- 已存在目录为什么是 `skip`
- 已存在普通文件为什么是 `fail`
- dry-run 为什么也会把 `Created` 计数加 1

这里的关键理解是:

- `Created` 在 dry-run 里代表“计划创建数量”, 不是“已经创建成功”

#### 4.2 `link`

`linker` 是执行层里最复杂的部分.

建议重点关注:

- `applyOne`

它的判断顺序大致是:

1. 校验 source
2. 检查 target 父目录是否满足 `create` 语义
3. 看 target 当前是缺失 / symlink / 普通文件目录
4. 再决定是新建, relink, force 覆盖还是失败

这里最容易出错的点有:

- `relative=true` 时实际落盘的 symlink 目标如何计算
- `relink` 和 `force` 的边界
- 受保护目标为什么即使在 `relink` 路径下也需要确认

如果你要 review 风险边界, 这里是第一优先级.

#### 4.3 `clean`

`cleaner` 的重点不是“删除能力”, 而是“默认保守”.

建议重点看:

- `Apply`
- `maybeRemoveDeadLink`
- `riskyRootReason`

这里要理解的核心问题是:

- 什么叫 dead symlink
- 为什么 `force=false` 时只删仓库内 dead target
- 为什么 symlink root 被视为高风险 clean root

`clean` 的真正删除边界最后落在:

- [`maybeRemoveDeadLink` in `internal/cleaner/cleaner.go`](../internal/cleaner/cleaner.go)

### 5. 展示层

展示层分成两块:

- 终端文本输出: `internal/output`
- 交互式 TUI: `internal/tui`

#### 5.1 文本输出

建议按这个顺序看:

1. [`internal/output/model.go`](../internal/output/model.go)
2. [`internal/output/terminal.go`](../internal/output/terminal.go)
3. [`internal/output/review_text.go`](../internal/output/review_text.go)

关键理解:

- `Entry` 是 create/link/clean 共享的统一输出模型
- `ReviewData` 是 dry-run/check 的统一展示模型
- `terminal.go` 负责普通执行输出
- `review_text.go` 负责非交互环境下的审阅文本回退

如果你发现 TUI 和文本输出有差异, 先看它们是不是共享了同一份 `ReviewData`.

#### 5.2 TUI

建议按这个顺序看:

1. [`internal/tui/tui.go`](../internal/tui/tui.go)
2. [`internal/tui/review_panels.go`](../internal/tui/review_panels.go)
3. [`internal/tui/helpers.go`](../internal/tui/helpers.go)
4. [`internal/tui/confirm.go`](../internal/tui/confirm.go)
5. [`internal/tui/runtime.go`](../internal/tui/runtime.go)

这里建议你把 TUI 拆成两种界面来理解:

- review 界面: 给 `--dry-run` / `--check` 用
- confirm 界面: 给高风险正式执行确认用

重点要看懂:

- 为什么 `reviewModel.View()` 只装配页头、viewport、页脚
- 为什么具体 panel 渲染被拆到 `review_panels.go`
- 为什么 `helpers.go` 里有这么多宽度和换行 helper

答案其实很简单:

- 这个项目的 TUI 最大难点不是状态管理, 而是长路径和窄终端下的布局稳定性

## 测试怎么读

如果你不是要补实现, 而是要理解“当前行为到底被什么锁住了”, 测试文件反而很值得读.

建议顺序:

1. [`internal/config/load_test.go`](../internal/config/load_test.go)
2. [`internal/linker/linker_test.go`](../internal/linker/linker_test.go)
3. [`internal/cleaner/cleaner_test.go`](../internal/cleaner/cleaner_test.go)
4. [`internal/runner/review_mode_test.go`](../internal/runner/review_mode_test.go)
5. [`internal/runner/risk_test.go`](../internal/runner/risk_test.go)
6. [`internal/tui/review_test.go`](../internal/tui/review_test.go)
7. [`internal/tui/layout_test.go`](../internal/tui/layout_test.go)

可以把这些测试理解成 4 组问题:

- 配置层有没有把输入正确归一化
- 执行层有没有做对文件系统动作
- runner 有没有把模式和风险处理对
- TUI 和文本输出有没有把结果稳定展示出来

## 如果你只想快速 review 一轮

最短路线建议是:

1. 读 [`internal/runner/run.go`](../internal/runner/run.go)
2. 读 [`internal/config/build.go`](../internal/config/build.go)
3. 读 [`internal/linker/linker.go`](../internal/linker/linker.go)
4. 读 [`internal/cleaner/cleaner.go`](../internal/cleaner/cleaner.go)
5. 读 [`internal/tui/review_panels.go`](../internal/tui/review_panels.go)
6. 再看 [`internal/runner/risk_test.go`](../internal/runner/risk_test.go)

这样你会最快抓到:

- 配置边界
- 文件系统风险边界
- review / confirm 的交互边界

## 最后一个建议

如果你之后继续自己维护这个项目, 最值得保持的习惯是:

- 新功能先看 `runner` 会不会受影响
- 新配置先看 `config` 是否还能保持“执行层只拿最终值”
- 新输出先尽量复用 `Entry` 或 `ReviewData`
- 新高风险行为先放进现有的 protected/risky 模型里, 不要另起一套确认逻辑

这样后续代码不会重新散掉.
