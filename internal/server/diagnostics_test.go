package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestPublishParserDiagnosticsMapsUTF16RangeAndVersion(t *testing.T) {
	instance := New(nil)
	var output bytes.Buffer
	publisher := &frameDiagnosticPublisher{writer: transport.NewWriter(&output)}
	state := document.DocumentState{URI: "file:///sample.cpp", Version: 2, Text: "int 😀 = ;"}
	result := parser.ParseResult{
		DocumentVersion: 2,
		ContextVersion:  4,
		Diagnostics: []parser.ParserDiagnostic{{
			Code:     parser.DiagnosticUnexpectedToken,
			Severity: parser.SeverityError,
			Message:  "unexpected token",
			Range:    parser.SourceRange{Start: len("int "), End: len("int 😀")},
		}},
	}

	if err := instance.publishParserDiagnostics(context.Background(), publisher, state, result); err != nil {
		t.Fatalf("publishParserDiagnostics() error = %v", err)
	}
	body, err := transport.NewReader(bytes.NewReader(output.Bytes())).ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	var message struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &message); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if message.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("method = %q", message.Method)
	}
	var params protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(message.Params, &params); err != nil {
		t.Fatalf("params decode error = %v", err)
	}
	if params.Version == nil || *params.Version != 2 || len(params.Diagnostics) != 1 {
		t.Fatalf("params = %+v", params)
	}
	diagnostic := params.Diagnostics[0]
	if diagnostic.Range.Start.Character != 4 || diagnostic.Range.End.Character != 6 {
		t.Fatalf("range = %+v", diagnostic.Range)
	}
	if diagnostic.Code != parser.DiagnosticUnexpectedToken || diagnostic.Source != "tops-lsp" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestPublishParserDiagnosticsCarriesParserMetadata(t *testing.T) {
	instance := New(nil)
	publisher := &captureDiagnosticPublisher{}
	state := document.DocumentState{URI: "file:///sample.cpp", Version: 3, Text: "#if MAYBE\nint value = @;\n#endif\n"}
	result := parser.ParseResult{
		DocumentVersion: 3,
		ContextVersion:  9,
		Diagnostics: []parser.ParserDiagnostic{{
			Code:             parser.DiagnosticUnexpectedToken,
			Severity:         parser.SeverityError,
			Message:          "unrecognized character",
			Range:            parser.SourceRange{Start: 23, End: 24},
			ConditionalState: parser.ConditionalUnknown,
			Recoverable:      true,
			Incomplete:       true,
			ContextVersion:   9,
		}},
	}

	if err := instance.publishParserDiagnostics(context.Background(), publisher, state, result); err != nil {
		t.Fatalf("publishParserDiagnostics() error = %v", err)
	}
	if len(publisher.params) != 1 || len(publisher.params[0].Diagnostics) != 1 {
		t.Fatalf("published params = %+v", publisher.params)
	}
	metadata := publisher.params[0].Diagnostics[0].Data
	if metadata == nil || metadata.ConditionalState != "unknown" || !metadata.Recoverable || !metadata.Incomplete || metadata.ContextVersion != 9 {
		t.Fatalf("diagnostic metadata = %+v", metadata)
	}
	if publisher.params[0].ContextVersion == nil || *publisher.params[0].ContextVersion != 9 {
		t.Fatalf("published context version = %+v", publisher.params[0].ContextVersion)
	}
}

func TestPublishParserDiagnosticsMapsUnsupportedAndRelatedInformation(t *testing.T) {
	instance := New(nil)
	publisher := &captureDiagnosticPublisher{}
	state := document.DocumentState{URI: "file:///sample.cpp", Version: 4, Text: "concept Number = true;"}
	result := parser.ParseResult{
		DocumentVersion: 4,
		ContextVersion:  6,
		Diagnostics: []parser.ParserDiagnostic{{
			Code:             parser.DiagnosticUnsupported,
			Severity:         parser.SeverityInformation,
			Message:          "syntax is outside the configured C++ baseline",
			Range:            parser.SourceRange{Start: 0, End: 7},
			Related:          []parser.RelatedLocation{{Range: parser.SourceRange{Start: 0, End: 7}, Message: "unsupported construct"}},
			ConditionalState: parser.ConditionalActive,
			Recoverable:      true,
			ContextVersion:   6,
		}},
	}
	if err := instance.publishParserDiagnostics(context.Background(), publisher, state, result); err != nil {
		t.Fatalf("publishParserDiagnostics() error = %v", err)
	}
	if len(publisher.params) != 1 || len(publisher.params[0].Diagnostics) != 1 {
		t.Fatalf("published params = %+v", publisher.params)
	}
	diagnostic := publisher.params[0].Diagnostics[0]
	if diagnostic.Code != parser.DiagnosticUnsupported || diagnostic.Severity != protocol.DiagnosticSeverityInfo || len(diagnostic.RelatedInformation) != 1 || diagnostic.Data == nil || !diagnostic.Data.Recoverable {
		t.Fatalf("unsupported diagnostic = %+v", diagnostic)
	}
}

func TestParseAndPublishLogsSummaryWithoutSourceText(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	publisher := &captureDiagnosticPublisher{}
	state := document.DocumentState{URI: "file:///private/sample.cpp", LanguageID: "cpp", Version: 1, Text: "int value"}
	if err := instance.documents.Open(state.URI, state.LanguageID, state.Version, state.Text); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	instance.parseAndPublish(context.Background(), publisher, state)
	if !strings.Contains(logs.String(), `"event":"document_parsed"`) || !strings.Contains(logs.String(), `"diagnostic_count"`) {
		t.Fatalf("parse summary missing: %s", logs.String())
	}
	if strings.Contains(logs.String(), "file:///private/sample.cpp") || strings.Contains(logs.String(), "int value") {
		t.Fatalf("parse log contains source data: %s", logs.String())
	}
}

func TestPartialParseContextCarriesProvenanceAndVersion(t *testing.T) {
	instance := New(nil)
	state := document.DocumentState{URI: "file:///sample.cpp", LanguageID: "cpp", Version: 4}
	contextValue := instance.partialParseContextAt(state, instance.currentContextVersion())
	if contextValue.DriverKind != "unknown" || contextValue.CompilerContextStatus != "partial" || contextValue.ArgumentProvenance != "unknown" {
		t.Fatalf("context provenance = %+v", contextValue)
	}
	if contextValue.DocumentVersion != 4 || contextValue.ContextVersion == 0 {
		t.Fatalf("context versions = %+v", contextValue)
	}
}

func TestPublishParserDiagnosticsClearsWithEmptyArray(t *testing.T) {
	instance := New(nil)
	publisher := &captureDiagnosticPublisher{}
	state := document.DocumentState{URI: "file:///sample.cpp", Version: 3, Text: "int value;"}
	if err := instance.documents.Open(state.URI, "cpp", state.Version, state.Text); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	instance.parseAndPublish(context.Background(), publisher, state)
	if len(publisher.params) != 1 || len(publisher.params[0].Diagnostics) != 0 {
		t.Fatalf("empty diagnostics publication = %+v", publisher.params)
	}
}

type failingDiagnosticPublisher struct{}

func (failingDiagnosticPublisher) Publish(context.Context, protocol.PublishDiagnosticsParams) error {
	return errors.New("publisher payload contains source text")
}

func TestPublisherFailureDoesNotLogErrorText(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	state := document.DocumentState{URI: "file:///private/sample.cpp", LanguageID: "cpp", Version: 1, Text: "int value;"}
	if err := instance.documents.Open(state.URI, state.LanguageID, state.Version, state.Text); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	instance.parseAndPublish(context.Background(), failingDiagnosticPublisher{}, state)
	if strings.Contains(logs.String(), "publisher payload contains source text") || strings.Contains(logs.String(), "int value;") {
		t.Fatalf("publisher error leaked into logs: %s", logs.String())
	}
}
