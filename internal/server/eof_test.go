package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/logging"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestRunEOFCancelsActiveRequestAndWaits(t *testing.T) {
	reader, input := io.Pipe()
	var output bytes.Buffer
	var logs bytes.Buffer
	instance := New(logging.New(&logs, slog.LevelDebug))
	started := make(chan struct{})
	finished := make(chan struct{})
	instance.registerHandler("test/block", func(ctx context.Context, _ protocol.Message) []byte {
		close(started)
		<-ctx.Done()
		close(finished)
		return nil
	})

	status := make(chan int, 1)
	go func() {
		status <- instance.Run(context.Background(), transport.NewReader(reader), transport.NewWriter(&output))
	}()
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "test/block"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := input.Write(transport.Frame(body)); err != nil {
		t.Fatalf("write request error = %v", err)
	}
	<-started
	if err := input.Close(); err != nil {
		t.Fatalf("close input error = %v", err)
	}
	if got := <-status; got != 1 {
		t.Fatalf("Run() status = %d, want 1", got)
	}
	select {
	case <-finished:
	default:
		t.Fatal("active request was not cancelled before Run returned")
	}
	if !strings.Contains(logs.String(), `"event":"stdin_eof"`) {
		t.Fatalf("stdin_eof log missing: %s", logs.String())
	}
}
