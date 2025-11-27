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
dateFormat YYYY-MM-DD
excludes weekends
title 示例项目路线图
section 开发
任务A :a1, 2025-01-02, 3d
任务B :after a1, 4d
section 测试
测试用例准备 :t1, 2025-01-10, 2d
执行回归 :after t1, 3d`,
	OutputPath: out,
	Theme:      ganttmermaid.DefaultTheme(),
	Timezone:   "UTC",
	DisableTodayMarker: true, // 可选：禁用今日标记便于回归可复现
	// 可选：指定中文字体
	// FontPath: "/path/to/your/chinese.ttf",
}

	res, err := ganttmermaid.Render(context.Background(), input)
	if err != nil {
		panic(err)
	}
	fmt.Println("渲染完成:", res.OutputPath)
}
