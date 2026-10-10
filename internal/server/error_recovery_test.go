package server_test

import (
	"context"
	"testing"

	"tops-lsp/internal/protocol"
	"tops-lsp/internal/server"
)

func TestUnknownRequestAndInvalidOrderReturnStableErrors(t *testing.T) {
	instance := server.New(nil)
	response, _ := instance.Handle(context.Background(), requestWithRawID(t, "1", "textDocument/hover", nil))
	beforeInitialize := decodeResponse(t, response)
	if beforeInitialize.Error == nil || beforeInitialize.Error.Code != protocol.ServerNotInitialized {
		t.Fatalf("before initialize response = %+v", beforeInitialize)
	}

	instance.Handle(context.Background(), requestWithRawID(t, "2", "initialize", nil))
	response, _ = instance.Handle(context.Background(), requestWithRawID(t, "3", "textDocument/hover", nil))
	unknown := decodeResponse(t, response)
	if unknown.Error == nil || unknown.Error.Code != protocol.MethodNotFound {
		t.Fatalf("unknown request response = %+v", unknown)
	}
}

func TestInvalidNotificationDoesNotProduceResponse(t *testing.T) {
	instance := server.New(nil)
	instance.Handle(context.Background(), requestWithRawID(t, "1", "initialize", nil))
	response, exit := instance.Handle(context.Background(), notificationMessage(t, "textDocument/didOpen", "invalid"))
	if response != nil || exit {
		t.Fatalf("invalid notification response = %s exit = %v", response, exit)
	}
}
