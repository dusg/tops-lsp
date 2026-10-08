package protocol_test

import (
	"errors"
	"testing"

	"tops-lsp/internal/protocol"
)

func TestDecodeRequest(t *testing.T) {
	message, err := protocol.DecodeMessage([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"rootUri":"file:///workspace"}}`))
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if message.Kind != protocol.RequestMessage || message.Method != "initialize" || string(message.ID) != "1" {
		t.Fatalf("message = %+v", message)
	}
	if len(message.Params) == 0 {
		t.Fatal("request params were dropped")
	}
}

func TestDecodeNotification(t *testing.T) {
	message, err := protocol.DecodeMessage([]byte(`{"jsonrpc":"2.0","method":"exit"}`))
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if message.Kind != protocol.NotificationMessage || message.Method != "exit" || len(message.ID) != 0 {
		t.Fatalf("message = %+v", message)
	}
}

func TestDecodeErrorsDistinguishParseAndInvalidRequest(t *testing.T) {
	_, err := protocol.DecodeMessage([]byte(`{"jsonrpc":`))
	var decodeErr *protocol.DecodeError
	if !errors.As(err, &decodeErr) || decodeErr.Code != protocol.ParseError {
		t.Fatalf("parse error = %v", err)
	}

	_, err = protocol.DecodeMessage([]byte(`{"jsonrpc":"1.0","id":1,"method":"x"}`))
	if !errors.As(err, &decodeErr) || decodeErr.Code != protocol.InvalidRequest {
		t.Fatalf("invalid request error = %v", err)
	}
}

func TestDecodeRejectsInvalidRequestID(t *testing.T) {
	_, err := protocol.DecodeMessage([]byte(`{"jsonrpc":"2.0","id":null,"method":"x"}`))
	var decodeErr *protocol.DecodeError
	if !errors.As(err, &decodeErr) || decodeErr.Code != protocol.InvalidRequest {
		t.Fatalf("invalid ID error = %v", err)
	}
}
