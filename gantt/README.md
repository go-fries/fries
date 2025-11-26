# go-ganttmermaid

纯 Go Mermaid Gantt 渲染器：解析 Mermaid Gantt 语法，输出 PNG（无 mermaid-cli/Node 依赖），支持主题切换、周末高亮、今日基准线、自适应画布、中文字体。

## 特性
- 纯 Go 渲染：无外部命令依赖，可嵌入任意 Go 应用。
- 时间轴与日期：支持 `dateFormat`、`excludes weekends`，周末自动高亮；日期刻度贯穿全图；今日红线按当天百分比定位，超出范围夹紧到起止边界。
- 布局：任务/分区/刻度对齐，自适应宽高；任务文字不足时移到条外并用条色确保可读。
- 主题：内置默认/暗色主题，支持 `MergeTheme` 自定义；任务文字色与其他文字色分离。
- 字体：可指定 `FontPath` 或 `GGM_FONT_PATH`，内置回退避免渲染失败（中文友好）。

## 安装
```bash
cd go-ganttmermaid
go mod tidy
```

## 快速开始
```go
in := ganttmermaid.Input{
    Source: `gantt
dateFormat YYYY-MM-DD
excludes weekends
title 示例项目路线图
section 开发
任务A :a1, 2025-01-01, 3d
任务B :after a1, 2d
section 测试
用例 :t1, 2025-01-06, 2d`,
    OutputPath: "out.png",
    Theme:      ganttmermaid.DefaultTheme(),
    // FontPath: "/path/to/your/chinese.ttf",
}
_, err := ganttmermaid.Render(context.Background(), in)
if err != nil {
    panic(err)
}
```

## 示例
- 运行示例程序（输出到临时目录）：
```bash
go run ./examples/basic
```

## API 概览
- `Input`：`Source`（字符串或文件）、`OutputPath` 或 `Writer`、`Theme`、`FontPath`、`Width/Height/Scale`（可选，默认自适应）。
- `Render(ctx, Input)`：返回 `RenderResult{Bytes, OutputPath}`。
- 主题：`DefaultTheme()`、`DarkTheme()`，或 `MergeTheme(base, custom)` 自定义颜色。

## 字体（中文支持）
- 指定 `FontPath` 或设置 `GGM_FONT_PATH`。
- 未指定时尝试常见中文字体路径；失败则回退到内置基本字体（ASCII 友好，中文可能方框但不 panic）。

## 渲染规则
- 时间轴：按任务日期范围自适应，周末高亮，日期刻度全幅。
- 今日线：按当天时间百分比落在日格内，超出范围夹到起/止。
- 排除周末：`excludes weekends` 时持续天数跳过周末，结束日落在下一个工作日。
- 文本：任务内优先居中，空间不足移到条外，条外文字用任务背景色。

## 测试
```bash
GOCACHE=$(pwd)/.gocache go test ./...
```
（若默认缓存有权限限制，使用上面命令将缓存放到当前目录。）

## 版权
按项目需要补充 LICENSE。
