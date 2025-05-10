package queue

import "github.com/google/uuid"

type Generator interface {
	Generate() string
}

type GeneratorFunc func() string

func (f GeneratorFunc) Generate() string {
	return f()
}

func UUIDGenerator() Generator {
	return GeneratorFunc(func() string {
		return uuid.New().String()
	})
}
