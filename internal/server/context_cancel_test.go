package server

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"tops-lsp/internal/transport"
)

func TestRunContextCancellationClosesBlockedReader(t *testing.T) {
	reader, input := io.Pipe()
	instance := New(nil)
	result := make(chan int, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		result <- instance.Run(ctx, transport.NewReader(reader), transport.NewWriter(&bytes.Buffer{}))
	}()
	cancel()
	select {
	case status := <-result:
		if status != 1 {
			t.Fatalf("Run() status = %d, want 1", status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
	_ = input.Close()
}
