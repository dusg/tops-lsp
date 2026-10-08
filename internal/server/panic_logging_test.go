package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/logging"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestPanicDiagnosticStaysInControlledLogs(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	instance.registerHandler("test/panic", func(context.Context, protocol.Message) []byte {
		panic("controlled panic")
	})
	message := protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "test/panic"}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), message, transport.NewWriter(&output))
	instance.waitGroup.Wait()

	if !strings.Contains(logs.String(), `"event":"request_handler_panic"`) || !strings.Contains(logs.String(), "controlled panic") {
		t.Fatalf("panic diagnostic missing: %s", logs.String())
	}
	if strings.Contains(output.String(), "controlled panic") || strings.Contains(output.String(), "goroutine") {
		t.Fatalf("panic diagnostic leaked to response: %s", output.String())
	}
}
