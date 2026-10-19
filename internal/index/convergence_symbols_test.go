package index

import (
	"path/filepath"
	"strings"
	"testing"

	"tops-lsp/internal/parser"
)

func TestBuildClassifiesAttributesFieldsMacrosAndHeaderOrigins(t *testing.T) {
	source := "#define BUILD_FLAG 1\nstruct Point { int x; }; [[deprecated]] int make_point();"
	context := resolvedParseContext("file:///workspace/symbols.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	if !hasSymbolKind(snapshot, SymbolKindMacro, "BUILD_FLAG") {
		t.Fatalf("macro symbols = %+v", snapshot.Symbols)
	}
	if !hasSymbolKind(snapshot, SymbolKindField, "x") {
		t.Fatalf("field symbols = %+v", snapshot.Symbols)
	}
	if !hasSymbolKind(snapshot, SymbolKindAttribute, "deprecated") && !hasSymbolKind(snapshot, SymbolKindAttribute, "[[") {
		t.Fatalf("attribute symbols = %+v", snapshot.Symbols)
	}

	headerContext := resolvedParseContext("file:///workspace/include/api.h")
	headerContext.LanguageID = "tops-header"
	headerSource := "int api_value();"
	headerSnapshot := Build(headerSource, parser.Parse(headerSource, headerContext), headerContext)
	for _, symbol := range headerSnapshot.Symbols {
		if symbol.OriginKind != "header" {
			t.Fatalf("header symbol origin = %+v", symbol)
		}
	}
}

func TestBuildMergesDeclarationAndDefinitionLocations(t *testing.T) {
	source := "int add(int left, int right); int add(int left, int right) { return left + right; }"
	context := resolvedParseContext("file:///workspace/merge.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	var add *SymbolRecord
	for _, symbol := range snapshot.Symbols {
		if symbol.Name == "add" {
			copy := symbol
			add = &copy
			break
		}
	}
	if add == nil || len(add.Declarations) != 1 || len(add.Definitions) != 1 {
		t.Fatalf("merged symbol = %+v, all = %+v", add, snapshot.Symbols)
	}
	for _, location := range []SymbolLocation{add.Declarations[0], add.Definitions[0]} {
		if definitions := snapshot.DefinitionAtURI(context.DocumentURI, location.SelectionRange.Start); len(definitions) != 1 || definitions[0].URI != context.DocumentURI {
			t.Fatalf("definition lookup at %q = %+v", location.Role, definitions)
		}
	}
}

func TestBuildKeepsDeclarationIdentityAcrossParameterNames(t *testing.T) {
	source := "int add(int left); int add(int right) { return right; }"
	context := resolvedParseContext("file:///workspace/stable-id.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	add := symbolByName(snapshot, "add")
	if add == nil || len(add.Declarations) != 1 || len(add.Definitions) != 1 {
		t.Fatalf("parameter-name-stable symbol = %+v", add)
	}
}

func TestBuildClassifiesBuiltinAndAttributeSources(t *testing.T) {
	source := "[[deprecated]] int threadIdx;"
	context := resolvedParseContext("file:///workspace/builtin.tops")
	context.LanguageID = "tops-header"
	context.TargetProfile = "gcu400"
	context.PassKind = "device"
	context.PredefinedMacros = map[string]parser.MacroValue{"__GCU_ARCH__": {Kind: parser.MacroInteger, Value: 400}}
	provider := verifiedSymbolProvider{
		SourceProvider: NewLocalSourceProvider(map[string]SourceDocument{
			"file:///workspace/tops-attributes.h": {URI: "file:///workspace/tops-attributes.h", Text: "[[deprecated]]", Version: 1},
			"file:///workspace/tops-builtins.h":   {URI: "file:///workspace/tops-builtins.h", Text: "int threadIdx;", Version: 1},
		}, nil),
		attributeSource: SymbolLocation{URI: "file:///workspace/tops-attributes.h", Range: parser.SourceRange{Start: 0, End: len("[[deprecated]]")}, SelectionRange: parser.SourceRange{Start: 0, End: len("[[deprecated]]")}},
		builtinSource:   SymbolLocation{URI: "file:///workspace/tops-builtins.h", Range: parser.SourceRange{Start: 0, End: len("int threadIdx;")}, SelectionRange: parser.SourceRange{Start: 0, End: len("int threadIdx;")}},
	}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	attribute := symbolByName(snapshot, "deprecated")
	if attribute == nil || attribute.Kind != SymbolKindAttribute || attribute.OriginKind != "header" || len(attribute.AttributeSources) != 1 || len(attribute.AttributeUsages) != 1 || attribute.AttributeUsages[0].SelectionRange.Start >= attribute.AttributeUsages[0].SelectionRange.End {
		t.Fatalf("attribute usage = %+v", attribute)
	}
	if definitions := snapshot.DefinitionAt(attribute.NameRange.Start); len(definitions) != 1 || definitions[0].URI != "file:///workspace/tops-attributes.h" {
		t.Fatalf("verified attribute definition = %+v", definitions)
	}
	builtin := symbolByName(snapshot, "threadIdx")
	if builtin == nil || builtin.Kind != SymbolKindBuiltin || len(builtin.AttributeSources) != 1 || builtin.AttributeSources[0].Role != SymbolLocationBuiltin || builtin.Builtin.TargetProfile != "gcu400" || builtin.Builtin.PassKind != "device" || builtin.Builtin.SourceURI != "file:///workspace/tops-builtins.h" || builtin.Builtin.MacroContext["__GCU_ARCH__"].Value != 400 || builtin.Builtin.Availability != "available" {
		t.Fatalf("builtin source = %+v", builtin)
	}
	if definitions := snapshot.DefinitionAtURI(context.DocumentURI, builtin.NameRange.Start); len(definitions) != 1 || definitions[0].URI != "file:///workspace/tops-builtins.h" || definitions[0].Role != SymbolLocationBuiltin {
		t.Fatalf("builtin definition source = %+v", definitions)
	}
	if references := snapshot.ReferencesAtURI(context.DocumentURI, builtin.NameRange.Start, true); len(references) != 1 || references[0].URI != "file:///workspace/tops-builtins.h" {
		t.Fatalf("builtin declaration source = %+v", references)
	}
}

func TestBuildOnlyIndexesDefineAsMacro(t *testing.T) {
	source := "#ifdef FLAG\nint value();\n#endif\n#define REAL_FLAG 1\n"
	context := resolvedParseContext("file:///workspace/macros.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	if symbolByName(snapshot, "FLAG") != nil || symbolByName(snapshot, "ifdef") != nil || !hasSymbolKind(snapshot, SymbolKindMacro, "REAL_FLAG") {
		t.Fatalf("preprocessor symbols = %+v", snapshot.Symbols)
	}
}

func TestBuildRecordsUnresolvedBuiltinProvenanceWithoutFakeSource(t *testing.T) {
	source := "int use() { return threadIdx.x; }"
	context := resolvedParseContext("file:///workspace/builtin-use.tops")
	context.TargetProfile = "gcu400"
	context.PassKind = "device"
	context.PredefinedMacros = map[string]parser.MacroValue{"__GCU_ARCH__": {Kind: parser.MacroInteger, Value: 400}}
	snapshot := Build(source, parser.Parse(source, context), context)
	if symbolByName(snapshot, "threadIdx") != nil {
		t.Fatalf("builtin use created a fake symbol: %+v", snapshot.Symbols)
	}
	for _, reference := range snapshot.References {
		if reference.Name != "threadIdx" {
			continue
		}
		if reference.Kind != "builtin-use" || reference.Status != ReferenceStatusUnresolved || reference.Builtin.TargetProfile != "gcu400" || reference.Builtin.PassKind != "device" || reference.Builtin.SourceURI != "" || reference.Builtin.Availability != "unresolved" {
			t.Fatalf("builtin reference provenance = %+v", reference)
		}
		return
	}
	t.Fatal("builtin reference not found")
}

func TestBuildUsesVerifiedHeaderProvenanceForAttributeAndBuiltin(t *testing.T) {
	root := t.TempDir()
	mainURI := FileURI(filepath.Join(root, "main.tops"))
	attributeURI := FileURI(filepath.Join(root, "attributes.h"))
	builtinURI := FileURI(filepath.Join(root, "builtins.h"))
	opened := map[string]SourceDocument{
		attributeURI: {URI: attributeURI, Text: "[[deprecated]] int attribute_marker;\n", Version: 4},
		builtinURI:   {URI: builtinURI, Text: "int threadIdx;\n", Version: 5},
	}
	context := resolvedParseContext(mainURI)
	context.IncludeRoots = []string{root}
	context.TargetProfile = "gcu400"
	context.PassKind = "device"
	context.PredefinedMacros = map[string]parser.MacroValue{"__GCU_ARCH__": {Kind: parser.MacroInteger, Value: 400}}
	source := "#include \"attributes.h\"\n#include \"builtins.h\"\n[[deprecated]] int value;\nint use() { return threadIdx; }\n"
	provider := verifiedSymbolProvider{
		SourceProvider: NewLocalSourceProviderWithOptions(opened, []string{root}, LocalSourceProviderOptions{TrustedHeaderURIs: []string{builtinURI}}),
		attributeSource: SymbolLocation{
			URI: attributeURI, Range: parser.SourceRange{Start: 0, End: len("[[deprecated]]")}, SelectionRange: parser.SourceRange{Start: 0, End: len("[[deprecated]]")},
		},
		builtinSource: SymbolLocation{
			URI: builtinURI, Range: parser.SourceRange{Start: 0, End: len("int threadIdx;\n")}, SelectionRange: parser.SourceRange{Start: 4, End: 13},
		},
	}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	attribute := symbolByName(snapshot, "deprecated")
	if attribute == nil || len(attribute.AttributeSources) != 1 || attribute.AttributeSources[0].URI != attributeURI || attribute.AttributeSources[0].SourceVersion.DocumentVersion != 4 || attribute.AttributeSources[0].SourceVersion.ContextVersion != 1 || attribute.AttributeSources[0].SourceVersion.SourceVersion == "" || attribute.AttributeProvenance.SourceURI != attributeURI || attribute.AttributeProvenance.TargetProfile != "gcu400" || attribute.AttributeProvenance.PassKind != "device" {
		t.Fatalf("verified attribute provenance = %+v", attribute)
	}
	builtin := symbolByName(snapshot, "threadIdx")
	if builtin == nil || builtin.Builtin.Availability != "available" || builtin.Builtin.SourceURI != builtinURI || builtin.Builtin.TargetProfile != "gcu400" || builtin.Builtin.PassKind != "device" || builtin.Builtin.MacroContext["__GCU_ARCH__"].Value != 400 || len(builtin.AttributeSources) != 1 || builtin.AttributeSources[0].SourceVersion.DocumentVersion != 5 || builtin.AttributeSources[0].SourceVersion.ContextVersion != 1 {
		t.Fatalf("verified builtin provenance = %+v", builtin)
	}
	usageOffset := strings.Index(source, "deprecated")
	definitions := snapshot.DefinitionAtURI(mainURI, usageOffset)
	if len(definitions) != 1 || definitions[0].URI != attributeURI {
		t.Fatalf("attribute usage definition = %+v", definitions)
	}
}

func TestBuildRejectsNonHeaderBuiltinSourceAndKeepsOrdinaryVariable(t *testing.T) {
	context := resolvedParseContext("file:///workspace/ordinary.tops")
	context.TargetProfile = "gcu400"
	context.PassKind = "device"
	source := "int threadIdx; int use() { return threadIdx; }"
	provider := verifiedSymbolProvider{
		SourceProvider: NewLocalSourceProvider(map[string]SourceDocument{
			context.DocumentURI: {URI: context.DocumentURI, Text: source, Version: 1},
		}, nil),
		builtinSource: SymbolLocation{URI: context.DocumentURI, Range: parser.SourceRange{Start: 0, End: len("int threadIdx;")}, SelectionRange: parser.SourceRange{Start: 4, End: 13}},
	}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	threadIdx := symbolByName(snapshot, "threadIdx")
	if threadIdx == nil || threadIdx.Kind != SymbolKindVariable || threadIdx.Builtin.Availability != "" {
		t.Fatalf("ordinary same-name variable = %+v", threadIdx)
	}
	if definitions := snapshot.DefinitionAtURI(context.DocumentURI, strings.Index(source, "threadIdx")); len(definitions) != 1 || definitions[0].URI != context.DocumentURI {
		t.Fatalf("ordinary variable definition = %+v", definitions)
	}
}

func TestAttributeUsagesAreReferencesAndVerifiedSourceIsDeclaration(t *testing.T) {
	source := "[[deprecated]] [[deprecated]] int value();"
	context := resolvedParseContext("file:///workspace/attributes.tops")
	provider := verifiedSymbolProvider{
		SourceProvider: NewLocalSourceProvider(map[string]SourceDocument{
			"file:///workspace/attributes.h": {URI: "file:///workspace/attributes.h", Text: "[[deprecated]]", Version: 1},
		}, nil),
		attributeSource: SymbolLocation{
			URI:            "file:///workspace/attributes.h",
			Range:          parser.SourceRange{Start: 0, End: len("[[deprecated]]")},
			SelectionRange: parser.SourceRange{Start: 0, End: len("[[deprecated]]")},
		},
	}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, provider)
	attribute := symbolByName(snapshot, "deprecated")
	if attribute == nil || len(attribute.AttributeUsages) != 2 || len(attribute.AttributeSources) != 1 {
		t.Fatalf("attribute symbol = %+v", attribute)
	}
	attributeReferences := 0
	for _, reference := range snapshot.References {
		if reference.Kind == "attribute-use" && reference.Name == "deprecated" {
			attributeReferences++
		}
	}
	if attributeReferences != 2 {
		t.Fatalf("attribute references = %d, all = %+v", attributeReferences, snapshot.References)
	}
	usageOffset := attribute.AttributeUsages[1].SelectionRange.Start
	definitions := snapshot.DefinitionAt(usageOffset)
	if len(definitions) != 1 || definitions[0].URI != "file:///workspace/attributes.h" {
		t.Fatalf("attribute definition = %+v", definitions)
	}
	references := snapshot.ReferencesAt(usageOffset, false)
	if len(references) != 2 {
		t.Fatalf("attribute references without declaration = %+v", references)
	}
	allLocations := snapshot.ReferencesAt(usageOffset, true)
	if len(allLocations) != 3 {
		t.Fatalf("attribute references with declaration = %+v", allLocations)
	}
}

func TestBuildRecordsUnresolvedAndInactiveReferences(t *testing.T) {
	source := "#if 0\nint inactive_decl();\n#endif\nint use() { return inactive_decl() + missing_name; }"
	context := resolvedParseContext("file:///workspace/reference-states.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	statuses := make(map[string]ReferenceStatus)
	for _, reference := range snapshot.References {
		if reference.Name == "inactive_decl" || reference.Name == "missing_name" {
			statuses[reference.Name] = reference.Status
		}
	}
	if statuses["inactive_decl"] != ReferenceStatusUnresolved || statuses["missing_name"] != ReferenceStatusUnresolved {
		t.Fatalf("reference statuses = %+v", statuses)
	}
}

func hasSymbolKind(snapshot IndexSnapshot, kind SymbolKind, name string) bool {
	for _, symbol := range snapshot.Symbols {
		if symbol.Kind == kind && strings.EqualFold(symbol.Name, name) {
			return true
		}
	}
	return false
}
