package output

// Mode 控制普通终端输出的详细程度.
// 这套模式只作用于文本输出层, 不改变 create/link/clean 的执行行为.
type Mode int

const (
	ModeDefault Mode = iota
	ModeVerbose
	ModeQuiet
)

// Status 表示单条执行记录最终落到的状态.
// 普通输出, 审阅输出和测试断言都会复用这一组状态值.
type Status string

const (
	StatusInfo     Status = "info"
	StatusCreated  Status = "created"
	StatusLinked   Status = "linked"
	StatusSkipped  Status = "skipped"
	StatusReplaced Status = "replaced"
	StatusDeleted  Status = "deleted"
	StatusFailed   Status = "failed"
)

// Entry 是单条 create/link/clean 动作对应的统一输出模型.
// 执行层把差异很大的 create/link/clean 结果都压平到同一模型,
// 这样输出层就不需要知道条目来自哪个具体包.
type Entry struct {
	Stage    string
	Target   string
	Source   string
	Decision string
	Status   Status
	Message  string
}

// Summary 是执行结束后的聚合统计.
// 它既用于真实执行后的 summary, 也用于 dry-run/check 的审阅摘要.
type Summary struct {
	Created  int
	Linked   int
	Skipped  int
	Replaced int
	Deleted  int
	Failed   int
}

// Options 控制普通输出的详细程度和着色行为.
// 这里不包含业务开关, 只包含“怎么显示”.
type Options struct {
	Mode        Mode
	DryRun      bool
	EnableColor bool
}

// Add 会忽略纯信息型状态, 只统计最终结果型状态.
func (s *Summary) Add(status Status) {
	switch status {
	case StatusCreated:
		s.Created++
	case StatusInfo:
	case StatusLinked:
		s.Linked++
	case StatusSkipped:
		s.Skipped++
	case StatusReplaced:
		s.Replaced++
	case StatusDeleted:
		s.Deleted++
	case StatusFailed:
		s.Failed++
	}
}

// ReviewMode 区分 dry-run 和 check 两种审阅语义.
type ReviewMode string

const (
	ReviewModeDryRun ReviewMode = "dry-run"
	ReviewModeCheck  ReviewMode = "check"
)

// StageCounts 用于概览区展示三个固定阶段的动作数量.
type StageCounts struct {
	Create int
	Link   int
	Clean  int
}

// RiskItem 是审阅界面和确认界面共用的风险展示模型.
// Allowed 用来表达“风险仍然存在, 但当前命令已经显式放行”.
type RiskItem struct {
	Kind    string
	Path    string
	Allowed bool
}

type ConfigField struct {
	Key   string
	Value string
}

type ConfigGroup struct {
	Scope  string
	Fields []ConfigField
}

// ReviewData 是 dry-run/check 在文本和 TUI 两种视图下共享的数据模型.
// runner 先把执行结果压平到这里, 输出层再决定如何展示, 这样文本和 TUI 可以共享同一份事实来源.
type ReviewData struct {
	Mode         ReviewMode
	ConfigPath   string
	BaseDir      string
	StageCounts  StageCounts
	Entries      []Entry
	Risks        []RiskItem
	Summary      Summary
	Result       string
	ConfigGroups []ConfigGroup
}
