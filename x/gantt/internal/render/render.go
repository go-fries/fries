package render

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"time"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/go-fries/fries/x/gantt/v3/internal/font"
	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
)

const (
	defaultScale          = 1.0
	leftMarginPx          = 220
	topMarginPx           = 60
	rowHeightPx           = 36
	barHeightPx           = 18
	axisHeightPx          = 30
	sectionGapPx          = 6
	bottomMarginPx        = 60
	minDayWidthPx         = 28
	maxDayWidthPx         = 64
	rightMarginPx         = 100
	autoMinDayWidthPx     = 40
	contentPaddingPx      = 10
	minGridWidthPx        = 1200
	halfDivisor           = 2
	thirdDivisor          = 3
	doubleMultiplier      = 2
	hoursPerDay           = 24
	secondsPerHour        = 3600
	titleFontSize         = 18
	sectionFontSize       = 14
	taskFontSize          = 13
	axisFontSize          = 12
	minTickWidthPx        = 32.0
	longRangeDayThreshold = 30
	minTickStep           = 7
	tickLabelOffsetPx     = 4
	sectionShadeStep      = 0.04
	weekendBrightenFactor = 1.08
	maxColorValue         = 255.0
	opaqueAlpha           = 0xff
	labelPaddingPx        = 6
	hexShortLength        = 6
	hexLongLength         = 8
)

const (
	secondsPerDay    = hoursPerDay * secondsPerHour
	floatHoursPerDay = float64(hoursPerDay)
)

// Options 控制绘制。
type Options struct {
	Width    int
	Height   int
	Scale    float64
	Theme    ThemeColors
	FontPath string
}

// ThemeColors 绘制时用到的颜色。
type ThemeColors struct {
	Background color.Color
	Grid       color.Color
	TaskFill   color.Color
	TaskBorder color.Color
	TaskText   color.Color
	Text       color.Color
	Emphasis   color.Color
	Milestone  color.Color
	TodayLine  color.Color
}

// RenderModel 绘制解析后的模型为 PNG 字节。
func RenderModel(_ context.Context, m parser.Model, opt Options) ([]byte, error) {
	scale := opt.Scale
	if scale <= 0 {
		scale = defaultScale
	}

	leftMargin := int(float64(leftMarginPx) * scale) // 预留文字区
	topMargin := int(float64(topMarginPx) * scale)
	rowHeight := int(float64(rowHeightPx) * scale)
	barHeight := int(float64(barHeightPx) * scale)
	axisHeight := int(float64(axisHeightPx) * scale)
	secGap := int(float64(sectionGapPx) * scale)
	bottomMargin := int(float64(bottomMarginPx) * scale)

	// 计算时间范围与时间轴宽度（包含周末，只做颜色区分）
	minStart, maxEnd := timelineBounds(m)
	totalDays := calendarSpanDays(minStart, maxEnd)
	if totalDays <= 0 {
		totalDays = 1
	}
	minDayWidth := int(float64(minDayWidthPx) * scale)
	maxDayWidth := int(float64(maxDayWidthPx) * scale)
	rightMargin := int(float64(rightMarginPx) * scale)
	if opt.Width == 0 {
		// 自适应时增加最小日宽，确保日期刻度有足够空间
		autoMin := int(float64(autoMinDayWidthPx) * scale)
		if autoMin > minDayWidth {
			minDayWidth = autoMin
		}
	}

	// dayWidth 计算：若外部未指定宽度，则以最小宽度铺满内容；否则尝试适配可用宽度。
	var dayWidth int
	var gridWidth int
	width := opt.Width
	if width > 0 {
		available := width - leftMargin - rightMargin
		if available <= 0 {
			available = minDayWidth * totalDays
		}
		dayWidth = int(math.Floor(float64(available) / float64(totalDays)))
		dayWidth = clampInt(dayWidth, minDayWidth, maxDayWidth)
		gridWidth = dayWidth * totalDays
		target := width - leftMargin - rightMargin
		if target > gridWidth {
			dayWidth = clampInt(int(float64(target)/float64(totalDays)), minDayWidth, maxDayWidth)
			gridWidth = dayWidth * totalDays
			width = leftMargin + gridWidth + rightMargin
		} else {
			width = leftMargin + gridWidth + rightMargin
		}
	} else {
		dayWidth = minDayWidth
		gridWidth = dayWidth * totalDays
		minGridWidth := minGridWidthPx
		if gridWidth < minGridWidth {
			dayWidth = clampInt(int(math.Ceil(float64(minGridWidth)/float64(totalDays))), minDayWidth, maxDayWidth)
			gridWidth = dayWidth * totalDays
		}
		width = leftMargin + gridWidth + rightMargin
	}

	// 自适应高度：未指定时按内容计算
	height := opt.Height
	if height <= 0 {
		contentHeight := topMargin + axisHeight + int(float64(contentPaddingPx)*scale)
		for _, sec := range m.Sections {
			contentHeight += rowHeight / halfDivisor
			contentHeight += len(sec.Tasks) * rowHeight
			contentHeight += secGap
		}
		contentHeight += bottomMargin
		height = contentHeight
	}

	w := int(math.Round(float64(width) * scale))
	h := int(math.Round(float64(height) * scale))

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{opt.Theme.Background}, image.Point{}, draw.Src)

	// 标题
	if m.Title != "" {
		drawBoldText(img, opt.Theme.Emphasis, w/halfDivisor, topMargin/halfDivisor, m.Title, opt.FontPath, int(float64(titleFontSize)*scale))
	}

	// 预计算 section 布局
	startY := topMargin + axisHeight + int(float64(contentPaddingPx)*scale)
	type secInfo struct {
		section parser.Section
		start   int
		end     int
	}
	infos := make([]secInfo, 0, len(m.Sections))
	y := startY
	for _, sec := range m.Sections {
		secStart := y
		y += rowHeight / halfDivisor
		y += len(sec.Tasks) * rowHeight
		secEnd := y
		y += secGap
		infos = append(infos, secInfo{section: sec, start: secStart, end: secEnd})
	}
	contentBottom := y

	// 画 section 背景
	for idx, info := range infos {
		secColor := sectionBgColor(opt.Theme.Background, idx)
		fillRect(img, image.Rect(0, info.start, w, info.end+secGap), secColor)
	}

	// 当前日期位置（按当天百分比放置，超出范围则夹紧到起/止边界）
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayX int
	switch {
	case now.Before(minStart):
		todayX = leftMargin
	case now.After(maxEnd.Add(time.Duration(hoursPerDay) * time.Hour)):
		todayX = leftMargin + gridWidth
	default:
		offset := calendarOffset(minStart, dayStart)
		within := now.Sub(dayStart).Seconds() / float64(secondsPerDay)
		if within < 0 {
			within = 0
		}
		if within > 1 {
			within = 1
		}
		todayX = leftMargin + offset*dayWidth + int(float64(dayWidth)*within)
		if todayX > leftMargin+gridWidth {
			todayX = leftMargin + gridWidth
		}
		if todayX < leftMargin {
			todayX = leftMargin
		}
	}
	hasToday := true

	// 时间轴与纵向网格（贯穿全图，周末着色、今日红线）
	drawTimeline(img, leftMargin, topMargin, totalDays, axisHeight, dayWidth, minStart, opt.Theme, m.ExcludeWeekends, contentBottom, hasToday, todayX)

	// 绘制 section 标题与任务
	y = startY
	for _, sec := range m.Sections {
		drawBoldText(img, opt.Theme.Emphasis, leftMargin/halfDivisor, y+rowHeight/halfDivisor, sec.Name, opt.FontPath, int(float64(sectionFontSize)*scale))
		y += rowHeight / halfDivisor
		for _, task := range sec.Tasks {
			x := leftMargin
			if !task.Start.IsZero() {
				offset := calendarOffset(minStart, task.Start)
				if offset < 0 {
					offset = 0
				}
				x = leftMargin + offset*dayWidth
			}
			duration := task.DurationDays
			if duration <= 0 {
				duration = 1
			}
			durationSpan := durationWithExcludes(task.Start, duration, m.ExcludeWeekends)
			widthPx := durationSpan * dayWidth
			if widthPx < dayWidth {
				widthPx = dayWidth
			}

			barTop := y + (rowHeight-barHeight)/halfDivisor
			rect := image.Rect(x, barTop, x+widthPx, barTop+barHeight)
			draw.Draw(img, rect, &image.Uniform{opt.Theme.TaskFill}, image.Point{}, draw.Src)
			drawBorder(img, rect, opt.Theme.TaskBorder)

			// 文本：优先放条内，空间不足则放在条右侧
			padding := int(float64(labelPaddingPx) * scale)
			labelMeasured := measureTextWidth(task.Name, opt.Theme.TaskText, opt.FontPath, int(float64(taskFontSize)*scale))
			innerRoom := rect.Dx() - padding*doubleMultiplier
			labelX := rect.Min.X + rect.Dx()/halfDivisor
			labelY := rect.Min.Y + rect.Dy()/halfDivisor
			labelColor := opt.Theme.TaskText
			if labelMeasured > innerRoom {
				labelX = rect.Max.X + padding + labelMeasured/halfDivisor
				labelColor = opt.Theme.TaskFill // 写在外侧时用任务背景色，避免与背景重叠难读
			}
			var label string
			if labelMeasured > innerRoom {
				label = task.Name
			} else {
				label = fitText(img, task.Name, innerRoom, opt.Theme.TaskText, opt.FontPath, int(float64(taskFontSize)*scale))
			}
			drawText(img, labelColor, labelX, labelY, label, opt.FontPath, int(float64(taskFontSize)*scale))
			y += rowHeight
		}
		y += secGap // section 间隔
	}

	buf := bytes.NewBuffer(nil)
	if err := pngEncode(buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawBorder(img *image.RGBA, rect image.Rectangle, c color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, c)
		img.Set(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, c)
		img.Set(rect.Max.X-1, y, c)
	}
}

func pngEncode(buf *bytes.Buffer, img image.Image) error {
	// 避免依赖外部库，使用标准库 PNG 编码
	return (&pngEncoder{}).Encode(buf, img)
}

// pngEncoder 使用标准库编码 PNG（重新封装便于测试或未来替换）。
type pngEncoder struct{}

func (pngEncoder) Encode(w *bytes.Buffer, img image.Image) error {
	return pngEncodeStd(w, img)
}

// go embed 不可用标准库？直接使用 image/png
func pngEncodeStd(w *bytes.Buffer, img image.Image) error {
	return png.Encode(w, img)
}

func timelineBounds(m parser.Model) (min time.Time, max time.Time) {
	first := true
	for _, sec := range m.Sections {
		for _, task := range sec.Tasks {
			if task.Start.IsZero() {
				continue
			}
			end := task.Start.Add(time.Duration(task.DurationDays) * time.Duration(hoursPerDay) * time.Hour)
			if first {
				min, max = task.Start, end
				first = false
				continue
			}
			if task.Start.Before(min) {
				min = task.Start
			}
			if end.After(max) {
				max = end
			}
		}
	}
	if first {
		// 没有日期则使用今天
		min = time.Now()
		max = min.Add(time.Duration(hoursPerDay) * time.Hour)
	}
	return
}

func calendarSpanDays(start, end time.Time) int {
	if start.After(end) {
		start, end = end, start
	}
	return int(end.Sub(start).Hours()/floatHoursPerDay) + 1
}

func calendarOffset(start, target time.Time) int {
	return int(target.Sub(start).Hours() / floatHoursPerDay)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// durationWithExcludes 在排除周末时为每个周末增加额外天数，使任务长度保持工作日数量。
func durationWithExcludes(start time.Time, duration int, excludeWeekends bool) int {
	if duration <= 0 {
		return 1
	}
	if !excludeWeekends || start.IsZero() {
		return duration
	}
	workdays := 0
	days := 0
	current := start
	for workdays < duration {
		if !isWeekend(current) {
			workdays++
		}
		days++
		current = current.Add(time.Duration(hoursPerDay) * time.Hour)
	}
	return days
}

func isWeekend(t time.Time) bool {
	wd := t.Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

func drawTimeline(img *image.RGBA, xStart, yStart, days, axisHeight, dayWidth int, minStart time.Time, theme ThemeColors, excludeWeekends bool, endY int, hasToday bool, todayX int) {
	width := dayWidth * days
	weekendFill := weekendColor(theme.Background)

	// 周末着色 + 垂直网格（贯穿内容区域）
	for i := 0; i < days; i++ {
		x := xStart + i*dayWidth
		dayDate := minStart.Add(time.Duration(i) * time.Duration(hoursPerDay) * time.Hour)
		if excludeWeekends && isWeekend(dayDate) {
			fillRect(img, image.Rect(x, yStart, x+dayWidth, endY), weekendFill)
		}
		for yy := yStart; yy < endY; yy++ {
			img.Set(x, yy, theme.Grid)
		}
	}
	// 右边界线
	for yy := yStart; yy < endY; yy++ {
		img.Set(xStart+width, yy, theme.Grid)
	}

	// 水平基准线
	y := yStart + axisHeight/halfDivisor
	for x := xStart; x < xStart+width; x++ {
		img.Set(x, y, theme.Grid)
	}

	// 刻度文本
	tickEvery := 1
	if dayWidth < int(minTickWidthPx) {
		tickEvery = int(math.Ceil(minTickWidthPx / float64(dayWidth)))
	}
	if days > longRangeDayThreshold {
		tickEvery = int(math.Max(float64(tickEvery), float64(minTickStep)))
	}
	face, _, _ := font.LoadFaceWithFallback(float64(axisFontSize), "")
	for i := 0; i <= days; i += tickEvery {
		x := xStart + i*dayWidth
		date := minStart.Add(time.Duration(i) * time.Duration(hoursPerDay) * time.Hour).Format("01-02")
		drawTextWithFace(img, theme.Text, x+tickLabelOffsetPx, yStart+axisHeight-tickLabelOffsetPx, face, date)
	}

	// 今日竖线
	if hasToday {
		for yy := yStart; yy < endY; yy++ {
			img.Set(todayX, yy, theme.TodayLine)
		}
	}
}

// drawText 简单绘制文字，使用 freetype 字体（如果提供）。
func drawText(img *image.RGBA, c color.Color, centerX, centerY int, text, fontPath string, size int) {
	col := c
	face, _, err := font.LoadFaceWithFallback(float64(size), fontPath)
	if err != nil && face == nil {
		face = font.DefaultFace()
	}

	// 计算文本宽度以居中
	d := &xfont.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
	}
	textWidth := d.MeasureString(text).Floor()
	x := centerX - textWidth/halfDivisor
	y := centerY + size/thirdDivisor
	d.Dot = fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}
	d.DrawString(text)
}

// drawBoldText 简单“加粗”实现：多次平移叠加绘制。
func drawBoldText(img *image.RGBA, c color.Color, centerX, centerY int, text, fontPath string, size int) {
	// 为避免虚影，直接使用稍大的字号单次绘制模拟加粗。
	drawText(img, c, centerX, centerY, text, fontPath, size+1)
}

func drawTextWithFace(img *image.RGBA, c color.Color, x, y int, face xfont.Face, text string) {
	if face == nil {
		face = font.DefaultFace()
	}
	d := &xfont.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y),
		},
	}
	d.DrawString(text)
}

// fitText 根据宽度截断字符串，超出时添加省略号。
func fitText(img *image.RGBA, text string, maxWidth int, c color.Color, fontPath string, size int) string {
	if maxWidth <= 0 {
		return text
	}
	face, _, _ := font.LoadFaceWithFallback(float64(size), fontPath)
	if face == nil {
		face = font.DefaultFace()
	}
	d := &xfont.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
	}
	if d.MeasureString(text).Round() <= maxWidth {
		return text
	}
	runes := []rune(text)
	ellipsis := "…"
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + ellipsis
		if d.MeasureString(candidate).Round() <= maxWidth {
			return candidate
		}
	}
	return text
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
}

func sectionBgColor(base color.Color, idx int) color.Color {
	rgba := color.RGBAModel.Convert(base).(color.RGBA)
	factor := sectionShadeStep * float64((idx%halfDivisor)+1) // 4% 或 8% 明度调整
	r := clampFloat(float64(rgba.R) * (1 - factor))
	g := clampFloat(float64(rgba.G) * (1 - factor))
	b := clampFloat(float64(rgba.B) * (1 - factor))
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: rgba.A}
}

func weekendColor(base color.Color) color.Color {
	rgba := color.RGBAModel.Convert(base).(color.RGBA)
	// 加亮 10% 作为周末高亮
	r := clampFloat(float64(rgba.R) * weekendBrightenFactor)
	g := clampFloat(float64(rgba.G) * weekendBrightenFactor)
	b := clampFloat(float64(rgba.B) * weekendBrightenFactor)
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: rgba.A}
}

func clampFloat(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > maxColorValue {
		return maxColorValue
	}
	return v
}

func measureTextWidth(text string, c color.Color, fontPath string, size int) int {
	face, _, _ := font.LoadFaceWithFallback(float64(size), fontPath)
	if face == nil {
		face = font.DefaultFace()
	}
	d := &xfont.Drawer{
		Src:  image.NewUniform(c),
		Face: face,
	}
	return d.MeasureString(text).Round()
}

// ThemeFromHex 构造主题颜色。
func ThemeFromHex(bg, grid, taskFill, taskBorder, taskText, text, milestone, todayLine string) ThemeColors {
	return ThemeColors{
		Background: mustColor(bg, color.White),
		Grid:       mustColor(grid, color.RGBA{0xe0, 0xe0, 0xe0, 0xff}),
		TaskFill:   mustColor(taskFill, color.RGBA{0x81, 0x90, 0xdd, 0xff}),
		TaskBorder: mustColor(taskBorder, color.RGBA{0x2c, 0x3e, 0x50, 0xff}),
		TaskText:   mustColor(taskText, color.White),
		Text:       mustColor(text, color.RGBA{0x33, 0x33, 0x33, 0xff}),
		Emphasis:   mustColor(text, color.RGBA{0x11, 0x11, 0x11, 0xff}),
		Milestone:  mustColor(milestone, color.RGBA{0xe6, 0x7e, 0x22, 0xff}),
		TodayLine:  mustColor(todayLine, color.RGBA{0xd0, 0x02, 0x1b, 0xff}),
	}
}

func mustColor(hex string, fallback color.Color) color.Color {
	c, err := parseHexColor(hex)
	if err != nil {
		return fallback
	}
	return c
}

func parseHexColor(s string) (color.RGBA, error) {
	c := color.RGBA{A: opaqueAlpha}
	if s == "" {
		return c, fmt.Errorf("empty color")
	}
	if s[0] == '#' {
		s = s[1:]
	}
	var err error
	switch len(s) {
	case hexShortLength:
		_, err = fmt.Sscanf(s, "%02x%02x%02x", &c.R, &c.G, &c.B)
	case hexLongLength:
		_, err = fmt.Sscanf(s, "%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A)
	default:
		return c, fmt.Errorf("invalid color: %s", s)
	}
	return c, err
}
