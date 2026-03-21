package output

// Mode 控制普通终端输出的详细程度.
type Mode int

const (
	ModeDefault Mode = iota
	ModeVerbose
	ModeQuiet
)

// Status 表示单条执行记录最终落到的状态.
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
type Entry struct {
	Stage    string
	Target   string
	Source   string
	Decision string
	Status   Status
	Message  string
}

// Summary 是执行结束后的聚合统计.
type Summary struct {
	Created  int
	Linked   int
	Skipped  int
	Replaced int
	Deleted  int
	Failed   int
}

// Options 控制普通输出的详细程度和着色行为.
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
type RiskItem struct {
	Kind string
	Path string
}

// ReviewData 是 dry-run/check 在文本和 TUI 两种视图下共享的数据模型.
type ReviewData struct {
	Mode         ReviewMode
	ConfigPath   string
	BaseDir      string
	StageCounts  StageCounts
	Entries      []Entry
	Risks        []RiskItem
	Summary      Summary
	Result       string
	VerboseLines []string
}
