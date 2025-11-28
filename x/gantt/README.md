# go-ganttmermaid

纯 Go 的 Mermaid 风格甘特图渲染器 / A pure-Go Mermaid-style Gantt renderer  
解析 Mermaid Gantt 语法直接输出 PNG，无 Node/mermaid-cli 依赖，支持多主题、中文字体、时间与周末定制。

## Features / 特性
- Pure Go rendering, no external CLI; embeddable in any Go app / 纯 Go 渲染，可嵌入任意 Go 应用。
- Flexible axis: `dateFormat`(dayjs 风格)、`axisFormat|tickFormat`(strftime 风格)、`tickInterval`(ms~month)、`weekday` 周起始、`todayMarker` 自定义或关闭 / 灵活时间轴。
- Calendar control: `excludes`/`includes`、自定义 `weekend`、`timezone` 时区 / 日历与时区控制。
- Tasks: 状态 `crit|done|active|milestone|vert`，依赖 `after|before|until`，进度百分比，持续时间单位 ms/min/hour/day/week/month，支持 HH:mm 时间 / 丰富任务语法。
- Layout: 自适应画布，刻度贯穿全图，今日线按百分比定位，支持无 section 场景 / Auto-sized canvas, full-width grid, today line.
- Themes & Fonts: `DefaultTheme`/`DarkTheme`/`MergeTheme`，`FontPath` 或 `GGM_FONT_PATH` 友好支持中文字体。

## Install / 安装
```bash
go get github.com/go-fries/fries/x/gantt/v3
```

## Quick Start / 快速开始
```go
package main

import (
    "context"
    gantt "github.com/go-fries/fries/x/gantt/v3"
)

func main() {
    in := gantt.Input{
        Source: `gantt
  title Demo Roadmap
  dateFormat YYYY-MM-DD
  excludes weekends
  section Build
  Task A :a1, 2025-01-02, 3d
  Task B :crit, after a1, 4d
  section Release
  Launch  :milestone, m1, after a1, 0d`,
        OutputPath: "out.png",
        Theme:      gantt.DefaultTheme(),
        // FontPath: "/path/to/chinese.ttf", // 可选：指定中文字体
    }
    _, _ = gantt.Render(context.Background(), in)
}
```

## Syntax Reference / 语法参考
### Directives / 指令
- `title <text>` 图表标题
- `dateFormat <dayjs>`: 支持 YYYY/YY/MM/DD/HH/mm/ss/SSS 等 dayjs token
- `axisFormat|tickFormat <strftime>`: `%Y %m %d %H %M %S %L %a %A %b %B ...`
- `todayMarker [off|YYYY-MM-DD]` 关闭或固定今日线
- `tickInterval <N><unit>` 单位：millisecond|second|minute|hour|day|week|month；结合 `weekday <mon..sun>` 控制周起始
- `timezone <IANA>` 例：`Asia/Shanghai`
- `excludes <weekends|fri sat|YYYY-MM-DD ...>` 排除周末或特定日期；`includes` 重新纳入
- `weekend [fri sat ...]` 自定义周末集合；缺省周六日
- `section <name>` 可选；缺省亦可渲染任务

### Task Line / 任务行
`Name : [crit|done|active|milestone|vert], [id], [start/date/time], [duration], [after X Y|before Z|until Z], [progress%], [resources...]`
- 状态 Status：`crit`、`done`、`active`、`milestone`（0d）、`vert`（垂直线，不占行）
- 时间 Time：日期或 `HH:mm`; 可给开始+结束，或开始+持续（ms/min/hour/day/week/month）
- 依赖 Dependencies：`after a b`、`before x`、`until x`（结束前）
- 进度 Progress：`40%`
- 资源 Resources：额外 token 视为资源标签；未显式 ID 时自动生成

## Themes & Fonts / 主题与字体
- 内置：`DefaultTheme()`、`DarkTheme()`；使用 `MergeTheme(base, override)` 覆盖非空字段（hex 色值）。
- 字体：`FontPath` 或环境变量 `GGM_FONT_PATH`；若未指定，尝试常见系统中文字体，失败回退内置字体（不 panic）。

## Examples / 示例
- Basic 示例：`cd x/gantt && go run ./examples/basic`（输出到临时目录）
- Full 语法示例：查看 `x/gantt/examples/full_mermaid.gantt`，可作为 Render 源。

## Testing / 测试
```bash
cd x/gantt
GOCACHE=$(pwd)/.gocache go test ./...
```

## License / 许可证
遵循仓库根目录 LICENSE / Follow repo root LICENSE
