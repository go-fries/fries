package gantt

import "testing"

func TestMergeTheme(t *testing.T) {
	base := DefaultTheme()
	over := Theme{TaskFill: "#000000", TaskText: "#ffffff", Name: "custom"}
	merged := MergeTheme(base, over)
	if merged.TaskFill != "#000000" {
		t.Fatalf("task fill not merged")
	}
	if merged.TaskText != "#ffffff" {
		t.Fatalf("task text not merged")
	}
	if merged.Grid != base.Grid {
		t.Fatalf("grid should keep base")
	}
	if merged.Name != "custom" {
		t.Fatalf("name not merged")
	}
}
