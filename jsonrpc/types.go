package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
)

const Version = "2.0"

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
