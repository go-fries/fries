package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_ChineseText(t *testing.T) {
	mermaid := `
gantt
    dateFormat HH:mm
    axisFormat %H:%M
    Initial vert : vert, v1, 17:30, 2m
    Task A : 3m
    Task B : 8m
    Final vert : vert, v2, 17:58, 4m

`
	out := filepath.Join("/tmp", "gantt.png")
	t.Logf("writing %s", out)
	in := Input{
		Source:     mermaid,
		OutputPath: out,
	}
	res, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render chinese failed: %v", err)
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
}
