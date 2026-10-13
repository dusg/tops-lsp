package index

import (
	"os"
	"path/filepath"
	"testing"

	"tops-lsp/internal/parser"
)

func TestBuildIndexesConditionalBranchProvenance(t *testing.T) {
	source := "#if defined(ACTIVE)\nint active_symbol();\n#else\nint inactive_symbol();\n#endif\n"
	context := resolvedParseContext("file:///workspace/conditions.tops")
	context.PredefinedMacros = map[string]parser.MacroValue{"ACTIVE": {Kind: parser.MacroDefined}}
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)
	active := symbolByName(snapshot, "active_symbol")
	inactive := symbolByName(snapshot, "inactive_symbol")
	if active == nil || inactive == nil {
		t.Fatalf("conditional symbols = %+v", snapshot.Symbols)
	}
	if active.Condition.SourceURI != context.DocumentURI || inactive.Condition.SourceURI != context.DocumentURI || active.Condition.State != parser.ConditionalActive || inactive.Condition.State != parser.ConditionalInactive || active.Condition.ExpressionRange.End <= active.Condition.ExpressionRange.Start || source[active.Condition.ExpressionRange.Start:active.Condition.ExpressionRange.End] != "defined(ACTIVE)" || inactive.Condition.BranchIndex != 1 {
		t.Fatalf("conditional states = active:%+v inactive:%+v", active.Condition, inactive.Condition)
	}
	symbols := snapshot.DocumentSymbols()
	if !containsDocumentSymbol(symbols, "active_symbol") || containsDocumentSymbol(symbols, "inactive_symbol") {
		t.Fatalf("document symbols = %+v", symbols)
	}
}

func TestBuildKeepsUnknownConditionalSymbolsForDocumentSymbols(t *testing.T) {
	source := "#if defined(UNKNOWN_MACRO)\nint unknown_symbol();\n#endif\n"
	context := resolvedParseContext("file:///workspace/unknown.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)
	unknown := symbolByName(snapshot, "unknown_symbol")
	if unknown == nil || unknown.Condition.State != parser.ConditionalUnknown || unknown.Condition.RegionID < 0 {
		t.Fatalf("unknown symbol = %+v", unknown)
	}
	if !containsDocumentSymbol(snapshot.DocumentSymbols(), "unknown_symbol") {
		t.Fatalf("document symbols = %+v", snapshot.DocumentSymbols())
	}
}

func TestBuildNestedUnknownParentDoesNotPromoteInactiveChild(t *testing.T) {
	source := "#if OUTER\n#if 0\nint inactive_child;\n#else\nint unknown_child;\n#endif\n#endif\n"
	context := resolvedParseContext("file:///workspace/nested-conditions.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	inactive := symbolByName(snapshot, "inactive_child")
	unknown := symbolByName(snapshot, "unknown_child")
	if inactive == nil || unknown == nil {
		t.Fatalf("nested conditional symbols = %+v", snapshot.Symbols)
	}
	if inactive.Condition.State != parser.ConditionalInactive || unknown.Condition.State != parser.ConditionalUnknown {
		t.Fatalf("nested conditional states = inactive:%+v unknown:%+v", inactive.Condition, unknown.Condition)
	}
	if inactive.Condition.ParentRegionID != 0 || len(inactive.Condition.ParentConditions) == 0 {
		t.Fatalf("inactive child provenance = %+v", inactive.Condition)
	}
	if definitions := snapshot.DefinitionAtURI(context.DocumentURI, inactive.NameRange.Start); len(definitions) != 0 {
		t.Fatalf("inactive child definition = %+v", definitions)
	}
	symbols := snapshot.DocumentSymbols()
	if containsDocumentSymbol(symbols, "inactive_child") || !containsDocumentSymbol(symbols, "unknown_child") {
		t.Fatalf("nested document symbols = %+v", symbols)
	}
}

func TestIndexDiagnosticPreservesNestedConditionalProvenance(t *testing.T) {
	source := "#if OUTER\n#if INNER\nint nested_value;\n#endif\n#endif\n"
	context := resolvedParseContext("file:///workspace/nested-diagnostic.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code != "analysis-degraded" || diagnostic.Reason != "unknown-condition" {
			continue
		}
		if diagnostic.Condition.State != parser.ConditionalUnknown || diagnostic.Condition.ParentRegionID != -1 || diagnostic.Condition.ExpressionRange.End <= diagnostic.Condition.ExpressionRange.Start || len(diagnostic.Condition.ParentConditions) != 2 {
			t.Fatalf("nested diagnostic provenance = %+v", diagnostic)
		}
		return
	}
	t.Fatalf("nested conditional diagnostic missing: %+v", snapshot.Diagnostics)
}

func TestQueriesExcludeInactiveSymbolsAndReferences(t *testing.T) {
	source := "#if 0\nint hidden();\n#endif\nint use() { return hidden(); }\n"
	context := resolvedParseContext("file:///workspace/inactive-query.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	hidden := symbolByName(snapshot, "hidden")
	if hidden == nil || hidden.Condition.State != parser.ConditionalInactive {
		t.Fatalf("inactive symbol = %+v", hidden)
	}
	if definitions := snapshot.DefinitionAt(hidden.NameRange.Start); len(definitions) != 0 {
		t.Fatalf("inactive definition = %+v", definitions)
	}
	if references := snapshot.ReferencesAt(hidden.NameRange.Start, true); len(references) != 0 {
		t.Fatalf("inactive references = %+v", references)
	}
	for id, reference := range snapshot.References {
		if reference.Name != "hidden" || reference.Status != ReferenceStatusUnresolved {
			continue
		}
		reference.Status = ReferenceStatusInactive
		reference.Candidates = []string{hidden.SymbolID}
		snapshot.References[id] = reference
		if definitions := snapshot.DefinitionAt(reference.Range.Start); len(definitions) != 0 {
			t.Fatalf("inactive reference candidates = %+v", definitions)
		}
		if references := snapshot.ReferencesAt(reference.Range.Start, true); len(references) != 0 {
			t.Fatalf("inactive reference output = %+v", references)
		}
		return
	}
	t.Fatal("inactive reference not found")
}

func TestBuildWithProviderPropagatesUnknownIncludeCondition(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int header_value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := "#if defined(UNKNOWN_INCLUDE)\n#include \"api.h\"\n#endif\nint use() { return header_value; }\n"
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(nil, []string{root}))

	header := symbolByName(snapshot, "header_value")
	if header == nil || header.Condition.State != parser.ConditionalUnknown || header.Condition.SourceURI != uri {
		t.Fatalf("unknown include condition = %+v", header)
	}
	usage := sourceOffset(source, "header_value", 0)
	definitions := snapshot.DefinitionAtURI(uri, usage)
	if len(definitions) != 1 || definitions[0].URI != fileURI(headerPath) {
		t.Fatalf("unknown include definition = %+v", definitions)
	}
	references := snapshot.ReferencesAtURI(uri, usage, true)
	if len(references) != 2 || references[0].URI != fileURI(headerPath) {
		t.Fatalf("unknown include references = %+v", references)
	}
	foundUnknownDiagnostic := false
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == "analysis-degraded" && diagnostic.ConditionalState == parser.ConditionalUnknown {
			foundUnknownDiagnostic = true
			break
		}
	}
	if !foundUnknownDiagnostic || snapshot.IndexStatus != IndexStatusDegraded {
		t.Fatalf("unknown include diagnostics/status = %+v, status = %q", snapshot.Diagnostics, snapshot.IndexStatus)
	}
}

func TestBuildWithProviderKeepsNestedConditionalSourceURIs(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("#if INNER\nint header_value;\n#endif\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mainURI := fileURI(filepath.Join(root, "main.tops"))
	headerURI := fileURI(headerPath)
	source := "#if OUTER\n#include \"api.h\"\n#endif\nint root_value;\n"
	context := resolvedParseContext(mainURI)
	context.IncludeRoots = []string{root}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(nil, []string{root}))
	header := symbolByName(snapshot, "header_value")
	if header == nil {
		t.Fatalf("header symbol = %+v", snapshot.Symbols)
	}
	sources := make(map[string]struct{})
	var collect func(ConditionalProvenance)
	collect = func(condition ConditionalProvenance) {
		if condition.State == "" {
			return
		}
		sources[condition.SourceURI] = struct{}{}
		for _, parent := range condition.ParentConditions {
			collect(parent)
		}
	}
	collect(header.Condition)
	if _, ok := sources[mainURI]; !ok {
		t.Fatalf("nested condition did not retain main URI: %+v", header.Condition)
	}
	if _, ok := sources[headerURI]; !ok {
		t.Fatalf("nested condition did not retain header URI: %+v", header.Condition)
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Condition.State == "" {
			continue
		}
		if diagnostic.SourceURI == headerURI && diagnostic.Condition.SourceURI == "" {
			t.Fatalf("header diagnostic lost condition source URI: %+v", diagnostic)
		}
	}
}

func TestBuildWithProviderDoesNotLoadInactiveInclude(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "api.h"), []byte("int hidden_header;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := fileURI(filepath.Join(root, "main.tops"))
	source := "#if 0\n#include \"api.h\"\n#endif\nint use() { return 0; }\n"
	context := resolvedParseContext(uri)
	context.IncludeRoots = []string{root}
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(nil, []string{root}))
	if symbolByName(snapshot, "hidden_header") != nil {
		t.Fatalf("inactive include loaded header symbols = %+v", snapshot.Symbols)
	}
	if snapshot.IndexStatus != IndexStatusComplete {
		t.Fatalf("inactive include status = %q, diagnostics = %+v", snapshot.IndexStatus, snapshot.Diagnostics)
	}
}

func TestCombineConditionalProvenanceKeepsNestedRegions(t *testing.T) {
	parent := ConditionalProvenance{SourceURI: "file:///workspace/main.tops", RegionID: 7, BranchIndex: 1, ExpressionRange: parser.SourceRange{Start: 2, End: 10}, State: parser.ConditionalUnknown}
	child := ConditionalProvenance{SourceURI: "file:///workspace/api.h", RegionID: 3, BranchIndex: 0, ExpressionRange: parser.SourceRange{Start: 4, End: 12}, State: parser.ConditionalUnknown}
	combined := combineConditionalProvenance(parent, child)
	if combined.State != parser.ConditionalUnknown || combined.RegionID != parent.RegionID || combined.SourceURI != parent.SourceURI {
		t.Fatalf("combined primary condition = %+v", combined)
	}
	if len(combined.ParentConditions) != 2 || combined.ParentConditions[0].RegionID != parent.RegionID || combined.ParentConditions[1].RegionID != child.RegionID {
		t.Fatalf("combined nested conditions = %+v", combined.ParentConditions)
	}
}
