package server

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"tops-lsp/internal/transport"
)

type closeTrackingReader struct {
	reader io.Reader
	closed chan struct{}
}

type blockingReader struct {
	started chan struct{}
	block   chan struct{}
}

func (reader *blockingReader) Read([]byte) (int, error) {
	select {
	case <-reader.started:
	default:
		close(reader.started)
	}
	<-reader.block
	return 0, io.EOF
}

func (reader *closeTrackingReader) Read(buffer []byte) (int, error) {
	return reader.reader.Read(buffer)
}

func (reader *closeTrackingReader) Close() error {
	select {
	case <-reader.closed:
	default:
		close(reader.closed)
	}
	return nil
}

func TestRunClosesReaderOnEOF(t *testing.T) {
	underlying := &closeTrackingReader{reader: strings.NewReader(""), closed: make(chan struct{})}
	status := make(chan int, 1)
	go func() {
		status <- New(nil).Run(context.Background(), transport.NewReader(underlying), transport.NewWriter(io.Discard))
	}()
	select {
	case got := <-status:
		if got != 1 {
			t.Fatalf("Run() status = %d, want 1", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return on EOF")
	}
	select {
	case <-underlying.closed:
	case <-time.After(time.Second):
		t.Fatal("Run() did not close the reader on EOF")
	}
}

func TestRunContextCancellationReturnsWithReaderWithoutCloser(t *testing.T) {
	underlying := &blockingReader{started: make(chan struct{}), block: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	status := make(chan int, 1)
	go func() {
		status <- New(nil).Run(ctx, transport.NewReader(underlying), transport.NewWriter(io.Discard))
	}()
	<-underlying.started
	cancel()
	select {
	case got := <-status:
		if got != 1 {
			t.Fatalf("Run() status = %d, want 1", got)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not return after context cancellation without a reader closer")
	}
	close(underlying.block)
}
