package server

import (
	"context"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
)

type testSourceProvider struct {
	roots []string
}

type recordingDiagnosticPublisher struct {
	params []protocol.PublishDiagnosticsParams
}

func (publisher *recordingDiagnosticPublisher) Publish(_ context.Context, params protocol.PublishDiagnosticsParams) error {
	publisher.params = append(publisher.params, params)
	return nil
}

func (provider testSourceProvider) OpenDocument(string) (index.SourceDocument, bool) {
	return index.SourceDocument{}, false
}

func (provider testSourceProvider) IncludeRoots() ([]string, bool) {
	return provider.roots, true
}

func (provider testSourceProvider) ResolveInclude(string, string) (index.SourceDocument, index.HeaderDependencyStatus) {
	return index.SourceDocument{}, index.HeaderDependencyMissing
}

func TestServerCopiesSourceProviderRootsIntoParseContext(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{SourceProvider: testSourceProvider{roots: []string{"/workspace/include"}}})
	state := document.DocumentState{URI: "file:///workspace/main.tops", LanguageID: "tops", Version: 1, Text: "int value;"}
	context := server.partialParseContextAt(state, 4)
	if len(context.IncludeRoots) != 1 || context.IncludeRoots[0] != "/workspace/include" || !context.IncludeRootsAvailable {
		t.Fatalf("context = %+v", context)
	}
}

func TestServerUsesAndUpdatesCompilationContext(t *testing.T) {
	initial := CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "compile_commands",
		PredefinedMacros:      map[string]parser.MacroValue{"FLAG": {Kind: parser.MacroDefined}},
		TargetProfile:         "gcu400",
		PassKind:              "device",
	}
	server := NewWithOptions(nil, ServerOptions{
		SourceProvider:     testSourceProvider{roots: []string{"/workspace/include"}},
		CompilationContext: &initial,
	})
	state := document.DocumentState{URI: "file:///workspace/context.tops", LanguageID: "tops", Version: 1, Text: "#if defined(FLAG)\nint active;\n#endif\n"}
	parseContext := server.partialParseContextAt(state, 1)
	if parseContext.CompilerContextStatus != "resolved" || parseContext.TargetProfile != "gcu400" || parseContext.PassKind != "device" || parseContext.PredefinedMacros["FLAG"].Kind != parser.MacroDefined {
		t.Fatalf("initial parse context = %+v", parseContext)
	}
	if _, err := server.openDocument(state.URI, state.LanguageID, state.Version, state.Text); err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	oldVersion := server.currentContextVersion()
	updated := initial
	updated.PredefinedMacros = map[string]parser.MacroValue{"FLAG": {Kind: parser.MacroUndefined}}
	updated.TargetProfile = "gcu500"
	updated.PassKind = "host"
	server.UpdateCompilationContext(updated)
	if server.currentContextVersion() != oldVersion+1 {
		t.Fatalf("context version = %d, want %d", server.currentContextVersion(), oldVersion+1)
	}
	updatedContext := server.partialParseContextAt(state, server.currentContextVersion())
	if updatedContext.TargetProfile != "gcu500" || updatedContext.PassKind != "host" || updatedContext.PredefinedMacros["FLAG"].Kind != parser.MacroUndefined {
		t.Fatalf("updated parse context = %+v", updatedContext)
	}
	snapshot, ok := server.symbolIndex.Get(state.URI)
	if !ok || snapshot.ContextVersion != server.currentContextVersion() || snapshot.TargetProfile != "gcu500" || snapshot.PassKind != "host" {
		t.Fatalf("updated snapshot = %+v, ok = %v", snapshot, ok)
	}
}

func TestServerRejectsIndexSnapshotForStaleDocumentVersion(t *testing.T) {
	server := New(nil)
	if err := server.documents.Open("file:///workspace/main.tops", "tops", 2, "int value;"); err != nil {
		t.Fatal(err)
	}
	snapshot := index.IndexSnapshot{
		RootURI:       "file:///workspace/main.tops",
		SourceVersion: index.SourceVersion{URI: "file:///workspace/main.tops", DocumentVersion: 1, ContextVersion: 1},
	}
	if server.commitIndexSnapshot(snapshot) {
		t.Fatal("stale snapshot was committed")
	}
}

func TestParseSnapshotCommitsIndexAndDiagnosticsForSameVersion(t *testing.T) {
	server := New(nil)
	state, err := server.openDocument("file:///workspace/analyzed.tops", "tops", 1, "int value() { return 1; }")
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingDiagnosticPublisher{}
	server.parseAndPublish(context.Background(), publisher, state)
	snapshot, ok := server.symbolIndex.Get(state.URI)
	if !ok || snapshot.SourceVersion.DocumentVersion != state.Version || snapshot.ContextVersion != server.currentContextVersion() {
		t.Fatalf("snapshot = %+v, ok = %v", snapshot, ok)
	}
	if len(publisher.params) != 1 || publisher.params[0].URI != state.URI || publisher.params[0].Version == nil || *publisher.params[0].Version != state.Version {
		t.Fatalf("published diagnostics = %+v", publisher.params)
	}
}

func TestPublishIndexDiagnosticsUsesStableCode(t *testing.T) {
	server := New(nil)
	state := document.DocumentState{URI: "file:///workspace/degraded.tops", Version: 1, Text: "int value;"}
	publisher := &recordingDiagnosticPublisher{}
	snapshot := index.IndexSnapshot{
		RootURI:        state.URI,
		ContextVersion: 1,
		Diagnostics:    []index.IndexDiagnostic{{Code: "analysis-degraded", Message: "header is missing"}},
	}
	if err := server.publishIndexDiagnostics(context.Background(), publisher, state, snapshot); err != nil {
		t.Fatal(err)
	}
	if len(publisher.params) != 1 || len(publisher.params[0].Diagnostics) != 1 || publisher.params[0].Diagnostics[0].Code != "analysis-degraded" {
		t.Fatalf("published index diagnostics = %+v", publisher.params)
	}
}

func TestPublishIndexDiagnosticsUsesSourceURIAndText(t *testing.T) {
	server := New(nil)
	rootURI := "file:///workspace/main.tops"
	headerURI := "file:///workspace/include/api.h"
	state := document.DocumentState{URI: rootURI, Version: 1, Text: "int root;"}
	publisher := &recordingDiagnosticPublisher{}
	snapshot := index.IndexSnapshot{
		RootURI:        rootURI,
		ContextVersion: 1,
		Sources:        map[string]string{rootURI: state.Text, headerURI: "int header;"},
		Diagnostics:    []index.IndexDiagnostic{{SourceURI: headerURI, Code: "analysis-degraded", Message: "header is missing", Range: parser.SourceRange{Start: 0, End: 3}}},
	}
	if err := server.publishIndexDiagnostics(context.Background(), publisher, state, snapshot); err != nil {
		t.Fatal(err)
	}
	if len(publisher.params) != 2 {
		t.Fatalf("source diagnostic publication = %+v", publisher.params)
	}
	var headerDiagnostics protocol.PublishDiagnosticsParams
	for _, params := range publisher.params {
		if params.URI == headerURI {
			headerDiagnostics = params
		}
	}
	if len(headerDiagnostics.Diagnostics) != 1 || headerDiagnostics.Diagnostics[0].Range.End.Character != 3 {
		t.Fatalf("header diagnostic publication = %+v", publisher.params)
	}
}

func TestParseAndPublishMergesIndexAndParserDiagnostics(t *testing.T) {
	compilationContext := CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "test",
		TargetProfile:         "gcu400",
		PassKind:              "device",
	}
	server := NewWithOptions(nil, ServerOptions{
		SourceProvider:     testSourceProvider{roots: []string{"/workspace/include"}},
		CompilationContext: &compilationContext,
	})
	state, err := server.openDocument("file:///workspace/degraded.tops", "tops", 1, "#include \"missing.h\"\nint broken(\n")
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingDiagnosticPublisher{}
	server.parseAndPublish(context.Background(), publisher, state)
	if len(publisher.params) != 1 || len(publisher.params[0].Diagnostics) < 2 {
		t.Fatalf("merged diagnostics = %+v", publisher.params)
	}
	seenIndex := false
	seenParser := false
	for _, diagnostic := range publisher.params[0].Diagnostics {
		seenIndex = seenIndex || diagnostic.Code == "analysis-degraded"
		seenParser = seenParser || diagnostic.Code == parser.DiagnosticMissingToken
	}
	if !seenIndex || !seenParser {
		t.Fatalf("merged diagnostics codes = %+v", publisher.params[0].Diagnostics)
	}
}
