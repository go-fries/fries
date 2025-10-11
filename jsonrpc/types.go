package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
)

const JSONRPCVersion = "2.0"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      *ID             `json:"id,omitempty"`
}

// Response 表示一个 JSON-RPC 2.0 响应对象
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      *ID             `json:"id"`
}

// Error 表示一个 JSON-RPC 错误对象
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	return fmt.Sprintf("code: %d, message: %s, data: %s", e.Code, e.Message, string(e.Data))
}

func (e *Error) Unwrap() error {
	return errors.New(e.Message)
}

type ID struct {
	str   *string
	num   *float64
	isNil bool
}

func (i *ID) String() string {
	if i.isNil {
		return "null"
	}
	if i.str != nil {
		return *i.str
	}
	if i.num != nil {
		return fmt.Sprintf("%v", *i.num)
	}
	return "null"
}

type id interface {
	~string | ~float64 | ~int
}

func NewID[T id](v T) *ID {
	var id ID
	switch any(v).(type) {
	case string:
		s := any(v).(string)
		id.str = &s
	case float64:
		n := any(v).(float64)
		id.num = &n
	case int:
		n := float64(any(v).(int))
		id.num = &n
	default:
		return nil
	}
	return &id
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
