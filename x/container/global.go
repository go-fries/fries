package container

import (
	"errors"
	"sync/atomic"
)

var ErrTypeMismatch = errors.New("type mismatch")

var global atomic.Value

func SetContainer(c Container) {
	global.Store(c)
}

func GetContainer() Container {
	if global.Load() == nil {
		SetContainer(New())
	}
	return global.Load().(Container)
}

func Set[T any](key any, value T) {
	GetContainer().Set(key, value)
}

func Get[T any](key any) (T, error) {
	v, err := GetContainer().Get(key)
	if err != nil {
		var zero T
		return zero, err
	}
	if vv, ok := v.(T); ok {
		return vv, nil
	}
	var zero T
	return zero, ErrTypeMismatch
}

func MustGet[T any](key any) T {
	v, err := Get[T](key)
	if err != nil {
		panic(err)
	}
	return v
}
