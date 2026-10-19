package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
)

type verifiedServerSourceProvider struct {
	attributeSource index.SymbolLocation
	builtinSource   index.SymbolLocation
}

func (provider verifiedServerSourceProvider) OpenDocument(string) (index.SourceDocument, bool) {
	return index.SourceDocument{}, false
}

func (provider verifiedServerSourceProvider) IncludeRoots() ([]string, bool) {
	return nil, true
}

func (provider verifiedServerSourceProvider) ResolveInclude(string, string) (index.SourceDocument, index.HeaderDependencyStatus) {
	return index.SourceDocument{}, index.HeaderDependencyUnresolved
}

func (provider verifiedServerSourceProvider) ResolveAttribute(string, string) (index.SymbolLocation, bool) {
	return provider.attributeSource, true
}

func (provider verifiedServerSourceProvider) ResolveBuiltin(string, string) (index.SymbolLocation, bool) {
	return provider.builtinSource, true
}

func (provider verifiedServerSourceProvider) ResolveAttributeSource(index.SourceResolutionContext, string) (index.SymbolLocation, bool) {
	return provider.attributeSource, true
}

func (provider verifiedServerSourceProvider) ResolveBuiltinSource(index.SourceResolutionContext, string) (index.SymbolLocation, bool) {
	return provider.builtinSource, true
}

type legacyServerSourceProvider struct {
	attributeSource index.SymbolLocation
	builtinSource   index.SymbolLocation
}

func (provider legacyServerSourceProvider) OpenDocument(string) (index.SourceDocument, bool) {
	return index.SourceDocument{}, false
}

func (provider legacyServerSourceProvider) IncludeRoots() ([]string, bool) {
	return nil, true
}

func (provider legacyServerSourceProvider) ResolveInclude(string, string) (index.SourceDocument, index.HeaderDependencyStatus) {
	return index.SourceDocument{}, index.HeaderDependencyUnresolved
}

func (provider legacyServerSourceProvider) ResolveAttribute(string, string) (index.SymbolLocation, bool) {
	return provider.attributeSource, true
}

func (provider legacyServerSourceProvider) ResolveBuiltin(string, string) (index.SymbolLocation, bool) {
	return provider.builtinSource, true
}

func TestHeaderChangeInvalidatesDependentIndex(t *testing.T) {
	server := New(nil)
	headerURI := "file:///workspace/include/api.h"
	rootURI := "file:///workspace/main.tops"
	if !server.symbolIndex.Replace(index.IndexSnapshot{
		RootURI:       rootURI,
		SourceVersion: index.SourceVersion{URI: rootURI, DocumentVersion: 1, ContextVersion: 1},
		Dependencies:  []index.HeaderDependency{{RequesterURI: rootURI, TargetURI: headerURI, Status: index.HeaderDependencyResolved}},
	}) {
		t.Fatal("dependent index was not stored")
	}
	stale := server.symbolIndex.MarkHeaderStale(headerURI)
	if len(stale) != 1 || stale[0] != rootURI {
		t.Fatalf("stale = %v", stale)
	}
}

func TestServerSourceProviderInvalidatesIndirectHeaderSnapshotCache(t *testing.T) {
	server := New(nil)
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		t.Fatalf("source provider type = %T", server.sourceProvider)
	}
	childURI := "file:///workspace/include/child.h"
	parentURI := "file:///workspace/include/parent.h"
	rootURI := "file:///workspace/main.tops"
	parentText := "#include \"child.h\"\nint parent_value;\n"
	rootText := "#include \"parent.h\"\nint root_value;\n"
	provider.PutHeaderSnapshot(index.IndexSnapshot{
		RootURI:        parentURI,
		ContextVersion: 1,
		SourceVersion:  index.SourceVersion{URI: parentURI, SourceVersion: index.SourceDigest(parentText)},
		Sources:        map[string]string{parentURI: parentText},
		Dependencies:   []index.HeaderDependency{{RequesterURI: parentURI, TargetURI: childURI, Status: index.HeaderDependencyResolved}},
	})
	provider.PutHeaderSnapshot(index.IndexSnapshot{
		RootURI:        rootURI,
		ContextVersion: 1,
		SourceVersion:  index.SourceVersion{URI: rootURI, SourceVersion: index.SourceDigest(rootText)},
		Sources:        map[string]string{rootURI: rootText},
		Dependencies:   []index.HeaderDependency{{RequesterURI: rootURI, TargetURI: parentURI, Status: index.HeaderDependencyResolved}},
	})
	provider.invalidateHeaderSnapshots(childURI)
	if _, ok := provider.GetHeaderSnapshot(parentURI, index.SourceDigest(parentText), 1); ok {
		t.Fatal("parent header cache remained after child invalidation")
	}
	if _, ok := provider.GetHeaderSnapshot(rootURI, index.SourceDigest(rootText), 1); ok {
		t.Fatal("root header cache remained after indirect child invalidation")
	}
}

func TestServerSourceProviderUsesCurrentOpenedHeaderVersion(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{SourceProvider: testSourceProvider{roots: []string{"/workspace/include"}}})
	headerURI := "file:///workspace/include/api.h"
	if err := server.documents.Open(headerURI, "tops-header", 1, "int old_value;"); err != nil {
		t.Fatal(err)
	}
	source, ok := server.sourceProvider.OpenDocument(headerURI)
	if !ok || source.Version != 1 || source.Text != "int old_value;" {
		t.Fatalf("source = %+v, ok = %v", source, ok)
	}
	if err := server.documents.Change(headerURI, 2, []document.Change{{Range: &document.Range{Start: document.Position{Line: 0, Character: 4}, End: document.Position{Line: 0, Character: 7}}, Text: "new"}}); err != nil {
		t.Fatal(err)
	}
	source, ok = server.sourceProvider.OpenDocument(headerURI)
	if !ok || source.Version != 2 || source.Text != "int new_value;" {
		t.Fatalf("updated source = %+v, ok = %v", source, ok)
	}
}

func TestServerSourceProviderPreservesCompleteSourceVersion(t *testing.T) {
	attributeURI := "file:///workspace/attributes.h"
	builtinURI := "file:///workspace/builtins.h"
	server := NewWithOptions(nil, ServerOptions{SourceProvider: verifiedServerSourceProvider{
		attributeSource: index.SymbolLocation{URI: attributeURI, Range: parser.SourceRange{Start: 0, End: 14}, SelectionRange: parser.SourceRange{Start: 2, End: 12}},
		builtinSource:   index.SymbolLocation{URI: builtinURI, Range: parser.SourceRange{Start: 0, End: 14}, SelectionRange: parser.SourceRange{Start: 4, End: 13}},
	}})
	if _, err := server.openDocument(attributeURI, "tops-header", 4, "[[deprecated]]"); err != nil {
		t.Fatal(err)
	}
	if _, err := server.openDocument(builtinURI, "tops-header", 5, "int threadIdx;"); err != nil {
		t.Fatal(err)
	}
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		t.Fatalf("source provider type = %T", server.sourceProvider)
	}
	attribute, found := provider.ResolveAttributeSource(index.SourceResolutionContext{RequesterURI: "file:///workspace/main.tops", ContextVersion: 7, TargetProfile: "gcu400", PassKind: "device"}, "deprecated")
	if !found || attribute.URI != attributeURI || attribute.SourceVersion.URI != attributeURI || attribute.SourceVersion.DocumentVersion != 4 || attribute.SourceVersion.ContextVersion != 7 || attribute.SourceVersion.SourceVersion != index.SourceDigest("[[deprecated]]") {
		t.Fatalf("attribute source version = %+v, found = %v", attribute, found)
	}
	builtin, found := provider.ResolveBuiltinSource(index.SourceResolutionContext{RequesterURI: "file:///workspace/main.tops", ContextVersion: 7, TargetProfile: "gcu400", PassKind: "device"}, "threadIdx")
	if !found || builtin.URI != builtinURI || builtin.SourceVersion.URI != builtinURI || builtin.SourceVersion.DocumentVersion != 5 || builtin.SourceVersion.ContextVersion != 7 || builtin.SourceVersion.SourceVersion != index.SourceDigest("int threadIdx;") {
		t.Fatalf("builtin source version = %+v, found = %v", builtin, found)
	}
}

func TestServerSourceProviderResolvesRequesterRelativeAndRejectsAmbiguousHeaders(t *testing.T) {
	server := New(nil)
	localURI := "file:///workspace/src/api.h"
	otherURI := "file:///workspace/other/api.h"
	if err := server.documents.Open(localURI, "tops-header", 1, "int local_api;"); err != nil {
		t.Fatal(err)
	}
	if err := server.documents.Open(otherURI, "tops-header", 1, "int other_api;"); err != nil {
		t.Fatal(err)
	}
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		t.Fatalf("source provider type = %T", server.sourceProvider)
	}
	source, status := provider.ResolveInclude("file:///workspace/src/main.tops", "api.h")
	if status != index.HeaderDependencyResolved || source.URI != localURI {
		t.Fatalf("requester-relative source = %+v, status = %q", source, status)
	}
	if _, status := provider.ResolveInclude("file:///workspace/unrelated/main.tops", "api.h"); status != index.HeaderDependencyUnresolved {
		t.Fatalf("ambiguous source status = %q", status)
	}
}

func TestServerSourceProviderCombinesOpenedAndIncludeRootRules(t *testing.T) {
	root := t.TempDir()
	rootHeader := filepath.Join(root, "api.h")
	if err := os.WriteFile(rootHeader, []byte("int root_api;"), 0o644); err != nil {
		t.Fatal(err)
	}
	server := NewWithOptions(nil, ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, []string{root})})
	if err := server.documents.Open("file:///workspace/other/api.h", "tops-header", 1, "int other_api;"); err != nil {
		t.Fatal(err)
	}
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		t.Fatalf("source provider type = %T", server.sourceProvider)
	}
	requester := "file:///workspace/src/main.tops"
	source, status := provider.ResolveInclude(requester, "api.h")
	if status != index.HeaderDependencyResolved || source.URI != index.FileURI(rootHeader) {
		t.Fatalf("include-root source = %+v, status = %q", source, status)
	}
	if err := server.documents.Open("file:///workspace/third/api.h", "tops-header", 1, "int third_api;"); err != nil {
		t.Fatal(err)
	}
	if _, status := provider.ResolveInclude(requester, "api.h"); status != index.HeaderDependencyUnresolved {
		t.Fatalf("opened ambiguity status = %q", status)
	}
}

func TestServerSourceProviderSyncsOpenedHeaderIntoLocalResolver(t *testing.T) {
	root := t.TempDir()
	headerURI := index.FileURI(filepath.Join(root, "builtins.h"))
	mainURI := index.FileURI(filepath.Join(root, "main.tops"))
	delegate := index.NewLocalSourceProviderWithOptions(nil, nil, index.LocalSourceProviderOptions{TrustedHeaderURIs: []string{headerURI}})
	server := NewWithOptions(nil, ServerOptions{
		SourceProvider:     delegate,
		CompilationContext: &CompilationContext{CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "device"},
	})
	if _, err := server.openDocument(headerURI, "tops-header", 1, "int threadIdx;\n"); err != nil {
		t.Fatal(err)
	}
	mainState, err := server.openDocument(mainURI, "tops", 1, "#include \"builtins.h\"\nint use() { return threadIdx; }\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, mainState)
	snapshot, ok := server.symbolIndex.Get(mainURI)
	if !ok {
		t.Fatal("main index was not stored")
	}
	builtin := index.SymbolRecord{}
	for _, symbol := range snapshot.Symbols {
		if symbol.Name == "threadIdx" {
			builtin = symbol
			break
		}
	}
	if builtin.Builtin.Availability != "available" || builtin.Builtin.SourceURI != headerURI {
		t.Fatalf("opened header builtin = %+v", builtin)
	}
}

func TestServerSourceProviderForwardsVerifiedSymbolResolvers(t *testing.T) {
	attributeURI := "file:///workspace/attributes.h"
	builtinURI := "file:///workspace/builtins.h"
	server := NewWithOptions(nil, ServerOptions{SourceProvider: verifiedServerSourceProvider{
		attributeSource: index.SymbolLocation{URI: attributeURI, Range: parser.SourceRange{Start: 0, End: len("[[deprecated]]")}, SelectionRange: parser.SourceRange{Start: 0, End: len("[[deprecated]]")}},
		builtinSource:   index.SymbolLocation{URI: builtinURI, Range: parser.SourceRange{Start: 0, End: len("int threadIdx;")}, SelectionRange: parser.SourceRange{Start: 0, End: len("int threadIdx;")}},
	}, CompilationContext: &CompilationContext{LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "device"}})
	if _, err := server.openDocument(attributeURI, "tops-header", 1, "[[deprecated]]"); err != nil {
		t.Fatal(err)
	}
	if _, err := server.openDocument(builtinURI, "tops-header", 1, "int threadIdx;"); err != nil {
		t.Fatal(err)
	}
	attributeState, err := server.openDocument("file:///workspace/main.tops", "tops", 1, "[[deprecated]] int value;")
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingDiagnosticPublisher{}
	server.parseAndPublish(context.Background(), publisher, attributeState)
	attributeSnapshot, ok := server.symbolIndex.Get(attributeState.URI)
	if !ok || len(attributeSnapshot.Symbols) == 0 {
		t.Fatalf("attribute snapshot = %+v, ok = %v", attributeSnapshot, ok)
	}
	attribute := attributeSnapshot.Symbols
	foundAttribute := false
	for _, symbol := range attribute {
		if symbol.Kind == index.SymbolKindAttribute && len(symbol.AttributeSources) == 1 && symbol.AttributeSources[0].URI == attributeURI {
			foundAttribute = true
		}
	}
	if !foundAttribute {
		t.Fatalf("verified attribute source was not forwarded: %+v", attributeSnapshot.Symbols)
	}

	builtinState, ok := server.documents.Get(builtinURI)
	if !ok {
		t.Fatalf("builtin document lookup = %+v", builtinState)
	}
	server.parseAndPublish(context.Background(), publisher, builtinState)
	builtinSnapshot, ok := server.symbolIndex.Get(builtinURI)
	if !ok {
		t.Fatalf("builtin snapshot missing = %+v", builtinSnapshot)
	}
	builtin := builtinSnapshot.Symbols
	foundBuiltin := false
	for _, symbol := range builtin {
		if symbol.Kind == index.SymbolKindBuiltin && symbol.Builtin.Availability == "available" && symbol.Builtin.SourceURI == builtinURI {
			foundBuiltin = true
		}
	}
	if !foundBuiltin {
		t.Fatalf("verified builtin source was not forwarded: %+v", builtin)
	}
}

func TestInvalidateHeaderDependentsRemovesAndRequeuesCurrentDocuments(t *testing.T) {
	server := New(nil)
	headerURI := "file:///workspace/include/api.h"
	rootURI := "file:///workspace/main.tops"
	if err := server.documents.Open(rootURI, "tops", 1, "int use();"); err != nil {
		t.Fatal(err)
	}
	if !server.symbolIndex.Replace(index.IndexSnapshot{
		RootURI:       rootURI,
		SourceVersion: index.SourceVersion{URI: rootURI, DocumentVersion: 1, ContextVersion: 1},
		Dependencies:  []index.HeaderDependency{{RequesterURI: rootURI, TargetURI: headerURI, Status: index.HeaderDependencyResolved}},
	}) {
		t.Fatal("dependent index was not stored")
	}
	server.invalidateHeaderDependents(context.Background(), nil, headerURI, false)
	snapshot, ok := server.symbolIndex.Get(rootURI)
	if !ok || snapshot.SourceVersion.DocumentVersion != 1 {
		t.Fatalf("dependent index was not rebuilt: %+v, ok = %v", snapshot, ok)
	}
}

func TestHeaderDidCloseClearsOverlayBeforeDependentReparse(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int disk_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	headerURI := index.FileURI(headerPath)
	mainURI := index.FileURI(filepath.Join(root, "main.tops"))
	server := NewWithOptions(nil, ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, []string{root})})
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if _, err := server.openDocument(headerURI, "tops-header", 1, "int overlay_value;\n"); err != nil {
		t.Fatal(err)
	}
	mainState, err := server.openDocument(mainURI, "tops", 1, "#include \"api.h\"\nint use() { return overlay_value; }\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, mainState)
	initial, ok := server.symbolIndex.Get(mainURI)
	if !ok || symbolByNameForServer(initial, "overlay_value") == nil {
		t.Fatalf("initial overlay snapshot = %+v, ok = %v", initial, ok)
	}
	if err := os.WriteFile(headerPath, []byte("int disk_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	server.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didClose", protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: headerURI}}), nil)
	updated, ok := server.symbolIndex.Get(mainURI)
	if !ok || symbolByNameForServer(updated, "disk_value") == nil || symbolByNameForServer(updated, "overlay_value") != nil {
		t.Fatalf("post-close overlay snapshot = %+v, ok = %v", updated, ok)
	}
}

func TestIndirectHeaderCloseRefreshesParentAndRootSnapshots(t *testing.T) {
	root := t.TempDir()
	childPath := filepath.Join(root, "child.h")
	parentPath := filepath.Join(root, "parent.h")
	mainPath := filepath.Join(root, "main.tops")
	if err := os.WriteFile(childPath, []byte("int disk_child;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parentPath, []byte("#include \"child.h\"\nint parent_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainText := "#include \"parent.h\"\nint use() { return overlay_child; }\n"
	mainURI := index.FileURI(mainPath)
	childURI := index.FileURI(childPath)
	server := NewWithOptions(nil, ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, []string{root}), CompilationContext: &CompilationContext{
		LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "host",
	}})
	if !server.tryInitialize() {
		t.Fatal("server did not initialize")
	}
	if _, err := server.openDocument(childURI, "tops-header", 1, "int overlay_child;\n"); err != nil {
		t.Fatal(err)
	}
	mainState, err := server.openDocument(mainURI, "tops", 1, mainText)
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, mainState)
	initial, ok := server.symbolIndex.Get(mainURI)
	if !ok || symbolByNameForServer(initial, "overlay_child") == nil {
		t.Fatalf("initial indirect header snapshot = %+v, ok = %v", initial, ok)
	}
	if err := os.WriteFile(childPath, []byte("int disk_child;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	server.handleNotificationWithPublisher(context.Background(), staleNotification(t, "textDocument/didClose", protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: childURI}}), nil)
	updated, ok := server.symbolIndex.Get(mainURI)
	if !ok || symbolByNameForServer(updated, "disk_child") == nil || symbolByNameForServer(updated, "overlay_child") != nil {
		t.Fatalf("updated indirect header snapshot = %+v, ok = %v", updated, ok)
	}
}

func TestLegacySourceResolversRequireCompleteProvenance(t *testing.T) {
	headerURI := "file:///workspace/builtins.h"
	server := NewWithOptions(nil, ServerOptions{SourceProvider: legacyServerSourceProvider{
		attributeSource: index.SymbolLocation{URI: headerURI, Range: parser.SourceRange{Start: 0, End: 14}, SelectionRange: parser.SourceRange{Start: 0, End: 14}},
		builtinSource:   index.SymbolLocation{URI: headerURI, Range: parser.SourceRange{Start: 0, End: 14}, SelectionRange: parser.SourceRange{Start: 4, End: 13}},
	}})
	if _, err := server.openDocument(headerURI, "tops-header", 1, "int threadIdx;\n"); err != nil {
		t.Fatal(err)
	}
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		t.Fatalf("source provider type = %T", server.sourceProvider)
	}
	base := index.SourceResolutionContext{RequesterURI: "file:///workspace/main.tops", ContextVersion: 1, TargetProfile: "gcu400", PassKind: "device", MacroContext: map[string]parser.MacroValue{"ARCH": {Kind: parser.MacroInteger, Value: 400}}, AllowedSourceURIs: []string{headerURI}}
	cases := []struct {
		name   string
		mutate func(*index.SourceResolutionContext)
	}{
		{name: "untrusted-header", mutate: func(context *index.SourceResolutionContext) { context.AllowedSourceURIs = nil }},
		{name: "missing-target", mutate: func(context *index.SourceResolutionContext) { context.TargetProfile = "" }},
		{name: "missing-pass", mutate: func(context *index.SourceResolutionContext) { context.PassKind = "" }},
		{name: "missing-macros", mutate: func(context *index.SourceResolutionContext) { context.MacroContext = nil }},
		{name: "missing-context-version", mutate: func(context *index.SourceResolutionContext) { context.ContextVersion = 0 }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			context := base
			testCase.mutate(&context)
			if _, found := provider.ResolveAttributeSource(context, "deprecated"); found {
				t.Fatal("legacy attribute source was verified without complete provenance")
			}
			if _, found := provider.ResolveBuiltinSource(context, "threadIdx"); found {
				t.Fatal("legacy builtin source was verified without complete provenance")
			}
		})
	}
}

func symbolByNameForServer(snapshot index.IndexSnapshot, name string) *index.SymbolRecord {
	for _, symbol := range snapshot.Symbols {
		if symbol.Name == name {
			copy := symbol
			return &copy
		}
	}
	return nil
}
