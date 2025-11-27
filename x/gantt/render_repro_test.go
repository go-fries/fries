package gantt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
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
