package transport_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"tops-lsp/internal/transport"
)

type chunkReader struct {
	reader *strings.Reader
	size   int
}

func (reader *chunkReader) Read(buffer []byte) (int, error) {
	if len(buffer) > reader.size {
		buffer = buffer[:reader.size]
	}
	return reader.reader.Read(buffer)
}

func TestReaderHandlesFragmentedFramesAndOptionalContentType(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	input := transport.Frame(body)
	input = append([]byte("Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n"), input...)

	reader := transport.NewReader(&chunkReader{reader: strings.NewReader(string(input)), size: 2})
	actual, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !bytes.Equal(actual, body) {
		t.Fatalf("body = %s, want %s", actual, body)
	}
}

func TestReaderHandlesMultipleFrames(t *testing.T) {
	first := []byte(`{"id":1}`)
	second := []byte(`{"id":2}`)
	reader := transport.NewReader(bytes.NewReader(append(transport.Frame(first), transport.Frame(second)...)))

	actualFirst, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("first ReadFrame() error = %v", err)
	}
	actualSecond, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("second ReadFrame() error = %v", err)
	}
	if !bytes.Equal(actualFirst, first) || !bytes.Equal(actualSecond, second) {
		t.Fatalf("frames = %s, %s", actualFirst, actualSecond)
	}
	if _, err := reader.ReadFrame(); err != io.EOF {
		t.Fatalf("third ReadFrame() error = %v, want EOF", err)
	}
}

func TestReaderRejectsInvalidHeadersAndTruncatedBody(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "missing length", input: "\r\n"},
		{name: "invalid length", input: "Content-Length: no\r\n\r\n"},
		{name: "duplicate length", input: "Content-Length: 1\r\nContent-Length: 1\r\n\r\na"},
		{name: "truncated body", input: "Content-Length: 3\r\n\r\na"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := transport.NewReader(strings.NewReader(testCase.input)).ReadFrame()
			if err == nil {
				t.Fatal("ReadFrame() error = nil")
			}
		})
	}
}

func TestReaderRejectsFramesAboveConfiguredLimit(t *testing.T) {
	input := "Content-Length: 5\r\n\r\nhello"
	_, err := transport.NewReaderWithMaxFrameSize(strings.NewReader(input), 4).ReadFrame()
	if !errors.Is(err, transport.ErrFrameTooLarge) {
		t.Fatalf("ReadFrame() error = %v", err)
	}
}

func TestWriterUsesUTF8ByteLength(t *testing.T) {
	body := []byte(`{"text":"中文"}`)
	var output bytes.Buffer
	if err := transport.NewWriter(&output).WriteFrame(body); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}
	if !bytes.Contains(output.Bytes(), []byte("Content-Length: 17\r\n\r\n")) {
		t.Fatalf("frame header = %q", output.String())
	}
	actual, err := transport.NewReader(&output).ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !bytes.Equal(actual, body) {
		t.Fatalf("body = %s, want %s", actual, body)
	}
}
