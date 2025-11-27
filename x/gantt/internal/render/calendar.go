package render

import (
	"time"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
)

const (
	hoursPerWeek  = 7 * hoursPerDay
	hoursPerMonth = 30 * hoursPerDay
)

// DurationHours 将 DurationSpec 转换为小时。
func DurationHours(d parser.DurationSpec) float64 {
	if d.Value <= 0 {
		return 0
	}
	switch d.Unit {
	case parser.DurationHour:
		return float64(d.Value)
	case parser.DurationWeek:
		return float64(d.Value * hoursPerWeek)
	case parser.DurationMonth:
		return float64(d.Value * hoursPerMonth)
	case parser.DurationDay:
		fallthrough
	default:
		return float64(d.Value * hoursPerDay)
	}
}

// ApplyCalendar 计算持续时间在日历下的结束时间与天数跨度。
// daysSpan 用于渲染时计算宽度（至少 1 天）。
func ApplyCalendar(start time.Time, dur parser.DurationSpec, cal parser.Calendar) (end time.Time, daysSpan int) {
	loc := time.UTC
	if cal.Timezone != "" {
		if tz, err := time.LoadLocation(cal.Timezone); err == nil {
			loc = tz
		}
	}
	start = start.In(loc)

	// milestone 或零时长
	if dur.Value <= 0 {
		return start, 0
	}

	hours := DurationHours(dur)
	if hours == 0 {
		return start, 0
	}

	current := start
	remaining := hours
	daysSpan = 0

	for remaining > 0 {
		// 检查是否跳过当前日
		if shouldSkipDay(current, cal) {
			current = nextDay(current)
			daysSpan++
			continue
		}

		if remaining >= hoursPerDay {
			remaining -= hoursPerDay
			current = nextDay(current)
			daysSpan++
			continue
		}

		// 剩余不足一天的小时，仍占用当日
		current = current.Add(time.Duration(remaining * float64(time.Hour)))
		remaining = 0
		daysSpan++
	}

	if daysSpan == 0 {
		daysSpan = 1
	}
	return current, daysSpan
}

func shouldSkipDay(t time.Time, cal parser.Calendar) bool {
	if cal.ExcludeWeekend && (t.Weekday() == time.Saturday || t.Weekday() == time.Sunday) {
		// includeDates 可以强制包含
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

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func nextDay(t time.Time) time.Time {
	return t.AddDate(0, 0, 1)
}
