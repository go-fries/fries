package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_EmptySource(t *testing.T) {
	_, err := Render(t.Context(), Input{OutputPath: filepath.Join(os.TempDir(), "out.png")})
	if err == nil {
		t.Fatalf("expected error for empty source")
	}
}

func TestRender_MissingOutput(t *testing.T) {
	_, err := Render(t.Context(), Input{Source: "gantt\nsection A\nTask :a, 1d"})
	if err == nil {
		t.Fatalf("expected error when no output target")
	}
}
