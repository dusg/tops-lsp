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
	"sync"
	"testing"
	"time"

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
	type readResult struct {
		body []byte
		err  error
	}
	result := make(chan readResult, 1)
	go func() {
		body, err := reader.ReadFrame()
		result <- readResult{body: body, err: err}
	}()
	select {
	case value := <-result:
		if value.err != nil {
			t.Fatalf("read LSP response error = %v", value.err)
		}
		var response map[string]json.RawMessage
		if err := json.Unmarshal(value.body, &response); err != nil {
			t.Fatalf("json.Unmarshal() error = %v; body = %s", err, value.body)
		}
		return response
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for LSP frame")
	}
	return nil
}

type serverProcess struct {
	command   *exec.Cmd
	stdin     io.WriteCloser
	stderr    io.ReadCloser
	reader    *transport.Reader
	closeOnce sync.Once
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
	process := &serverProcess{command: command, stdin: stdin, stderr: stderr, reader: transport.NewReader(stdout)}
	t.Cleanup(func() {
		process.close()
	})
	return process
}

func (process *serverProcess) finish(t *testing.T, wantFailure bool) []byte {
	t.Helper()
	_ = process.stdin.Close()
	waitResult := make(chan error, 1)
	stderrResult := make(chan struct {
		bytes []byte
		err   error
	}, 1)
	go func() {
		waitResult <- process.command.Wait()
	}()
	go func() {
		stderrBytes, readErr := io.ReadAll(process.stderr)
		stderrResult <- struct {
			bytes []byte
			err   error
		}{bytes: stderrBytes, err: readErr}
	}()
	var waitErr error
	select {
	case waitErr = <-waitResult:
	case <-time.After(5 * time.Second):
		if process.command.Process != nil {
			_ = process.command.Process.Kill()
		}
		waitErr = <-waitResult
		t.Fatalf("server process did not exit within timeout")
	}
	stderr := <-stderrResult
	if stderr.err != nil {
		t.Fatalf("read stderr error = %v", stderr.err)
	}
	process.close()
	if wantFailure && waitErr == nil {
		t.Fatal("server exited successfully, want failure")
	}
	if !wantFailure && waitErr != nil {
		t.Fatalf("server exit error = %v", waitErr)
	}
	return stderr.bytes
}

func (process *serverProcess) close() {
	process.closeOnce.Do(func() {
		_ = process.stdin.Close()
		if process.command.Process != nil && process.command.ProcessState == nil {
			_ = process.command.Process.Kill()
			_, _ = process.command.Process.Wait()
		}
		_ = process.stderr.Close()
	})
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
	openDiagnostics := readResponse(t, reader)
	if string(openDiagnostics["method"]) != "\"textDocument/publishDiagnostics\"" {
		t.Fatalf("didOpen diagnostics = %+v", openDiagnostics)
	}
	var openParams struct {
		Diagnostics []struct {
			Data *struct {
				ConditionalState string `json:"conditionalState"`
				Recoverable      bool   `json:"recoverable"`
				Incomplete       bool   `json:"incomplete"`
				ContextVersion   int    `json:"contextVersion"`
			} `json:"data"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(openDiagnostics["params"], &openParams); err != nil {
		t.Fatalf("open diagnostics params decode = %v", err)
	}
	if len(openParams.Diagnostics) != 1 || openParams.Diagnostics[0].Data == nil || !openParams.Diagnostics[0].Data.Recoverable || !openParams.Diagnostics[0].Data.Incomplete || openParams.Diagnostics[0].Data.ContextVersion != 1 {
		t.Fatalf("open diagnostic metadata = %+v", openParams)
	}
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
	changeDiagnostics := readResponse(t, reader)
	if string(changeDiagnostics["method"]) != "\"textDocument/publishDiagnostics\"" {
		t.Fatalf("didChange diagnostics = %+v", changeDiagnostics)
	}
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

func TestLSPProcessPublishesConditionalRecoveryMetadata(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	initialize := readResponse(t, reader)
	if initialize["error"] != nil {
		t.Fatalf("initialize response = %+v", initialize)
	}
	source := "#if MAYBE\nint main( { return 1; }\n#endif\n"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri": "file:///tmp/conditional.cpp", "languageId": "cpp", "version": 1, "text": source,
		}},
	})
	publication := readResponse(t, reader)
	var params struct {
		Version        *int `json:"version"`
		ContextVersion *int `json:"contextVersion"`
		Diagnostics    []struct {
			Code               string `json:"code"`
			Severity           int    `json:"severity"`
			Source             string `json:"source"`
			RelatedInformation []struct {
				Message string `json:"message"`
			} `json:"relatedInformation"`
			Data *struct {
				ConditionalState string `json:"conditionalState"`
				Recoverable      bool   `json:"recoverable"`
				Incomplete       bool   `json:"incomplete"`
				ContextVersion   int    `json:"contextVersion"`
			} `json:"data"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(publication["params"], &params); err != nil {
		t.Fatalf("diagnostic params decode = %v", err)
	}
	if params.Version == nil || *params.Version != 1 || params.ContextVersion == nil || *params.ContextVersion != 1 {
		t.Fatalf("diagnostic versions = %+v", params)
	}
	unknownCondition := false
	unknownRecovery := false
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Source != "tops-lsp" || diagnostic.Data == nil || diagnostic.Data.ContextVersion != 1 {
			t.Fatalf("diagnostic metadata = %+v", diagnostic)
		}
		if diagnostic.Code == "tops-syntax-unknown-condition" {
			unknownCondition = diagnostic.Severity == 3 && diagnostic.Data.ConditionalState == "unknown"
		}
		if diagnostic.Code == "tops-syntax-missing-token" && len(diagnostic.RelatedInformation) > 0 {
			unknownRecovery = diagnostic.Data.ConditionalState == "unknown" && diagnostic.Data.Recoverable && diagnostic.Data.Incomplete
		}
	}
	if !unknownCondition || !unknownRecovery {
		t.Fatalf("conditional recovery diagnostics = %+v", params.Diagnostics)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	shutdown := readResponse(t, reader)
	if shutdown["error"] != nil {
		t.Fatalf("shutdown response = %+v", shutdown)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
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
