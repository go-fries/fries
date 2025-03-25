//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
	_ "github.com/wadey/gocovmerge"
	_ "go.opentelemetry.io/build-tools/multimod"
	_ "golang.org/x/exp/cmd/gorelease"
)
