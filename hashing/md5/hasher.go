package md5

import (
	"crypto/md5"
	"fmt"

	"github.com/go-fires/fires/v3/hashing"
)

type hasher struct{}

var global *hasher

func New() hashing.Hasher {
	if global == nil {
		global = &hasher{}
	}

	return global
}

func init() {
	hashing.Register(hashing.MD5, New)
}

// Make generates a new hashed value.
func (h *hasher) Make(value string) (string, error) {
	hashedValue := md5.Sum([]byte(value))

	return fmt.Sprintf("%x", hashedValue), nil
}

// MustMake generates a new hashed value.
func (h *hasher) MustMake(value string) string {
	hashedValue, err := h.Make(value)
	if err != nil {
		panic(err)
	}

	return hashedValue
}

// Check checks the given value and hashed value.
func (h *hasher) Check(value, hashedValue string) bool {
	hv, err := h.Make(value)
	if err != nil {
		return false
	}

	return hv == hashedValue
}
