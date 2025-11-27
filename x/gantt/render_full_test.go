package gantt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
)

func TestRenderFullFeatures(t *testing.T) {
	base := filepath.Join("testdata", "mermaid_full", "valid_full.gantt")
	data, err := os.ReadFile(base)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	model, err := parser.Parse(string(data))
	if err != nil {
		t.Fatalf("parse sample: %v", err)
	}
	model, err = parser.ResolveSchedule(model)
	if err != nil {
		t.Fatalf("resolve schedule: %v", err)
	}

	taskByID := make(map[string]parser.Task)
	for _, sec := range model.Sections {
		for _, task := range sec.Tasks {
			taskByID[task.ID] = task
		}
	}

	assertTask := func(id, startDate, endDate string) {
		t.Helper()
		task, ok := taskByID[id]
		if !ok {
			t.Fatalf("task %s not found", id)
		}
		if task.Start.Format("2006-01-02") != startDate {
			t.Fatalf("task %s start mismatch, got %s", id, task.Start.Format("2006-01-02"))
		}
		endStr := task.End.Format("2006-01-02")
		if endStr != endDate {
			t.Fatalf("task %s end mismatch, got %s", id, endStr)
		}
	}

	assertTask("a1", "2025-01-06", "2025-01-07")
	// after a1, spans周末且排除 2025-01-10，最晚工作日落在 2025-01-11
	assertTask("a2", "2025-01-08", "2025-01-11")
	if taskByID["m1"].DurationDays != 0 || !taskByID["m1"].IsMilestone {
		t.Fatalf("milestone duration incorrect")
	}

	buf := &bytes.Buffer{}
	_, err = Render(t.Context(), Input{
		Source:             string(data),
		Writer:             buf,
		Timezone:           "UTC",
		DisableTodayMarker: true,
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected png bytes")
	}
}
