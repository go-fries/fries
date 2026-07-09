//go:build !windows && !plan9

package syslog

// Version returns the current release version of this component.
func Version() string {
	return "4.0.0-beta.1"
}
