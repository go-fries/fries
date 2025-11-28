package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	minTaskParts          = 2
	minDirectiveParts     = 2
	minAxisFields         = 2
	tickIntervalPartCount = 3
	taskSplitParts        = 2
	minLowerLen           = 2
	maxProgressPercent    = 100
	hoursPerDayInt        = 24
	minPartsForLayout     = 2
	minFieldsForTimezone  = 2
	minMatchesForDate     = 3
	minDurationParts      = 2
	percentDivisor        = 100.0
)

var tickIntervalRe = regexp.MustCompile(`^([1-9][0-9]*)(millisecond|second|minute|hour|day|week|month)$`)

// Parse 从字符串解析 Gantt。
// nolint:gocyclo // 复杂度较高，后续按 refactor-design.md 拆分
func Parse(src string) (Model, error) {
	if strings.TrimSpace(src) == "" {
		return Model{}, fmt.Errorf("source is empty")
	}
	scanner := bufio.NewScanner(strings.NewReader(src))
	model := Model{
		DateFormat: "2006-01-02",
		Calendar: Calendar{
			Timezone:       "UTC",
			ExcludeWeekend: false,
			WeekendDays:    nil,
		},
		Today: TodayMarker{Enabled: true},
	}
	sectionName := ""
	index := 0
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "gantt"):
			continue
		case strings.HasPrefix(lower, "title"):
			model.Title = strings.TrimSpace(line[len("title"):])
			continue
		case strings.HasPrefix(lower, "dateformat"):
			model.DateFormat = dateLayout(strings.TrimSpace(line[len("dateformat"):]))
			continue
		case strings.HasPrefix(lower, "axisformat"), strings.HasPrefix(lower, "tickformat"):
			parts := strings.Fields(line)
			if len(parts) >= minAxisFields {
				model.AxisFormat = convertStrftimeLayout(strings.TrimSpace(parts[1]))
			}
			continue
		case strings.HasPrefix(lower, "todaymarker"):
			parseTodayMarker(strings.TrimSpace(line[len("todaymarker"):]), model.DateFormat, &model)
			continue
		case strings.HasPrefix(lower, "tickinterval"):
			parseTickInterval(strings.TrimSpace(line[len("tickinterval"):]), &model)
			continue
		case strings.HasPrefix(lower, "weekday"):
			parseWeekday(strings.TrimSpace(line[len("weekday"):]), &model)
			continue
		case strings.HasPrefix(lower, "timezone"):
			fields := strings.Fields(line)
			if len(fields) >= minAxisFields {
				model.Calendar.Timezone = strings.TrimSpace(fields[1])
			}
			continue
		case strings.HasPrefix(lower, "excludes"):
			parseCalendarDates(strings.TrimSpace(line[len("excludes"):]), true, model.DateFormat, &model)
			continue
		case strings.HasPrefix(lower, "includes"):
			parseCalendarDates(strings.TrimSpace(line[len("includes"):]), false, model.DateFormat, &model)
			continue
		case strings.HasPrefix(lower, "weekend"):
			parseWeekendDirective(strings.TrimSpace(line[len("weekend"):]), &model)
			continue
		case strings.HasPrefix(lower, "section"):
			sectionName = strings.TrimSpace(line[len("section"):])
			model.Sections = append(model.Sections, Section{Name: sectionName})
			continue
		default:
			task, err := parseTaskLine(line, lineNo, sectionName, model.DateFormat, model.Calendar)
			if err != nil {
				return Model{}, err
			}
			if len(model.Sections) == 0 {
				model.Sections = append(model.Sections, Section{Name: sectionName})
			}
			task.Index = index
			if task.IsVertical {
				model.Verticals = append(model.Verticals, task)
			} else {
				model.Sections[len(model.Sections)-1].Tasks = append(model.Sections[len(model.Sections)-1].Tasks, task)
				index++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Model{}, err
	}

	totalTasks := 0
	for _, s := range model.Sections {
		totalTasks += len(s.Tasks)
	}
	if totalTasks == 0 {
		return Model{}, fmt.Errorf("no tasks parsed")
	}
	return model, nil
}

func parseTodayMarker(expr, layout string, model *Model) {
	fields := strings.Fields(expr)
	if len(fields) == 0 {
		model.Today.Enabled = true
		return
	}
	if strings.EqualFold(fields[0], "off") || strings.EqualFold(fields[0], "false") {
		model.Today.Enabled = false
		return
	}
	dateStr := fields[0]
	if t, err := time.Parse(layoutOrDefault(layout), dateStr); err == nil {
		model.Today.Enabled = true
		model.Today.Date = t
		model.Today.HasDate = true
		return
	}
	model.Today.Enabled = true
}

func parseCalendarDates(expr string, exclude bool, layout string, model *Model) {
	parts := strings.FieldsFunc(expr, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t'
	})
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(p), "weekend") {
			if exclude {
				model.Calendar.ExcludeWeekend = true
				if len(model.Calendar.WeekendDays) == 0 {
					model.Calendar.WeekendDays = defaultWeekend()
				}
			}
			continue
		}
		if wd, ok := weekdayFromString(strings.ToLower(p)); ok {
			if exclude {
				model.Calendar.ExcludeWeekend = true
				if !weekdayExists(model.Calendar.WeekendDays, wd) {
					model.Calendar.WeekendDays = append(model.Calendar.WeekendDays, wd)
				}
			}
			continue
		}
		if t, err := time.Parse(layoutOrDefault(layout), p); err == nil {
			if exclude {
				model.Calendar.ExcludeDates = append(model.Calendar.ExcludeDates, t)
			} else {
				model.Calendar.IncludeDates = append(model.Calendar.IncludeDates, t)
			}
		}
	}
}

func parseWeekendDirective(expr string, model *Model) {
	model.Calendar.ExcludeWeekend = true
	tokens := strings.Fields(expr)
	// 语义：保留默认的周六为周末，周日只有在显式声明时才作为周末；显式 token 追加为周末
	weekend := []time.Weekday{time.Saturday}
	for _, p := range tokens {
		if wd, ok := weekdayFromString(strings.ToLower(p)); ok {
			if !weekdayExists(weekend, wd) {
				weekend = append(weekend, wd)
			}
		}
	}
	// 若未声明任何 token，使用默认周六/周日
	if len(tokens) == 0 {
		weekend = defaultWeekend()
	}
	model.Calendar.WeekendDays = weekend
}

func weekdayFromString(s string) (time.Weekday, bool) {
	switch s {
	case "sun", "sunday":
		return time.Sunday, true
	case "mon", "monday":
		return time.Monday, true
	case "tue", "tues", "tuesday":
		return time.Tuesday, true
	case "wed", "wednesday":
		return time.Wednesday, true
	case "thu", "thur", "thurs", "thursday":
		return time.Thursday, true
	case "fri", "friday":
		return time.Friday, true
	case "sat", "saturday":
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func weekdayExists(days []time.Weekday, wd time.Weekday) bool {
	for _, d := range days {
		if d == wd {
			return true
		}
	}
	return false
}

func defaultWeekend() []time.Weekday {
	return []time.Weekday{time.Saturday, time.Sunday}
}

const defaultDateLayout = "2006-01-02"

func parseTickInterval(expr string, model *Model) {
	fields := strings.Fields(expr)
	if len(fields) == 0 {
		return
	}
	matches := tickIntervalRe.FindStringSubmatch(fields[0])
	if len(matches) != tickIntervalPartCount {
		return
	}
	val, err := strconv.Atoi(matches[1])
	if err != nil || val <= 0 {
		return
	}
	model.Tick = TickInterval{Value: val, Unit: matches[2], Valid: true}
}

func parseWeekday(expr string, model *Model) {
	parts := strings.Fields(expr)
	if len(parts) == 0 {
		return
	}
	if wd, ok := weekdayFromString(strings.ToLower(parts[0])); ok {
		model.WeekStart = &wd
	}
}

func convertStrftimeLayout(format string) string {
	if strings.TrimSpace(format) == "" {
		return defaultDateLayout
	}
	var b strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			i++
			switch format[i] {
			case 'Y':
				b.WriteString("2006")
			case 'y':
				b.WriteString("06")
			case 'm':
				b.WriteString("01")
			case 'b':
				b.WriteString("Jan")
			case 'B':
				b.WriteString("January")
			case 'd':
				b.WriteString("02")
			case 'e':
				b.WriteString("2")
			case 'H':
				b.WriteString("15")
			case 'I':
				b.WriteString("03")
			case 'M':
				b.WriteString("04")
			case 'S':
				b.WriteString("05")
			case 'L':
				b.WriteString(".000")
			case 'p':
				b.WriteString("PM")
			case 'a':
				b.WriteString("Mon")
			case 'A':
				b.WriteString("Monday")
			case 'x':
				b.WriteString("01/02/2006")
			case 'X':
				b.WriteString("15:04:05")
			case 'c':
				b.WriteString("Mon Jan 2 15:04:05 2006")
			case 'Z', 'z':
				b.WriteString("-0700")
			case '%':
				b.WriteByte('%')
			default:
				b.WriteByte('%')
				b.WriteByte(format[i])
			}
		} else {
			b.WriteByte(format[i])
		}
	}
	return b.String()
}

// nolint:gocyclo // 复杂度较高，后续按 refactor-design.md 拆分
func parseTaskLine(line string, lineNo int, section, layout string, cal Calendar) (Task, error) {
	parseDeps := func(field string, typ DependencyType) []Dependency {
		depStr := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(field, "after"), "before"))
		depStr = strings.TrimSpace(strings.TrimPrefix(depStr, "until"))
		var deps []Dependency
		for _, tok := range strings.Fields(depStr) {
			deps = append(deps, Dependency{Type: typ, Target: tok})
		}
		return deps
	}

	addDate := func(task *Task, val string, isStart bool) {
		if t, err := parseDate(val, layout, cal.Timezone); err == nil {
			if isStart && !task.HasStart {
				task.Start = t
				task.HasStart = true
				task.StartExpr = val
				task.HasTime = hasClock(val)
			} else if !isStart && !task.HasEnd {
				task.End = t
				task.HasEnd = true
				task.EndExpr = val
			}
		}
	}

	parts := strings.SplitN(line, ":", taskSplitParts)
	if len(parts) < minTaskParts {
		return Task{}, newParseError(lineNo, 1, fmt.Sprintf("invalid task line: %s", line))
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return Task{}, newParseError(lineNo, 1, "task name is empty")
	}

	task := Task{
		Name:    name,
		Section: section,
		Line:    lineNo,
		Column:  1,
		Status:  StatusNormal,
		Duration: DurationSpec{
			Value: 1,
			Unit:  DurationDay,
		},
	}

	fields := strings.Split(parts[1], ",")
	dateCount := 0
	var deps []Dependency
	for _, raw := range fields {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		lower := strings.ToLower(field)
		switch {
		case lower == "crit" || lower == "critical":
			task.Status = StatusCritical
		case lower == "done":
			task.Status = StatusDone
		case lower == "active":
			task.Status = StatusActive
		case lower == "milestone":
			task.Status = StatusMilestone
			task.IsMilestone = true
		case lower == "vert":
			task.IsVertical = true
			task.IsMilestone = false
		case strings.HasPrefix(lower, "after"):
			deps = append(deps, parseDeps(field, DepAfter)...)
			task.StartExpr = strings.TrimSpace(field)
		case strings.HasPrefix(lower, "before"), strings.HasPrefix(lower, "until"):
			deps = append(deps, parseDeps(field, DepBefore)...)
		case strings.Contains(field, "%"):
			if p := parseProgress(field); p >= 0 {
				task.Progress = p
			}
		case looksLikeDuration(field):
			task.Duration = parseDurationSpec(field)
			task.DurationExplicit = true
		case isDate(field, layout):
			dateCount++
			if !task.HasStart {
				addDate(&task, field, true)
			} else if !task.HasEnd {
				addDate(&task, field, false)
			}
		default:
			if task.ID == "" && isIdentifierCandidate(field) {
				task.ID = field
				task.ExplicitID = true
			} else {
				task.Resources = append(task.Resources, field)
			}
			if !task.HasStart && strings.Contains(field, ":") {
				task.StartExpr = field
				if t, err := parseDate(field, layout, cal.Timezone); err == nil {
					task.Start = t
					task.HasStart = true
					task.HasTime = true
				}
			}
		}
	}

	if task.Status == StatusMilestone {
		task.IsMilestone = true
		if !task.DurationExplicit {
			task.Duration = DurationSpec{Value: 0, Unit: DurationDay}
			task.DurationExplicit = true
		}
	}
	if task.IsVertical {
		if !task.DurationExplicit {
			task.Duration = DurationSpec{Value: 0, Unit: DurationDay}
			task.DurationExplicit = true
		}
		task.IsMilestone = false
	}
	// 若提供开始和结束日期，转换为持续时间
	if task.HasStart && task.HasEnd {
		spanDays := int(task.End.Sub(task.Start).Hours()/hoursPerDayInt) + 1 // inclusive of end date
		if spanDays <= 0 {
			spanDays = 1
		}
		task.Duration = DurationSpec{Value: spanDays, Unit: DurationDay}
		task.DurationExplicit = true
	}

	if task.ID == "" {
		task.ID = fmt.Sprintf("auto_%d", lineNo)
	}
	if task.HasStart && task.Start.Hour() == 0 && strings.Contains(task.StartExpr, ":") {
		if t, err := parseClock(task.StartExpr, cal.Timezone, time.Time{}); err == nil {
			task.Start = t
		}
	}
	task.Dependencies = deps
	return task, nil
}

// ParseFile 从文件读取并解析。
func ParseFile(path string) (Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Model{}, fmt.Errorf("read source file: %w", err)
	}
	return Parse(string(data))
}

func dateLayout(format string) string {
	return convertDayjsLayout(format)
}

func layoutOrDefault(layout string) string {
	if strings.TrimSpace(layout) == "" {
		return "2006-01-02"
	}
	return layout
}

func parseDate(val, layout, tz string) (time.Time, error) {
	loc := time.UTC
	if tz != "" {
		if locLoaded, err := time.LoadLocation(tz); err == nil {
			loc = locLoaded
		}
	}
	layout = layoutOrDefault(layout)
	t, err := time.ParseInLocation(layout, val, loc)
	if err != nil {
		return t, err
	}
	if t.Year() <= 1 {
		now := time.Now().In(loc)
		t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
	}
	return t, nil
}

func parseClock(val, tz string, base time.Time) (time.Time, error) {
	loc := time.UTC
	if tz != "" {
		if locLoaded, err := time.LoadLocation(tz); err == nil {
			loc = locLoaded
		}
	}
	if base.IsZero() {
		base = time.Now().In(loc)
	}
	layouts := []string{"15:04:05.000", "15:04:05", "15:04"}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, val, loc); err == nil {
			return time.Date(base.Year(), base.Month(), base.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("clock parse failed")
}

func hasClock(val string) bool {
	return strings.Contains(val, ":")
}

func convertDayjsLayout(format string) string {
	f := strings.TrimSpace(format)
	if f == "" {
		return "2006-01-02"
	}
	// 处理毫秒与时区等长 token 时应优先替换长字符串
	replacements := []struct {
		src string
		dst string
	}{
		{"YYYY", "2006"},
		{"YY", "06"},
		{"MMMM", "January"},
		{"MMM", "Jan"},
		{"MM", "01"},
		{"M", "1"},
		{"DDDD", "002"},
		{"DDD", "002"},
		{"DD", "02"},
		{"D", "2"},
		{"HH", "15"},
		{"H", "15"},
		{"hh", "03"},
		{"h", "3"},
		{"mm", "04"},
		{"m", "4"},
		{"ss", "05"},
		{"s", "5"},
		{"SSS", "000"},
		{"SS", "00"},
		{"S", "0"},
		{"A", "PM"},
		{"a", "pm"},
		{"ZZ", "-0700"},
		{"Z", "-07:00"},
	}
	out := f
	for _, rep := range replacements {
		out = strings.ReplaceAll(out, rep.src, rep.dst)
	}
	return out
}

func looksLikeDuration(val string) bool {
	lower := strings.ToLower(strings.TrimSpace(val))
	if len(lower) < minLowerLen {
		return false
	}
	// 必须以数字开头，且仅允许数字后跟单位
	if lower[0] < '0' || lower[0] > '9' {
		return false
	}
	// quick reject if any space inside
	if strings.ContainsAny(lower, " \t") {
		return false
	}
	if strings.HasSuffix(lower, "mo") {
		_, err := strconv.Atoi(strings.TrimSuffix(lower, "mo"))
		return err == nil
	}
	unit := lower[len(lower)-1]
	switch unit {
	case 'h', 'd', 'w', 'm':
		_, err := strconv.Atoi(strings.TrimSuffix(lower, string(unit)))
		return err == nil
	default:
		return false
	}
}

func parseDurationSpec(val string) DurationSpec {
	lower := strings.ToLower(strings.TrimSpace(val))
	unit := DurationDay
	switch {
	case strings.HasSuffix(lower, "mo"):
		unit = DurationMonth
		lower = strings.TrimSuffix(lower, "mo")
	case strings.HasSuffix(lower, "w"):
		unit = DurationWeek
		lower = strings.TrimSuffix(lower, "w")
	case strings.HasSuffix(lower, "h"):
		unit = DurationHour
		lower = strings.TrimSuffix(lower, "h")
	case strings.HasSuffix(lower, "m"):
		unit = DurationMinute
		lower = strings.TrimSuffix(lower, "m")
	default:
		lower = strings.TrimSuffix(lower, "d")
	}

	value := 1
	if _, err := fmt.Sscanf(strings.TrimSpace(lower), "%d", &value); err != nil {
		value = 1
	}
	if value < 0 {
		value = 1
	}
	return DurationSpec{Value: value, Unit: unit}
}

func isDate(val, layout string) bool {
	_, err := time.Parse(layoutOrDefault(layout), val)
	return err == nil
}

func parseProgress(val string) int {
	val = strings.TrimSpace(val)
	val = strings.TrimSuffix(val, "%")
	var p int
	if _, err := fmt.Sscanf(val, "%d", &p); err != nil {
		return -1
	}
	if p < 0 {
		p = 0
	}
	if p > maxProgressPercent {
		p = maxProgressPercent
	}
	return p
}

func isIdentifierCandidate(val string) bool {
	if val == "" {
		return false
	}
	// 粗略判断：无空白即可视为 id/资源
	return !strings.ContainsAny(val, " \t")
}
