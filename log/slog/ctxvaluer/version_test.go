package ctxvaluer_test

import (
	"regexp"
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
)

func TestVersionSemver(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(ctxvaluer.Version()) {
		t.Fatalf("invalid release version: %q", ctxvaluer.Version())
	}
}
