package parser

import (
	"testing"
	"time"
)

func TestParseBasic(t *testing.T) {
	src := `gantt
section 开发
任务A :a1, 2024-01-01, 3d
任务B :after a1, 2d`
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(m.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(m.Sections))
	}
	if len(m.Sections[0].Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(m.Sections[0].Tasks))
	}
	if m.Sections[0].Tasks[0].Name != "任务A" {
		t.Fatalf("unexpected task name %s", m.Sections[0].Tasks[0].Name)
	}
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatalf("expected error on empty source")
	}
}

func TestParse_ExcludesWeekends(t *testing.T) {
	src := `gantt
dateFormat YYYY-MM-DD
excludes weekends
section 开发
任务A :a1, 2024-01-01, 3d
section 测试
任务B :a2, 2024-01-04, 2d`
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !m.Calendar.ExcludeWeekend {
		t.Fatalf("expected ExcludeWeekends true")
	}
	if len(m.Sections) != 2 {
		t.Fatalf("expected 2 sections")
	}
	if m.Sections[0].Tasks[0].Section != "开发" {
		t.Fatalf("section name mismatch: %s", m.Sections[0].Tasks[0].Section)
	}
}

func TestSchedule_ExcludeWeekends_CustomWeekend(t *testing.T) {
	src := `gantt
dateFormat YYYY-MM-DD
excludes weekends
weekend friday
section Section
    A task :a1, 2024-01-01, 30d
`
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	m, err = ResolveSchedule(m)
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}
	if len(m.Sections) == 0 || len(m.Sections[0].Tasks) == 0 {
		t.Fatalf("no tasks")
	}
	task := m.Sections[0].Tasks[0]
	if got := task.End.Format("2006-01-02"); got != "2024-02-11" {
		t.Fatalf("expected end 2024-02-11, got %s", got)
	}
}

func TestSchedule_ExcludesWeekdayName(t *testing.T) {
	src := `gantt
dateFormat YYYY-MM-DD
excludes friday
section S
    A task :a1, 2024-01-01, 5d
`
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !m.Calendar.ExcludeWeekend {
		t.Fatalf("expected ExcludeWeekend true when weekday excluded")
	}
	if len(m.Calendar.WeekendDays) != 1 || m.Calendar.WeekendDays[0] != time.Friday {
		t.Fatalf("expected only friday in excluded weekdays, got %v", m.Calendar.WeekendDays)
	}
	m, err = ResolveSchedule(m)
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}
	task := m.Sections[0].Tasks[0]
	if got := task.End.Format("2006-01-02"); got != "2024-01-06" {
		t.Fatalf("expected end 2024-01-06, got %s", got)
	}
}
