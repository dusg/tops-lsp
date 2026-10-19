package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"tops-lsp/internal/protocol"
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

type typedLSPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type typedLSPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *typedLSPError  `json:"error"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func readTypedResponse(t *testing.T, reader *transport.Reader) typedLSPResponse {
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
			t.Fatalf("read typed LSP response error = %v", value.err)
		}
		var response typedLSPResponse
		if err := json.Unmarshal(value.body, &response); err != nil {
			t.Fatalf("decode typed LSP response = %v; body = %s", err, value.body)
		}
		if response.JSONRPC != "2.0" {
			t.Fatalf("jsonrpc = %q", response.JSONRPC)
		}
		return response
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for typed LSP frame")
	}
	return typedLSPResponse{}
}

func readTypedUntilID(t *testing.T, reader *transport.Reader, id string) typedLSPResponse {
	t.Helper()
	for {
		response := readTypedResponse(t, reader)
		if typedIDMatches(response.ID, id) {
			return response
		}
	}
}

func typedIDMatches(raw json.RawMessage, id string) bool {
	return string(raw) == id || string(raw) == `"`+id+`"`
}

func readTypedUntilMethod(t *testing.T, reader *transport.Reader, method string) typedLSPResponse {
	t.Helper()
	for {
		response := readTypedResponse(t, reader)
		if response.Method == method {
			return response
		}
	}
}

func readUntilID(t *testing.T, reader *transport.Reader, id string) map[string]json.RawMessage {
	t.Helper()
	for {
		response := readResponse(t, reader)
		if string(response["id"]) == id {
			return response
		}
	}
}

func readIDs(t *testing.T, reader *transport.Reader, ids ...string) map[string]map[string]json.RawMessage {
	t.Helper()
	wanted := make(map[string]struct{}, len(ids))
	responses := make(map[string]map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	for len(responses) < len(wanted) {
		response := readResponse(t, reader)
		id := string(response["id"])
		if _, ok := wanted[id]; ok {
			responses[id] = response
		}
	}
	return responses
}

func readUntilMethod(t *testing.T, reader *transport.Reader, method string) map[string]json.RawMessage {
	t.Helper()
	for {
		response := readResponse(t, reader)
		if string(response["method"]) == `"`+method+`"` {
			return response
		}
	}
}

func readUntilDiagnosticsURI(t *testing.T, reader *transport.Reader, uri string) map[string]json.RawMessage {
	t.Helper()
	for {
		response := readResponse(t, reader)
		if string(response["method"]) != `"textDocument/publishDiagnostics"` {
			continue
		}
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(response["params"], &params); err != nil {
			t.Fatalf("diagnostic params decode = %v", err)
		}
		if params.URI == uri {
			return response
		}
	}
}

func readUntilDiagnosticsVersions(t *testing.T, reader *transport.Reader, wanted map[string]int) {
	t.Helper()
	remaining := make(map[string]int, len(wanted))
	for uri, version := range wanted {
		remaining[uri] = version
	}
	for len(remaining) > 0 {
		response := readResponse(t, reader)
		if string(response["method"]) != `"textDocument/publishDiagnostics"` {
			continue
		}
		var params struct {
			URI     string `json:"uri"`
			Version *int   `json:"version"`
		}
		if err := json.Unmarshal(response["params"], &params); err != nil {
			t.Fatalf("diagnostic params decode = %v", err)
		}
		if expected, ok := remaining[params.URI]; ok && params.Version != nil && *params.Version == expected {
			delete(remaining, params.URI)
		}
	}
}

func readDefinitionUntilLocation(t *testing.T, writer io.Writer, reader *transport.Reader, uri, source string, offset int, expectedURI string) []protocol.Location {
	t.Helper()
	for attempt := 0; attempt < 8; attempt++ {
		id := 7 + attempt*10
		writeJSON(t, writer, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  "textDocument/definition",
			"params": map[string]any{
				"textDocument": map[string]any{"uri": uri},
				"position":     sourceLSPPosition(source, offset),
			},
		})
		response := readUntilID(t, reader, strconv.Itoa(id))
		var locations []protocol.Location
		if err := json.Unmarshal(response["result"], &locations); err != nil {
			t.Fatalf("definition response decode = %v; response = %+v", err, response)
		}
		if len(locations) == 1 && locations[0].URI == expectedURI {
			return locations
		}
		if attempt < 7 {
			readUntilDiagnosticsURI(t, reader, uri)
		}
	}
	t.Fatalf("definition did not become available for %q", uri)
	return nil
}

func utf16Units(value string) int {
	units := 0
	for _, character := range value {
		if character > 0xffff {
			units += 2
			continue
		}
		units++
	}
	return units
}

func sourceLSPPosition(source string, offset int) protocol.Position {
	line := strings.Count(source[:offset], "\n")
	lineStart := strings.LastIndex(source[:offset], "\n") + 1
	return protocol.Position{Line: line, Character: utf16Units(source[lineStart:offset])}
}

func lspPositionBefore(leftLine, leftCharacter, rightLine, rightCharacter int) bool {
	return leftLine < rightLine || (leftLine == rightLine && leftCharacter < rightCharacter)
}

type serverProcess struct {
	command    *exec.Cmd
	stdin      io.WriteCloser
	stderr     io.ReadCloser
	reader     *transport.Reader
	stderrBuf  bytes.Buffer
	stderrDone chan struct{}
	closeOnce  sync.Once
}

func startServer(t *testing.T) *serverProcess {
	return startServerWithEnv(t, nil)
}

func startServerWithEnv(t *testing.T, environment map[string]string) *serverProcess {
	t.Helper()
	command := exec.Command(goBinary(t), "run", "./cmd/tops-lsp")
	command.Dir = repositoryRoot(t)
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	for key, value := range environment {
		command.Env = append(command.Env, key+"="+value)
	}
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
	process := &serverProcess{command: command, stdin: stdin, stderr: stderr, reader: transport.NewReader(stdout), stderrDone: make(chan struct{})}
	go func() {
		_, _ = io.Copy(&process.stderrBuf, stderr)
		close(process.stderrDone)
	}()
	t.Cleanup(func() {
		process.close()
	})
	return process
}

func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func fullTextChangeParams(uri string, version int, oldText, newText string) map[string]any {
	end := sourceLSPPosition(oldText, len(oldText))
	return map[string]any{
		"textDocument": map[string]any{"uri": uri, "version": version},
		"contentChanges": []map[string]any{{
			"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": end.Line, "character": end.Character}},
			"text":  newText,
		}},
	}
}

func (process *serverProcess) finish(t *testing.T, wantFailure bool) []byte {
	t.Helper()
	_ = process.stdin.Close()
	waitResult := make(chan error, 1)
	go func() {
		waitResult <- process.command.Wait()
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
	<-process.stderrDone
	stderr := append([]byte(nil), process.stderrBuf.Bytes()...)
	process.close()
	if wantFailure && waitErr == nil {
		t.Fatal("server exited successfully, want failure")
	}
	if !wantFailure && waitErr != nil {
		t.Fatalf("server exit error = %v", waitErr)
	}
	return stderr
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
	foundRecovery := false
	for _, diagnostic := range openParams.Diagnostics {
		if diagnostic.Data != nil && diagnostic.Data.Recoverable && diagnostic.Data.Incomplete && diagnostic.Data.ContextVersion == 1 {
			foundRecovery = true
		}
	}
	if !foundRecovery {
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
	updatedDiagnostics := readResponse(t, reader)
	if string(updatedDiagnostics["method"]) != "\"textDocument/publishDiagnostics\"" {
		t.Fatalf("updated didChange diagnostics = %+v", updatedDiagnostics)
	}
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didClose",
		"params":  map[string]any{"textDocument": map[string]any{"uri": "file:///tmp/tops-lsp-integration.cpp"}},
	})

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	shutdown := readUntilID(t, reader, "2")
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

func TestLSPProcessPublishesSemanticContextMetadata(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if response := readResponse(t, reader); response["error"] != nil {
		t.Fatalf("initialize response = %+v", response)
	}
	uri := "file:///tmp/semantic-context.tops"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri": uri, "languageId": "tops", "version": 1, "text": "int use() { return missing_name; }",
		}},
	})
	publication := readUntilDiagnosticsURI(t, reader, uri)
	var params struct {
		Diagnostics []struct {
			Code string `json:"code"`
			Data *struct {
				ProfileID     string `json:"profileId"`
				ContextStatus string `json:"contextStatus"`
				PassKind      string `json:"passKind"`
			} `json:"data"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(publication["params"], &params); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Code != "tops-semantic-context-degraded" || diagnostic.Data == nil {
			continue
		}
		found = diagnostic.Data.ProfileID == "gcu400-default" && diagnostic.Data.ContextStatus == "partial" && diagnostic.Data.PassKind == "not-applicable"
	}
	if !found {
		t.Fatalf("semantic diagnostics = %+v", params.Diagnostics)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	if response := readResponse(t, reader); response["error"] != nil {
		t.Fatalf("shutdown response = %+v", response)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessPublishesHostDeviceBoundaryDiagnostic(t *testing.T) {
	process := startServerWithEnv(t, map[string]string{"TOPS_LSP_TARGET_PROFILE": "gcu400", "TOPS_LSP_PASS_KIND": "host+device"})
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if response := readResponse(t, reader); response["error"] != nil {
		t.Fatalf("initialize response = %+v", response)
	}
	uri := "file:///tmp/host-device-boundary.tops"
	source := "__device__ int device_value() { return 1; }\nint host_call() { return device_value(); }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "tops", "version": 1, "text": source}}})
	publication := readUntilDiagnosticsURI(t, reader, uri)
	var params struct {
		Diagnostics []struct {
			Code string `json:"code"`
			Data *struct {
				PassKind string `json:"passKind"`
			} `json:"data"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(publication["params"], &params); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Code == "tops-semantic-host-device-boundary" && diagnostic.Data != nil && diagnostic.Data.PassKind == "host" {
			found = true
		}
	}
	if !found {
		t.Fatalf("host/device diagnostics = %+v", params.Diagnostics)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"})
	if response := readResponse(t, reader); response["error"] != nil {
		t.Fatalf("shutdown response = %+v", response)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
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
	for {
		var codes struct {
			Diagnostics []struct {
				Code string `json:"code"`
			} `json:"diagnostics"`
		}
		if err := json.Unmarshal(publication["params"], &codes); err != nil {
			t.Fatalf("diagnostic code decode = %v", err)
		}
		parserDiagnostic := false
		for _, diagnostic := range codes.Diagnostics {
			if diagnostic.Code == "tops-syntax-unknown-condition" || diagnostic.Code == "tops-syntax-missing-token" {
				parserDiagnostic = true
				break
			}
		}
		if parserDiagnostic {
			break
		}
		publication = readResponse(t, reader)
	}
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

func TestLSPProcessTypedNavigationContract(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	initialize := readTypedUntilID(t, reader, "1")
	if initialize.Error != nil {
		t.Fatalf("initialize error = %+v", initialize.Error)
	}
	var capabilities struct {
		DefinitionProvider     bool `json:"definitionProvider"`
		ReferencesProvider     bool `json:"referencesProvider"`
		DocumentSymbolProvider bool `json:"documentSymbolProvider"`
	}
	var initializeResult struct {
		Capabilities struct {
			DefinitionProvider     bool `json:"definitionProvider"`
			ReferencesProvider     bool `json:"referencesProvider"`
			DocumentSymbolProvider bool `json:"documentSymbolProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(initialize.Result, &initializeResult); err != nil {
		t.Fatal(err)
	}
	capabilities = initializeResult.Capabilities
	if !capabilities.DefinitionProvider || !capabilities.ReferencesProvider || !capabilities.DocumentSymbolProvider {
		t.Fatalf("navigation capabilities = %+v", capabilities)
	}

	headerURI := "file:///tmp/typed-navigation/api.h"
	headerText := "int api_value;\r\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": headerURI, "languageId": "tops-header", "version": 1, "text": headerText}}})
	readTypedUntilMethod(t, reader, "textDocument/publishDiagnostics")
	mainURI := "file:///tmp/typed-navigation/main.tops"
	mainText := "#include \"api.h\"\r\nconst char* text = \"中文\"; int use() { int local_value; return api_value; }\r\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "tops", "version": 1, "text": mainText}}})
	readTypedUntilMethod(t, reader, "textDocument/publishDiagnostics")

	apiOffset := strings.Index(mainText, "api_value")
	apiPosition := sourceLSPPosition(mainText, apiOffset)
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": apiPosition}})
	definition := readTypedUntilID(t, reader, "2")
	var definitions []protocol.Location
	if err := json.Unmarshal(definition.Result, &definitions); err != nil || len(definitions) != 1 {
		t.Fatalf("definition = %+v, err = %v", definition, err)
	}
	if definitions[0].URI != headerURI || definitions[0].Range.Start != (protocol.Position{}) || definitions[0].Range.End != (protocol.Position{Line: 0, Character: utf16Units(strings.TrimSuffix(headerText, "\r\n"))}) {
		t.Fatalf("definition range = %+v", definitions[0])
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": apiPosition, "context": map[string]any{"includeDeclaration": true}}})
	references := readTypedUntilID(t, reader, "3")
	var referenceLocations []protocol.Location
	if err := json.Unmarshal(references.Result, &referenceLocations); err != nil || len(referenceLocations) != 2 {
		t.Fatalf("references = %+v, err = %v", references, err)
	}
	foundHeader := false
	foundMain := false
	for _, location := range referenceLocations {
		if location.URI == headerURI && location.Range.Start == (protocol.Position{}) && location.Range.End.Character == 14 {
			foundHeader = true
		}
		if location.URI == mainURI && location.Range.Start == apiPosition && location.Range.End == (protocol.Position{Line: apiPosition.Line, Character: apiPosition.Character + len("api_value")}) {
			foundMain = true
		}
	}
	if !foundHeader || !foundMain {
		t.Fatalf("reference locations = %+v", referenceLocations)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}}})
	documentSymbols := readTypedUntilID(t, reader, "4")
	var symbols []protocol.DocumentSymbol
	if err := json.Unmarshal(documentSymbols.Result, &symbols); err != nil {
		t.Fatalf("document symbols = %+v, err = %v", documentSymbols, err)
	}
	useOffset := strings.Index(mainText, "use")
	localOffset := strings.Index(mainText, "local_value")
	usePosition := sourceLSPPosition(mainText, useOffset)
	localPosition := sourceLSPPosition(mainText, localOffset)
	foundUse := false
	foundLocal := false
	for _, symbol := range symbols {
		if symbol.Name != "use" {
			continue
		}
		foundUse = true
		if symbol.SelectionRange.Start != usePosition || symbol.SelectionRange.End != (protocol.Position{Line: usePosition.Line, Character: usePosition.Character + len("use")}) {
			t.Fatalf("use selectionRange = %+v", symbol.SelectionRange)
		}
		for _, child := range symbol.Children {
			if child.Name == "local_value" {
				foundLocal = true
				if child.SelectionRange.Start != localPosition || child.SelectionRange.End != (protocol.Position{Line: localPosition.Line, Character: localPosition.Character + len("local_value")}) {
					t.Fatalf("local selectionRange = %+v", child.SelectionRange)
				}
			}
		}
	}
	if !foundUse || !foundLocal {
		t.Fatalf("typed document symbols = %+v", symbols)
	}

	for id, params := range map[string]map[string]any{
		"5": {"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": 0}},
		"6": {"textDocument": map[string]any{}, "position": map[string]any{"line": 0, "character": 0}},
		"8": {"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": 99, "character": 0}},
	} {
		writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": id, "method": "textDocument/definition", "params": params})
		response := readTypedUntilID(t, reader, id)
		if response.Error == nil || response.Error.Code != -32602 {
			t.Fatalf("malformed/invalid response %s = %+v", id, response)
		}
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 7, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": "file:///tmp/typed-navigation/missing.tops"}, "position": map[string]any{"line": 0, "character": 0}}})
	missing := readTypedUntilID(t, reader, "7")
	if missing.Error != nil || string(missing.Result) != "null" {
		t.Fatalf("unopened document response = %+v", missing)
	}

	mainLines := strings.Split(strings.TrimSuffix(mainText, "\r\n"), "\r\n")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "version": 2}, "contentChanges": []map[string]any{{"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": len(mainLines) - 1, "character": utf16Units(mainLines[len(mainLines)-1])}}, "text": "int replacement;\r\n"}}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 9, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": 0, "character": 4}}})
	stale := readTypedUntilID(t, reader, "9")
	if strings.Contains(string(stale.Result), headerURI) {
		t.Fatalf("stale typed result returned old header location = %s", stale.Result)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 10, "method": "shutdown"})
	if response := readTypedUntilID(t, reader, "10"); response.Error != nil {
		t.Fatalf("shutdown response = %+v", response)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessConfiguredIncludeRootLoadsHeaderWithoutOpeningIt(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int configured_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headerURI := fileURI(headerPath)
	process := startServerWithEnv(t, map[string]string{"TOPS_LSP_INCLUDE_ROOTS": root})
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if response := readUntilID(t, reader, "1"); response["error"] != nil {
		t.Fatalf("initialize response = %+v", response)
	}
	mainURI := fileURI(filepath.Join(root, "main.tops"))
	mainText := "#include \"api.h\"\nint use() { return configured_value; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "tops", "version": 1, "text": mainText}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	offset := strings.LastIndex(mainText, "configured_value")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": sourceLSPPosition(mainText, offset)}})
	definition := readUntilID(t, reader, "2")
	var locations []protocol.Location
	if err := json.Unmarshal(definition["result"], &locations); err != nil || len(locations) != 1 || locations[0].URI != headerURI {
		t.Fatalf("configured include-root definition = %+v, err = %v", definition, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "shutdown"})
	readUntilID(t, reader, "3")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessNavigationConditionBuiltinAndHeaderChangeMatrix(t *testing.T) {
	root := t.TempDir()
	builtinPath := filepath.Join(root, "builtins.h")
	if err := os.WriteFile(builtinPath, []byte("int threadIdx;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	builtinURI := fileURI(builtinPath)
	process := startServerWithEnv(t, map[string]string{
		"TOPS_LSP_INCLUDE_ROOTS":        root,
		"TOPS_LSP_VERIFIED_HEADER_URIS": builtinURI,
		"TOPS_LSP_TARGET_PROFILE":       "gcu400",
		"TOPS_LSP_PASS_KIND":            "device",
		"TOPS_LSP_PREDEFINED_MACROS":    "__GCU_ARCH__",
	})
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if response := readUntilID(t, reader, "1"); response["error"] != nil {
		t.Fatalf("initialize response = %+v", response)
	}

	unknownURI := fileURI(filepath.Join(root, "unknown.tops"))
	unknownText := "#if UNKNOWN_MACRO\nint unknown_only;\n#endif\nint use_unknown() { return unknown_only; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": unknownURI, "languageId": "tops", "version": 1, "text": unknownText}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	unknownOffset := strings.LastIndex(unknownText, "unknown_only")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": unknownURI}, "position": sourceLSPPosition(unknownText, unknownOffset)}})
	unknownDefinition := readUntilID(t, reader, "2")
	var unknownLocations []protocol.Location
	if err := json.Unmarshal(unknownDefinition["result"], &unknownLocations); err != nil || len(unknownLocations) != 1 || unknownLocations[0].URI != unknownURI {
		t.Fatalf("unknown-only definition = %+v, err = %v", unknownDefinition, err)
	}

	activeUnknownURI := fileURI(filepath.Join(root, "active-unknown.tops"))
	activeUnknownText := "#if 1\nint candidate(int value);\n#endif\n#if UNKNOWN_MACRO\nint candidate(double value);\n#endif\nint use_candidate() { return candidate(1); }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": activeUnknownURI, "languageId": "tops", "version": 1, "text": activeUnknownText}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	activeOffset := strings.LastIndex(activeUnknownText, "candidate")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": activeUnknownURI}, "position": sourceLSPPosition(activeUnknownText, activeOffset)}})
	activeDefinition := readUntilID(t, reader, "3")
	var activeLocations []protocol.Location
	if err := json.Unmarshal(activeDefinition["result"], &activeLocations); err != nil || len(activeLocations) != 1 || activeLocations[0].URI != activeUnknownURI || activeLocations[0].Range.Start.Line != 1 {
		t.Fatalf("active+unknown definition = %+v, err = %v", activeDefinition, err)
	}

	builtinURIForMain := fileURI(filepath.Join(root, "builtin-main.tops"))
	builtinText := "#include \"builtins.h\"\nint use_builtin() { return threadIdx; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": builtinURIForMain, "languageId": "tops", "version": 1, "text": builtinText}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	builtinOffset := strings.LastIndex(builtinText, "threadIdx")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": builtinURIForMain}, "position": sourceLSPPosition(builtinText, builtinOffset)}})
	builtinDefinition := readUntilID(t, reader, "4")
	var builtinLocations []protocol.Location
	if err := json.Unmarshal(builtinDefinition["result"], &builtinLocations); err != nil || len(builtinLocations) != 1 || builtinLocations[0].URI != builtinURI {
		t.Fatalf("verified builtin definition = %+v, err = %v", builtinDefinition, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": builtinURIForMain}, "position": sourceLSPPosition(builtinText, builtinOffset), "context": map[string]any{"includeDeclaration": false}}})
	builtinReferences := readUntilID(t, reader, "5")
	var builtinReferenceLocations []protocol.Location
	if err := json.Unmarshal(builtinReferences["result"], &builtinReferenceLocations); err != nil || len(builtinReferenceLocations) != 1 || builtinReferenceLocations[0].URI != builtinURIForMain {
		t.Fatalf("verified builtin references without declaration = %+v, err = %v", builtinReferences, err)
	}

	apiPath := filepath.Join(root, "api.h")
	oldHeader := "int old_value;\n"
	newHeader := "int new_value;\n"
	if err := os.WriteFile(apiPath, []byte(oldHeader), 0o644); err != nil {
		t.Fatal(err)
	}
	apiURI := fileURI(apiPath)
	apiMainURI := fileURI(filepath.Join(root, "api-main.tops"))
	oldMain := "#include \"api.h\"\nint use_api() { return old_value; }\n"
	newMain := "#include \"api.h\"\nint use_api() { return new_value; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": apiMainURI, "languageId": "tops", "version": 1, "text": oldMain}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	oldOffset := strings.LastIndex(oldMain, "old_value")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 6, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": apiMainURI}, "position": sourceLSPPosition(oldMain, oldOffset)}})
	oldDefinition := readUntilID(t, reader, "6")
	var oldLocations []protocol.Location
	if err := json.Unmarshal(oldDefinition["result"], &oldLocations); err != nil || len(oldLocations) != 1 || oldLocations[0].URI != apiURI {
		t.Fatalf("initial header definition = %+v, err = %v", oldDefinition, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": apiURI, "languageId": "tops-header", "version": 1, "text": oldHeader}}})
	readUntilDiagnosticsVersions(t, reader, map[string]int{apiURI: 1, apiMainURI: 1})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": fullTextChangeParams(apiURI, 2, oldHeader, newHeader)})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": fullTextChangeParams(apiMainURI, 2, oldMain, newMain)})
	newOffset := strings.LastIndex(newMain, "new_value")
	readDefinitionUntilLocation(t, process.stdin, reader, apiMainURI, newMain, newOffset, apiURI)
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 8, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": apiMainURI}, "position": sourceLSPPosition(newMain, newOffset), "context": map[string]any{"includeDeclaration": false}}})
	newReferences := readUntilID(t, reader, "8")
	var newReferenceLocations []protocol.Location
	if err := json.Unmarshal(newReferences["result"], &newReferenceLocations); err != nil || len(newReferenceLocations) != 1 || newReferenceLocations[0].URI != apiMainURI {
		t.Fatalf("updated header references = %+v, err = %v", newReferences, err)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 9, "method": "shutdown"})
	readUntilID(t, reader, "9")
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

func TestLSPProcessNavigationContract(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	initialize := readResponse(t, reader)
	if initialize["error"] != nil {
		t.Fatalf("initialize response = %+v", initialize)
	}
	var initializeResult struct {
		Capabilities struct {
			DefinitionProvider     bool `json:"definitionProvider"`
			ReferencesProvider     bool `json:"referencesProvider"`
			DocumentSymbolProvider bool `json:"documentSymbolProvider"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(initialize["result"], &initializeResult); err != nil {
		t.Fatal(err)
	}
	if !initializeResult.Capabilities.DefinitionProvider || !initializeResult.Capabilities.ReferencesProvider || !initializeResult.Capabilities.DocumentSymbolProvider {
		t.Fatalf("navigation capabilities = %+v", initializeResult.Capabilities)
	}

	uri := "file:///tmp/navigation.tops"
	source := "int add(int left, int right) { return left + right; } int use() { return add(1, 2); }"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri": uri, "languageId": "tops", "version": 1, "text": source,
		}},
	})
	publication := readResponse(t, reader)
	if string(publication["method"]) != `"textDocument/publishDiagnostics"` {
		t.Fatalf("diagnostics publication = %+v", publication)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 73}},
	})
	definition := readResponse(t, reader)
	var locations []struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(definition["result"], &locations); err != nil {
		t.Fatal(err)
	}
	if len(locations) != 1 || locations[0].URI != uri {
		t.Fatalf("definition result = %+v", locations)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "textDocument/references",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 73}, "context": map[string]any{"includeDeclaration": true}},
	})
	references := readResponse(t, reader)
	var referenceLocations []map[string]any
	if err := json.Unmarshal(references["result"], &referenceLocations); err != nil {
		t.Fatal(err)
	}
	if len(referenceLocations) < 2 {
		t.Fatalf("references result = %+v", referenceLocations)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}},
	})
	documentSymbols := readResponse(t, reader)
	var symbols []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(documentSymbols["result"], &symbols); err != nil {
		t.Fatal(err)
	}
	if len(symbols) == 0 {
		t.Fatalf("document symbols = %+v", symbols)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "shutdown"})
	shutdown := readResponse(t, reader)
	if shutdown["error"] != nil {
		t.Fatalf("shutdown response = %+v", shutdown)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessNavigationMatrix(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	if response := readUntilID(t, reader, "1"); response["error"] != nil {
		t.Fatalf("initialize response = %+v", response)
	}

	headerURI := "file:///tmp/lsp-matrix/api.h"
	headerText := "int api_value;\r\n"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": headerURI, "languageId": "tops-header", "version": 1, "text": headerText}},
	})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")

	mainURI := "file:///tmp/lsp-matrix/main.tops"
	mainText := "#include \"api.h\"\r\nconst char* text = \"中文\"; int use() { int local_value; return api_value; }\r\n"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "tops", "version": 1, "text": mainText}},
	})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	apiOffset := strings.Index(mainText, "api_value")
	line := strings.Count(mainText[:apiOffset], "\n")
	character := utf16Units(mainText[strings.LastIndex(mainText[:apiOffset], "\n")+1 : apiOffset])

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition",
		"params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line, "character": character}},
	})
	definition := readUntilID(t, reader, "2")
	var definitionLocations []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"start"`
			End struct {
				Line      int `json:"line"`
				Character int `json:"character"`
			} `json:"end"`
		} `json:"range"`
	}
	if err := json.Unmarshal(definition["result"], &definitionLocations); err != nil || len(definitionLocations) != 1 || definitionLocations[0].URI != headerURI || definitionLocations[0].Range.Start.Line != 0 || definitionLocations[0].Range.Start.Character != 0 || definitionLocations[0].Range.End.Character != 14 {
		t.Fatalf("cross-file definition = %+v, err = %v", definition, err)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "textDocument/references",
		"params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line, "character": character}, "context": map[string]any{"includeDeclaration": true}},
	})
	references := readUntilID(t, reader, "3")
	var referenceLocations []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct{ Line, Character int } `json:"start"`
			End   struct{ Line, Character int } `json:"end"`
		} `json:"range"`
	}
	if err := json.Unmarshal(references["result"], &referenceLocations); err != nil || len(referenceLocations) < 2 {
		t.Fatalf("cross-file references = %+v, err = %v", references, err)
	}
	foundHeaderReference := false
	foundMainReference := false
	for _, reference := range referenceLocations {
		if reference.URI == headerURI && reference.Range.Start.Line == 0 && reference.Range.Start.Character == 0 && reference.Range.End.Character == 14 {
			foundHeaderReference = true
		}
		if reference.URI == mainURI && reference.Range.Start.Line == line && reference.Range.Start.Character == character && reference.Range.End.Character == character+len("api_value") {
			foundMainReference = true
		}
	}
	if !foundHeaderReference || !foundMainReference {
		t.Fatalf("cross-file reference ranges = %+v", referenceLocations)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 9, "method": "textDocument/references",
		"params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line, "character": character}, "context": map[string]any{"includeDeclaration": false}},
	})
	declarationExcluded := readUntilID(t, reader, "9")
	var referenceOnly []struct {
		URI   string `json:"uri"`
		Range struct {
			Start struct{ Line, Character int } `json:"start"`
			End   struct{ Line, Character int } `json:"end"`
		} `json:"range"`
	}
	if err := json.Unmarshal(declarationExcluded["result"], &referenceOnly); err != nil || len(referenceOnly) != 1 || referenceOnly[0].URI != mainURI || referenceOnly[0].Range.Start.Line != line || referenceOnly[0].Range.Start.Character != character || referenceOnly[0].Range.End.Character != character+len("api_value") {
		t.Fatalf("references without declaration = %+v, err = %v", declarationExcluded, err)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}}})
	documentSymbols := readUntilID(t, reader, "4")
	var symbols []struct {
		Name           string                                             `json:"name"`
		Range          struct{ Start, End struct{ Line, Character int } } `json:"range"`
		SelectionRange struct{ Start, End struct{ Line, Character int } } `json:"selectionRange"`
		Children       []struct {
			Name           string                                             `json:"name"`
			Range          struct{ Start, End struct{ Line, Character int } } `json:"range"`
			SelectionRange struct{ Start, End struct{ Line, Character int } } `json:"selectionRange"`
		} `json:"children"`
	}
	if err := json.Unmarshal(documentSymbols["result"], &symbols); err != nil {
		t.Fatalf("document symbols = %+v, err = %v", documentSymbols, err)
	}
	foundUse := false
	foundNestedLocal := false
	for _, symbol := range symbols {
		if symbol.Name != "use" {
			continue
		}
		foundUse = true
		if symbol.Range.End.Line < symbol.Range.Start.Line || symbol.SelectionRange.End.Line < symbol.SelectionRange.Start.Line || lspPositionBefore(symbol.SelectionRange.Start.Line, symbol.SelectionRange.Start.Character, symbol.Range.Start.Line, symbol.Range.Start.Character) || lspPositionBefore(symbol.Range.End.Line, symbol.Range.End.Character, symbol.SelectionRange.End.Line, symbol.SelectionRange.End.Character) {
			t.Fatalf("use symbol ranges = %+v", symbol)
		}
		for _, child := range symbol.Children {
			if child.Name == "local_value" {
				foundNestedLocal = true
				if child.Range.End.Line < child.Range.Start.Line || child.SelectionRange.End.Line < child.SelectionRange.Start.Line || lspPositionBefore(child.SelectionRange.Start.Line, child.SelectionRange.Start.Character, child.Range.Start.Line, child.Range.Start.Character) || lspPositionBefore(child.Range.End.Line, child.Range.End.Character, child.SelectionRange.End.Line, child.SelectionRange.End.Character) {
					t.Fatalf("nested symbol ranges = %+v", child)
				}
			}
		}
	}
	if !foundUse || !foundNestedLocal {
		t.Fatalf("document symbols = %+v", symbols)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line}}})
	malformed := readUntilID(t, reader, "5")
	var malformedError struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(malformed["error"], &malformedError); err != nil || malformedError.Code != -32602 {
		t.Fatalf("malformed navigation response = %+v, err = %v", malformed, err)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 10, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": line, "character": character}, "context": map[string]any{}}})
	malformedReferences := readUntilID(t, reader, "10")
	if err := json.Unmarshal(malformedReferences["error"], &malformedError); err != nil || malformedError.Code != -32602 {
		t.Fatalf("malformed references response = %+v, err = %v", malformedReferences, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 11, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{}}})
	malformedSymbols := readUntilID(t, reader, "11")
	if err := json.Unmarshal(malformedSymbols["error"], &malformedError); err != nil || malformedError.Code != -32602 {
		t.Fatalf("malformed documentSymbol response = %+v, err = %v", malformedSymbols, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 12, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": 0, "character": 0}}})
	emptyDefinition := readUntilID(t, reader, "12")
	if result := string(emptyDefinition["result"]); result != "[]" {
		t.Fatalf("fresh empty definition = %s", result)
	}

	mainLines := strings.Split(strings.TrimSuffix(mainText, "\r\n"), "\r\n")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "version": 2}, "contentChanges": []map[string]any{{"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": len(mainLines) - 1, "character": utf16Units(mainLines[len(mainLines)-1])}}, "text": "int replacement;\r\n"}}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 6, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": 0, "character": 4}}})
	stale := readUntilID(t, reader, "6")
	if result := string(stale["result"]); strings.Contains(result, headerURI) {
		t.Fatalf("stale definition returned old header location = %s", result)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 8, "method": "shutdown"})
	if response := readUntilID(t, reader, "8"); response["error"] != nil {
		t.Fatalf("shutdown response = %+v", response)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessNavigationCancellation(t *testing.T) {
	process := startServerWithEnv(t, map[string]string{"TOPS_LSP_NAVIGATION_BLOCK": "1"})
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	readUntilID(t, reader, "1")
	uri := "file:///tmp/navigation-cancel.tops"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "tops", "version": 1, "text": "int value;\n"}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 4}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "$/cancelRequest", "params": map[string]any{"id": 2}})
	response := readUntilID(t, reader, "2")
	var responseError struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response["error"], &responseError); err != nil || responseError.Code != -32800 {
		t.Fatalf("navigation cancellation response = %+v, err = %v", response, err)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "shutdown"})
	readUntilID(t, reader, "3")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessEmptyNavigationResults(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	readUntilID(t, reader, "1")
	uri := "file:///tmp/empty-navigation.tops"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "tops", "version": 1, "text": "// no symbols\n"}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 0}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]any{"line": 0, "character": 0}, "context": map[string]any{"includeDeclaration": false}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{"uri": uri}}})
	responses := readIDs(t, reader, "2", "3", "4")
	definition := responses["2"]
	references := responses["3"]
	symbols := responses["4"]
	if string(definition["result"]) != "[]" || string(references["result"]) != "[]" || string(symbols["result"]) != "[]" {
		t.Fatalf("empty navigation results: definition=%s references=%s symbols=%s", definition["result"], references["result"], symbols["result"])
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "shutdown"})
	readUntilID(t, reader, "5")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessConditionAndBuiltinNavigationContract(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	readUntilID(t, reader, "1")

	uri := "file:///tmp/condition-builtin.tops"
	source := "#if 0\nint hidden_symbol;\n#else\nint visible_symbol;\n#endif\nint use() { return visible_symbol; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "tops", "version": 1, "text": source}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")

	hiddenOffset := strings.Index(source, "hidden_symbol")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": sourceLSPPosition(source, hiddenOffset)}})
	hiddenDefinition := readUntilID(t, reader, "2")
	if string(hiddenDefinition["result"]) != "[]" {
		t.Fatalf("inactive definition = %s", hiddenDefinition["result"])
	}

	visibleOffset := strings.Index(source, "visible_symbol")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": sourceLSPPosition(source, visibleOffset)}})
	visibleDefinition := readUntilID(t, reader, "3")
	var visibleLocations []protocol.Location
	if err := json.Unmarshal(visibleDefinition["result"], &visibleLocations); err != nil || len(visibleLocations) != 1 || visibleLocations[0].URI != uri {
		t.Fatalf("active definition = %s, err = %v", visibleDefinition["result"], err)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{"uri": uri}}})
	documentSymbols := readUntilID(t, reader, "4")
	var symbols []protocol.DocumentSymbol
	if err := json.Unmarshal(documentSymbols["result"], &symbols); err != nil {
		t.Fatalf("conditional document symbols = %s, err = %v", documentSymbols["result"], err)
	}
	for _, symbol := range symbols {
		if symbol.Name == "hidden_symbol" {
			t.Fatalf("inactive symbol was published = %+v", symbol)
		}
	}

	headerURI := "file:///tmp/condition-builtin/builtins.h"
	headerText := "int threadIdx;\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": headerURI, "languageId": "tops-header", "version": 1, "text": headerText}}})
	readUntilDiagnosticsURI(t, reader, headerURI)
	mainURI := "file:///tmp/condition-builtin/main.tops"
	mainText := "int use_builtin() { return threadIdx.x; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "tops", "version": 1, "text": mainText}}})
	readUntilDiagnosticsURI(t, reader, mainURI)
	builtinOffset := strings.Index(mainText, "threadIdx")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": sourceLSPPosition(mainText, builtinOffset)}})
	builtinDefinition := readUntilID(t, reader, "5")
	if string(builtinDefinition["result"]) != "[]" {
		t.Fatalf("unverified builtin definition = %s", builtinDefinition["result"])
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 6, "method": "shutdown"})
	if response := readUntilID(t, reader, "6"); response["error"] != nil {
		t.Fatalf("shutdown response = %+v", response)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

func TestLSPProcessHeaderChangeDropsOldDefinition(t *testing.T) {
	process := startServer(t)
	reader := process.reader
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	readUntilID(t, reader, "1")
	headerURI := "file:///tmp/lsp-header-change/api.h"
	mainURI := "file:///tmp/lsp-header-change/main.tops"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": headerURI, "languageId": "tops-header", "version": 1, "text": "int old_value;\n"}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	mainText := strings.Repeat("// padding\n", 20000) + "#include \"api.h\"\nint use() { return old_value; }\n"
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI, "languageId": "tops", "version": 1, "text": mainText}}})
	readUntilMethod(t, reader, "textDocument/publishDiagnostics")
	oldOffset := strings.Index(mainText, "old_value")
	oldLine := strings.Count(mainText[:oldOffset], "\n")
	oldCharacter := oldOffset - strings.LastIndex(mainText[:oldOffset], "\n") - 1
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": oldLine, "character": oldCharacter}}})
	oldDefinition := readUntilID(t, reader, "2")
	var oldLocations []struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(oldDefinition["result"], &oldLocations); err != nil || len(oldLocations) != 1 || oldLocations[0].URI != headerURI {
		t.Fatalf("initial header definition = %+v, err = %v", oldDefinition, err)
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange", "params": map[string]any{"textDocument": map[string]any{"uri": headerURI, "version": 2}, "contentChanges": []map[string]any{{"range": map[string]any{"start": map[string]any{"line": 0, "character": 0}, "end": map[string]any{"line": 0, "character": len("int old_value;")}}, "text": "int new_value;\n"}}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "textDocument/definition", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": oldLine, "character": oldCharacter}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "textDocument/references", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}, "position": map[string]any{"line": oldLine, "character": oldCharacter}, "context": map[string]any{"includeDeclaration": true}}})
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 6, "method": "textDocument/documentSymbol", "params": map[string]any{"textDocument": map[string]any{"uri": mainURI}}})
	staleResponses := readIDs(t, reader, "3", "5", "6")
	newDefinition := staleResponses["3"]
	newReferences := staleResponses["5"]
	newSymbols := staleResponses["6"]
	var newLocations []protocol.Location
	if err := json.Unmarshal(newDefinition["result"], &newLocations); err != nil || len(newLocations) != 0 {
		t.Fatalf("definition after header change must not return old data: %s", newDefinition["result"])
	}
	var newReferenceLocations []protocol.Location
	if err := json.Unmarshal(newReferences["result"], &newReferenceLocations); err != nil || len(newReferenceLocations) != 0 {
		t.Fatalf("references after header change must not return old data: %s", newReferences["result"])
	}
	var newDocumentSymbols []protocol.DocumentSymbol
	if err := json.Unmarshal(newSymbols["result"], &newDocumentSymbols); err != nil {
		t.Fatalf("document symbols after header change = %s, err = %v", newSymbols["result"], err)
	}
	for _, symbol := range newDocumentSymbols {
		if symbol.Name == "old_value" {
			t.Fatalf("document symbols after header change leak old header data: %s", newSymbols["result"])
		}
	}

	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "shutdown"})
	readUntilID(t, reader, "4")
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
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
