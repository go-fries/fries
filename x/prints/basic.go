package prints

import (
	"strings"

	"github.com/fatih/color"
)

var (
	line    = color.New()
	info    = color.New(color.FgCyan)
	comment = color.New(color.FgWhite)
	errs    = color.New(color.FgRed)
	warn    = color.New(color.FgYellow)
	alert   = color.New(color.FgYellow, color.ReverseVideo)
	success = color.New(color.FgGreen)
)

func Line(a ...any) (int, error) {
	return line.Println(a...)
}

func Linef(format string, a ...any) (int, error) {
	return line.Printf(format, a...)
}

func NewLine(length ...int) (int, error) {
	var brs []string

	if len(length) > 0 {
		for i := 0; i < length[0]-1; i++ {
			brs = append(brs, "\n")
		}
	}

	return line.Println(strings.Join(brs, ""), "")
}

func Info(a ...any) (int, error) {
	return info.Println(a...)
}

func Infof(format string, a ...any) (int, error) {
	return info.Printf(format, a...)
}

func Comment(a ...any) (int, error) {
	return comment.Println(a...)
}

func Commentf(format string, a ...any) (int, error) {
	return comment.Printf(format, a...)
}

func Error(a ...any) (int, error) {
	return errs.Println(a...)
}

func Errorf(format string, a ...any) (int, error) {
	return errs.Printf(format, a...)
}

func Warn(a ...any) (int, error) {
	return warn.Println(a...)
}

func Warnf(format string, a ...any) (int, error) {
	return warn.Printf(format, a...)
}

func Alert(a ...any) (int, error) {
	return alert.Println(a...)
}

func Alertf(format string, a ...any) (int, error) {
	return alert.Printf(format, a...)
}

func Success(a ...any) (int, error) {
	return success.Println(a...)
}

func Successf(format string, a ...any) (int, error) {
	return success.Printf(format, a...)
}
