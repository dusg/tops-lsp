package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestRunContinuesWhileParserIsBlocked(t *testing.T) {
	instance := New(nil)
	if !instance.tryInitialize() {
		t.Fatal("tryInitialize() = false")
	}
	started := make(chan struct{})
	release := make(chan struct{})
	instance.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		close(started)
		<-release
		return parser.Parse(text, parseContext)
	}

	reader, input := io.Pipe()
	status := make(chan int, 1)

	params, err := json.Marshal(protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{
		URI: "file:///blocked.cpp", LanguageID: "cpp", Version: 1, Text: "int value;",
	}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	message, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": json.RawMessage(params),
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	exit, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "exit"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	go func() {
		status <- instance.Run(context.Background(), transport.NewReader(reader), transport.NewWriter(io.Discard))
	}()
	didOpenWritten := make(chan error, 1)
	go func() {
		_, writeErr := input.Write(transport.Frame(message))
		didOpenWritten <- writeErr
	}()
	select {
	case got := <-status:
		t.Fatalf("Run() returned before parser started with status %d", got)
	case <-started:
	}
	if err := <-didOpenWritten; err != nil {
		t.Fatalf("write didOpen frame error = %v", err)
	}
	exitWritten := make(chan error, 1)
	go func() {
		_, writeErr := input.Write(transport.Frame(exit))
		exitWritten <- writeErr
	}()

	if err := <-exitWritten; err != nil {
		t.Fatalf("write exit frame error = %v", err)
	}
	select {
	case got := <-status:
		close(release)
		_ = input.Close()
		t.Fatalf("Run() returned before parser completed with status %d", got)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case got := <-status:
		if got != 1 {
			t.Fatalf("Run() status = %d, want 1", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after parser completed")
	}
	_ = input.Close()
}

func TestTerminateWaitsForAnalysisWorker(t *testing.T) {
	instance := New(nil)
	uri := "file:///terminate.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int value;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	instance.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		close(started)
		<-release
		return parser.Parse(text, parseContext)
	}
	publisher := &channelDiagnosticPublisher{values: make(chan protocol.PublishDiagnosticsParams, 1)}
	state, _ := instance.documents.Get(uri)
	instance.enqueueAnalysis(context.Background(), publisher, state)
	<-started

	terminated := make(chan int, 1)
	go func() {
		terminated <- instance.terminate(context.Background(), 1, slog.LevelInfo, "test_terminate")
	}()
	select {
	case status := <-terminated:
		close(release)
		t.Fatalf("terminate returned before parser completed with status %d", status)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case status := <-terminated:
		if status != 1 {
			t.Fatalf("terminate status = %d, want 1", status)
		}
	case <-time.After(time.Second):
		t.Fatal("terminate did not return after parser completed")
	}
	instance.analysisWG.Wait()
}

type channelDiagnosticPublisher struct {
	values chan protocol.PublishDiagnosticsParams
}

func (publisher *channelDiagnosticPublisher) Publish(_ context.Context, params protocol.PublishDiagnosticsParams) error {
	publisher.values <- params
	return nil
}

func TestAnalysisQueuePublishesOnlyLatestDocumentSnapshot(t *testing.T) {
	instance := New(nil)
	uri := "file:///latest.cpp"
	if err := instance.documents.Open(uri, "cpp", 1, "int first;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	secondParse := make(chan struct{})
	parseCount := 0
	instance.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		parseCount++
		if parseCount == 1 {
			close(started)
			<-release
		} else if parseCount == 2 {
			close(secondParse)
		}
		return parser.Parse(text, parseContext)
	}
	publisher := &channelDiagnosticPublisher{values: make(chan protocol.PublishDiagnosticsParams, 1)}
	state, _ := instance.documents.Get(uri)
	instance.enqueueAnalysis(context.Background(), publisher, state)
	<-started

	if err := instance.documents.Open(uri, "cpp", 2, "int second;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	state, _ = instance.documents.Get(uri)
	instance.enqueueAnalysis(context.Background(), publisher, state)
	if err := instance.documents.Open(uri, "cpp", 3, "int third;"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	state, _ = instance.documents.Get(uri)
	instance.enqueueAnalysis(context.Background(), publisher, state)
	close(release)

	select {
	case <-secondParse:
	case <-time.After(time.Second):
		t.Fatal("latest analysis was not parsed")
	}
	select {
	case publication := <-publisher.values:
		if publication.Version == nil || *publication.Version != 3 {
			t.Fatalf("published version = %+v", publication.Version)
		}
	case <-time.After(time.Second):
		t.Fatal("latest analysis was not published")
	}
	instance.analysisWG.Wait()
}
