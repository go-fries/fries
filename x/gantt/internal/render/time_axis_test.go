package render

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
)

func TestTimelineMinutesRangeAndOrder(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "mermaid_full", "minute_axis.gantt"))
	if err != nil {
		t.Fatalf("read minute axis sample: %v", err)
	}
	model, err := parser.Parse(string(data))
	if err != nil {
		t.Fatalf("parse minute sample: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule minute sample: %v", err)
	}

	minStart, maxEnd := timelineBounds(model)
	if minStart.Hour() != 17 || minStart.Minute() != 49 {
		t.Fatalf("unexpected min start: %v", minStart)
	}
	if maxEnd.Before(minStart.Add(20 * time.Minute)) {
		t.Fatalf("unexpected max end (too short): %v", maxEnd)
	}

	if len(model.Sections) == 0 || len(model.Sections[0].Tasks) < 4 {
		t.Fatalf("expect at least 4 tasks, got %d", len(model.Sections))
	}
	tasks := model.Sections[0].Tasks
	m1, a, b, m2 := tasks[0], tasks[1], tasks[2], tasks[3]

	if !a.Start.After(m1.End) {
		t.Fatalf("task A should start after m1 end: %v <= %v", a.Start, m1.End)
	}
	if !b.Start.After(a.End) {
		t.Fatalf("task B should start after A end: %v <= %v", b.Start, a.End)
	}
	if !m2.Start.After(b.End) {
		t.Fatalf("m2 should start after task B: %v <= %v", m2.Start, b.End)
	}

	aDur := a.End.Sub(a.Start).Minutes()
	bDur := b.End.Sub(b.Start).Minutes()
	if bDur <= 0 || aDur <= 0 {
		t.Fatalf("durations should be positive: A %.2f, B %.2f", aDur, bDur)
	}
	ratio := aDur / bDur
	if ratio < 1.8 || ratio > 2.2 {
		t.Fatalf("duration ratio unexpected, want about 2:1, got %.2f (A %.2f, B %.2f)", ratio, aDur, bDur)
	}
}

func TestWeekStartDefaultsAndCustom(t *testing.T) {
	var weekStart *time.Weekday
	tick := parser.TickInterval{Value: 1, Unit: "week", Valid: true}

	if weekStart != nil {
		t.Fatalf("expected nil start initially")
	}

	if tick.Valid && tick.Unit == "week" {
		if weekStart == nil {
			ws := time.Sunday
			weekStart = &ws
		}
	}
	if weekStart == nil || *weekStart != time.Sunday {
		t.Fatalf("expected default week start Sunday, got %v", weekStart)
	}

	custom := time.Monday
	weekStart = &custom
	aligned := alignToWeekStart(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), weekStart) // Wed -> Mon
	if aligned.Weekday() != time.Monday {
		t.Fatalf("expected align to Monday, got %s", aligned.Weekday())
	}
	if aligned.Day() != 1 {
		t.Fatalf("expected align to start of week (1st), got %d", aligned.Day())
	}
}
