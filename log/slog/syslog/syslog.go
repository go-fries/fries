//go:build !windows && !plan9

package syslog

import stdsyslog "log/syslog"

// Dial connects to syslog and returns a [Handler] backed by the syslog writer.
func Dial(network, addr string, opts ...Option) (*Handler, error) {
	cfg := newConfig(opts...)
	writer, err := stdsyslog.Dial(network, addr, cfg.facility, cfg.tag)
	if err != nil {
		return nil, err
	}
	return newHandler(writer, cfg), nil
}
