package parser

import (
	"fmt"
	"strings"
	"time"
)

const (
	hoursPerDay   = 24
	daysPerWeek   = 7
	hoursPerWeek  = daysPerWeek * hoursPerDay
	hoursPerMonth = 30 * hoursPerDay
)

// ResolveSchedule 解析依赖并计算起止时间与持续天数。
// nolint:gocyclo // 核心逻辑复杂，计划按 refactor-design 拆分
func ResolveSchedule(m Model) (Model, error) {
	taskMap := make(map[string]*Task)
	for si := range m.Sections {
		for ti := range m.Sections[si].Tasks {
			task := &m.Sections[si].Tasks[ti]
			if task.IsVertical {
				continue
			}
			if _, exists := taskMap[task.ID]; exists {
				orig := task.ID
				suffix := 1
				for {
					candidate := fmt.Sprintf("%s_%d", orig, suffix)
					if _, exists := taskMap[candidate]; !exists {
						task.ID = candidate
						task.ExplicitID = false
						taskMap[task.ID] = task
						break
					}
					suffix++
				}
				continue
			}
			taskMap[task.ID] = task
		}
	}

	loc := time.UTC
	if m.Calendar.Timezone != "" {
		if tz, err := time.LoadLocation(m.Calendar.Timezone); err == nil {
			loc = tz
		}
	}
	baseStart := baselineStart(m, loc)

	visited := make(map[string]bool)
	resolving := make(map[string]bool)
	prevInSection := make(map[string]*Task)
	for si := range m.Sections {
		var prev *Task
		for ti := range m.Sections[si].Tasks {
			t := &m.Sections[si].Tasks[ti]
			if t.IsVertical {
				continue
			}
			if prev != nil {
				prevInSection[t.ID] = prev
			}
			prev = t
		}
	}

	var resolve func(*Task) error
	resolve = func(t *Task) error {
		if visited[t.ID] {
			return nil
		}
		if resolving[t.ID] {
			return ParseError{Line: t.Line, Column: t.Column, Message: fmt.Sprintf("circular dependency around %s", t.ID)}
		}
		resolving[t.ID] = true

		isTimeTask := t.HasTime || t.Duration.Unit == DurationMinute || t.Duration.Unit == DurationHour

		if t.HasStart && !t.HasTime {
			y, mth, d := t.Start.Date()
			t.Start = time.Date(y, mth, d, 0, 0, 0, 0, loc)
		}
		if t.HasEnd && !t.HasTime {
			y, mth, d := t.End.Date()
			t.End = time.Date(y, mth, d, 0, 0, 0, 0, loc)
		}

		var start time.Time
		if t.HasStart {
			start = t.Start
			if isTimeTask && start.Hour() == 0 && strings.Contains(t.StartExpr, ":") {
				if parsed, err := parseClock(t.StartExpr, m.Calendar.Timezone, baseStart); err == nil {
					start = parsed
					t.Start = parsed
				}
			}
		} else if strings.Contains(t.StartExpr, ":") {
			if parsed, err := parseClock(t.StartExpr, m.Calendar.Timezone, baseStart); err == nil {
				start = parsed
				t.Start = parsed
				t.HasStart = true
				isTimeTask = true
			}
		}

		beforeStarts := []time.Time{}
		maxAfter := time.Time{}
		for _, dep := range t.Dependencies {
			target, ok := taskMap[dep.Target]
			if !ok {
				return ParseError{Line: t.Line, Column: t.Column, Message: fmt.Sprintf("dependency not found: %s", dep.Target)}
			}
			if err := resolve(target); err != nil {
				return err
			}
			switch dep.Type {
			case DepAfter:
				if isTimeTask || target.HasTime || target.Duration.Unit == DurationMinute || target.Duration.Unit == DurationHour {
					if candidate := target.End.Add(time.Nanosecond); candidate.After(maxAfter) {
						maxAfter = candidate
					}
				} else {
					if candidate := startOfNextDay(target.End); candidate.After(maxAfter) {
						maxAfter = candidate
					}
				}
			case DepBefore:
				// 结束不晚于目标开始；选取更靠后的可行起点以贴近约束
				if target.Start.IsZero() {
					continue
				}
				startCandidate := target.Start.Add(-durationToDuration(t.Duration))
				if !t.HasStart { // 只有在未显式指定开始时间时才调整起点
					if start.IsZero() || start.Before(startCandidate) {
						start = startCandidate
					}
				}
				beforeStarts = append(beforeStarts, target.Start)
			}
		}

		// 若存在 before/until 依赖，按最早依赖开始前一天结束，保持显式起点不被移动
		if len(beforeStarts) > 0 && !start.IsZero() {
			minBefore := beforeStarts[0]
			for _, ts := range beforeStarts[1:] {
				if ts.Before(minBefore) {
					minBefore = ts
				}
			}
			// 结束在依赖开始日的前一日
			customEnd := time.Date(minBefore.Year(), minBefore.Month(), minBefore.Day(), 0, 0, 0, 0, minBefore.Location()).Add(-time.Nanosecond)
			span := int(customEnd.Sub(start).Hours()/hoursPerDay) + 1
			if span <= 0 {
				span = 1
			}
			t.Duration = DurationSpec{Value: span, Unit: DurationDay}
			t.DurationExplicit = true
			t.Start = start
			t.End = customEnd
			t.HasEnd = true
			t.DurationDays = span
			visited[t.ID] = true
			resolving[t.ID] = false
			return nil
		}

		if start.IsZero() {
			// 若仍未确定起点，优先使用同 section 前一任务的结束时间，否则基准起点
			prev := prevInSection[t.ID]
			if prev == nil {
				prev = findPrevInSection(m, t)
			}
			if prev != nil {
				if err := resolve(prev); err != nil {
					return err
				}
				prevEnd := prev.End
				if prevEnd.IsZero() {
					prevEnd = prev.Start.Add(durationToDuration(prev.Duration))
				}
				if isTimeTask || prev.HasTime || prev.Duration.Unit == DurationMinute || prev.Duration.Unit == DurationHour {
					start = prevEnd.Add(time.Nanosecond)
				} else {
					start = startOfNextDay(prevEnd)
				}
			}
			if start.IsZero() {
				start = baseStart
			}
		}

		if !maxAfter.IsZero() && (start.IsZero() || maxAfter.After(start)) {
			start = maxAfter
		}

		origDur := t.Duration
		end, days := applyCalendar(start, t.Duration, m.Calendar)
		// 对周/月等单位使用天数期望，避免包容端偏差
		if origDur.Unit == DurationWeek && origDur.Value > 0 {
			days = origDur.Value * daysPerWeek
			end = start.Add(durationToDuration(DurationSpec{Value: days, Unit: DurationDay}) - time.Nanosecond)
		}
		t.Start = start
		t.End = end
		t.DurationDays = days
		visited[t.ID] = true
		resolving[t.ID] = false
		return nil
	}

	for i := range m.Sections {
		for j := range m.Sections[i].Tasks {
			if m.Sections[i].Tasks[j].IsVertical {
				continue
			}
			if err := resolve(&m.Sections[i].Tasks[j]); err != nil {
				return Model{}, err
			}
		}
	}

	return m, nil
}

func baselineStart(m Model, loc *time.Location) time.Time {
	var min time.Time
	add := func(t time.Time) {
		if t.IsZero() {
			return
		}
		t = t.In(loc)
		if min.IsZero() || t.Before(min) {
			min = t
		}
	}
	for _, sec := range m.Sections {
		for _, task := range sec.Tasks {
			if task.HasStart {
				add(task.Start)
			}
		}
	}
	for _, v := range m.Verticals {
		if v.HasStart {
			end := v.Start.In(loc)
			if v.Duration.Value > 0 {
				end = end.Add(durationToDuration(v.Duration))
				if v.Duration.Unit == DurationMinute || v.Duration.Unit == DurationHour {
					end = end.Add(time.Minute)
				}
			}
			add(end)
		}
	}
	if min.IsZero() {
		min = time.Now().In(loc)
		min = time.Date(min.Year(), min.Month(), min.Day(), 0, 0, 0, 0, loc)
	}
	return min
}

func durationToDuration(d DurationSpec) time.Duration {
	switch d.Unit {
	case DurationMinute:
		return time.Duration(d.Value) * time.Minute
	case DurationHour:
		return time.Duration(d.Value) * time.Hour
	case DurationWeek:
		return time.Duration(d.Value*hoursPerWeek) * time.Hour
	case DurationMonth:
		return time.Duration(d.Value*hoursPerMonth) * time.Hour
	case DurationDay:
		fallthrough
	default:
		return time.Duration(d.Value*hoursPerDay) * time.Hour
	}
}

func applyCalendar(start time.Time, dur DurationSpec, cal Calendar) (time.Time, int) {
	loc := time.UTC
	if cal.Timezone != "" {
		if tz, err := time.LoadLocation(cal.Timezone); err == nil {
			loc = tz
		}
	}
	start = start.In(loc)
	if dur.Value <= 0 {
		return start, 0
	}

	remaining := durationToDuration(dur)
	current := start
	timeBased := dur.Unit == DurationMinute || (dur.Unit == DurationHour && dur.Value < hoursPerDay)

	// 跳过起始日若为排除日
	for shouldSkipDay(current, cal) {
		current = startOfNextDay(current)
		start = current
	}

	for remaining > 0 {
		if shouldSkipDay(current, cal) {
			current = startOfNextDay(current)
			continue
		}
		endOfDay := startOfNextDay(current)
		span := endOfDay.Sub(current)
		if remaining <= span {
			// 将结束定位在当前工作区间内
			if timeBased {
				current = current.Add(remaining)
			} else {
				current = current.Add(remaining - time.Nanosecond)
			}
			break
		}
		remaining -= span
		current = endOfDay
	}

	end := current
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
	days := int(endDay.Sub(startDay).Hours()/hoursPerDay) + 1
	if days <= 0 {
		days = 1
	}
	return end, days
}

func startOfNextDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
}

func findPrevInSection(m Model, t *Task) *Task {
	for si := range m.Sections {
		for idx := range m.Sections[si].Tasks {
			cur := &m.Sections[si].Tasks[idx]
			if cur == t && idx > 0 {
				return &m.Sections[si].Tasks[idx-1]
			}
		}
	}
	return nil
}

func shouldSkipDay(t time.Time, cal Calendar) bool {
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

func isWeekendDay(t time.Time, cal Calendar) bool {
	if len(cal.WeekendDays) == 0 {
		return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
	}
	for _, wd := range cal.WeekendDays {
		if t.Weekday() == wd {
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
