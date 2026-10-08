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
