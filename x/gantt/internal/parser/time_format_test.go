package parser

import (
	"testing"
	"time"
)

func TestConvertDayjsLayout(t *testing.T) {
	in := "YYYY-MM-DD HH:mm:ss.SSS Z"
	got := convertDayjsLayout(in)
	expect := "2006-01-02 15:04:05.000 -07:00"
	if got != expect {
		t.Fatalf("layout mismatch got %s, expect %s", got, expect)
	}
}

func TestParse_TimeAndAxisFormats(t *testing.T) {
	src := `gantt
dateFormat HH:mm
axisFormat %H:%M
Initial milestone : milestone, m1, 17:49, 2m
Task A : 10m
Task B : 5m
Final milestone : milestone, m2, 18:08, 4m
`
	model, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if model.DateFormat != "15:04" {
		t.Fatalf("date format converted incorrectly: %s", model.DateFormat)
	}
	if model.AxisFormat != "15:04" {
		t.Fatalf("axis format converted incorrectly: %s", model.AxisFormat)
	}
	model, err = ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}
	m := model
	find := func(name string) Task {
		for _, sec := range m.Sections {
			for _, task := range sec.Tasks {
				if task.Name == name {
					return task
				}
			}
		}
		return Task{}
	}
	m1 := find("Initial milestone")
	if !m1.HasStart || m1.Start.Hour() != 17 || m1.Start.Minute() != 49 {
		t.Fatalf("m1 start incorrect: %v (expr %s)", m1.Start, m1.StartExpr)
	}
	a := find("Task A")
	b := find("Task B")
	m2 := find("Final milestone")
	t.Logf("m1 %v->%v, a %v->%v, b %v->%v, m2 %v", m1.Start, m1.End, a.Start, a.End, b.Start, b.End, m2.Start)
	m1End := m1.Start.Add(2 * time.Minute)
	m2Start := m2.Start
	if a.Start.Before(m1End) || a.End.After(m2Start) || a.End.Before(a.Start) {
		t.Fatalf("Task A not between m1 end and m2 start: %v -> %v (m1End %v, m2Start %v)", a.Start, a.End, m1End, m2.Start)
	}
	if b.Start.Before(a.End) || b.End.After(m2Start) || b.End.Before(b.Start) {
		t.Fatalf("Task B not after A and before m2: %v -> %v (m2Start %v)", b.Start, b.End, m2.Start)
	}
}

func TestParse_TickIntervalPattern(t *testing.T) {
	src := `gantt
tickInterval 2week
weekday monday
section S
  a : 2024-01-01, 1d
`
	m, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !m.Tick.Valid || m.Tick.Value != 2 || m.Tick.Unit != "week" {
		t.Fatalf("unexpected tick interval: %+v", m.Tick)
	}
	if m.WeekStart == nil || *m.WeekStart != time.Monday {
		t.Fatalf("expected week start Monday, got %v", m.WeekStart)
	}

	invalid := `gantt
tickInterval 0week
section S
  a : 2024-01-01, 1d
`
	m2, err := Parse(invalid)
	if err != nil {
		t.Fatalf("parse invalid failed: %v", err)
	}
	if m2.Tick.Valid {
		t.Fatalf("expected invalid tick to be ignored, got %+v", m2.Tick)
	}
}
