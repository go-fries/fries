package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_InvalidFile(t *testing.T) {
	_, err := Render(t.Context(), Input{FromFile: true, Source: filepath.Join("not", "exist"), OutputPath: filepath.Join("/tmp", "x.png")})
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}

func TestRender_InvalidOutput(t *testing.T) {
	// missing output target
	_, err := Render(t.Context(), Input{Source: "gantt\nsection A\nTask :a, 1d"})
	if err == nil {
		t.Fatalf("expected error for missing output")
	}
}

func TestRender_InvalidContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mermaid_full", "invalid_cases.gantt"))
	if err != nil {
		t.Fatalf("read invalid cases: %v", err)
	}
	_, err = Render(t.Context(), Input{Source: string(data), Writer: &mockWriter{}})
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
