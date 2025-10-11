package jsonrpc

import "github.com/google/uuid"

type IDGenerator interface {
	Generate() *ID
}

type idGenerator struct {
	counter int64
}

func NewIDGenerator() IDGenerator {
	return &idGenerator{}
}

func (g *idGenerator) Generate() *ID {
	return NewID(uuid.New().String())
}
