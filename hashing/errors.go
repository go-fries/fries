package hashing

import "errors"

// ErrNilReader is returned when a nil reader is passed to SumReader.
var ErrNilReader = errors.New("hashing: nil reader")
