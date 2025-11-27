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
	"strings"
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
	Calendar parser.Calendar
	Today    parser.TodayMarker
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
	Vertical   color.Color
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

	calendar := m.Calendar
	if opt.Calendar.Timezone != "" {
		calendar.Timezone = opt.Calendar.Timezone
	}
	if opt.Calendar.ExcludeWeekend {
		calendar.ExcludeWeekend = true
	}
	if len(opt.Calendar.ExcludeDates) > 0 {
		calendar.ExcludeDates = opt.Calendar.ExcludeDates
	}
	if len(opt.Calendar.IncludeDates) > 0 {
		calendar.IncludeDates = opt.Calendar.IncludeDates
	}

	today := m.Today
	if opt.Today.HasDate || !opt.Today.Enabled {
		today = opt.Today
	}
	if !today.HasDate {
		loc := time.UTC
		if calendar.Timezone != "" {
			if tz, err := time.LoadLocation(calendar.Timezone); err == nil {
				loc = tz
			}
		}
		now := time.Now().In(loc)
		today.Date = now
		today.HasDate = true
	}

	timeMode := hasTimeGranularity(m)

	// 是否存在命名的 section
	hasSectionHeader := false
	for _, sec := range m.Sections {
		if strings.TrimSpace(sec.Name) != "" {
			hasSectionHeader = true
			break
		}
	}
	if !hasSectionHeader {
		leftMargin = int(20 * scale) // 无 section 时尽量贴近轴
		if leftMargin < 12 {
			leftMargin = 12
		}
	}

	// 计算时间范围与时间轴宽度
	minStart, maxEnd := timelineBounds(m)
	minStart, maxEnd = normalizeSpan(minStart, maxEnd)
	// 为右侧添加平滑缓冲，避免视觉截断
	span := maxEnd.Sub(minStart)
	if span > 0 {
		var pad time.Duration
		if timeMode {
			pad = span / 20 // 5%
			if pad < 30*time.Minute {
				pad = 30 * time.Minute
			}
		} else {
			pad = span / 20
			if pad < 24*time.Hour {
				pad = 24 * time.Hour
			}
		}
		maxEnd = maxEnd.Add(pad)
	}

	// 若包含小时/分钟但跨度超过 48 小时，则强制退回日级刻度以避免分钟轴过密
	if timeMode {
		spanHours := maxEnd.Sub(minStart).Hours()
		if spanHours > 48 {
			timeMode = false
		}
	}
	minSpan := minStart
	maxSpan := maxEnd
	rightMargin := int(float64(rightMarginPx) * scale)
	var gridWidth int
	var dayWidth int
	width := opt.Width

	tickMinutes := tickToMinutes(m.Tick)
	tickDays := tickToDays(m.Tick)
	if !m.Tick.Valid {
		autoMin, autoDay := autoTickInterval(minSpan, maxSpan, timeMode)
		if tickMinutes == 0 {
			tickMinutes = autoMin
		}
		if tickDays == 0 {
			tickDays = autoDay
		}
	}

	if timeMode {
		totalMinutes := int(maxSpan.Sub(minSpan).Minutes()) + 1
		if totalMinutes <= 0 {
			totalMinutes = 1
		}
		if width > 0 {
			gridWidth = width - leftMargin - rightMargin
			if gridWidth < minGridWidthPx {
				gridWidth = minGridWidthPx
			}
			width = leftMargin + gridWidth + rightMargin
		} else {
			gridWidth = minGridWidthPx
			width = leftMargin + gridWidth + rightMargin
		}
		dayWidth = gridWidth // used as total span for minutes mode
	} else {
		totalDays := calendarSpanDays(minStart, maxEnd)
		if totalDays <= 0 {
			totalDays = 1
		}
		minDayWidth := int(float64(minDayWidthPx) * scale)
		maxDayWidth := int(float64(maxDayWidthPx) * scale)
		if opt.Width == 0 {
			autoMin := int(float64(autoMinDayWidthPx) * scale)
			if autoMin > minDayWidth {
				minDayWidth = autoMin
			}
		}

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
			if gridWidth < minGridWidthPx {
				dayWidth = clampInt(int(math.Ceil(float64(minGridWidthPx)/float64(totalDays))), minDayWidth, maxDayWidth)
				gridWidth = dayWidth * totalDays
			}
			width = leftMargin + gridWidth + rightMargin
		}
	}

	// 自适应高度：未指定时按内容计算
	height := opt.Height
	if height <= 0 {
		if !hasSectionHeader {
			topMargin = int(float64(axisHeightPx) * scale)
		}
		contentHeight := topMargin + axisHeight
		if hasSectionHeader {
			contentHeight += int(float64(contentPaddingPx) * scale)
		}
		for _, sec := range m.Sections {
			if hasSectionHeader {
				contentHeight += rowHeight / halfDivisor
			}
			contentHeight += len(sec.Tasks) * rowHeight
			if hasSectionHeader {
				contentHeight += secGap
			}
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
	startY := topMargin + axisHeight
	if hasSectionHeader {
		startY += int(float64(contentPaddingPx) * scale)
	}
	type secInfo struct {
		section parser.Section
		start   int
		end     int
	}
	infos := make([]secInfo, 0, len(m.Sections))
	y := startY
	for _, sec := range m.Sections {
		secStart := y
		if hasSectionHeader {
			y += rowHeight / halfDivisor
		}
		y += len(sec.Tasks) * rowHeight
		secEnd := y
		if hasSectionHeader {
			y += secGap
		}
		infos = append(infos, secInfo{section: sec, start: secStart, end: secEnd})
	}
	contentBottom := y

	// 画 section 背景
	for idx, info := range infos {
		if !hasSectionHeader {
			break
		}
		secColor := sectionBgColor(opt.Theme.Background, idx)
		fillRect(img, image.Rect(0, info.start, w, info.end+secGap), secColor)
	}

	// 当前日期位置（按当天百分比放置，超出范围则夹紧到起/止边界）
	todayTime := today.Date
	loc := time.Now().Location()
	if calendar.Timezone != "" {
		if tz, err := time.LoadLocation(calendar.Timezone); err == nil {
			loc = tz
		}
	} else if today.HasDate {
		loc = today.Date.Location()
	}
	now := time.Now().In(loc)
	if todayTime.IsZero() {
		todayTime = now
	}
	// 仅标记到当天开始
	todayTime = time.Date(todayTime.Year(), todayTime.Month(), todayTime.Day(), 0, 0, 0, 0, loc)
	var todayX int
	if timeMode {
		todayX = leftMargin
	} else {
		spanStart := minStart.In(loc)
		spanEnd := maxEnd.In(loc)
		if spanEnd.Before(spanStart) {
			todayX = leftMargin
		} else {
			nowDayStart := todayTime
			if nowDayStart.Before(spanStart) {
				nowDayStart = spanStart
			}
			if nowDayStart.After(spanEnd) {
				nowDayStart = spanEnd
			}
			spanTotal := spanEnd.Sub(spanStart)
			ratio := nowDayStart.Sub(spanStart).Seconds() / spanTotal.Seconds()
			todayX = leftMargin + int(float64(gridWidth)*ratio)
		}
	}
	hasToday := today.Enabled && !timeMode

	// 时间轴与纵向网格（贯穿全图，周末着色、今日红线）
	weekendFill := weekendColor(opt.Theme.Background)
	if !hasSectionHeader {
		// 无 section 背景时，排除日统一用稍暗底色，含 weekend 与 excludes
		weekendFill = darkenColor(opt.Theme.Background, 0.9)
	}

	if timeMode {
		drawTimelineMinutes(img, leftMargin, topMargin, gridWidth, axisHeight, minSpan, maxSpan, m.AxisFormat, opt.Theme, calendar, contentBottom, tickMinutes, weekendFill)
	} else {
		totalDays := calendarSpanDays(minStart, maxEnd)
		if totalDays <= 0 {
			totalDays = 1
		}
		weekStart := m.WeekStart
		if m.Tick.Valid && m.Tick.Unit == "week" {
			if weekStart == nil {
				ws := time.Sunday
				weekStart = &ws
			}
		} else {
			weekStart = nil
		}
		drawTimeline(img, leftMargin, topMargin, totalDays, axisHeight, dayWidth, minStart, m.AxisFormat, opt.Theme, calendar, contentBottom, hasToday, todayX, tickDays, weekStart, weekendFill)
	}

	// 垂直标记（不占用行）
	if timeMode {
		drawVerticalMarkers(img, leftMargin, topMargin, contentBottom, minSpan, maxSpan, gridWidth, dayWidth, timeMode, opt.Theme, m.Verticals)
	} else {
		drawVerticalMarkers(img, leftMargin, topMargin, contentBottom, minStart, maxEnd, gridWidth, dayWidth, timeMode, opt.Theme, m.Verticals)
	}

	// 绘制 section 标题与任务
	y = startY
	for _, sec := range m.Sections {
		drawBoldText(img, opt.Theme.Emphasis, leftMargin/halfDivisor, y+rowHeight/halfDivisor, sec.Name, opt.FontPath, int(float64(sectionFontSize)*scale))
		y += rowHeight / halfDivisor
		for _, task := range sec.Tasks {
			x := leftMargin
			widthPx := dayWidth
			if timeMode {
				totalMinutes := maxSpan.Sub(minSpan).Minutes()
				if totalMinutes <= 0 {
					totalMinutes = 1
				}
				offsetMinutes := task.Start.Sub(minSpan).Minutes()
				if offsetMinutes < 0 {
					offsetMinutes = 0
				}
				x = leftMargin + int(float64(gridWidth)*(offsetMinutes/totalMinutes))
				durationMinutes := task.End.Sub(task.Start).Minutes()
				if durationMinutes <= 0 {
					durationMinutes = 1
				}
				widthPx = int(float64(gridWidth) * (durationMinutes / totalMinutes))
				if widthPx < 4 {
					widthPx = 4
				}
			} else {
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
				widthPx = duration * dayWidth
				if widthPx < dayWidth {
					widthPx = dayWidth
				}
			}

			barTop := y + (rowHeight-barHeight)/halfDivisor
			if task.IsMilestone || task.Duration.Value == 0 {
				markerWidth := widthPx
				if markerWidth < barHeight {
					markerWidth = barHeight
				}
				drawMilestone(img, opt.Theme.Milestone, x, barTop, markerWidth, barHeight)
				drawText(img, opt.Theme.Text, x+markerWidth/halfDivisor, barTop-barHeight/halfDivisor, task.Name, opt.FontPath, int(float64(taskFontSize)*scale))
				y += rowHeight
				continue
			}

			fill, border := statusColors(opt.Theme, task.Status)
			rect := image.Rect(x, barTop, x+widthPx, barTop+barHeight)
			draw.Draw(img, rect, &image.Uniform{fill}, image.Point{}, draw.Src)
			drawBorder(img, rect, border)

			if task.Progress > 0 {
				progressWidth := int(float64(rect.Dx()) * float64(task.Progress) / 100.0)
				if progressWidth > 0 {
					progRect := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+progressWidth, rect.Max.Y)
					draw.Draw(img, progRect, &image.Uniform{opt.Theme.Milestone}, image.Point{}, draw.Over)
				}
			}

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
			if task.IsVertical {
				continue
			}
			if task.Start.IsZero() {
				continue
			}
			end := task.End
			if end.IsZero() {
				end = task.Start.Add(time.Duration(task.DurationDays) * time.Duration(hoursPerDay) * time.Hour)
			}
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
	for _, v := range m.Verticals {
		if v.Start.IsZero() {
			continue
		}
		if first {
			min, max = v.Start, v.Start
			first = false
			continue
		}
		if v.Start.Before(min) {
			min = v.Start
		}
		if v.Start.After(max) {
			max = v.Start
		}
	}
	if first {
		// 没有日期则使用今天
		min = time.Now()
		max = min.Add(time.Duration(hoursPerDay) * time.Hour)
	}
	return
}

func normalizeSpan(min, max time.Time) (time.Time, time.Time) {
	if min.IsZero() && max.IsZero() {
		return min, max
	}
	if max.Before(min) {
		min, max = max, min
	}
	if max.Equal(min) {
		// 保证至少有一个单位跨度，避免分母为 0
		max = min.Add(time.Minute)
	}
	return min, max
}

func autoTickInterval(min, max time.Time, timeMode bool) (tickMinutes int, tickDays int) {
	span := max.Sub(min)
	if span <= 0 {
		span = time.Minute
	}
	hours := span.Hours()
	days := span.Hours() / floatHoursPerDay

	if timeMode {
		switch {
		case hours <= 2:
			return 5, 0 // 分钟刻度
		case hours <= 48:
			return 60, 0 // 小时刻度
		case days <= 31:
			return 60, 0 // 小时刻度
		case days <= 365:
			return 7 * 24 * 60, 0 // 周刻度
		default:
			return 30 * 24 * 60, 0 // 月刻度（按 30 天）
		}
	}

	switch {
	case days <= 30:
		return 0, 1 // 日
	case days <= 365:
		return 0, 7 // 周
	default:
		return 0, 30 // 月
	}
}

// AutoTickIntervalForTest 暴露给测试验证自动刻度选择。
func AutoTickIntervalForTest(min, max time.Time, timeMode bool) (int, int) {
	return autoTickInterval(min, max, timeMode)
}

func normalizeMinuteTicks(totalMinutes int, tickMinutes int, forced bool) (int, int) {
	if tickMinutes <= 0 {
		tickMinutes = 1
	}
	if totalMinutes <= 0 {
		totalMinutes = 1
	}

	if !forced {
		maxTicks := 60
		ticks := totalMinutes / tickMinutes
		if ticks > maxTicks {
			target := int(math.Ceil(float64(totalMinutes) / float64(maxTicks)))
			tickMinutes = snapNiceMinutes(target)
		}
	}

	labelStep := tickMinutes
	maxLabels := 30
	labels := totalMinutes / labelStep
	if labels > maxLabels {
		multiplier := int(math.Ceil(float64(labels) / float64(maxLabels)))
		labelStep = tickMinutes * multiplier
		labelStep = snapNiceMinutes(labelStep)
	}

	return tickMinutes, labelStep
}

func snapNiceMinutes(v int) int {
	nice := []int{1, 5, 10, 15, 30, 60, 120, 180, 240, 360, 720, 1440, 10080, 20160, 43200} // up to month
	for _, n := range nice {
		if v <= n {
			return n
		}
	}
	return v
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

func alignTickOffset(minStart time.Time, stepMinutes int) int {
	if stepMinutes <= 0 {
		return 0
	}
	step := time.Duration(stepMinutes) * time.Minute
	aligned := minStart.Truncate(step)
	if aligned.Before(minStart) {
		aligned = aligned.Add(step)
	} else if aligned.Equal(minStart) && (minStart.Second() > 0 || minStart.Nanosecond() > 0) {
		aligned = aligned.Add(step)
	}
	offset := aligned.Sub(minStart)
	if offset < 0 {
		return 0
	}
	return int(offset.Minutes())
}

func hasTimeGranularity(m parser.Model) bool {
	for _, sec := range m.Sections {
		for _, t := range sec.Tasks {
			if t.HasTime || t.Duration.Unit == parser.DurationMinute || t.Duration.Unit == parser.DurationHour {
				return true
			}
		}
	}
	for _, v := range m.Verticals {
		if v.HasTime {
			return true
		}
	}
	return false
}

func statusColors(theme ThemeColors, status parser.TaskStatus) (color.Color, color.Color) {
	switch status {
	case parser.StatusCritical:
		return theme.Milestone, theme.TodayLine
	case parser.StatusDone:
		return weekendColor(theme.TaskFill), theme.TaskBorder
	case parser.StatusActive:
		return theme.TaskFill, theme.TodayLine
	case parser.StatusMilestone:
		return theme.Milestone, theme.Milestone
	default:
		return theme.TaskFill, theme.TaskBorder
	}
}

func drawMilestone(img *image.RGBA, c color.Color, x, y, dayWidth, barHeight int) {
	size := barHeight
	if size < 6 {
		size = 6
	}
	centerX := x + dayWidth/halfDivisor
	centerY := y + barHeight/halfDivisor
	half := size / halfDivisor
	for dy := -half; dy <= half; dy++ {
		span := half - abs(dy)
		for dx := -span; dx <= span; dx++ {
			img.Set(centerX+dx, centerY+dy, c)
		}
	}
}

func isExcludedDay(t time.Time, cal parser.Calendar) bool {
	if cal.ExcludeWeekend && isWeekendDay(t, cal) {
		for _, d := range cal.IncludeDates {
			if sameDay(t, d) {
				return false
			}
		}
		return true
	}
	for _, d := range cal.ExcludeDates {
		if sameDay(t, d) {
			return true
		}
	}
	return false
}

func isWeekendDay(t time.Time, cal parser.Calendar) bool {
	weekends := cal.WeekendDays
	if len(weekends) == 0 {
		weekends = []time.Weekday{time.Saturday, time.Sunday}
	}
	for _, wd := range weekends {
		if t.Weekday() == wd {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func drawTimelineMinutes(img *image.RGBA, xStart, yStart, width, axisHeight int, minStart, maxEnd time.Time, axisFormat string, theme ThemeColors, calendar parser.Calendar, endY int, forcedTickMinutes int, weekendFill color.Color) {
	totalMinutes := int(maxEnd.Sub(minStart).Minutes()) + 1
	if totalMinutes <= 0 {
		totalMinutes = 1
	}
	pixelsPerMinute := float64(width) / float64(totalMinutes)

	// 背景网格（粗略每小时），并在自动模式下控制刻度密度
	tickMinutes := forcedTickMinutes
	isForced := forcedTickMinutes > 0
	if tickMinutes <= 0 {
		tickMinutes = 60
	}
	tickMinutes, labelStep := normalizeMinuteTicks(totalMinutes, tickMinutes, isForced)
	tickOffset := alignTickOffset(minStart, tickMinutes)
	labelOffset := alignTickOffset(minStart, labelStep)

	// 起点线
	for yy := yStart; yy < endY; yy++ {
		img.Set(xStart, yy, theme.Grid)
	}

	for i := tickOffset; i <= totalMinutes; i += tickMinutes {
		x := xStart + int(float64(i)*pixelsPerMinute)
		current := minStart.Add(time.Duration(i) * time.Minute)
		if isExcludedDay(current, calendar) {
			fillRect(img, image.Rect(x, yStart, x+int(float64(tickMinutes)*pixelsPerMinute), endY), weekendFill)
		}
		for yy := yStart; yy < endY; yy++ {
			img.Set(x, yy, theme.Grid)
		}
	}

	y := yStart + axisHeight/halfDivisor
	for x := xStart; x < xStart+width; x++ {
		img.Set(x, y, theme.Grid)
	}
	for yy := yStart; yy < endY; yy++ {
		img.Set(xStart+width, yy, theme.Grid)
	}

	format := axisFormat
	if strings.TrimSpace(format) == "" {
		format = "15:04"
	}
	face, _, _ := font.LoadFaceWithFallback(float64(axisFontSize), "")
	step := labelStep
	for i := labelOffset; i <= totalMinutes; i += step {
		x := xStart + int(float64(i)*pixelsPerMinute)
		date := minStart.Add(time.Duration(i) * time.Minute).Format(format)
		drawTextWithFace(img, theme.Text, x+tickLabelOffsetPx, yStart+axisHeight-tickLabelOffsetPx, face, date)
	}
}

func drawVerticalMarkers(img *image.RGBA, xStart, yStart, endY int, spanStart, spanEnd time.Time, gridWidth, dayWidth int, timeMode bool, theme ThemeColors, verts []parser.Task) {
	if len(verts) == 0 {
		return
	}
	spanMinutes := spanEnd.Sub(spanStart).Minutes()
	for _, v := range verts {
		if v.Start.IsZero() {
			continue
		}
		var x int
		if timeMode {
			if spanMinutes <= 0 {
				x = xStart
			} else {
				offset := v.Start.Sub(spanStart).Minutes()
				if offset < 0 {
					offset = 0
				}
				x = xStart + int(float64(gridWidth)*(offset/spanMinutes))
			}
		} else {
			offset := calendarOffset(spanStart, v.Start)
			if offset < 0 {
				offset = 0
			}
			x = xStart + offset*dayWidth
		}
		for yy := yStart; yy < endY; yy++ {
			img.Set(x, yy, theme.Vertical)
		}
	}
}

func drawTimeline(img *image.RGBA, xStart, yStart, days, axisHeight, dayWidth int, minStart time.Time, axisFormat string, theme ThemeColors, calendar parser.Calendar, endY int, hasToday bool, todayX int, forcedTickDays int, weekStart *time.Weekday, weekendFill color.Color) {
	width := dayWidth * days

	tickEvery := 1
	if forcedTickDays > 0 {
		tickEvery = forcedTickDays
	} else {
		if dayWidth < int(minTickWidthPx) {
			tickEvery = int(math.Ceil(minTickWidthPx / float64(dayWidth)))
		}
		if days > longRangeDayThreshold {
			tickEvery = int(math.Max(float64(tickEvery), float64(minTickStep)))
		}
	}
	// 限制最大刻度数量，避免网格/标签过密
	maxDayTicks := 12
	autoStep := int(math.Ceil(float64(days) / float64(maxDayTicks)))
	if autoStep < 1 {
		autoStep = 1
	}
	if autoStep > tickEvery {
		tickEvery = autoStep
	}

	// 周末着色 + 垂直网格（贯穿内容区域），仅在刻度日画主网格
	for i := 0; i < days; i++ {
		x := xStart + i*dayWidth
		dayDate := minStart.Add(time.Duration(i) * time.Duration(hoursPerDay) * time.Hour)
		if isExcludedDay(dayDate, calendar) {
			fillRect(img, image.Rect(x, yStart, x+dayWidth, endY), weekendFill)
		}
		if i%tickEvery == 0 {
			for yy := yStart; yy < endY; yy++ {
				img.Set(x, yy, theme.Grid)
			}
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
	face, _, _ := font.LoadFaceWithFallback(float64(axisFontSize), "")
	format := axisFormat
	if strings.TrimSpace(format) == "" {
		format = "01-02"
	}
	startDay := minStart
	if weekStart != nil {
		startDay = alignToWeekStart(minStart, weekStart)
	}
	for cur := startDay; ; cur = cur.Add(time.Duration(tickEvery) * 24 * time.Hour) {
		offsetDays := calendarSpanDays(minStart, cur) - 1
		if offsetDays > days {
			break
		}
		if offsetDays >= 0 {
			x := xStart + offsetDays*dayWidth
			date := cur.Format(format)
			drawTextWithFace(img, theme.Text, x+tickLabelOffsetPx, yStart+axisHeight-tickLabelOffsetPx, face, date)
		}
		if offsetDays >= days {
			break
		}
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

func darkenColor(base color.Color, factor float64) color.Color {
	if factor <= 0 {
		factor = 1
	}
	rgba := color.RGBAModel.Convert(base).(color.RGBA)
	r := clampFloat(float64(rgba.R) * factor)
	g := clampFloat(float64(rgba.G) * factor)
	b := clampFloat(float64(rgba.B) * factor)
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
		Vertical:   mustColor(taskBorder, color.RGBA{0x00, 0x7a, 0xcc, 0xff}),
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
