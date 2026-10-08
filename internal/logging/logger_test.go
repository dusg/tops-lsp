package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"tops-lsp/internal/logging"
)

func TestDocumentIDIsStableAndDoesNotExposeURI(t *testing.T) {
	uri := "file:///home/user/private/sample.cpp"
	first := logging.DocumentID(uri)
	second := logging.DocumentID(uri)
	if first == "" || first != second || strings.Contains(first, "sample") || strings.Contains(first, "/") {
		t.Fatalf("document ID = %q", first)
	}
}

func TestNewWritesStructuredFields(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(&output, slog.LevelDebug)
	logger.Info("document_changed", "document_id", logging.DocumentID("file:///sample.cpp"), "document_version", 2)

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("log is not JSON: %v; output = %q", err, output.String())
	}
	if record["msg"] != "document_changed" || record["document_version"] != float64(2) {
		t.Fatalf("record = %+v", record)
	}
	if strings.Contains(output.String(), "file:///sample.cpp") {
		t.Fatalf("log contains raw URI: %s", output.String())
	}
}
