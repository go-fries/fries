package gantt

// Theme 定义渲染主题颜色。
type Theme struct {
	Name       string
	Background string
	Grid       string
	TaskFill   string
	TaskBorder string
	TaskText   string
	Text       string
	Milestone  string
	TodayLine  string
	Vertical   string
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
