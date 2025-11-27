package gantt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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
