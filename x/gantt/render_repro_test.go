package gantt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
	"github.com/go-fries/fries/x/gantt/v3/internal/render"
)

func TestRender_ReproducibleWithTodayOff(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mermaid_full", "repro_cases.gantt"))
	if err != nil {
		t.Fatalf("read repro: %v", err)
	}
	input := Input{
		Source:             string(data),
		Writer:             &bytes.Buffer{},
		Timezone:           "UTC",
		DisableTodayMarker: true,
	}

	buf1 := &bytes.Buffer{}
	input.Writer = buf1
	if _, err := Render(t.Context(), input); err != nil {
		t.Fatalf("render first: %v", err)
	}

	buf2 := &bytes.Buffer{}
	input.Writer = buf2
	if _, err := Render(t.Context(), input); err != nil {
		t.Fatalf("render second: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Fatalf("expected deterministic output with todayMarker off")
	}
}

func TestParse_TimezoneOverride(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mermaid_full", "repro_cases.gantt"))
	if err != nil {
		t.Fatalf("read repro: %v", err)
	}
	model, err := parser.Parse(string(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model.Calendar.Timezone = "Asia/Shanghai"
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	found := false
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			if task.ID == "r1" {
				if task.Start.Location().String() != "Asia/Shanghai" {
					t.Fatalf("timezone not applied")
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("task r1 not found")
	}
}

func TestRender_ManualCase_SectionSplit(t *testing.T) {
	src := `
gantt
    title A Gantt Diagram
    dateFormat YYYY-MM-DD
    section Section
        A task          :a1, 2014-01-01, 30d
        Another task    :after a1, 20d
    section Another
        Task in Another :2014-01-12, 12d
        another task    :24d
`
	out := filepath.Join(os.TempDir(), "gantt_case1.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render manual case failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}
	t.Logf("rendered manual case to %s", out)

	// 解析并验证起止时间，确保调度符合预期
	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse manual case: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule manual case: %v", err)
	}
	type span struct {
		start time.Time
		end   time.Time
	}
	expect := map[string]span{
		"A task":          {start: time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC), end: time.Date(2014, 1, 31, 0, 0, 0, -1, time.UTC)},
		"Another task":    {start: time.Date(2014, 1, 31, 0, 0, 0, 0, time.UTC), end: time.Date(2014, 2, 20, 0, 0, 0, -1, time.UTC)}, // after a1
		"Task in Another": {start: time.Date(2014, 1, 12, 0, 0, 0, 0, time.UTC), end: time.Date(2014, 1, 24, 0, 0, 0, -1, time.UTC)},
		"another task":    {start: time.Date(2014, 1, 24, 0, 0, 0, 0, time.UTC), end: time.Date(2014, 2, 17, 0, 0, 0, -1, time.UTC)}, // chained in same section
	}
	checked := 0
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			exp, ok := expect[task.Name]
			if !ok {
				continue
			}
			checked++
			if !task.Start.Equal(exp.start) {
				t.Fatalf("task %s start mismatch: got %v expect %v", task.Name, task.Start, exp.start)
			}
			if !task.End.Equal(exp.end) {
				t.Fatalf("task %s end mismatch: got %v expect %v", task.Name, task.End, exp.end)
			}
		}
	}
	if checked != len(expect) {
		t.Fatalf("expected to check %d tasks, checked %d", len(expect), checked)
	}
}

func durationToDuration(d parser.DurationSpec) time.Duration {
	switch d.Unit {
	case parser.DurationMinute:
		return time.Duration(d.Value) * time.Minute
	case parser.DurationHour:
		return time.Duration(d.Value) * time.Hour
	case parser.DurationWeek:
		return time.Duration(d.Value*7*24) * time.Hour
	case parser.DurationMonth:
		return time.Duration(d.Value*30*24) * time.Hour
	case parser.DurationDay:
		fallthrough
	default:
		return time.Duration(d.Value*24) * time.Hour
	}
}

func TestRender_ManualCase_Case2WeekendExcludes(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mermaid_full", "case2.gantt"))
	if err != nil {
		t.Fatalf("read case2: %v", err)
	}
	src := string(data)
	out := filepath.Join(os.TempDir(), "gantt_case2.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render case2 failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("output not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}
	t.Logf("rendered case2 to %s", out)

	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse case2: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule case2: %v", err)
	}

	// 验证自动刻度在日模式下不会过密
	minStart, maxEnd := minMax(model)
	autoMin, autoDay := render.AutoTickIntervalForTest(minStart, maxEnd, false)
	if autoDay != 1 {
		t.Fatalf("expected auto day tick 1 for case2, got min %d day %d", autoMin, autoDay)
	}

	expect := map[string]struct {
		start string
		end   string
	}{
		"Completed task":                        {"2014-01-06", "2014-01-08"},
		"Active task":                           {"2014-01-09", "2014-01-13"},
		"Future task":                           {"2014-01-14", "2014-01-20"},
		"Future task2":                          {"2014-01-21", "2014-01-27"},
		"Completed task in the critical line":   {"2014-01-06", "2014-01-06"},
		"Implement parser and jison":            {"2014-01-09", "2014-01-10"},
		"Create tests for parser":               {"2014-01-11", "2014-01-15"},
		"Future task in critical line":          {"2014-01-16", "2014-01-22"},
		"Create tests for renderer":             {"2014-01-23", "2014-01-24"},
		"Add to mermaid":                        {"2014-01-24", "2014-01-24"},
		"Functionality added":                   {"2014-01-25", "2014-01-25"},
		"Describe gantt syntax":                 {"2014-01-09", "2014-01-13"},
		"Add gantt diagram to demo page":        {"2014-01-14", "2014-01-14"},
		"Add another diagram to demo page":      {"2014-01-14", "2014-01-16"},
		"Describe gantt syntax Last section":    {"2014-01-16", "2014-01-21"},
		"Add gantt diagram to demo page Last":   {"2014-01-21", "2014-01-22"},
		"Add another diagram to demo page Last": {"2014-01-22", "2014-01-24"},
	}

	findEnd := func(task parser.Task) time.Time {
		if !task.End.IsZero() {
			return task.End
		}
		return task.Start.Add(durationToDuration(task.Duration) - time.Nanosecond)
	}

	checked := 0
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			key := task.Name
			// disambiguate duplicates by suffixing section for last section tasks
			if task.Name == "Describe gantt syntax" && sec.Name == "Last section" {
				key = "Describe gantt syntax Last section"
			}
			if task.Name == "Add gantt diagram to demo page" && sec.Name == "Last section" {
				key = "Add gantt diagram to demo page Last"
			}
			if task.Name == "Add another diagram to demo page" && sec.Name == "Last section" {
				key = "Add another diagram to demo page Last"
			}
			exp, ok := expect[key]
			if !ok {
				continue
			}
			checked++
			start := task.Start.Format("2006-01-02")
			end := findEnd(task).Format("2006-01-02")
			if start != exp.start || end != exp.end {
				t.Fatalf("task %s start/end mismatch: got %s -> %s, expect %s -> %s", key, start, end, exp.start, exp.end)
			}
		}
	}
	if checked != len(expect) {
		t.Fatalf("expected to check %d tasks, checked %d", len(expect), checked)
	}
}

func minMax(m parser.Model) (time.Time, time.Time) {
	var min time.Time
	var max time.Time
	first := true
	for _, sec := range m.Sections {
		for _, task := range sec.Tasks {
			if task.Start.IsZero() {
				continue
			}
			end := task.End
			if end.IsZero() {
				end = task.Start.Add(durationToDuration(task.Duration) - time.Nanosecond)
			}
			if first {
				min, max = task.Start, end
				first = false
				continue
			}
			if task.Start.Before(min) {
				min = task.Start
			}
			if end.After(max) {
				max = end
			}
		}
	}
	return min, max
}

func TestRender_ManualCase_UntilAndAfter(t *testing.T) {
	src := `
gantt
    apple :a, 2017-07-20, 1w
    banana :crit, b, 2017-07-23, 1d
    cherry :active, c, after b a, 1d
    kiwi   :d, 2017-07-20, until b c
`
	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	expectDays := map[string]int{
		"a": 7,
		"b": 1,
		"c": 1,
		"d": 3,
	}
	expectStart := map[string]string{
		"a": "2017-07-20",
		"b": "2017-07-23",
		"c": "2017-07-27", // after both a (ends 7/26) and b (7/23)
		"d": "2017-07-20",
	}
	expectEnd := map[string]string{
		"d": "2017-07-22", // until min(start b, start c) - 1 day
	}

	checked := 0
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			id := task.ID
			if id == "" {
				continue
			}
			if want, ok := expectDays[id]; ok {
				checked++
				if task.DurationDays != want {
					t.Fatalf("task %s duration days mismatch: got %d want %d", id, task.DurationDays, want)
				}
				start := task.Start.Format("2006-01-02")
				if start != expectStart[id] {
					t.Fatalf("task %s start mismatch: got %s want %s", id, start, expectStart[id])
				}
				if endWant, ok := expectEnd[id]; ok {
					end := task.End.Format("2006-01-02")
					if end != endWant {
						t.Fatalf("task %s end mismatch: got %s want %s", id, end, endWant)
					}
				}
			}
		}
	}
	if checked != len(expectDays) {
		t.Fatalf("expected to check %d tasks, checked %d", len(expectDays), checked)
	}

	// 渲染生成图片，便于肉眼校对
	out := filepath.Join(os.TempDir(), "gantt_until_after.png")
	resOut, renderErr := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if renderErr != nil {
		t.Fatalf("render until/after failed: %v", renderErr)
	}
	if resOut.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	info, statErr := os.Stat(out)
	if statErr != nil {
		t.Fatalf("output not found: %v", statErr)
	}
	if info.Size() == 0 {
		t.Fatalf("output file is empty")
	}
	t.Logf("rendered until/after case to %s", out)
}

func TestRender_ManualCase_MinuteMilestones(t *testing.T) {
	src := `
gantt
    dateFormat HH:mm
    axisFormat %H:%M
    Initial milestone : milestone, m1, 17:49, 2m
    Task A : 10m
    Task B : 5m
    Final milestone : milestone, m2, 18:08, 4m
`
	out := filepath.Join(os.TempDir(), "gantt_minute_axis.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render minute case failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("output file missing or empty: %v", err)
	}
	t.Logf("rendered minute axis case to %s", out)

	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse minute case: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule minute case: %v", err)
	}
	type span struct {
		start string
		end   string
	}
	expect := map[string]span{
		"Initial milestone": {"17:49", "17:51"},
		"Task A":            {"17:51", "18:01"},
		"Task B":            {"18:01", "18:06"},
		"Final milestone":   {"18:08", "18:12"},
	}

	checked := 0
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			exp, ok := expect[task.Name]
			if !ok {
				continue
			}
			checked++
			start := task.Start.Format("15:04")
			end := task.End.Format("15:04")
			if start != exp.start || end != exp.end {
				t.Fatalf("task %s start/end mismatch: got %s -> %s, want %s -> %s", task.Name, start, end, exp.start, exp.end)
			}
		}
	}
	if checked != len(expect) {
		t.Fatalf("expected to check %d tasks, checked %d", len(expect), checked)
	}
}

func TestRender_ManualCase_VertMarkers(t *testing.T) {
	src := `
gantt
    dateFormat HH:mm
    axisFormat %H:%M
    Initial vert : vert, v1, 17:30, 2m
    Task A : 3m
    Task B : 8m
    Final vert : vert, v2, 17:58, 4m
`
	out := filepath.Join(os.TempDir(), "gantt_vert_axis.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render vert case failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("output file missing or empty: %v", err)
	}
	t.Logf("rendered vert case to %s", out)

	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse vert case: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule vert case: %v", err)
	}
	type span struct {
		start string
		end   string
	}
	expect := map[string]span{
		"Task A": {"17:33", "17:36"},
		"Task B": {"17:36", "17:44"},
	}
	checked := 0
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			exp, ok := expect[task.Name]
			if !ok {
				continue
			}
			checked++
			start := task.Start.Format("15:04")
			end := task.End.Format("15:04")
			if start != exp.start || end != exp.end {
				t.Fatalf("task %s start/end mismatch: got %s -> %s, want %s -> %s", task.Name, start, end, exp.start, exp.end)
			}
		}
	}
	if checked != len(expect) {
		t.Fatalf("expected to check %d tasks, checked %d", len(expect), checked)
	}
	if len(model.Verticals) != 2 {
		t.Fatalf("expected 2 vertical markers, got %d", len(model.Verticals))
	}
	v1 := model.Verticals[0]
	v2 := model.Verticals[1]
	if v1.Start.Format("15:04") != "17:30" || v2.Start.Format("15:04") != "17:58" {
		t.Fatalf("vertical markers start mismatch: v1 %s v2 %s", v1.Start.Format("15:04"), v2.Start.Format("15:04"))
	}
}

func TestRender_ManualCase_CustomWeekendExclude(t *testing.T) {
	src := `
gantt
    title A Gantt Diagram Excluding Fri - Sat weekends
    dateFormat YYYY-MM-DD
    excludes weekends
    weekend friday
    section Section
        A task          :a1, 2024-01-01, 30d
        Another task    :after a1, 20d
`
	out := filepath.Join(os.TempDir(), "gantt_weekend_custom.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render custom weekend failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("output file missing or empty: %v", err)
	}
	t.Logf("rendered custom weekend case to %s", out)

	model, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse custom weekend: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("schedule custom weekend: %v", err)
	}

	var a1, a2 *parser.Task
	for si := range model.Sections {
		for ti := range model.Sections[si].Tasks {
			task := &model.Sections[si].Tasks[ti]
			if task.ID == "a1" {
				a1 = task
			}
			if task.Name == "Another task" {
				a2 = task
			}
		}
	}
	if a1 == nil || a2 == nil {
		t.Fatalf("tasks not found: a1=%v a2=%v", a1 != nil, a2 != nil)
	}
	if a1.End.Format("2006-01-02") != "2024-02-11" {
		t.Fatalf("a1 end mismatch: %s", a1.End.Format("2006-01-02"))
	}
	if a2.Start.Format("2006-01-02") != "2024-02-12" {
		t.Fatalf("a2 start mismatch: %s", a2.Start.Format("2006-01-02"))
	}
}
