package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
)

func TestNavigationRejectsRequestBeforeInitialize(t *testing.T) {
	server := New(nil)
	body := server.handleRequest(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition"})
	if code := responseErrorCode(t, body); code != int(protocol.ServerNotInitialized) {
		t.Fatalf("error code = %d", code)
	}
}

func TestNavigationReturnsNullWithoutFreshSnapshot(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if err := server.documents.Open("file:///workspace/not-ready.tops", "tops", 1, "int value;"); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: "file:///workspace/not-ready.tops"}, Position: protocol.Position{Line: 0, Character: 4}})
	if err != nil {
		t.Fatal(err)
	}
	body := server.handleRequest(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	result, ok := response["result"]
	if !ok || string(result) != "null" {
		t.Fatalf("result = %s", result)
	}
	assertStaleResultEventFields(t, output.String(), "file:///workspace/not-ready.tops", 0, 1, 0, 1)
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}

func TestNavigationRejectsInvalidPosition(t *testing.T) {
	server := New(nil)
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if err := server.documents.Open("file:///workspace/position.tops", "tops", 1, "int value;"); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: "file:///workspace/position.tops"}, Position: protocol.Position{Line: 5}})
	if err != nil {
		t.Fatal(err)
	}
	body := server.handleRequest(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
	if code := responseErrorCode(t, body); code != int(protocol.InvalidParams) {
		t.Fatalf("error code = %d", code)
	}
}

func TestNavigationHonorsCancelledContext(t *testing.T) {
	server := New(nil)
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if err := server.documents.Open("file:///workspace/cancel.tops", "tops", 1, "int value;"); err != nil {
		t.Fatal(err)
	}
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: "file:///workspace/cancel.tops"}, Position: protocol.Position{Line: 0, Character: 4}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	body := server.handleRequest(ctx, protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
	if code := responseErrorCode(t, body); code != int(protocol.RequestCancelled) {
		t.Fatalf("error code = %d", code)
	}
}

func TestProtocolSymbolKindPreservesTypeKind(t *testing.T) {
	if got := protocolSymbolKind(index.SymbolKindType, "enum"); got != protocol.SymbolKindEnum {
		t.Fatalf("enum symbol kind = %d", got)
	}
	if got := protocolSymbolKind(index.SymbolKindType, "struct"); got != protocol.SymbolKindStruct {
		t.Fatalf("struct symbol kind = %d", got)
	}
	if got := protocolSymbolKind(index.SymbolKindType, "class"); got != protocol.SymbolKindClass {
		t.Fatalf("class symbol kind = %d", got)
	}
}

func TestNavigationResponseGateDropsResultAfterDocumentChange(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	uri := "file:///workspace/gated.tops"
	state, err := server.openDocument(uri, "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	started := make(chan struct{})
	release := make(chan struct{})
	server.registerHandler("test/navigation", func(ctx context.Context, message protocol.Message) []byte {
		state, ok := ctx.Value(requestStateKey{}).(*requestState)
		if !ok || state == nil {
			t.Fatal("request state was not attached to context")
		}
		state.SetFreshnessCheck(func() bool {
			current, exists := server.documents.Get(uri)
			return exists && current.Version == 1
		})
		close(started)
		<-release
		return server.successResponse(message.ID, []string{"old-result"})
	})
	response := make(chan []byte, 1)
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		body, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "test/navigation", Params: params})
		response <- body
	}()
	<-started
	if _, err := server.changeDocument(uri, 2, []document.Change{{Range: &document.Range{Start: document.Position{Line: 0, Character: 0}, End: document.Position{Line: 0, Character: len("int value;")}}, Text: "int changed;"}}); err != nil {
		t.Fatal(err)
	}
	close(release)
	body := <-response
	var decoded struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded.Result) != "null" {
		t.Fatalf("gated result = %s", decoded.Result)
	}
	assertStaleResultEventFields(t, output.String(), uri, 1, 2, 1, 1)
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}

func TestNavigationQueryStaleResultLogsRequestURIAndVersions(t *testing.T) {
	var output bytes.Buffer
	started := make(chan struct{})
	release := make(chan struct{})
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{
		NavigationGate: func(ctx context.Context, operation string) error {
			if operation != "definition" {
				return nil
			}
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	uri := "file:///workspace/in-flight.tops"
	state, err := server.openDocument(uri, "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	params, err := json.Marshal(protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     protocol.Position{Line: 0, Character: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := make(chan []byte, 1)
	go func() {
		body, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
		response <- body
	}()
	<-started
	if _, err := server.changeDocument(uri, 2, []document.Change{{
		Range: &document.Range{Start: document.Position{Line: 0, Character: 0}, End: document.Position{Line: 0, Character: len("int value;")}},
		Text:  "int changed;",
	}}); err != nil {
		t.Fatal(err)
	}
	close(release)
	body := <-response
	var decoded struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded.Result) != "null" {
		t.Fatalf("stale navigation result = %s", decoded.Result)
	}
	assertStaleResultEventFields(t, output.String(), uri, 1, 2, 1, 1)
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}

func TestNavigationHeaderChangeStaleResultLogsHeaderVersionTransition(t *testing.T) {
	var output bytes.Buffer
	started := make(chan struct{})
	release := make(chan struct{})
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{
		SourceProvider: index.NewLocalSourceProvider(nil, nil),
		NavigationGate: func(ctx context.Context, operation string) error {
			if operation != "definition" {
				return nil
			}
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	headerURI := "file:///workspace/api.h"
	rootURI := "file:///workspace/main.tops"
	oldHeader := "int old_value;\n"
	newHeader := "int new_value;\n"
	if _, err := server.openDocument(headerURI, "tops-header", 1, oldHeader); err != nil {
		t.Fatal(err)
	}
	rootText := "#include \"api.h\"\nint use() { return old_value; }\n"
	rootState, err := server.openDocument(rootURI, "tops", 1, rootText)
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, rootState)
	oldSnapshot, ok := server.CurrentIndexSnapshot(rootURI)
	if !ok || len(oldSnapshot.Dependencies) != 1 {
		t.Fatalf("initial root snapshot = %+v, ok = %v", oldSnapshot, ok)
	}
	params, err := json.Marshal(protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: rootURI},
		Position:     protocol.Position{Line: 1, Character: 19},
	})
	if err != nil {
		t.Fatal(err)
	}
	response := make(chan []byte, 1)
	go func() {
		body, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
		response <- body
	}()
	<-started
	if _, err := server.changeDocument(headerURI, 2, []document.Change{{
		Range: &document.Range{Start: document.Position{}, End: document.Position{Line: 1, Character: 0}},
		Text:  newHeader,
	}}); err != nil {
		t.Fatal(err)
	}
	server.markHeaderDependentsStale(headerURI)
	close(release)
	body := <-response
	var decoded struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if string(decoded.Result) != "null" {
		t.Fatalf("stale header navigation result = %s", decoded.Result)
	}
	logged := output.String()
	assertStaleResultEventFields(t, logged, rootURI, 1, 1, 1, 1)
	oldHeaderVersion := index.SourceDigest(oldHeader)
	newHeaderVersion := index.SourceDigest(newHeader)
	if !strings.Contains(logged, `"old_header_versions":"`+logging.DocumentID(headerURI)+`:`+oldHeaderVersion+`"`) {
		t.Fatalf("stale log lacks old header version: %s", logged)
	}
	if !strings.Contains(logged, `"new_header_versions":"`+logging.DocumentID(headerURI)+`:`+newHeaderVersion+`"`) {
		t.Fatalf("stale log lacks new header version: %s", logged)
	}
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}

func TestLateIndexStaleResultLogsGenerationEpochAndHeaderVersions(t *testing.T) {
	var output bytes.Buffer
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, nil)})
	headerURI := "file:///workspace/api.h"
	rootURI := "file:///workspace/main.tops"
	oldHeader := "int old_value;\n"
	newHeader := "int new_value;\n"
	if _, err := server.openDocument(headerURI, "tops-header", 1, oldHeader); err != nil {
		t.Fatal(err)
	}
	rootState, err := server.openDocument(rootURI, "tops", 1, "#include \"api.h\"\nint use() { return old_value; }\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, rootState)
	started := make(chan struct{})
	release := make(chan struct{})
	var parseCalls atomic.Int32
	server.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		if parseCalls.Add(1) == 1 {
			close(started)
			<-release
		}
		return parser.Parse(text, parseContext)
	}
	done := make(chan struct{})
	go func() {
		server.parseAndPublish(context.Background(), nil, rootState)
		close(done)
	}()
	<-started
	if _, err := server.changeDocument(headerURI, 2, []document.Change{{
		Range: &document.Range{Start: document.Position{}, End: document.Position{Line: 1, Character: 0}},
		Text:  newHeader,
	}}); err != nil {
		t.Fatal(err)
	}
	server.markHeaderDependentsStale(headerURI)
	server.nextAnalysis(rootURI)
	close(release)
	<-done
	logged := output.String()
	if !strings.Contains(logged, `"operation":"index"`) || !strings.Contains(logged, `"reason":"snapshot-mismatch"`) {
		t.Fatalf("late index stale log = %s", logged)
	}
	for _, field := range []string{"old_generation", "new_generation", "old_epoch", "new_epoch", "old_header_versions", "new_header_versions"} {
		if !strings.Contains(logged, `"`+field+`":`) {
			t.Fatalf("late index stale log missing %q: %s", field, logged)
		}
	}
	if strings.Contains(logged, `"old_context_version":0`) || strings.Contains(logged, `"old_header_versions":""`) {
		t.Fatalf("late index stale log has empty old version fields: %s", logged)
	}
	if !strings.Contains(logged, index.SourceDigest(oldHeader)) && !strings.Contains(logged, index.SourceDigest(newHeader)) {
		t.Fatalf("late index stale log lacks header digest: %s", logged)
	}
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("late stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}

func responseErrorCode(t *testing.T, body []byte) int {
	t.Helper()
	var response struct {
		Error *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == nil {
		t.Fatalf("response has no error: %s", body)
	}
	return int(response.Error.Code)
}
