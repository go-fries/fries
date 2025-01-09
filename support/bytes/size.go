package bytes

import (
	"fmt"
	"strconv"
)

type Bytes int64

const (
	KB Bytes = 1 << (10 * (iota + 1)) //nolint:mnd
	MB
	GB
	TB
	PB
	EB
)

func (b Bytes) Unit() string {
	switch {
	case b >= EB:
		return "EB"
	case b >= PB:
		return "PB"
	case b >= TB:
		return "TB"
	case b >= GB:
		return "GB"
	case b >= MB:
		return "MB"
	case b >= KB:
		return "KB"
	default:
		return "B"
	}
}

func (b Bytes) Format() float64 {
	switch {
	case b >= EB:
		return b.EB()
	case b >= PB:
		return b.PB()
	case b >= TB:
		return b.TB()
	case b >= GB:
		return b.GB()
	case b >= MB:
		return b.MB()
	case b >= KB:
		return b.KB()
	default:
		return b.Bytes()
	}
}

func (b Bytes) Bytes() float64 {
	return float64(b)
}

func (b Bytes) KB() float64 {
	return float64(b) / float64(KB)
}

func (b Bytes) MB() float64 {
	return float64(b) / float64(MB)
}

func (b Bytes) GB() float64 {
	return float64(b) / float64(GB)
}

func (b Bytes) TB() float64 {
	return float64(b) / float64(TB)
}

func (b Bytes) PB() float64 {
	return float64(b) / float64(PB)
}

func (b Bytes) EB() float64 {
	return float64(b) / float64(EB)
}

func (b Bytes) HumanizeValue(lens ...int) string {
	l := 2
	if len(lens) > 0 {
		l = lens[0]
	}

	return fmt.Sprintf("%."+strconv.Itoa(l)+"f", b.Format())
}

func (b Bytes) Humanize(lens ...int) string {
	return b.HumanizeValue(lens...) + b.Unit()
}
