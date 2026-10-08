package protocol_test

import (
	"encoding/json"
	"testing"

	"tops-lsp/internal/protocol"
)

func TestMarshalErrorPreservesRequestID(t *testing.T) {
	body, err := protocol.MarshalError(json.RawMessage(`7`), protocol.NewError(protocol.InvalidParams, "bad params", map[string]string{"field": "params"}))
	if err != nil {
		t.Fatalf("MarshalError() error = %v", err)
	}

	var response struct {
		JSONRPC string                `json:"jsonrpc"`
		ID      json.RawMessage       `json:"id"`
		Result  json.RawMessage       `json:"result"`
		Error   *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.JSONRPC != "2.0" || string(response.ID) != "7" {
		t.Fatalf("response envelope = %+v", response)
	}
	if len(response.Result) != 0 || response.Error == nil || response.Error.Code != protocol.InvalidParams {
		t.Fatalf("response error = %+v result = %s", response.Error, response.Result)
	}
}

func TestMarshalSuccessIncludesNullResult(t *testing.T) {
	body, err := protocol.MarshalSuccess(json.RawMessage(`"request"`), nil)
	if err != nil {
		t.Fatalf("MarshalSuccess() error = %v", err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(response["id"]) != `"request"` || string(response["result"]) != "null" {
		t.Fatalf("response = %s", body)
	}
}

func TestIsValidRequestID(t *testing.T) {
	valid := []string{`1`, `-2`, `1.5`, `"request"`, `""`}
	for _, value := range valid {
		if !protocol.IsValidRequestID(json.RawMessage(value)) {
			t.Errorf("IsValidRequestID(%s) = false", value)
		}
	}
	invalid := []string{`null`, `true`, `{}`, `[]`}
	for _, value := range invalid {
		if protocol.IsValidRequestID(json.RawMessage(value)) {
			t.Errorf("IsValidRequestID(%s) = true", value)
		}
	}
}
