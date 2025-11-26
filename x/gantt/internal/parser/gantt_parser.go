package parser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Task 表示解析后的任务。
type Task struct {
	Name         string
	Section      string
	Index        int
	Start        time.Time
	DurationDays int
}

// Section 表示分组。
type Section struct {
	Name  string
	Tasks []Task
}

// Model 表示解析结果。
type Model struct {
	Title           string
	Sections        []Section
	DateFormatLine  string
	ExcludeWeekends bool
}

const (
	minTaskParts      = 2
	minRightParts     = 2
	minDurationFields = 3
)

// dateLayout 将 Mermaid 的 dateFormat 转换为 Go layout。
func dateLayout(line string) string {
	format := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "dateformat"))
	if format == "" {
		return "2006-01-02"
	}
	// 支持常见片段
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"mm", "01",
		"dd", "02",
	)
	layout := replacer.Replace(strings.ToLower(format))
	return layout
}

func parseDuration(val string) int {
	val = strings.TrimSpace(strings.TrimSuffix(val, "d"))
	if val == "" {
		return 1
	}
	var days int
	if _, err := fmt.Sscanf(val, "%d", &days); err != nil {
		return 1
	}
	if days <= 0 {
		return 1
	}
	return days
}

// Parse 从字符串解析 Gantt。
func Parse(src string) (Model, error) {
	if strings.TrimSpace(src) == "" {
		return Model{}, fmt.Errorf("source is empty")
	}
	scanner := bufio.NewScanner(strings.NewReader(src))
	model := Model{}
	sectionName := ""
	index := 0
	layout := "2006-01-02"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "gantt") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "title") {
			model.Title = strings.TrimSpace(line[5:])
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "dateformat") {
			layout = dateLayout(line)
			model.DateFormatLine = layout
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "excludes") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "weekend") || strings.Contains(lower, "weekends") {
				model.ExcludeWeekends = true
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "section") {
			sectionName = strings.TrimSpace(line[len("section"):])
			model.Sections = append(model.Sections, Section{Name: sectionName})
			continue
		}
		// 任务行格式： 任务名 :标识, 开始日期, 2d
		parts := strings.Split(line, ":")
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		task := Task{Name: name, Section: sectionName, Index: index}
		if len(parts) >= minTaskParts {
			right := strings.Split(parts[1], ",")
			if len(right) >= minRightParts {
				startStr := strings.TrimSpace(right[1])
				if t, err := time.Parse(layout, startStr); err == nil {
					task.Start = t
				}
			}
			if len(right) >= minDurationFields {
				task.DurationDays = parseDuration(strings.TrimSpace(right[2]))
			}
		}
		if task.DurationDays == 0 {
			task.DurationDays = 1
		}
		if len(model.Sections) == 0 {
			model.Sections = append(model.Sections, Section{Name: ""})
		}
		model.Sections[len(model.Sections)-1].Tasks = append(model.Sections[len(model.Sections)-1].Tasks, task)
		index++
	}
	if err := scanner.Err(); err != nil {
		return Model{}, err
	}
	totalTasks := 0
	for _, s := range model.Sections {
		totalTasks += len(s.Tasks)
	}
	if totalTasks == 0 {
		return Model{}, fmt.Errorf("no tasks parsed")
	}
	return model, nil
}

// ParseFile 从文件读取并解析。
func ParseFile(path string) (Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Model{}, fmt.Errorf("read source file: %w", err)
	}
	return Parse(string(data))
}
