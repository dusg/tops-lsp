package server

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestCancelledDispatchEmitsOneCancellationResponse(t *testing.T) {
	instance := New(nil)
	started := make(chan struct{})
	instance.registerHandler("test/block", func(ctx context.Context, _ protocol.Message) []byte {
		close(started)
		<-ctx.Done()
		return nil
	})

	message := protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "test/block"}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), message, transport.NewWriter(&output))
	<-started
	cancelParams, err := json.Marshal(protocol.CancelRequestParams{ID: message.ID})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	instance.handleNotification(context.Background(), protocol.Message{
		Kind:    protocol.NotificationMessage,
		JSONRPC: "2.0",
		Method:  "$/cancelRequest",
		Params:  cancelParams,
	})
	instance.waitGroup.Wait()

	reader := transport.NewReader(bytes.NewReader(output.Bytes()))
	body, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	var response struct {
		ID    json.RawMessage       `json:"id"`
		Error *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if string(response.ID) != "1" || response.Error == nil || response.Error.Code != protocol.RequestCancelled {
		t.Fatalf("response = %+v", response)
	}
	if _, err := reader.ReadFrame(); err == nil {
		t.Fatal("cancelled request produced more than one response")
	}
}

func TestCancelledNavigationHandlerEmitsOnlyRequestCancelled(t *testing.T) {
	instance := NewWithOptions(nil, ServerOptions{NavigationGate: func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}})
	if !instance.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	uri := "file:///workspace/navigation-cancel.tops"
	state, err := instance.openDocument(uri, "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	instance.parseAndPublish(context.Background(), nil, state)
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: uri}, Position: protocol.Position{Line: 0, Character: 4}})
	if err != nil {
		t.Fatal(err)
	}
	message := protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "textDocument/definition", Params: params}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), message, transport.NewWriter(&output))
	if instance.navigationGate == nil {
		t.Fatal("navigation gate was not installed")
	}
	instance.handleCancel(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, JSONRPC: "2.0", Method: "$/cancelRequest", Params: json.RawMessage(`{"id":1}`)})
	instance.waitGroup.Wait()

	reader := transport.NewReader(bytes.NewReader(output.Bytes()))
	body, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	var response struct {
		Error *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Error == nil || response.Error.Code != protocol.RequestCancelled {
		t.Fatalf("navigation cancellation response = %+v", response)
	}
	if _, err := reader.ReadFrame(); err == nil {
		t.Fatal("cancelled navigation produced more than one response")
	}
}

func TestCancelUnknownOrCompletedRequestHasNoEffect(t *testing.T) {
	instance := New(nil)
	if instance.requests.Cancel(json.RawMessage(`99`)) {
		t.Fatal("unknown request was cancelled")
	}
	state := newRequestState(func() {})
	if !instance.requests.Add(json.RawMessage(`1`), state) {
		t.Fatal("Add() = false")
	}
	if cancelled, ok := state.Complete(); !ok || cancelled {
		t.Fatalf("Complete() = cancelled:%v ok:%v", cancelled, ok)
	}
	if instance.requests.Cancel(json.RawMessage(`1`)) {
		t.Fatal("completed request was cancelled")
	}
}

func TestPanickingHandlerProducesInternalError(t *testing.T) {
	instance := New(nil)
	instance.registerHandler("test/panic", func(context.Context, protocol.Message) []byte {
		panic("test panic")
	})
	message := protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "test/panic"}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), message, transport.NewWriter(&output))
	instance.waitGroup.Wait()

	body, err := transport.NewReader(bytes.NewReader(output.Bytes())).ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	var response struct {
		Error *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Error == nil || response.Error.Code != protocol.InternalError {
		t.Fatalf("response = %+v", response)
	}
}
