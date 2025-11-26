package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_ChineseText(t *testing.T) {
	mermaid := `
gantt
    title Gantt任务预览
    dateFormat YYYY-MM-DD
    excludes weekends
    section 研发
    任务1 :A同学, 2025-11-24, 2d
    任务2 :B同学, 2025-11-20, 2d
    section 测试
    测试任务1 :C同学, 2025-11-28, 2d
`
	out := filepath.Join(os.TempDir(), "gantt.png")
	in := Input{
		Source:     mermaid,
		OutputPath: out,
		Scale:      1,
		// FontPath 可留空使用默认字体，确保不 panic
	}
	_, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render chinese failed: %v", err)
	}
	_ = os.Remove(out)
}
