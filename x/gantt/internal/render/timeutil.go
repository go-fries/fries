package render

import (
	"time"

	"github.com/go-fries/fries/x/gantt/v3/internal/parser"
)

func tickDuration(t parser.TickInterval) time.Duration {
	if !t.Valid || t.Value <= 0 {
		return 0
	}
	switch t.Unit {
	case "millisecond":
		return time.Duration(t.Value) * time.Millisecond
	case "second":
		return time.Duration(t.Value) * time.Second
	case "minute":
		return time.Duration(t.Value) * time.Minute
	case "hour":
		return time.Duration(t.Value) * time.Hour
	case "day":
		return time.Duration(t.Value*24) * time.Hour
	case "week":
		return time.Duration(t.Value*7*24) * time.Hour
	case "month":
		// approximate month as 30 days for tick spacing
		return time.Duration(t.Value*30*24) * time.Hour
	default:
		return 0
	}
}

func tickToMinutes(t parser.TickInterval) int {
	d := tickDuration(t)
	if d == 0 {
		return 0
	}
	if d < time.Minute {
		return 1
	}
	return int(d.Minutes())
}

func tickToDays(t parser.TickInterval) int {
	if !t.Valid || t.Value <= 0 {
		return 0
	}
	switch t.Unit {
	case "day":
		return t.Value
	case "week":
		return t.Value * 7
	case "month":
		return t.Value * 30
	default:
		return 0
	}
}

func alignToWeekStart(t time.Time, wd *time.Weekday) time.Time {
	if wd == nil {
		return t
	}
	offset := (int(t.Weekday()) - int(*wd) + 7) % 7
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, -offset)
}
