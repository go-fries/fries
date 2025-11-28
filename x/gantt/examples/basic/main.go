package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	ganttmermaid "github.com/go-fries/fries/x/gantt/v3"
)

func main() {
	out := filepath.Join(os.TempDir(), "gantt_basic.png")
	input := ganttmermaid.Input{
		Source: `gantt
  title 示例项目路线图
  dateFormat YYYY-MM-DD
  axisFormat %m-%d
  excludes weekends

  section 开发
  需求评审 :done, a1, 2025-01-06, 2d
  功能开发 :crit, a2, after a1, 4d
  联调验证 :after a2, 2d

  section 发布
  发布里程碑 :milestone, m1, after a2, 0d`,
		OutputPath:         out,
		Theme:              ganttmermaid.DefaultTheme(),
		Timezone:           "UTC",
		DisableTodayMarker: true, // 可选：禁用今日标记便于回归可复现
		// FontPath: "/path/to/chinese.ttf", // 如需中文字体可指定
	}

	res, err := ganttmermaid.Render(context.Background(), input)
	if err != nil {
		panic(err)
	}
	fmt.Println("渲染完成:", res.OutputPath)
}
