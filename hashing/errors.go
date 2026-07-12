package hashing

import "errors"

// ErrNilReader indicates that SumReader received a nil reader.
var ErrNilReader = errors.New("hashing: nil reader")
