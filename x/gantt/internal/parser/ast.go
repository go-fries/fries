package parser

import "time"

// DurationUnit 表示持续时间单位。
type DurationUnit int

const (
	DurationDay DurationUnit = iota
	DurationHour
	DurationMinute
	DurationWeek
	DurationMonth
)

// TaskStatus 表示任务状态标记。
type TaskStatus int

const (
	StatusNormal TaskStatus = iota
	StatusDone
	StatusActive
	StatusCritical
	StatusMilestone
)

// DependencyType 表示依赖类型。
type DependencyType int

const (
	DepAfter DependencyType = iota
	DepBefore
)

// DurationSpec 捕获 mermaid 中的持续时间定义。
type DurationSpec struct {
	Value int
	Unit  DurationUnit
}

// Dependency 描述任务间的依赖。
type Dependency struct {
	Type   DependencyType
	Target string
}

// TodayMarker 控制今日标记。
type TodayMarker struct {
	Enabled bool
	Date    time.Time
	HasDate bool
}

// TickInterval 控制轴刻度间隔。
type TickInterval struct {
	Value int
	Unit  string
	Valid bool
}

// Calendar 记录日历规则。
type Calendar struct {
	Timezone       string
	ExcludeWeekend bool
	WeekendDays    []time.Weekday // 可自定义周末集合；为空时使用默认周末
	ExcludeDates   []time.Time
	IncludeDates   []time.Time
}

// Task 表示解析后的任务。
type Task struct {
	Name         string
	ID           string
	ExplicitID   bool
	Section      string
	Index        int
	Status       TaskStatus
	IsMilestone  bool
	IsVertical   bool
	HasTime      bool
	Progress     int // 0-100
	Resources    []string
	Dependencies []Dependency

	StartExpr        string // 绝对日期或相对表达式
	EndExpr          string
	Duration         DurationSpec
	DurationDays     int
	DurationExplicit bool

	Start    time.Time
	End      time.Time
	HasStart bool
	HasEnd   bool

	Line   int
	Column int
}

// Section 表示分组。
type Section struct {
	Name  string
	Tasks []Task
}

// Model 表示解析结果。
type Model struct {
	Title      string
	DateFormat string
	AxisFormat string
	Tick       TickInterval
	WeekStart  *time.Weekday
	Today      TodayMarker
	Calendar   Calendar
	Sections   []Section
	Verticals  []Task
}

// ParseError 携带行列信息的错误。
type ParseError struct {
	Line    int
	Column  int
	Message string
}

func (e ParseError) Error() string {
	return e.Message
}
