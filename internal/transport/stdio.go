package transport

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const DefaultMaxFrameSize = 16 << 20

var ErrFrameTooLarge = errors.New("LSP frame exceeds maximum size")

type Reader struct {
	reader       *bufio.Reader
	closer       io.Closer
	closeOnce    sync.Once
	closeErr     error
	maxFrameSize int
}

func NewReader(reader io.Reader) *Reader {
	return NewReaderWithMaxFrameSize(reader, DefaultMaxFrameSize)
}

func NewReaderWithMaxFrameSize(reader io.Reader, maxFrameSize int) *Reader {
	if maxFrameSize <= 0 {
		maxFrameSize = DefaultMaxFrameSize
	}
	result := &Reader{reader: bufio.NewReader(reader), maxFrameSize: maxFrameSize}
	if closer, ok := reader.(io.Closer); ok {
		result.closer = closer
	}
	return result
}

func (reader *Reader) Close() error {
	reader.closeOnce.Do(func() {
		if reader.closer != nil {
			reader.closeErr = reader.closer.Close()
		}
	})
	return reader.closeErr
}

func (reader *Reader) ReadFrame() ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) == 0 && contentLength < 0 {
				return nil, io.EOF
			}
			if err == io.EOF {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			if contentLength < 0 {
				return nil, fmt.Errorf("missing Content-Length header")
			}
			body := make([]byte, contentLength)
			if _, err := io.ReadFull(reader.reader, body); err != nil {
				if err == io.EOF {
					return nil, io.ErrUnexpectedEOF
				}
				return nil, err
			}
			return body, nil
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("malformed header %q", line)
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		switch name {
		case "content-length":
			if contentLength >= 0 {
				return nil, fmt.Errorf("duplicate Content-Length header")
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return nil, fmt.Errorf("invalid Content-Length %q", value)
			}
			if parsed > reader.maxFrameSize {
				return nil, ErrFrameTooLarge
			}
			contentLength = parsed
		case "content-type":
			continue
		default:
			continue
		}
	}
}

type Writer struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{writer: writer}
}

func (writer *Writer) WriteFrame(body []byte) error {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if err := writeAll(writer.writer, []byte(header)); err != nil {
		return err
	}
	return writeAll(writer.writer, body)
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func Frame(body []byte) []byte {
	var buffer bytes.Buffer
	_ = NewWriter(&buffer).WriteFrame(body)
	return buffer.Bytes()
}
