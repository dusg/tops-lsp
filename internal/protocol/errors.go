package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type ErrorCode int

const (
	ParseError           ErrorCode = -32700
	InvalidRequest       ErrorCode = -32600
	MethodNotFound       ErrorCode = -32601
	InvalidParams        ErrorCode = -32602
	InternalError        ErrorCode = -32603
	ServerNotInitialized ErrorCode = -32002
	RequestCancelled     ErrorCode = -32800
)

type ErrorObject struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Data    any       `json:"data,omitempty"`
}

func NewError(code ErrorCode, message string, data any) ErrorObject {
	return ErrorObject{Code: code, Message: message, Data: data}
}

type DecodeError struct {
	Code    ErrorCode
	Message string
	ID      json.RawMessage
}

func (e *DecodeError) Error() string {
	if e == nil {
		return "protocol decode error"
	}
	return e.Message
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

func MarshalSuccess(id json.RawMessage, result any) ([]byte, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Response{
		JSONRPC: "2.0",
		ID:      normalizedID(id),
		Result:  resultBytes,
	})
}

func MarshalError(id json.RawMessage, protocolError ErrorObject) ([]byte, error) {
	return json.Marshal(Response{
		JSONRPC: "2.0",
		ID:      normalizedID(id),
		Error:   &protocolError,
	})
}

func normalizedID(id json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(id)) == 0 {
		return json.RawMessage("null")
	}
	return append(json.RawMessage(nil), id...)
}

func IsValidRequestID(raw json.RawMessage) bool {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 || bytes.Equal(value, []byte("null")) || !json.Valid(value) {
		return false
	}
	if value[0] == '"' {
		var stringID string
		return json.Unmarshal(value, &stringID) == nil
	}

	var numberID json.Number
	if err := json.Unmarshal(value, &numberID); err != nil {
		return false
	}
	return numberID.String() != ""
}

func IDKey(id json.RawMessage) string {
	return string(bytes.TrimSpace(id))
}

func InvalidRequestError(message string) *DecodeError {
	return &DecodeError{Code: InvalidRequest, Message: message}
}

func ParseErrorFrom(err error) *DecodeError {
	return &DecodeError{Code: ParseError, Message: fmt.Sprintf("failed to parse JSON: %v", err)}
}
