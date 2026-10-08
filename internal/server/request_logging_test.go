package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"tops-lsp/internal/logging"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestRequestCompletionLogContainsRequiredFields(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	message := protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"}
	var output bytes.Buffer
	instance.dispatchRequest(context.Background(), message, transport.NewWriter(&output))
	instance.waitGroup.Wait()

	var record map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte{'\n'}) {
		var candidate map[string]any
		if err := json.Unmarshal(line, &candidate); err != nil {
			t.Fatalf("log is not JSON: %v", err)
		}
		if candidate["event"] == "request_completed" {
			record = candidate
			break
		}
	}
	if record == nil {
		t.Fatalf("request_completed log missing: %s", logs.String())
	}
	for _, key := range []string{"timestamp", "level", "event", "session_id", "request_id", "method", "duration_ms", "result"} {
		if _, ok := record[key]; !ok {
			t.Errorf("log field %q missing from %+v", key, record)
		}
	}
	if record["method"] != "initialize" || record["result"] != "success" {
		t.Fatalf("request log = %+v", record)
	}
	if duration, ok := record["duration_ms"].(float64); !ok || duration < 0 || duration > float64((time.Hour).Milliseconds()) {
		t.Fatalf("duration_ms = %#v", record["duration_ms"])
	}
}
