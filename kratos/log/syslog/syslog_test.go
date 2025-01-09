//go:build !windows && !plan9

package syslog

import (
	"runtime"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestSyslogLogger(t *testing.T) {
	if runtime.GOOS == "windows" { // nolint:staticcheck
		t.Skip("skip syslog test")
	}

	logger := New(&Config{
		Network: "udp",
		Addr:    "192.168.8.92:30732",
		Tag:     "test",
	})
	defer logger.Close()

	err := logger.Log(log.LevelDebug, "test", "test")
	if err != nil {
		t.Fatal(err)
	}
}
