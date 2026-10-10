package protocol

import (
	"encoding/json"
	"testing"
)

func TestMarshalPublishDiagnosticsNotification(t *testing.T) {
	body, err := MarshalNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:     "file:///sample.cpp",
		Version: intPointer(2),
		Diagnostics: []Diagnostic{{
			Range:    Range{Start: Position{Line: 1, Character: 2}, End: Position{Line: 1, Character: 2}},
			Severity: DiagnosticSeverityError,
			Code:     "tops-syntax-missing-token",
			Source:   "tops-lsp",
			Message:  "expected ';'",
			Data: &DiagnosticData{
				ConditionalState: "unknown",
				Recoverable:      true,
				Incomplete:       true,
				ContextVersion:   3,
			},
		}},
	})
	if err != nil {
		t.Fatalf("MarshalNotification() error = %v", err)
	}

	var message struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if message.JSONRPC != "2.0" || message.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("message = %+v", message)
	}

	var params PublishDiagnosticsParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.Fatalf("diagnostics params decode error = %v", err)
	}
	if params.URI != "file:///sample.cpp" || params.Version == nil || *params.Version != 2 || len(params.Diagnostics) != 1 {
		t.Fatalf("params = %+v", params)
	}
	if params.Diagnostics[0].Severity != DiagnosticSeverityError || params.Diagnostics[0].Code != "tops-syntax-missing-token" {
		t.Fatalf("diagnostic = %+v", params.Diagnostics[0])
	}
	metadata := params.Diagnostics[0].Data
	if metadata == nil || metadata.ConditionalState != "unknown" || !metadata.Recoverable || !metadata.Incomplete || metadata.ContextVersion != 3 {
		t.Fatalf("diagnostic metadata = %+v", metadata)
	}
}

func TestMarshalPublishDiagnosticsAllowsEmptyDiagnostics(t *testing.T) {
	body, err := MarshalNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         "file:///sample.cpp",
		Diagnostics: []Diagnostic{},
	})
	if err != nil {
		t.Fatalf("MarshalNotification() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("MarshalNotification() returned an empty body")
	}
}

func intPointer(value int) *int {
	return &value
}
