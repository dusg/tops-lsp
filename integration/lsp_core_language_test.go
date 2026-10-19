package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tops-lsp/internal/protocol"
)

type fixtureDocument struct {
	URI  string
	Text string
}

type diagnosticData struct {
	DocumentVersion int    `json:"documentVersion"`
	ContextVersion  int    `json:"contextVersion"`
	ContextStatus   string `json:"contextStatus"`
	ProfileID       string `json:"profileId"`
	PassKind        string `json:"passKind"`
	SemanticKind    string `json:"semanticKind"`
}

type diagnosticView struct {
	Code     string          `json:"code"`
	Severity int             `json:"severity"`
	Source   string          `json:"source"`
	Message  string          `json:"message"`
	Range    protocol.Range  `json:"range"`
	Data     *diagnosticData `json:"data"`
}

type diagnosticsParams struct {
	URI            string           `json:"uri"`
	Version        *int             `json:"version"`
	ContextVersion *int             `json:"contextVersion"`
	Diagnostics    []diagnosticView `json:"diagnostics"`
}

func readFixture(t *testing.T, name string) fixtureDocument {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "testdata", "lsp", "fixtures", name)
	text, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s error = %v", name, err)
	}
	return fixtureDocument{URI: fileURI(path), Text: string(text)}
}

func initializeProcess(t *testing.T, process *serverProcess, params map[string]any) {
	t.Helper()
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": params})
	response := readTypedUntilID(t, process.reader, "1")
	if response.Error != nil {
		t.Fatalf("initialize error = %+v", response.Error)
	}
}

func openDocument(t *testing.T, process *serverProcess, document fixtureDocument, version int) {
	t.Helper()
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{
			"uri": document.URI, "languageId": "tops", "version": version, "text": document.Text,
		}},
	})
}

func changeDocument(t *testing.T, process *serverProcess, document fixtureDocument, version int, text string) {
	t.Helper()
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "textDocument/didChange",
		"params":  fullTextChangeParams(document.URI, version, document.Text, text),
	})
}

func shutdownProcess(t *testing.T, process *serverProcess) {
	t.Helper()
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 900, "method": "shutdown"})
	if response := readTypedUntilID(t, process.reader, "900"); response.Error != nil {
		t.Fatalf("shutdown error = %+v", response.Error)
	}
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "method": "exit"})
	process.finish(t, false)
}

// readDiagnosticsFrame reads frames until a publishDiagnostics notification for
// uri arrives.
func readDiagnosticsFrame(t *testing.T, process *serverProcess, uri string) diagnosticsParams {
	t.Helper()
	for {
		response := readTypedResponse(t, process.reader)
		if response.Method != "textDocument/publishDiagnostics" {
			continue
		}
		var params diagnosticsParams
		if err := json.Unmarshal(response.Params, &params); err != nil {
			t.Fatalf("diagnostic params decode = %v", err)
		}
		if params.URI != uri {
			continue
		}
		return params
	}
}

func waitDiagnostics(t *testing.T, process *serverProcess, uri string, version int, requireNonEmpty bool, deadline time.Duration) diagnosticsParams {
	t.Helper()
	started := time.Now()
	for {
		if time.Since(started) > deadline {
			t.Fatalf("timed out waiting for diagnostics uri=%s version=%d nonEmpty=%v", uri, version, requireNonEmpty)
		}
		params := readDiagnosticsFrame(t, process, uri)
		if params.Version == nil || *params.Version != version {
			continue
		}
		if requireNonEmpty && len(params.Diagnostics) == 0 {
			continue
		}
		return params
	}
}

// requestPosition sends a position-based request and returns its raw result.
func requestPosition(t *testing.T, process *serverProcess, id, method, uri, source string, offset int, extraParams map[string]any) (json.RawMessage, *typedLSPError) {
	t.Helper()
	params := map[string]any{"textDocument": map[string]any{"uri": uri}, "position": sourceLSPPosition(source, offset)}
	for key, value := range extraParams {
		params[key] = value
	}
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": jsonNumberOrString(id), "method": method,
		"params": params,
	})
	response := readTypedUntilID(t, process.reader, id)
	if response.Error != nil {
		return nil, response.Error
	}
	return response.Result, nil
}

func requestCompletion(t *testing.T, process *serverProcess, id string, uri string, source string, offset int) ([]protocol.CompletionItem, *typedLSPError) {
	t.Helper()
	result, lspError := requestPosition(t, process, id, "textDocument/completion", uri, source, offset, nil)
	if lspError != nil {
		return nil, lspError
	}
	var items []protocol.CompletionItem
	if err := json.Unmarshal(result, &items); err != nil {
		t.Fatalf("completion response decode = %v; result = %s", err, result)
	}
	return items, nil
}

func requestHover(t *testing.T, process *serverProcess, id string, uri string, source string, offset int) (*protocol.Hover, *typedLSPError) {
	t.Helper()
	result, lspError := requestPosition(t, process, id, "textDocument/hover", uri, source, offset, nil)
	if lspError != nil {
		return nil, lspError
	}
	if string(result) == "null" {
		return nil, nil
	}
	var hover protocol.Hover
	if err := json.Unmarshal(result, &hover); err != nil {
		t.Fatalf("hover response decode = %v; result = %s", err, result)
	}
	return &hover, nil
}

func requestDefinition(t *testing.T, process *serverProcess, id string, uri string, source string, offset int) ([]protocol.Location, *typedLSPError) {
	t.Helper()
	result, lspError := requestPosition(t, process, id, "textDocument/definition", uri, source, offset, nil)
	if lspError != nil {
		return nil, lspError
	}
	var locations []protocol.Location
	if string(result) != "null" {
		if err := json.Unmarshal(result, &locations); err != nil {
			t.Fatalf("definition response decode = %v; result = %s", err, result)
		}
	}
	return locations, nil
}

func requestReferences(t *testing.T, process *serverProcess, id string, uri string, source string, offset int) ([]protocol.Location, *typedLSPError) {
	t.Helper()
	result, lspError := requestPosition(t, process, id, "textDocument/references", uri, source, offset, map[string]any{"context": map[string]any{"includeDeclaration": true}})
	if lspError != nil {
		return nil, lspError
	}
	var locations []protocol.Location
	if err := json.Unmarshal(result, &locations); err != nil {
		t.Fatalf("references response decode = %v; result = %s", err, result)
	}
	return locations, nil
}

func requestDocumentSymbols(t *testing.T, process *serverProcess, id string, uri string) ([]protocol.DocumentSymbol, *typedLSPError) {
	t.Helper()
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": jsonNumberOrString(id), "method": "textDocument/documentSymbol",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}},
	})
	response := readTypedUntilID(t, process.reader, id)
	if response.Error != nil {
		return nil, response.Error
	}
	var symbols []protocol.DocumentSymbol
	if err := json.Unmarshal(response.Result, &symbols); err != nil {
		t.Fatalf("document symbol response decode = %v; result = %s", err, response.Result)
	}
	return symbols, nil
}

func jsonNumberOrString(id string) any {
	if number, err := strconv.Atoi(id); err == nil {
		return number
	}
	return id
}

func completionLabels(items []protocol.CompletionItem) map[string]protocol.CompletionItem {
	result := make(map[string]protocol.CompletionItem, len(items))
	for _, item := range items {
		result[item.Label] = item
	}
	return result
}

func hasCompletionLabel(items []protocol.CompletionItem, label string) bool {
	_, ok := completionLabels(items)[label]
	return ok
}

func diagnosticsContain(params diagnosticsParams, code string, severity int) bool {
	for _, diagnostic := range params.Diagnostics {
		if code != "" && diagnostic.Code != code {
			continue
		}
		if severity != 0 && diagnostic.Severity != severity {
			continue
		}
		return true
	}
	return false
}

func errorDiagnostics(params diagnosticsParams) []diagnosticView {
	result := make([]diagnosticView, 0)
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Severity == 1 {
			result = append(result, diagnostic)
		}
	}
	return result
}

func TestLSPProcessCoreLanguageValidKernel(t *testing.T) {
	process := startServer(t)
	initializeProcess(t, process, map[string]any{})
	document := readFixture(t, "valid-kernel.tops")
	openDocument(t, process, document, 1)
	initial := waitDiagnostics(t, process, document.URI, 1, true, 5*time.Second)
	if errors := errorDiagnostics(initial); len(errors) != 0 {
		t.Fatalf("valid kernel reported unexpected errors: %+v", errors)
	}

	completionOffset := strings.Index(document.Text, "tile_offset(rows, columns)") + len("tile_offset(rows, ")
	items, completionError := requestCompletion(t, process, "10", document.URI, document.Text, completionOffset)
	if completionError != nil {
		t.Fatalf("completion error = %+v", completionError)
	}
	for _, label := range []string{"shared_tile", "tile_offset", "rows", "columns", "scale_kernel", "output"} {
		if !hasCompletionLabel(items, label) {
			t.Fatalf("completion for valid kernel is missing %q: %+v", label, completionLabels(items))
		}
	}

	// Every resolvable name at the cursor must produce a hover with name and kind.
	for _, hoverCase := range []struct {
		name   string
		offset int
		kind   string
	}{
		{"rows", strings.Index(document.Text, "tile_offset(rows") + len("tile_offset(") + 2, "variable"},
		{"shared_tile", strings.LastIndex(document.Text, "shared_tile") + 2, "variable"},
		{"scale_kernel", strings.Index(document.Text, "scale_kernel") + 2, "function"},
	} {
		hoverResult, hoverErr := requestHover(t, process, "17"+hoverCase.name, document.URI, document.Text, hoverCase.offset)
		if hoverErr != nil || hoverResult == nil || !strings.Contains(hoverResult.Contents.Value, hoverCase.name) || !strings.Contains(hoverResult.Contents.Value, hoverCase.kind) {
			t.Fatalf("hover for %q error = %+v hover = %+v", hoverCase.name, hoverErr, hoverResult)
		}
	}

	// Comments and string literals never produce completions or hovers, and
	// never report an error. Whitespace produces no hover, but completion is
	// allowed there because it is a declaration-start position (FR-015, FR-022).
	for _, emptyCase := range []struct {
		name            string
		offset          int
		checkCompletion bool
	}{
		{"comment", strings.Index(document.Text, "Completion and hover") + 2, true},
		{"string", strings.Index(document.Text, `"tile_offset"`) + 2, true},
		{"whitespace", strings.Index(document.Text, "\n\n") + 1, false},
	} {
		if emptyCase.checkCompletion {
			emptyItems, emptyCompletionError := requestCompletion(t, process, "18"+emptyCase.name, document.URI, document.Text, emptyCase.offset)
			if emptyCompletionError != nil || len(emptyItems) != 0 {
				t.Fatalf("completion in %s = %+v error = %+v", emptyCase.name, emptyItems, emptyCompletionError)
			}
		}
		emptyHover, emptyHoverError := requestHover(t, process, "19"+emptyCase.name, document.URI, document.Text, emptyCase.offset)
		if emptyHoverError != nil || emptyHover != nil {
			t.Fatalf("hover in %s = %+v error = %+v", emptyCase.name, emptyHover, emptyHoverError)
		}
	}

	tileOffsetOffset := strings.Index(document.Text, "tile_offset(rows")
	hover, hoverError := requestHover(t, process, "11", document.URI, document.Text, tileOffsetOffset+2)
	if hoverError != nil || hover == nil {
		t.Fatalf("hover error = %+v hover = %+v", hoverError, hover)
	}
	if hover.Contents.Kind != protocol.MarkupKindMarkdown {
		t.Fatalf("hover markup kind = %q", hover.Contents.Kind)
	}
	for _, fragment := range []string{"tile_offset", "function", "Profile:", "Pass:"} {
		if !strings.Contains(hover.Contents.Value, fragment) {
			t.Fatalf("hover content %q is missing %q", hover.Contents.Value, fragment)
		}
	}

	locations, definitionError := requestDefinition(t, process, "12", document.URI, document.Text, tileOffsetOffset+2)
	if definitionError != nil || len(locations) != 1 || locations[0].URI != document.URI {
		t.Fatalf("definition error = %+v locations = %+v", definitionError, locations)
	}
	declarationLine := strings.Count(document.Text[:strings.Index(document.Text, "tile_offset(int row")], "\n")
	if locations[0].Range.Start.Line != declarationLine {
		t.Fatalf("definition line = %d want %d", locations[0].Range.Start.Line, declarationLine)
	}

	references, referencesError := requestReferences(t, process, "13", document.URI, document.Text, tileOffsetOffset+2)
	if referencesError != nil || len(references) < 2 {
		t.Fatalf("references error = %+v references = %+v", referencesError, references)
	}
	for _, location := range references {
		if location.URI != document.URI {
			t.Fatalf("references leak to %q", location.URI)
		}
	}

	symbols, symbolError := requestDocumentSymbols(t, process, "14", document.URI)
	if symbolError != nil {
		t.Fatalf("document symbols error = %+v", symbolError)
	}
	names := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		names[symbol.Name] = struct{}{}
	}
	for _, name := range []string{"scale_kernel", "shared_tile", "TileShape"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("document symbols missing %q: %+v", name, names)
		}
	}

	updated := strings.Replace(document.Text,
		"  int lane = tile_offset(rows, columns);",
		"  int extra_value = 1;\n  int lane = tile_offset(rows, columns) + extra_value;", 1)
	if updated == document.Text {
		t.Fatal("fixture edit did not apply")
	}
	changeStarted := time.Now()
	changeDocument(t, process, document, 2, updated)
	updatedDiagnostics := waitDiagnostics(t, process, document.URI, 2, true, 5*time.Second)
	elapsed := time.Since(changeStarted)
	if elapsed > 2*time.Second {
		t.Fatalf("new diagnostics took %s, want under 2s", elapsed)
	}
	if errors := errorDiagnostics(updatedDiagnostics); len(errors) != 0 {
		t.Fatalf("edited valid kernel reported errors: %+v", errors)
	}

	updatedCompletionOffset := strings.LastIndex(updated, "extra_value")
	updatedItems, updatedCompletionError := requestCompletion(t, process, "15", document.URI, updated, updatedCompletionOffset+2)
	if updatedCompletionError != nil || !hasCompletionLabel(updatedItems, "extra_value") {
		t.Fatalf("updated completion error = %+v items = %+v", updatedCompletionError, completionLabels(updatedItems))
	}
	extraOffset := strings.Index(updated, "extra_value")
	updatedHover, updatedHoverError := requestHover(t, process, "16", document.URI, updated, extraOffset+2)
	if updatedHoverError != nil || updatedHover == nil || !strings.Contains(updatedHover.Contents.Value, "extra_value") {
		t.Fatalf("updated hover error = %+v hover = %+v", updatedHoverError, updatedHover)
	}

	shutdownProcess(t, process)
}

func TestLSPProcessCoreLanguageStandardCppErrors(t *testing.T) {
	process := startServerWithEnv(t, map[string]string{"TOPS_LSP_TARGET_PROFILE": "gcu400", "TOPS_LSP_PASS_KIND": "host+device"})
	initializeProcess(t, process, map[string]any{})
	document := readFixture(t, "std-cpp-error.tops")
	openDocument(t, process, document, 1)
	initial := waitDiagnostics(t, process, document.URI, 1, true, 5*time.Second)
	if !diagnosticsContain(initial, "", 1) {
		t.Fatalf("std cpp error file has no Error diagnostics: %+v", initial.Diagnostics)
	}
	if !diagnosticsContain(initial, "tops-semantic-unknown-name", 1) {
		t.Fatalf("std cpp error file is missing tops-semantic-unknown-name: %+v", initial.Diagnostics)
	}
	for _, diagnostic := range initial.Diagnostics {
		if diagnostic.Severity != 1 {
			continue
		}
		if !strings.HasPrefix(diagnostic.Code, "tops-syntax-") && diagnostic.Code != "tops-semantic-unknown-name" {
			t.Fatalf("unexpected Error diagnostic in std cpp error file: %+v", diagnostic)
		}
	}

	// Version 2 fixes only the missing semicolon; the unknown name remains so
	// the non-empty result proves the new analysis completed.
	partial := strings.Replace(document.Text, "return total\n}", "return total;\n}", 1)
	if partial == document.Text {
		t.Fatal("syntax fix did not apply")
	}
	changeDocument(t, process, document, 2, partial)
	partialDiagnostics := waitDiagnostics(t, process, document.URI, 2, true, 5*time.Second)
	if diagnosticsContain(partialDiagnostics, "tops-syntax-missing-token", 1) {
		t.Fatalf("syntax error survived the fix: %+v", partialDiagnostics.Diagnostics)
	}
	if !diagnosticsContain(partialDiagnostics, "tops-semantic-unknown-name", 1) {
		t.Fatalf("unknown name disappeared too early: %+v", partialDiagnostics.Diagnostics)
	}

	// Completion, hover and definition keep working while errors are present.
	resolvableOffset := strings.Index(partial, "compute_total(input)")
	items, completionError := requestCompletion(t, process, "20", document.URI, partial, resolvableOffset+2)
	if completionError != nil || !hasCompletionLabel(items, "sample_value") {
		t.Fatalf("completion while erroneous error = %+v items = %+v", completionError, completionLabels(items))
	}
	hover, hoverError := requestHover(t, process, "21", document.URI, partial, resolvableOffset+2)
	if hoverError != nil || hover == nil || !strings.Contains(hover.Contents.Value, "compute_total") {
		t.Fatalf("hover while erroneous error = %+v hover = %+v", hoverError, hover)
	}
	if _, definitionError := requestDefinition(t, process, "22", document.URI, partial, resolvableOffset+2); definitionError != nil {
		t.Fatalf("definition while erroneous error = %+v", definitionError)
	}
	references, referencesError := requestReferences(t, process, "23", document.URI, partial, resolvableOffset+2)
	if referencesError != nil || len(references) < 2 {
		t.Fatalf("references while erroneous error = %+v references = %+v", referencesError, references)
	}
	symbols, symbolError := requestDocumentSymbols(t, process, "24", document.URI)
	if symbolError != nil {
		t.Fatalf("document symbols while erroneous error = %+v", symbolError)
	}
	symbolNames := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbolNames[symbol.Name] = struct{}{}
	}
	for _, name := range []string{"compute_total", "sample_value"} {
		if _, ok := symbolNames[name]; !ok {
			t.Fatalf("document symbols while erroneous missing %q: %+v", name, symbolNames)
		}
	}

	// Version 3 removes the unknown name; the diagnostics array becomes empty.
	fixed := strings.Replace(partial, "input + missing_value", "input + 1", 1)
	if fixed == partial {
		t.Fatal("semantic fix did not apply")
	}
	changeDocument(t, process, document, 3, fixed)
	cleaned := waitDiagnostics(t, process, document.URI, 3, false, 5*time.Second)
	if len(cleaned.Diagnostics) != 0 {
		t.Fatalf("fixed document still reports diagnostics: %+v", cleaned.Diagnostics)
	}

	shutdownProcess(t, process)
}

func TestLSPProcessCoreLanguageStaleAndCloseCleanup(t *testing.T) {
	process := startServer(t)
	initializeProcess(t, process, map[string]any{})
	document := readFixture(t, "valid-kernel.tops")
	openDocument(t, process, document, 1)
	baseline := waitDiagnostics(t, process, document.URI, 1, true, 5*time.Second)
	if len(baseline.Diagnostics) == 0 {
		t.Fatal("baseline diagnostics are empty")
	}

	// Make the intermediate version expensive to analyze so the final change
	// wins before the intermediate analysis can publish.
	intermediate := strings.Repeat("// padding\n", 4000) + document.Text
	final := document.Text + "\nint extra_declaration = 1;\n"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didChange",
		"params": fullTextChangeParams(document.URI, 2, document.Text, intermediate),
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didChange",
		"params": fullTextChangeParams(document.URI, 3, intermediate, final),
	})
	deadline := time.Now().Add(10 * time.Second)
	oldVersionDiagnostics := 0
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for version 3 diagnostics")
		}
		params := readDiagnosticsFrame(t, process, document.URI)
		if params.Version == nil {
			continue
		}
		if *params.Version == 2 && len(params.Diagnostics) > 0 {
			oldVersionDiagnostics++
		}
		if *params.Version == 3 && len(params.Diagnostics) > 0 {
			break
		}
	}
	if oldVersionDiagnostics != 0 {
		t.Fatalf("published %d diagnostics for stale version 2", oldVersionDiagnostics)
	}

	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "method": "textDocument/didClose",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}},
	})
	closed := readDiagnosticsFrame(t, process, document.URI)
	if len(closed.Diagnostics) != 0 {
		t.Fatalf("diagnostics after didClose = %+v", closed.Diagnostics)
	}
	if closed.Version == nil || *closed.Version != 3 {
		t.Fatalf("cleanup version = %+v, want 3", closed.Version)
	}

	// Requests for a closed document return the agreed empty shapes.
	unopened := fixtureDocument{URI: "file:///tmp/not-open.tops", Text: "int value = 1;\n"}
	if locations, definitionError := requestDefinition(t, process, "30", unopened.URI, unopened.Text, 4); definitionError != nil || len(locations) != 0 {
		t.Fatalf("closed definition = %+v error = %+v", locations, definitionError)
	}
	if items, completionError := requestCompletion(t, process, "31", unopened.URI, unopened.Text, 4); completionError != nil || len(items) != 0 {
		t.Fatalf("closed completion = %+v error = %+v", items, completionError)
	}
	if hover, hoverError := requestHover(t, process, "32", unopened.URI, unopened.Text, 4); hoverError != nil || hover != nil {
		t.Fatalf("closed hover = %+v error = %+v", hover, hoverError)
	}

	shutdownProcess(t, process)
}

func TestLSPProcessCoreLanguageTopsSemanticErrors(t *testing.T) {
	resolved := startServerWithEnv(t, map[string]string{
		"TOPS_LSP_TARGET_PROFILE":    "gcu400",
		"TOPS_LSP_PASS_KIND":         "host+device",
		"TOPS_LSP_PREDEFINED_MACROS": "TOPS_ACCEPTANCE=1",
	})
	initializeProcess(t, resolved, map[string]any{})
	document := readFixture(t, "tops-semantic-error.tops")
	openDocument(t, resolved, document, 1)
	params := waitDiagnostics(t, resolved, document.URI, 1, true, 5*time.Second)
	found := false
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Code != "tops-semantic-host-device-boundary" {
			continue
		}
		if diagnostic.Severity != 1 || diagnostic.Data == nil {
			continue
		}
		if diagnostic.Data.ProfileID == "gcu400" && diagnostic.Data.PassKind == "host" && diagnostic.Data.ContextStatus == "resolved" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tops semantic diagnostics = %+v", params.Diagnostics)
	}
	for _, diagnostic := range params.Diagnostics {
		if diagnostic.Severity != 1 {
			continue
		}
		if diagnostic.Code != "tops-semantic-host-device-boundary" {
			t.Fatalf("unexpected Error diagnostic in resolved context: %+v", diagnostic)
		}
	}
	callOffset := strings.Index(document.Text, "device_only();") + 2
	items, completionError := requestCompletion(t, resolved, "25", document.URI, document.Text, callOffset)
	if completionError != nil || !hasCompletionLabel(items, "device_only") {
		t.Fatalf("tops semantic completion error = %+v items = %+v", completionError, completionLabels(items))
	}
	hover, hoverError := requestHover(t, resolved, "26", document.URI, document.Text, callOffset)
	if hoverError != nil || hover == nil || !strings.Contains(hover.Contents.Value, "device_only") || !strings.Contains(hover.Contents.Value, "function") {
		t.Fatalf("tops semantic hover error = %+v hover = %+v", hoverError, hover)
	}
	locations, definitionError := requestDefinition(t, resolved, "27", document.URI, document.Text, callOffset)
	if definitionError != nil || len(locations) != 1 || locations[0].Range.Start.Line != 0 {
		t.Fatalf("tops semantic definition error = %+v locations = %+v", definitionError, locations)
	}
	shutdownProcess(t, resolved)

	degraded := startServer(t)
	initializeProcess(t, degraded, map[string]any{})
	openDocument(t, degraded, document, 1)
	degradedParams := waitDiagnostics(t, degraded, document.URI, 1, true, 5*time.Second)
	if errors := errorDiagnostics(degradedParams); len(errors) != 0 {
		t.Fatalf("degraded context published errors: %+v", errors)
	}
	degradedFound := false
	for _, diagnostic := range degradedParams.Diagnostics {
		if diagnostic.Code == "tops-semantic-context-degraded" && diagnostic.Severity == 2 {
			degradedFound = true
		}
	}
	if !degradedFound {
		t.Fatalf("missing Warning-level context-degraded diagnostic: %+v", degradedParams.Diagnostics)
	}
	shutdownProcess(t, degraded)
}

func TestLSPProcessCoreLanguageRequestContract(t *testing.T) {
	process := startServer(t)
	writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{}})
	initialize := readTypedUntilID(t, process.reader, "1")
	if initialize.Error != nil {
		t.Fatalf("initialize error = %+v", initialize.Error)
	}
	var result struct {
		Capabilities map[string]json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(initialize.Result, &result); err != nil {
		t.Fatalf("initialize result decode = %v", err)
	}
	expectedCapabilities := map[string]bool{
		"textDocumentSync": true, "definitionProvider": true, "referencesProvider": true,
		"documentSymbolProvider": true, "completionProvider": true, "hoverProvider": true,
	}
	for key := range result.Capabilities {
		if !expectedCapabilities[key] {
			t.Fatalf("unexpected capability %q", key)
		}
	}
	for key := range expectedCapabilities {
		if _, ok := result.Capabilities[key]; !ok {
			t.Fatalf("missing capability %q", key)
		}
	}
	if string(result.Capabilities["completionProvider"]) != "{}" {
		t.Fatalf("completionProvider = %s", result.Capabilities["completionProvider"])
	}

	document := readFixture(t, "valid-kernel.tops")
	openDocument(t, process, document, 1)
	waitDiagnostics(t, process, document.URI, 1, true, 5*time.Second)

	invalidRequests := []struct {
		id     string
		method string
		params map[string]any
	}{
		{"40", "textDocument/definition", map[string]any{}},
		{"41", "textDocument/references", map[string]any{}},
		{"42", "textDocument/completion", map[string]any{}},
		{"43", "textDocument/hover", map[string]any{}},
		{"44", "textDocument/documentSymbol", map[string]any{}},
		{"45", "textDocument/definition", map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": map[string]any{"line": 9999, "character": 0}}},
	}
	for _, request := range invalidRequests {
		writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": jsonNumberOrString(request.id), "method": request.method, "params": request.params})
		response := readTypedUntilID(t, process.reader, request.id)
		if response.Error == nil || response.Error.Code != int(protocol.InvalidParams) {
			t.Fatalf("%s invalid params response = %+v", request.method, response)
		}
	}

	unopened := fixtureDocument{URI: "file:///tmp/core-language-unopened.tops", Text: "int value = 1;\n"}
	if locations, err := requestDefinition(t, process, "46", unopened.URI, unopened.Text, 4); err != nil || len(locations) != 0 {
		t.Fatalf("unopened definition = %+v error = %+v", locations, err)
	}
	if locations, err := requestReferences(t, process, "47", unopened.URI, unopened.Text, 4); err != nil || len(locations) != 0 {
		t.Fatalf("unopened references = %+v error = %+v", locations, err)
	}
	if items, err := requestCompletion(t, process, "48", unopened.URI, unopened.Text, 4); err != nil || len(items) != 0 {
		t.Fatalf("unopened completion = %+v error = %+v", items, err)
	}
	if hover, err := requestHover(t, process, "49", unopened.URI, unopened.Text, 4); err != nil || hover != nil {
		t.Fatalf("unopened hover = %+v error = %+v", hover, err)
	}
	if symbols, err := requestDocumentSymbols(t, process, "50", unopened.URI); err != nil || len(symbols) != 0 {
		t.Fatalf("unopened document symbols = %+v error = %+v", symbols, err)
	}

	for index, method := range []string{"textDocument/diagnostic", "workspace/symbol", "completionItem/resolve"} {
		id := strconv.Itoa(60 + index)
		writeJSON(t, process.stdin, map[string]any{"jsonrpc": "2.0", "id": 60 + index, "method": method, "params": map[string]any{}})
		response := readTypedUntilID(t, process.reader, id)
		if response.Error == nil || response.Error.Code != int(protocol.MethodNotFound) {
			t.Fatalf("%s response = %+v", method, response)
		}
	}

	hover, hoverError := requestHover(t, process, "70", document.URI, document.Text, strings.Index(document.Text, "tile_offset(rows")+2)
	if hoverError != nil || hover == nil || hover.Contents.Kind != protocol.MarkupKindMarkdown {
		t.Fatalf("default hover markup = %+v error = %+v", hover, hoverError)
	}

	// A slow intermediate version keeps every stale publish from leaking.
	oldDiagnosticPublishes := 0
	oldResults := 0
	previousText := document.Text
	for iteration := 0; iteration < 100; iteration++ {
		raceDeadline := time.Now().Add(10 * time.Second)
		text := strings.Repeat("// race padding\n", iteration+1) + fmt.Sprintf("int f%d() { return %d; }\nint use_%d() { return f%d(); }\n", iteration, iteration, iteration, iteration)
		version := iteration + 2
		changeDocument(t, process, fixtureDocument{URI: document.URI, Text: previousText}, version, text)
		previousText = text
		callOffset := strings.Index(text, fmt.Sprintf("f%d();", iteration))
		writeJSON(t, process.stdin, map[string]any{
			"jsonrpc": "2.0", "id": 1000 + iteration, "method": "textDocument/definition",
			"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": sourceLSPPosition(text, callOffset)},
		})
		expectedLine := strings.Count(text[:strings.Index(text, fmt.Sprintf("int f%d()", iteration))], "\n")
		var response typedLSPResponse
		gotResponse := false
		sawVersion := false
		for !gotResponse || !sawVersion {
			if time.Now().After(raceDeadline) {
				t.Fatalf("race iteration %d did not settle", iteration)
			}
			frame := readTypedResponse(t, process.reader)
			if frame.Method == "textDocument/publishDiagnostics" {
				var params diagnosticsParams
				if json.Unmarshal(frame.Params, &params) == nil && params.URI == document.URI {
					if params.Version != nil && *params.Version < version && len(params.Diagnostics) > 0 {
						oldDiagnosticPublishes++
					}
					if params.Version != nil && *params.Version == version && len(params.Diagnostics) > 0 {
						sawVersion = true
					}
				}
				continue
			}
			if typedIDMatches(frame.ID, strconv.Itoa(1000+iteration)) {
				response = frame
				gotResponse = true
			}
		}
		if response.Error == nil {
			var locations []protocol.Location
			if err := json.Unmarshal(response.Result, &locations); err == nil {
				for _, location := range locations {
					if location.URI == document.URI && location.Range.Start.Line != expectedLine {
						oldResults++
					}
				}
			}
		} else if response.Error.Code != int(protocol.ContentModified) && response.Error.Code != int(protocol.RequestCancelled) {
			t.Fatalf("race definition error = %+v", response.Error)
		}
	}
	if oldResults != 0 || oldDiagnosticPublishes != 0 {
		t.Fatalf("stale results = %d, stale diagnostic publishes = %d", oldResults, oldDiagnosticPublishes)
	}
	shutdownProcess(t, process)

	// Cancellation and the fixed navigation timeout use a dedicated server so
	// the navigation gate does not affect the race loop above.
	gateProcess := startServerWithEnv(t, map[string]string{"TOPS_LSP_NAVIGATION_BLOCK": "1"})
	initializeProcess(t, gateProcess, map[string]any{})
	gateDocument := readFixture(t, "valid-kernel.tops")
	openDocument(t, gateProcess, gateDocument, 1)
	waitDiagnostics(t, gateProcess, gateDocument.URI, 1, true, 5*time.Second)

	// Cancellation is repeated 100 times, matching the version-race loop above.
	cancellationStarted := time.Now()
	for iteration := 0; iteration < 100; iteration++ {
		requestID := 2000 + iteration
		id := strconv.Itoa(requestID)
		writeJSON(t, gateProcess.stdin, map[string]any{
			"jsonrpc": "2.0", "id": requestID, "method": "textDocument/definition",
			"params": map[string]any{"textDocument": map[string]any{"uri": gateDocument.URI}, "position": sourceLSPPosition(gateDocument.Text, strings.Index(gateDocument.Text, "tile_offset(rows")+2)},
		})
		writeJSON(t, gateProcess.stdin, map[string]any{"jsonrpc": "2.0", "method": "$/cancelRequest", "params": map[string]any{"id": requestID}})
		cancelled := readTypedUntilID(t, gateProcess.reader, id)
		if cancelled.Error == nil || cancelled.Error.Code != int(protocol.RequestCancelled) {
			t.Fatalf("cancelled definition %d = %+v", iteration, cancelled)
		}
	}
	if time.Since(cancellationStarted) > 10*time.Second {
		t.Fatal("cancellation loop did not return promptly")
	}

	writeJSON(t, gateProcess.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 81, "method": "textDocument/definition",
		"params": map[string]any{"textDocument": map[string]any{"uri": gateDocument.URI}, "position": sourceLSPPosition(gateDocument.Text, strings.Index(gateDocument.Text, "tile_offset(rows")+2)},
	})
	blockedStarted := time.Now()
	blocked := readTypedUntilID(t, gateProcess.reader, "81")
	blockedElapsed := time.Since(blockedStarted)
	if blocked.Error == nil || blocked.Error.Code != int(protocol.ContentModified) {
		t.Fatalf("blocked definition = %+v", blocked)
	}
	if blockedElapsed < 1500*time.Millisecond || blockedElapsed > 4*time.Second {
		t.Fatalf("navigation timeout elapsed = %s", blockedElapsed)
	}
	shutdownProcess(t, gateProcess)
}

func TestLSPProcessCoreLanguageAnalyzingState(t *testing.T) {
	process := startServer(t)
	initializeProcess(t, process, map[string]any{})
	document := readFixture(t, "valid-kernel.tops")
	large := strings.Repeat("// analyzing padding\n", 50000) + document.Text
	document = fixtureDocument{URI: document.URI, Text: large}
	openDocument(t, process, document, 1)

	tileOffset := strings.Index(large, "tile_offset(rows")
	hoverID := "90"
	completionID := "91"
	definitionID := "92"
	referencesID := "93"
	symbolsID := "94"
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 90, "method": "textDocument/hover",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": sourceLSPPosition(large, tileOffset+2)},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 91, "method": "textDocument/completion",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": sourceLSPPosition(large, strings.Index(large, "  int lane = tile_offset")+2)},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 92, "method": "textDocument/definition",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": sourceLSPPosition(large, tileOffset+2)},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 93, "method": "textDocument/references",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}, "position": sourceLSPPosition(large, tileOffset+2), "context": map[string]any{"includeDeclaration": true}},
	})
	writeJSON(t, process.stdin, map[string]any{
		"jsonrpc": "2.0", "id": 94, "method": "textDocument/documentSymbol",
		"params": map[string]any{"textDocument": map[string]any{"uri": document.URI}},
	})

	navigationStarted := time.Now()
	collected := make(map[string]typedLSPResponse, 5)
	wanted := map[string]struct{}{hoverID: {}, completionID: {}, definitionID: {}, referencesID: {}, symbolsID: {}}
	for len(collected) < len(wanted) {
		response := readTypedResponse(t, process.reader)
		if _, ok := wanted[string(response.ID)]; ok {
			collected[string(response.ID)] = response
		}
	}
	navigationElapsed := time.Since(navigationStarted)

	var hover protocol.Hover
	if collected[hoverID].Error != nil || json.Unmarshal(collected[hoverID].Result, &hover) != nil || !strings.Contains(hover.Contents.Value, "Analyzing") {
		t.Fatalf("analyzing hover = %+v", collected[hoverID])
	}
	var items []protocol.CompletionItem
	if collected[completionID].Error != nil || json.Unmarshal(collected[completionID].Result, &items) != nil {
		t.Fatalf("analyzing completion = %+v", collected[completionID])
	}
	if len(items) != 1 || items[0].Kind != protocol.CompletionItemKindText || !strings.HasPrefix(items[0].Label, "Analyzing") {
		t.Fatalf("analyzing completion items = %+v", items)
	}

	for _, id := range []string{definitionID, referencesID, symbolsID} {
		response := collected[id]
		if response.Error != nil {
			if response.Error.Code != int(protocol.ContentModified) && response.Error.Code != int(protocol.RequestCancelled) {
				t.Fatalf("navigation during analysis error = %+v", response.Error)
			}
			continue
		}
		if len(response.Result) == 0 {
			t.Fatalf("navigation response has no result: %+v", response)
		}
	}
	if navigationElapsed > 4*time.Second {
		t.Fatalf("navigation wait exceeded the fixed limit: %s", navigationElapsed)
	}
	shutdownProcess(t, process)
}
