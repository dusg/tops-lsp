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

func TestDecodeRejectsNonObjectJSONAsInvalidRequest(t *testing.T) {
	for _, body := range []string{`[]`, `1`, `"message"`, `null`} {
		_, err := protocol.DecodeMessage([]byte(body))
		var decodeErr *protocol.DecodeError
		if !errors.As(err, &decodeErr) || decodeErr.Code != protocol.InvalidRequest {
			t.Fatalf("body %s error = %v", body, err)
		}
	}
}

func TestDecodeParamsRequiresDocumentFields(t *testing.T) {
	tests := []struct {
		name   string
		params string
		decode func(string) error
	}{
		{
			name:   "open requires text",
			params: `{"textDocument":{"uri":"file:///sample.cpp","languageId":"cpp","version":1}}`,
			decode: func(params string) error {
				return protocol.DecodeParams(protocol.Message{Params: []byte(params)}, &protocol.DidOpenTextDocumentParams{})
			},
		},
		{
			name:   "open rejects negative version",
			params: `{"textDocument":{"uri":"file:///sample.cpp","languageId":"cpp","version":-1,"text":""}}`,
			decode: func(params string) error {
				return protocol.DecodeParams(protocol.Message{Params: []byte(params)}, &protocol.DidOpenTextDocumentParams{})
			},
		},
		{
			name:   "change requires content changes",
			params: `{"textDocument":{"uri":"file:///sample.cpp","version":2}}`,
			decode: func(params string) error {
				return protocol.DecodeParams(protocol.Message{Params: []byte(params)}, &protocol.DidChangeTextDocumentParams{})
			},
		},
		{
			name:   "change requires text",
			params: `{"textDocument":{"uri":"file:///sample.cpp","version":2},"contentChanges":[{}]}`,
			decode: func(params string) error {
				return protocol.DecodeParams(protocol.Message{Params: []byte(params)}, &protocol.DidChangeTextDocumentParams{})
			},
		},
		{
			name:   "close requires uri",
			params: `{"textDocument":{}}`,
			decode: func(params string) error {
				return protocol.DecodeParams(protocol.Message{Params: []byte(params)}, &protocol.DidCloseTextDocumentParams{})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.decode(test.params)
			if err == nil {
				t.Fatal("DecodeParams() error = nil")
			}
		})
	}
}

func TestDecodeParamsRejectsMissingRequiredParams(t *testing.T) {
	tests := []struct {
		name   string
		decode func() error
	}{
		{name: "open", decode: func() error {
			return protocol.DecodeParams(protocol.Message{}, &protocol.DidOpenTextDocumentParams{})
		}},
		{name: "change", decode: func() error {
			return protocol.DecodeParams(protocol.Message{}, &protocol.DidChangeTextDocumentParams{})
		}},
		{name: "close", decode: func() error {
			return protocol.DecodeParams(protocol.Message{}, &protocol.DidCloseTextDocumentParams{})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.decode()
			if err == nil {
				t.Fatal("DecodeParams() error = nil")
			}
		})
	}
}
