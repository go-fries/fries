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
	const tickUnitWeek = "week"
	tick := parser.TickInterval{Value: 1, Unit: tickUnitWeek, Valid: true}

	if weekStart != nil {
		t.Fatalf("expected nil start initially")
	}

	if tick.Valid && tick.Unit == tickUnitWeek {
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

func TestAutoTickIntervalRules(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		span     time.Duration
		timeMode bool
		wantMin  int
		wantDay  int
	}{
		{2 * time.Hour, true, 5, 0},
		{10 * time.Hour, true, 60, 0},
		{5 * 24 * time.Hour, true, 60, 0},
		{40 * 24 * time.Hour, true, 7 * 24 * 60, 0},
		{400 * 24 * time.Hour, true, 30 * 24 * 60, 0},
		{10 * 24 * time.Hour, false, 0, 1},
		{50 * 24 * time.Hour, false, 0, 7},
		{800 * 24 * time.Hour, false, 0, 30},
	}
	for _, c := range cases {
		min := base
		max := base.Add(c.span)
		minTick, dayTick := autoTickInterval(min, max, c.timeMode)
		if minTick != c.wantMin || dayTick != c.wantDay {
			t.Fatalf("span %v timeMode=%v got min %d day %d, want %d %d", c.span, c.timeMode, minTick, dayTick, c.wantMin, c.wantDay)
		}
	}
}

func TestNormalizeMinuteTicksDensity(t *testing.T) {
	// 23 天跨度，分钟模式应提升到较大的间隔避免过密
	totalMinutes := 23 * 24 * 60
	tick, label := normalizeMinuteTicks(totalMinutes, 60, false)
	if totalMinutes/tick > 120 {
		t.Fatalf("ticks too dense: %d ticks for %d minutes", totalMinutes/tick, totalMinutes)
	}
	if totalMinutes/label > 60 {
		t.Fatalf("labels too dense: %d labels for %d minutes", totalMinutes/label, totalMinutes)
	}
	if tick < 60 {
		t.Fatalf("expected tick >= 60 minutes for long span, got %d", tick)
	}

	// 短 2 小时内仍保持小刻度
	totalMinutes = 120
	tick, label = normalizeMinuteTicks(totalMinutes, 5, false)
	if tick != 5 {
		t.Fatalf("expected 5-minute ticks for short span, got %d", tick)
	}
	if label != 5 {
		t.Fatalf("expected label step 5 for short span, got %d", label)
	}
}
