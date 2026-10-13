package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tops-lsp/internal/parser"
)

type verifiedSymbolProvider struct {
	SourceProvider
	attributeSource SymbolLocation
	builtinSource   SymbolLocation
}

func (provider verifiedSymbolProvider) ResolveAttribute(string, string) (SymbolLocation, bool) {
	return provider.attributeSource, provider.attributeSource.URI != ""
}

func (provider verifiedSymbolProvider) ResolveBuiltin(string, string) (SymbolLocation, bool) {
	return provider.builtinSource, provider.builtinSource.URI != ""
}

func (provider verifiedSymbolProvider) ResolveAttributeSource(SourceResolutionContext, string) (SymbolLocation, bool) {
	return provider.attributeSource, provider.attributeSource.URI != ""
}

func (provider verifiedSymbolProvider) ResolveBuiltinSource(SourceResolutionContext, string) (SymbolLocation, bool) {
	return provider.builtinSource, provider.builtinSource.URI != ""
}

func TestBuildIndexesLocalSymbolsAndFunctionMetadata(t *testing.T) {
	source := `namespace demo { using Value = int; struct Point { int x; }; __device__ int add(int left, int right) { return left + right; } int use() { Point point; return add(point.x, 2); } }`
	context := resolvedParseContext("file:///workspace/main.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)

	if snapshot.IndexStatus != IndexStatusComplete {
		t.Fatalf("index status = %q, diagnostics = %+v", snapshot.IndexStatus, snapshot.Diagnostics)
	}
	point := symbolByName(snapshot, "Point")
	if point == nil || point.Kind != SymbolKindType || point.TypeKind != "struct" {
		t.Fatalf("point symbol = %+v", point)
	}
	add := symbolByName(snapshot, "add")
	if add == nil || add.Kind != SymbolKindFunction {
		t.Fatalf("add symbol = %+v", add)
	}
	if add.ParameterSummary == "" || add.ReturnTypeSummary != "int" || len(add.ExecutionQualifiers) != 1 {
		t.Fatalf("add metadata = %+v", add)
	}
	value := symbolByName(snapshot, "Value")
	if value == nil || value.TypeKind != "alias" {
		t.Fatalf("alias symbol = %+v", value)
	}
}

func TestBuildTracksReferencesOutsideDeclarations(t *testing.T) {
	source := `int add(int left, int right) { return left + right; } int use() { return add(1, 2); }`
	context := resolvedParseContext("file:///workspace/main.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)
	add := symbolByName(snapshot, "add")
	if add == nil {
		t.Fatal("add symbol not found")
	}
	if len(add.References) == 0 {
		t.Fatalf("add references = %+v", add)
	}
}

func TestBuildWithProviderMergesHeaderSymbols(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int api_add(int left, int right) { return left + right; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := `#include "api.h"
int use() { return api_add(1, 2); }
`
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	provider := NewLocalSourceProvider(nil, []string{root})
	result := parser.Parse(source, context)
	snapshot := BuildWithProvider(source, result, context, provider)
	apiAdd := symbolByName(snapshot, "api_add")
	if apiAdd == nil || len(apiAdd.Definitions) != 1 {
		t.Fatalf("header symbol = %+v", apiAdd)
	}
	if apiAdd.Definitions[0].URI != fileURI(headerPath) {
		t.Fatalf("header definition = %+v", apiAdd.Definitions[0])
	}
	if len(snapshot.Dependencies) == 0 || snapshot.Dependencies[0].IncludeForm != IncludeFormQuoted {
		t.Fatalf("header include form = %+v", snapshot.Dependencies)
	}
	callOffset := sourceOffset(source, "api_add", 0)
	definitions := snapshot.DefinitionAt(callOffset)
	if len(definitions) != 1 || definitions[0].URI != fileURI(headerPath) {
		t.Fatalf("cross-file definitions = %+v", definitions)
	}
}

func TestBuildWithProviderKeepsHeaderReferences(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int api_value(); int header_use() { return api_value(); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := "#include \"api.h\"\nint use() { return header_use(); }\n"
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	provider := NewLocalSourceProvider(nil, []string{root})
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	headerURI := fileURI(headerPath)
	found := false
	for _, reference := range snapshot.References {
		if reference.URI == headerURI && reference.Name == "api_value" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("header references = %+v", snapshot.References)
	}
}

func TestBuildWithProviderKeepsHeaderConditionRegionsByURI(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "conditions.h")
	if err := os.WriteFile(headerPath, []byte("#if 0\nint hidden_header;\n#endif\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := "#include \"conditions.h\"\nint use() { return 0; }\n"
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(nil, []string{root}))
	headerURI := fileURI(headerPath)
	if len(snapshot.ConditionalRegionsByURI[headerURI]) == 0 {
		t.Fatalf("header condition regions = %+v", snapshot.ConditionalRegionsByURI)
	}
	hidden := symbolByName(snapshot, "hidden_header")
	if hidden == nil || hidden.Condition.SourceURI != headerURI || hidden.Condition.State != parser.ConditionalInactive {
		t.Fatalf("header condition provenance = %+v", hidden)
	}
}

func TestBuildLeavesUnverifiedBuiltinHeaderUnresolved(t *testing.T) {
	context := resolvedParseContext("file:///workspace/unverified.h")
	context.LanguageID = "tops-header"
	source := "int threadIdx;"
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(nil, nil))
	builtin := symbolByName(snapshot, "threadIdx")
	if builtin == nil || builtin.Kind != SymbolKindBuiltin || builtin.Builtin.Availability != "unresolved" {
		t.Fatalf("unverified builtin = %+v", builtin)
	}
	if definitions := snapshot.DefinitionAt(builtin.NameRange.Start); len(definitions) != 0 {
		t.Fatalf("unverified builtin definition = %+v", definitions)
	}
}

func TestBuildResolvesBuiltinFromLocalHeaderProvider(t *testing.T) {
	root := t.TempDir()
	headerURI := fileURI(filepath.Join(root, "builtins.h"))
	source := "int threadIdx;\n"
	context := resolvedParseContext(headerURI)
	context.LanguageID = "tops-header"
	provider := NewLocalSourceProviderWithOptions(map[string]SourceDocument{
		headerURI: {URI: headerURI, Text: source, Version: 7},
	}, nil, LocalSourceProviderOptions{TrustedHeaderURIs: []string{headerURI}})
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	builtin := symbolByName(snapshot, "threadIdx")
	if builtin == nil || builtin.Builtin.Availability != "available" || builtin.Builtin.SourceURI != headerURI {
		t.Fatalf("local builtin provenance = %+v", builtin)
	}
	if definitions := snapshot.DefinitionAtURI(headerURI, builtin.NameRange.Start); len(definitions) != 1 || definitions[0].URI != headerURI || definitions[0].Role != SymbolLocationBuiltin {
		t.Fatalf("local builtin definition = %+v", definitions)
	}
}

func TestBuildWithProviderMarksIncludeCycleDegraded(t *testing.T) {
	root := t.TempDir()
	aPath := filepath.Join(root, "a.h")
	bPath := filepath.Join(root, "b.h")
	if err := os.WriteFile(aPath, []byte("#include \"b.h\"\nstruct A { int value; };\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("#include \"a.h\"\nstruct B { int value; };\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := "#include \"a.h\"\nint use() { return 0; }\n"
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	provider := NewLocalSourceProvider(nil, []string{root})
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	if snapshot.IndexStatus != IndexStatusDegraded {
		t.Fatalf("index status = %q, dependencies = %+v", snapshot.IndexStatus, snapshot.Dependencies)
	}
	cycleFound := false
	for _, dependency := range snapshot.Dependencies {
		if dependency.Status == HeaderDependencyCycle {
			cycleFound = true
		}
	}
	if !cycleFound || symbolByName(snapshot, "A") == nil || symbolByName(snapshot, "B") == nil {
		t.Fatalf("cycle dependencies/symbols = %+v, symbols = %+v", snapshot.Dependencies, snapshot.Symbols)
	}
}

func TestApplyFindingStatusAddsRootDiagnosticForHeaderDegradedState(t *testing.T) {
	snapshot := IndexSnapshot{
		RootURI:     "file:///workspace/main.tops",
		IndexStatus: IndexStatusDegraded,
		Diagnostics: []IndexDiagnostic{{SourceURI: "file:///workspace/api.h", Code: "analysis-degraded", Reason: "unknown-condition"}},
	}
	applyFindingStatus(&snapshot)
	foundRoot := false
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.SourceURI == snapshot.RootURI && diagnostic.Code == "analysis-degraded" {
			foundRoot = true
		}
	}
	if !foundRoot {
		t.Fatalf("root degraded diagnostic = %+v", snapshot.Diagnostics)
	}
}

func resolvedParseContext(uri string) parser.ParseContext {
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

func symbolByName(snapshot IndexSnapshot, name string) *SymbolRecord {
	for _, symbol := range snapshot.Symbols {
		if symbol.Name == name {
			copy := symbol
			return &copy
		}
	}
	return nil
}

func sourceOffset(source, spelling string, occurrence int) int {
	start := 0
	for index := 0; index <= occurrence; index++ {
		offset := strings.Index(source[start:], spelling)
		if offset < 0 {
			return -1
		}
		start += offset
		if index == occurrence {
			return start
		}
		start += len(spelling)
	}
	return -1
}
