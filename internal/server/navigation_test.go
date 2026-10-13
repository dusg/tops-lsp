package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
)

func TestNavigationRequestReturnsLocalDefinition(t *testing.T) {
	uri := "file:///workspace/main.tops"
	source := "int add(int left, int right) { return left + right; } int use() { return add(1, 2); }"
	server := New(nil)
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if err := server.documents.Open(uri, "tops", 1, source); err != nil {
		t.Fatal(err)
	}
	parseContext := navigationParseContext(uri)
	result := parser.Parse(source, parseContext)
	if !server.commitIndexSnapshot(index.Build(source, result, parseContext)) {
		t.Fatal("index snapshot was not committed")
	}

	params, err := json.Marshal(protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Position:     protocol.Position{Line: 0, Character: 73},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := server.handleRequest(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
	var response struct {
		Result []protocol.Location   `json:"result"`
		Error  *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || len(response.Result) != 1 {
		t.Fatalf("response = %+v", response)
	}
	if response.Result[0].URI != uri {
		t.Fatalf("location = %+v", response.Result[0])
	}
}

func TestNavigationEncodesHeaderLocationWithHeaderSource(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int api_add();\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rootURI := "file://" + filepath.ToSlash(filepath.Join(root, "main.tops"))
	headerURI := "file://" + filepath.ToSlash(headerPath)
	source := "#include \"api.h\"\nint use() { return api_add(); }\n"
	server := NewWithOptions(nil, ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, []string{root})})
	if err := server.documents.Open(rootURI, "tops", 1, source); err != nil {
		t.Fatal(err)
	}
	parseContext := navigationParseContext(rootURI)
	parseContext.IncludeRoots = []string{root}
	snapshot := index.BuildWithProvider(source, parser.Parse(source, parseContext), parseContext, server.sourceProvider)
	if !server.commitIndexSnapshot(snapshot) {
		t.Fatal("header-aware snapshot was not committed")
	}
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	offset := strings.Index(source, "api_add")
	params, err := json.Marshal(protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: rootURI}, Position: protocol.Position{Line: 1, Character: offset - strings.LastIndex(source[:offset], "\n") - 1}})
	if err != nil {
		t.Fatal(err)
	}
	body := server.handleRequest(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "textDocument/definition", Params: params})
	var response struct {
		Result []protocol.Location `json:"result"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Result) != 1 || response.Result[0].URI != headerURI || response.Result[0].Range.Start.Line != 0 {
		t.Fatalf("header definition response = %+v", response)
	}
}

func navigationParseContext(uri string) parser.ParseContext {
	return parser.ParseContext{
		DocumentURI:           uri,
		LanguageID:            "tops",
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "compile_commands",
		TargetProfile:         "gcu400",
		PassKind:              "device",
		IncludeRoots:          []string{"/workspace/include"},
		IncludeRootsAvailable: true,
		DocumentVersion:       1,
		ContextVersion:        1,
	}
}

var _ = document.DocumentState{}
