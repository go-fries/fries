package gantt

import (
	"context"
	"io"
)

// Input 描述渲染所需的输入参数。
type Input struct {
	Source             string    // Mermaid Gantt 源（文本或文件路径）
	FromFile           bool      // 是否将 Source 视为文件路径
	Theme              Theme     // 主题配置，未设置则使用默认
	OutputPath         string    // 输出 PNG 路径（可选，与 Writer 至少一个）
	Writer             io.Writer // 输出目标 Writer（可选）
	Width              int       // 图像宽度，0 表示使用默认
	Height             int       // 图像高度，0 表示使用默认
	Scale              float64   // 缩放倍数，0 表示默认 1.0
	FontPath           string    // 自定义字体路径（支持中文），为空则使用内置/系统字体
	Timezone           string    // 时间计算使用的时区，空则使用 UTC
	Today              string    // 覆盖今日标记日期（YYYY-MM-DD），空则使用当前日期
	DisableTodayMarker bool      // 是否禁用今日标记
}

// RenderResult 返回渲染结果。
type RenderResult struct {
	OutputPath string
	Bytes      []byte
	Warnings   []string
}

// Renderer 定义渲染器接口，便于后续替换实现。
type Renderer interface {
	Render(ctx context.Context, in Input) (RenderResult, error)
}

// Render 使用默认渲染器执行渲染。
func Render(ctx context.Context, in Input) (RenderResult, error) {
	return defaultRenderer.Render(ctx, in)
}
