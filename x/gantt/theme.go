package gantt

// Theme 定义渲染主题颜色，字段使用十六进制字符串（如 "#ffffff"）。
type Theme struct {
	Name       string // 主题名称，便于区分或日志
	Background string // 画布背景色
	Grid       string // 坐标轴与网格线颜色
	TaskFill   string // 任务条填充色
	TaskBorder string // 任务条边框色
	TaskText   string // 任务条内部文字颜色
	Text       string // 轴刻度、标题等通用文字颜色
	Milestone  string // 里程碑标记颜色
	TodayLine  string // 今日基准线颜色
	Vertical   string // 垂直标记（vert）颜色
}

func DefaultTheme() Theme {
	return Theme{
		Name:       "default",
		Background: "#ffffff",
		Grid:       "#e0e0e0",
		TaskFill:   "#8190DD",
		TaskBorder: "#2c3e50",
		TaskText:   "#FFFFFF",
		Text:       "#333333",
		Milestone:  "#e67e22",
		TodayLine:  "#d0021b",
		Vertical:   "#007acc",
	}
}

func DarkTheme() Theme {
	return Theme{
		Name:       "dark",
		Background: "#1c1c1e",
		Grid:       "#3a3a3c",
		TaskFill:   "#9AA7F0",
		TaskBorder: "#ecf0f1",
		TaskText:   "#f5f5f7",
		Text:       "#f5f5f7",
		Milestone:  "#f1c40f",
		TodayLine:  "#e74c3c",
		Vertical:   "#29b6f6",
	}
}

// MergeTheme 将 override 中非空字段覆盖 base。
func MergeTheme(base, override Theme) Theme {
	out := base
	if override.Name != "" {
		out.Name = override.Name
	}
	if override.Background != "" {
		out.Background = override.Background
	}
	if override.Grid != "" {
		out.Grid = override.Grid
	}
	if override.TaskFill != "" {
		out.TaskFill = override.TaskFill
	}
	if override.TaskBorder != "" {
		out.TaskBorder = override.TaskBorder
	}
	if override.TaskText != "" {
		out.TaskText = override.TaskText
	}
	if override.Text != "" {
		out.Text = override.Text
	}
	if override.Milestone != "" {
		out.Milestone = override.Milestone
	}
	if override.TodayLine != "" {
		out.TodayLine = override.TodayLine
	}
	if override.Vertical != "" {
		out.Vertical = override.Vertical
	}
	return out
}
