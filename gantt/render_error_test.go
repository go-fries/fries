package gantt

import (
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
