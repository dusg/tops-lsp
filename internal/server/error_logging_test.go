package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/logging"
	"tops-lsp/internal/transport"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("sink closed")
}

func TestRunLogsResponseWriteFailureForParseError(t *testing.T) {
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	input := transport.Frame([]byte(`{"jsonrpc":`))
	status := instance.Run(context.Background(), transport.NewReader(bytes.NewReader(input)), transport.NewWriter(failingWriter{}))
	if status != 1 {
		t.Fatalf("Run() status = %d, want 1", status)
	}
	if !strings.Contains(logs.String(), `"event":"response_write_failed"`) {
		t.Fatalf("response write failure log missing: %s", logs.String())
	}
}
