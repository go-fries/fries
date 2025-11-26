package gantt

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderBasic(t *testing.T) {
	out := filepath.Join(os.TempDir(), "gantt_test_basic.png")
	in := Input{
		Source: `gantt
section 开发
任务A :a1, 2024-01-01, 3d
任务B :after a1, 2d`,
		OutputPath: out,
	}
	res, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render failed: %v", err)
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
	_ = os.Remove(out)
}

func TestRenderWithWriter(t *testing.T) {
	buf := &mockWriter{}
	in := Input{
		Source: `gantt
section 测试
任务A :a1, 1d`,
		Writer: buf,
	}
	res, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(res.Bytes) == 0 {
		t.Fatalf("expected bytes in result")
	}
	if buf.size == 0 {
		t.Fatalf("writer not written")
	}
}

func TestRenderThemeDifference(t *testing.T) {
	src := `gantt
section 主题
任务A :a1, 1d`

	var bufDefault bytes.Buffer
	var bufDark bytes.Buffer

	inDefault := Input{Source: src, Writer: &bufDefault, Theme: DefaultTheme()}
	inDark := Input{Source: src, Writer: &bufDark, Theme: DarkTheme()}

	if _, err := Render(t.Context(), inDefault); err != nil {
		t.Fatalf("default theme render failed: %v", err)
	}
	if _, err := Render(t.Context(), inDark); err != nil {
		t.Fatalf("dark theme render failed: %v", err)
	}

	imgDefault, err := png.Decode(&bufDefault)
	if err != nil {
		t.Fatalf("decode default png: %v", err)
	}
	imgDark, err := png.Decode(&bufDark)
	if err != nil {
		t.Fatalf("decode dark png: %v", err)
	}

	bgDefault := imgDefault.At(0, 0)
	bgDark := imgDark.At(0, 0)
	if bgDefault == bgDark {
		t.Fatalf("expected theme backgrounds differ, got same color")
	}
}

type mockWriter struct{ size int }

func (m *mockWriter) Write(p []byte) (int, error) {
	m.size += len(p)
	return len(p), nil
}

func TestRender_LongTimelineManyTasks(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("gantt\n")
	sb.WriteString("dateFormat YYYY-MM-DD\n")
	sb.WriteString("section 大规模\n")
	start := "2025-01-01"
	for i := 0; i < 120; i++ {
		day := i % 90
		sb.WriteString("任务")
		sb.WriteString(strings.TrimSpace(strings.Repeat("X", (i%5)+1)))
		sb.WriteString(" ")
		sb.WriteString(":t")
		sb.WriteString(strings.TrimSpace(strings.Repeat("a", (i%3)+1)))
		sb.WriteString(", ")
		sb.WriteString(addDays(start, day))
		sb.WriteString(", 5d\n")
	}
	out := filepath.Join(os.TempDir(), "gantt_test_long_timeline.png")
	in := Input{
		Source:     sb.String(),
		OutputPath: out,
	}
	res, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render long timeline failed: %v", err)
	}
	if len(res.Bytes) == 0 {
		t.Fatalf("expected png bytes")
	}
	_ = os.Remove(out)
}

func addDays(date string, days int) string {
	t, _ := time.Parse("2006-01-02", date)
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
