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

func TestMarshalPublishDiagnosticsCarriesConditionalProvenance(t *testing.T) {
	body, err := MarshalNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI: "file:///sample.cpp",
		Diagnostics: []Diagnostic{{
			Message: "condition is unknown",
			Data: &DiagnosticData{ConditionalProvenance: &ConditionalProvenance{
				SourceURI:       "file:///sample.cpp",
				RegionID:        2,
				BranchIndex:     1,
				State:           "unknown",
				ParentRegionID:  0,
				ExpressionRange: &Range{Start: Position{Line: 0, Character: 4}, End: Position{Line: 0, Character: 9}},
				ParentConditions: []ConditionalProvenance{{
					RegionID:    0,
					BranchIndex: 0,
					State:       "unknown",
				}},
			}},
		}},
	})
	if err != nil {
		t.Fatalf("MarshalNotification() error = %v", err)
	}
	var message struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	var params PublishDiagnosticsParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.Fatalf("diagnostics params decode error = %v", err)
	}
	provenance := params.Diagnostics[0].Data.ConditionalProvenance
	if provenance == nil || provenance.RegionID != 2 || provenance.ParentRegionID != 0 || len(provenance.ParentConditions) != 1 {
		t.Fatalf("conditional provenance = %+v", provenance)
	}
}

func TestMarshalPublishDiagnosticsCarriesSemanticData(t *testing.T) {
	documentVersion := 7
	body, err := MarshalNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:     "file:///sample.tops",
		Version: &documentVersion,
		Diagnostics: []Diagnostic{{
			Message: "host/device boundary",
			Data: &DiagnosticData{
				DocumentVersion: 7,
				ContextVersion:  3,
				SourceVersion:   "sha256:abc",
				ContextStatus:   "resolved",
				ProfileID:       "gcu400-default",
				PassKind:        "device",
				SemanticKind:    "host-device-boundary",
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		t.Fatal(err)
	}
	var params PublishDiagnosticsParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.Fatal(err)
	}
	data := params.Diagnostics[0].Data
	if data == nil || data.DocumentVersion != 7 || data.ContextVersion != 3 || data.SourceVersion != "sha256:abc" || data.ContextStatus != "resolved" || data.ProfileID != "gcu400-default" || data.PassKind != "device" || data.SemanticKind != "host-device-boundary" {
		t.Fatalf("semantic diagnostic data = %+v", data)
	}
}

func intPointer(value int) *int {
	return &value
}
