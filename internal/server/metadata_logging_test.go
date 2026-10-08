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

func TestInitializeLogsClientAndWorkspaceSummary(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	processID := 42
	params, err := json.Marshal(protocol.InitializeParams{
		ProcessID:  &processID,
		ClientInfo: &protocol.ClientInfo{Name: "test-client", Version: "1.0"},
		WorkspaceFolders: []protocol.WorkspaceFolder{
			{URI: "file:///workspace/one", Name: "one"},
			{URI: "file:///workspace/two", Name: "two"},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), protocol.Message{
		Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize", Params: params,
	}, transport.NewWriter(&output))
	instance.waitGroup.Wait()
	logText := logs.String()
	for _, field := range []string{`"process_id":42`, `"client_name":"test-client"`, `"client_version":"1.0"`, `"workspace_count":2`} {
		if !strings.Contains(logText, field) {
			t.Errorf("initialize log field %s missing: %s", field, logText)
		}
	}
}
