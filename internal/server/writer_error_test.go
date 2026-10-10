package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

type failingFrameWriter struct{}

func (failingFrameWriter) Write([]byte) (int, error) {
	return 0, errors.New("frame write failed")
}

func TestDuplicateRequestResponseWriteFailureIsFatal(t *testing.T) {
	instance := New(nil)
	active := newRequestState(func() {})
	if !instance.requests.Add(json.RawMessage(`1`), active) {
		t.Fatal("requests.Add() = false")
	}
	done := instance.dispatchRequest(context.Background(), protocol.Message{
		Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "test/duplicate",
	}, transport.NewWriter(failingFrameWriter{}))
	<-done
	select {
	case err := <-instance.fatalErrors:
		if err == nil || err.Error() != "frame write failed" {
			t.Fatalf("fatal error = %v", err)
		}
	default:
		t.Fatal("duplicate response write failure was not reported")
	}
}
