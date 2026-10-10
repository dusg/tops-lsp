package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
)

type captureDiagnosticPublisher struct {
	params []protocol.PublishDiagnosticsParams
}

func (publisher *captureDiagnosticPublisher) Publish(_ context.Context, params protocol.PublishDiagnosticsParams) error {
	publisher.params = append(publisher.params, params)
	return nil
}

type blockingFirstDiagnosticPublisher struct {
	entered chan struct{}
	release chan struct{}
	calls   chan protocol.PublishDiagnosticsParams
	count   int
}

func (publisher *blockingFirstDiagnosticPublisher) Publish(_ context.Context, params protocol.PublishDiagnosticsParams) error {
	publisher.count++
	if publisher.count == 1 {
		close(publisher.entered)
		<-publisher.release
	}
	publisher.calls <- params
	return nil
}

func TestDocumentChangeCannotInterleaveWithDiagnosticsPublish(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), staleRequest(t, "1", "initialize", nil))
	uri := "file:///sample.cpp"
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: "cpp", Version: 1, Text: "int value;"},
	}), nil)

	publisher := &blockingFirstDiagnosticPublisher{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		calls:   make(chan protocol.PublishDiagnosticsParams, 2),
	}
	firstDone := make(chan struct{})
	go func() {
		instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{URI: uri, Version: 2},
			ContentChanges: []protocol.ContentChange{{
				Range: &protocol.Range{Start: protocol.Position{}, End: protocol.Position{Line: 0, Character: len("int value;")}},
				Text:  "int next;",
			}},
		}), publisher)
		close(firstDone)
	}()
	<-publisher.entered

	secondDone := make(chan struct{})
	go func() {
		instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{URI: uri, Version: 3},
			ContentChanges: []protocol.ContentChange{{
				Range: &protocol.Range{Start: protocol.Position{}, End: protocol.Position{Line: 0, Character: len("int next;")}},
				Text:  "int final;",
			}},
		}), publisher)
		close(secondDone)
	}()

	select {
	case <-secondDone:
		t.Fatal("document change interleaved with diagnostics publication")
	case <-time.After(100 * time.Millisecond):
	}
	state, ok := instance.Documents().Get(uri)
	if !ok || state.Version != 2 {
		t.Fatalf("document state while publishing = %+v, exists = %v", state, ok)
	}

	close(publisher.release)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first change did not complete")
	}
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second change did not complete")
	}
	state, ok = instance.Documents().Get(uri)
	if !ok || state.Version != 3 {
		t.Fatalf("final document state = %+v, exists = %v", state, ok)
	}
}

func TestCloseReopenDropsLateDiagnostics(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), staleRequest(t, "1", "initialize", nil))
	uri := "file:///sample.cpp"
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: "cpp", Version: 1, Text: "int value;"},
	}), nil)

	started := make(chan struct{})
	release := make(chan struct{})
	parseCount := 0
	instance.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		parseCount++
		if parseCount == 1 {
			close(started)
			<-release
		}
		return parser.Parse(text, parseContext)
	}
	publisher := &channelDiagnosticPublisher{values: make(chan protocol.PublishDiagnosticsParams, 2)}
	instance.handleNotificationWithPublisherAsync(context.Background(), staleNotification(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{URI: uri, Version: 2},
		ContentChanges: []protocol.ContentChange{{
			Range: &protocol.Range{Start: protocol.Position{}, End: protocol.Position{Line: 0, Character: len("int value;")}},
			Text:  "int changed;",
		}},
	}), publisher)
	<-started

	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didClose", protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	}), nil)
	instance.handleNotificationWithPublisherAsync(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: "cpp", Version: 3, Text: "int reopened;"},
	}), publisher)
	close(release)

	select {
	case publication := <-publisher.values:
		if publication.Version == nil || *publication.Version != 3 {
			t.Fatalf("published late version = %+v", publication)
		}
	case <-time.After(time.Second):
		t.Fatal("reopened document was not published")
	}
	instance.analysisWG.Wait()
	select {
	case publication := <-publisher.values:
		t.Fatalf("unexpected additional publication = %+v", publication)
	default:
	}
}

func TestRejectedChangeDoesNotPublishNewDiagnostics(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), staleRequest(t, "1", "initialize", nil))
	publisher := &captureDiagnosticPublisher{}
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: "file:///sample.cpp", LanguageID: "cpp", Version: 1, Text: "int value"},
	}), publisher)
	if len(publisher.params) != 1 || publisher.params[0].Version == nil || *publisher.params[0].Version != 1 {
		t.Fatalf("open diagnostics = %+v", publisher.params)
	}

	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{URI: "file:///sample.cpp", Version: 1},
		ContentChanges: []protocol.ContentChange{{
			Range: &protocol.Range{Start: protocol.Position{Line: 4, Character: 0}, End: protocol.Position{Line: 4, Character: 0}},
			Text:  "invalid",
		}},
	}), publisher)
	if len(publisher.params) != 1 {
		t.Fatalf("rejected change published diagnostics: %+v", publisher.params)
	}
}

func TestCloseStopsParserDiagnosticsPublication(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), staleRequest(t, "1", "initialize", nil))
	publisher := &captureDiagnosticPublisher{}
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: "file:///sample.cpp", LanguageID: "cpp", Version: 1, Text: "int value"},
	}), publisher)
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didClose", protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///sample.cpp"},
	}), publisher)
	if len(publisher.params) != 1 {
		t.Fatalf("close changed diagnostics count: %+v", publisher.params)
	}
}

func TestOneHundredSequentialChangesPublishLatestVersion(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), staleRequest(t, "1", "initialize", nil))
	publisher := &captureDiagnosticPublisher{}
	uri := "file:///sample.cpp"
	text := "int value = 0;"
	instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: "cpp", Version: 1, Text: text},
	}), publisher)
	for version := 2; version <= 101; version++ {
		updated := "int value = " + string(rune('0'+version%10)) + ";"
		instance.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{URI: uri, Version: version},
			ContentChanges: []protocol.ContentChange{{
				Range: &protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: len(text)}},
				Text:  updated,
			}},
		}), publisher)
		text = updated
	}
	if len(publisher.params) != 101 || publisher.params[len(publisher.params)-1].Version == nil || *publisher.params[len(publisher.params)-1].Version != 101 {
		t.Fatalf("published versions = %+v", publisher.params)
	}
}

func TestSupersededAndClosedGenerationsCannotPublish(t *testing.T) {
	instance := New(nil)
	uri := "file:///sample.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	first := instance.nextAnalysis(uri)
	second := instance.nextAnalysis(uri)
	contextVersion := instance.currentContextVersion()
	if instance.analysisCurrent(uri, contextVersion, first) {
		t.Fatal("superseded generation remained current")
	}
	if !instance.analysisCurrent(uri, contextVersion, second) {
		t.Fatal("latest generation was not current")
	}
	if !instance.documents.Close(uri) {
		t.Fatal("Close() = false")
	}
	instance.invalidateAnalysis(uri)
	if instance.analysisCurrent(uri, contextVersion, second) {
		t.Fatal("closed generation remained publishable")
	}
}

func TestContextVersionInvalidatesAnalysisGeneration(t *testing.T) {
	instance := New(nil)
	uri := "file:///sample.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	generation := instance.nextAnalysis(uri)
	contextVersion := instance.currentContextVersion()
	if !instance.analysisCurrent(uri, contextVersion, generation) {
		t.Fatal("current context was rejected")
	}
	instance.InvalidateContext()
	if instance.analysisCurrent(uri, contextVersion, generation) {
		t.Fatal("stale context remained publishable")
	}
}

func TestLateParseResultIsDroppedAfterContextInvalidation(t *testing.T) {
	instance := New(nil)
	uri := "file:///sample.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	instance.parse = func(text string, context parser.ParseContext) parser.ParseResult {
		close(started)
		<-release
		return parser.Parse(text, context)
	}
	publisher := &captureDiagnosticPublisher{}
	done := make(chan struct{})
	go func() {
		instance.parseAndPublish(context.Background(), publisher, documentStateForTest(uri, 1, "int value;"))
		close(done)
	}()
	<-started
	oldContextVersion := instance.currentContextVersion()
	instance.InvalidateContext()
	if got := instance.currentContextVersion(); got != oldContextVersion+1 {
		t.Fatalf("context version = %d, want %d", got, oldContextVersion+1)
	}
	close(release)
	<-done
	if len(publisher.params) != 0 {
		t.Fatalf("late context result was published: %+v", publisher.params)
	}
}

func TestLateParseResultIsDroppedAfterCloseAndChange(t *testing.T) {
	instance := New(nil)
	uri := "file:///sample.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	instance.parse = func(text string, context parser.ParseContext) parser.ParseResult {
		close(started)
		<-release
		return parser.Parse(text, context)
	}
	publisher := &captureDiagnosticPublisher{}
	done := make(chan struct{})
	go func() {
		instance.parseAndPublish(context.Background(), publisher, documentStateForTest(uri, 1, "int value;"))
		close(done)
	}()
	<-started
	if err := instance.documents.Change(uri, 2, []document.Change{{Text: "int next;", Range: &document.Range{Start: document.Position{Line: 0, Character: 0}, End: document.Position{Line: 0, Character: len("int value;")}}}}); err != nil {
		t.Fatalf("Change() error = %v", err)
	}
	instance.nextAnalysis(uri)
	if !instance.documents.Close(uri) {
		t.Fatal("Close() = false")
	}
	instance.invalidateAnalysis(uri)
	close(release)
	<-done
	if len(publisher.params) != 0 {
		t.Fatalf("late closed result was published: %+v", publisher.params)
	}
}

func TestParseContextVersionIsUsedForNewAnalysis(t *testing.T) {
	instance := New(nil)
	uri := "file:///sample.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	instance.InvalidateContext()
	var parsedContext parser.ParseContext
	instance.parse = func(text string, context parser.ParseContext) parser.ParseResult {
		parsedContext = context
		return parser.Parse(text, context)
	}
	publisher := &captureDiagnosticPublisher{}
	instance.parseAndPublish(context.Background(), publisher, documentStateForTest(uri, 1, "int value;"))
	if parsedContext.ContextVersion != instance.currentContextVersion() || len(publisher.params) != 1 || publisher.params[0].ContextVersion == nil || *publisher.params[0].ContextVersion != parsedContext.ContextVersion {
		t.Fatalf("context lifecycle = context=%+v published=%+v", parsedContext, publisher.params)
	}
}

func documentStateForTest(uri string, version int, text string) document.DocumentState {
	return document.DocumentState{URI: uri, LanguageID: "cpp", Version: version, Text: text}
}

func staleRequest(t *testing.T, id, method string, params any) protocol.Message {
	t.Helper()
	return protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(id), Method: method, Params: staleParams(t, params)}
}

func staleNotification(t *testing.T, method string, params any) protocol.Message {
	t.Helper()
	return protocol.Message{Kind: protocol.NotificationMessage, JSONRPC: "2.0", Method: method, Params: staleParams(t, params)}
}

func staleParams(t *testing.T, params any) json.RawMessage {
	t.Helper()
	if params == nil {
		return nil
	}
	body, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return body
}
