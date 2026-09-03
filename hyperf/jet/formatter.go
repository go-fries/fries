package jet

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type FormatterKind string

const (
	FormatterKindJSONRPC FormatterKind = "jsonrpc"
)

var DefaultFormatter Formatter = NewJSONRPCFormatter()

type RPCRequest struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Params []byte `json:"params"`
}

type RPCResponse struct {
	ID     string `json:"id"`
	Result []byte `json:"result"`
}

type RPCResponseError struct {
	ID      string `json:"id"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"error"`
}

var _ error = (*RPCResponseError)(nil)

func (r *RPCResponseError) Error() string {
	return fmt.Sprintf("code: %d, message: %s, error: %v", r.Code, r.Message, r.Err)
}

type Formatter interface {
	Kind() FormatterKind

	// FormatRequest formats a request
	FormatRequest(req *RPCRequest) ([]byte, error)

	// FormatResponse formats a response
	FormatResponse(resp *RPCResponse, err *RPCResponseError) ([]byte, error)

	// ParseRequest parses a request
	ParseRequest(data []byte) (*RPCRequest, error)

	// ParseResponse parses a response
	ParseResponse(data []byte) (*RPCResponse, error)
}

// ============================================================

// JSONRPCVersion is the json rpc version
var JSONRPCVersion = "2.0"

// JSONRPCFormatter is a json rpc formatter
type JSONRPCFormatter struct{}

type JSONRPCFormatterRequest struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      string          `json:"id"`
}

type JSONRPCFormatterResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    error  `json:"data,omitempty"`
}

type jsonRPCFormatterResponseErrorWire struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type jsonRPCErrorData struct {
	raw json.RawMessage
}

func (e jsonRPCErrorData) Error() string {
	var message string
	if err := json.Unmarshal(e.raw, &message); err == nil {
		return message
	}
	return string(e.raw)
}

func (e jsonRPCErrorData) MarshalJSON() ([]byte, error) {
	return e.raw, nil
}

func (e JSONRPCFormatterResponseError) MarshalJSON() ([]byte, error) {
	var data json.RawMessage
	if e.Data != nil {
		var err error
		if _, ok := e.Data.(json.Marshaler); ok {
			data, err = json.Marshal(e.Data)
		} else {
			data, err = json.Marshal(e.Data)
			if err == nil && bytes.Equal(data, []byte("{}")) {
				data, err = json.Marshal(e.Data.Error())
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return json.Marshal(jsonRPCFormatterResponseErrorWire{
		Code:    e.Code,
		Message: e.Message,
		Data:    data,
	})
}

func (e *JSONRPCFormatterResponseError) UnmarshalJSON(data []byte) error {
	var decoded jsonRPCFormatterResponseErrorWire
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	e.Code = decoded.Code
	e.Message = decoded.Message
	e.Data = nil
	if len(decoded.Data) > 0 && !bytes.Equal(bytes.TrimSpace(decoded.Data), []byte("null")) {
		e.Data = jsonRPCErrorData{raw: decoded.Data}
	}
	return nil
}

type JSONRPCFormatterResponse struct {
	Jsonrpc string                         `json:"jsonrpc"`
	Result  json.RawMessage                `json:"result"`
	ID      string                         `json:"id"`
	Error   *JSONRPCFormatterResponseError `json:"error"`
}

func NewJSONRPCFormatter() *JSONRPCFormatter {
	return &JSONRPCFormatter{}
}

func (j *JSONRPCFormatter) Kind() FormatterKind {
	return FormatterKindJSONRPC
}

func (j *JSONRPCFormatter) FormatRequest(req *RPCRequest) ([]byte, error) {
	return json.Marshal(&JSONRPCFormatterRequest{
		Jsonrpc: JSONRPCVersion,
		Method:  req.Path,
		Params:  req.Params,
		ID:      req.ID,
	})
}

func (j *JSONRPCFormatter) FormatResponse(resp *RPCResponse, err *RPCResponseError) ([]byte, error) {
	if err != nil {
		return json.Marshal(&JSONRPCFormatterResponse{
			Jsonrpc: JSONRPCVersion,
			ID:      err.ID,
			Error: &JSONRPCFormatterResponseError{
				Code:    err.Code,
				Message: err.Message,
				Data:    err.Err,
			},
		})
	}
	return json.Marshal(&JSONRPCFormatterResponse{
		Jsonrpc: JSONRPCVersion,
		ID:      resp.ID,
		Result:  resp.Result,
	})
}

func (j *JSONRPCFormatter) ParseRequest(data []byte) (*RPCRequest, error) {
	var req JSONRPCFormatterRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &RPCRequest{
		ID:     req.ID,
		Path:   req.Method,
		Params: req.Params,
	}, nil
}

func (j *JSONRPCFormatter) ParseResponse(data []byte) (*RPCResponse, error) {
	var resp JSONRPCFormatterResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, &RPCResponseError{
			ID:      resp.ID,
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
			Err:     resp.Error.Data,
		}
	}
	return &RPCResponse{
		ID:     resp.ID,
		Result: resp.Result,
	}, nil
}
