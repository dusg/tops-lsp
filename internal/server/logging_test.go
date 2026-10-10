package server_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/logging"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/server"
)

func TestServerLogsDocumentEventsWithRedactedURI(t *testing.T) {
	var output bytes.Buffer
	instance := server.New(logging.New(&output, slog.LevelDebug))
	instance.Handle(context.Background(), requestWithRawID(t, "1", "initialize", nil))
	instance.Handle(context.Background(), notificationMessage(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI: "file:///private/sample.cpp", LanguageID: "cpp", Version: 1, Text: "secret-source",
		},
	}))

	if !strings.Contains(output.String(), `"event":"document_opened"`) || !strings.Contains(output.String(), `"session_id"`) {
		t.Fatalf("document event missing: %s", output.String())
	}
	if strings.Contains(output.String(), "file:///private/sample.cpp") || strings.Contains(output.String(), "secret-source") {
		t.Fatalf("log contains private document data: %s", output.String())
	}
}
