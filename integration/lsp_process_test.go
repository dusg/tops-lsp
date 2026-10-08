package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tops-lsp/internal/transport"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func goBinary(t *testing.T) string {
	t.Helper()
	if configured := os.Getenv("GO"); configured != "" {
		return configured
	}
	if configured, err := exec.LookPath("go"); err == nil {
		return configured
	}
	return "/home/carl.du/sdk/go1.26.8/bin/go"
}

func writeJSON(t *testing.T, writer io.Writer, value any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := writer.Write(transport.Frame(body)); err != nil {
		t.Fatalf("write LSP frame error = %v", err)
	}
}

func readResponse(t *testing.T, reader *transport.Reader) map[string]json.RawMessage {
	t.Helper()
	body, err := reader.ReadFrame()
	if err != nil {
		t.Fatalf("read LSP response error = %v", err)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, body)
	}
	return response
}

type serverProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	stderr  io.ReadCloser
	reader  *transport.Reader
}

func startServer(t *testing.T) *serverProcess {
	t.Helper()
	command := exec.Command(goBinary(t), "run", "./cmd/tops-lsp")
	command.Dir = repositoryRoot(t)
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe() error = %v", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		t.Fatalf("StderrPipe() error = %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start server error = %v", err)
	}
	return &serverProcess{command: command, stdin: stdin, stderr: stderr, reader: transport.NewReader(stdout)}
}

func (process *serverProcess) finish(t *testing.T, wantFailure bool) []byte {
	t.Helper()
	if err := process.stdin.Close(); err != nil {
		t.Fatalf("close stdin error = %v", err)
	}
	stderrBytes, err := io.ReadAll(process.stderr)
	if err != nil {
		t.Fatalf("read stderr error = %v", err)
	}
	waitErr := process.command.Wait()
	if wantFailure && waitErr == nil {
		t.Fatal("server exited successfully, want failure")
	}
	if !wantFailure && waitErr != nil {
		t.Fatalf("server exit error = %v", waitErr)
	}
	return stderrBytes
}

func TestLSPProcessLifecycleAndDocumentSync(t *testing.T) {
	process := startServer(t)
	reader := process.reader

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	initialize := readResponse(t, reader)
	if string(initialize["id"]) != "1" || initialize["error"] != nil {
		t.Fatalf("initialize response = %+v", initialize)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": "file:///tmp/tops-lsp-integration.cpp", "languageId": "cpp", "version": 1, "text": "hello",
			},
		},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///tmp/tops-lsp-integration.cpp", "version": 2},
			"contentChanges": []any{map[string]any{
				"range": map[string]any{
					"start": map[string]any{"line": 0, "character": 0},
					"end":   map[string]any{"line": 0, "character": 5},
				},
				"text": "world",
			}},
		},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didClose",
		"params":  map[string]any{"textDocument": map[string]any{"uri": "file:///tmp/tops-lsp-integration.cpp"}},
	})

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	shutdown := readResponse(t, reader)
	if string(shutdown["id"]) != "2" || string(shutdown["result"]) != "null" {
		t.Fatalf("shutdown response = %+v", shutdown)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	stderrBytes := process.finish(t, false)
	if bytes.Contains(stderrBytes, []byte("file:///tmp/tops-lsp-integration.cpp")) {
		t.Fatalf("stderr contains raw URI: %s", stderrBytes)
	}
	if !bytes.Contains(stderrBytes, []byte(`"event":"transport_ready"`)) || !bytes.Contains(stderrBytes, []byte("document_opened")) || !bytes.Contains(stderrBytes, []byte("shutdown_requested")) {
		t.Fatalf("stderr lacks lifecycle logs: %s", stderrBytes)
	}
	if strings.Contains(string(stderrBytes), "\"text\":\"hello\"") {
		t.Fatalf("stderr contains document payload: %s", stderrBytes)
	}
}

func TestLSPProcessRejectsUnknownRequestAndAbnormalExit(t *testing.T) {
	process := startServer(t)
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "textDocument/hover"})
	response := readResponse(t, process.reader)
	var responseError struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response["error"], &responseError); err != nil {
		t.Fatalf("error response decode = %v", err)
	}
	if responseError.Code != -32002 {
		t.Fatalf("error code = %d", responseError.Code)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "$/cancelRequest", "params": map[string]any{"id": 99}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, true)
}

func TestLSPProcessRecoversFromMalformedJSON(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	if _, err := process.stdin.Write(transport.Frame([]byte(`{"jsonrpc":`))); err != nil {
		t.Fatalf("write malformed frame error = %v", err)
	}
	parseError := readResponse(t, reader)
	if string(parseError["id"]) != "null" {
		t.Fatalf("parse error ID = %s", parseError["id"])
	}
	var errorBody struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(parseError["error"], &errorBody); err != nil {
		t.Fatalf("parse error decode = %v", err)
	}
	if errorBody.Code != -32700 {
		t.Fatalf("parse error code = %d", errorBody.Code)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "initialize", "params": map[string]any{}})
	initialize := readResponse(t, reader)
	if string(initialize["id"]) != "2" || initialize["error"] != nil {
		t.Fatalf("initialize after parse error = %+v", initialize)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "shutdown"})
	shutdown := readResponse(t, reader)
	if string(shutdown["id"]) != "3" || shutdown["error"] != nil {
		t.Fatalf("shutdown after parse error = %+v", shutdown)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	stderrBytes := process.finish(t, false)
	if !bytes.Contains(stderrBytes, []byte(`"event":"message_rejected"`)) {
		t.Fatalf("parse rejection log missing: %s", stderrBytes)
	}
	if !bytes.Contains(stderrBytes, []byte(`"body_length":11`)) {
		t.Fatalf("parse rejection body length missing: %s", stderrBytes)
	}
}
