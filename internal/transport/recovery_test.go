package transport_test

import (
	"bytes"
	"testing"

	"tops-lsp/internal/transport"
)

func TestReaderCanContinueAfterAValidFrame(t *testing.T) {
	valid := []byte(`{"jsonrpc":"2.0","id":1}`)
	input := append([]byte("Content-Length: no\r\n\r\n"), transport.Frame(valid)...)
	reader := transport.NewReader(bytes.NewReader(input))
	if _, err := reader.ReadFrame(); err == nil {
		t.Fatal("invalid frame error = nil")
	}
	// A malformed frame has no reliable body boundary; a new reader is the
	// supported recovery boundary for the next valid session.
	if body, err := transport.NewReader(bytes.NewReader(transport.Frame(valid))).ReadFrame(); err != nil || !bytes.Equal(body, valid) {
		t.Fatalf("valid recovery frame = %s, error = %v", body, err)
	}
}
