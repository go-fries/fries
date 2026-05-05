package jsonrpc

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type ID struct {
	str   *string
	num   *float64
	isNil bool
}

func (i *ID) String() string {
	if i.isNil {
		return "null" //nolint:goconst
	}
	if i.str != nil {
		return *i.str
	}
	if i.num != nil {
		return fmt.Sprintf("%v", *i.num)
	}
	return "null"
}

func NewID(v any) *ID {
	switch val := v.(type) {
	case nil:
		return &ID{isNil: true}
	case string:
		return &ID{str: &val}
	case int:
		f := float64(val)
		return &ID{num: &f}
	case int8:
		f := float64(val)
		return &ID{num: &f}
	case int16:
		f := float64(val)
		return &ID{num: &f}
	case int32:
		f := float64(val)
		return &ID{num: &f}
	case int64:
		f := float64(val)
		return &ID{num: &f}
	case uint:
		f := float64(val)
		return &ID{num: &f}
	case uint8:
		f := float64(val)
		return &ID{num: &f}
	case uint16:
		f := float64(val)
		return &ID{num: &f}
	case uint32:
		f := float64(val)
		return &ID{num: &f}
	case uint64:
		f := float64(val)
		return &ID{num: &f}
	case uintptr:
		f := float64(val)
		return &ID{num: &f}
	case float32:
		f := float64(val)
		return &ID{num: &f}
	case float64:
		return &ID{num: &val}
	case complex64:
		f := float64(real(val))
		return &ID{num: &f}
	case complex128:
		f := real(val)
		return &ID{num: &f}
	default:
		return nil
	}
}

func (i *ID) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.isNil = true
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		i.str = &s
		return nil
	}

	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		i.num = &n
		return nil
	}

	return nil
}

func (i ID) MarshalJSON() ([]byte, error) {
	if i.isNil {
		return []byte("null"), nil
	}
	if i.str != nil {
		return json.Marshal(i.str)
	}
	if i.num != nil {
		return json.Marshal(i.num)
	}
	return []byte("null"), nil
}

var DefaultIDGenerator = NewUUIDGenerator()

type IDGenerator interface {
	Generate() *ID
}

type uuidGenerator struct{}

var _ IDGenerator = (*uuidGenerator)(nil)

func NewUUIDGenerator() IDGenerator {
	return &uuidGenerator{}
}

func (g *uuidGenerator) Generate() *ID {
	return NewID(uuid.New().String())
}
