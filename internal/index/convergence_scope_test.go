package index

import (
	"strings"
	"testing"

	"tops-lsp/internal/parser"
)

func TestBuildResolvesReferencesByScopeAndQualifiedName(t *testing.T) {
	source := "namespace A { int f(); int use() { return f(); } } namespace B { int f(); int use() { return f(); } } int qualified() { return A::f(); }"
	context := resolvedParseContext("file:///workspace/scopes.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	localReferences := make([]ReferenceRecord, 0)
	qualifiedReference := ReferenceRecord{}
	for _, reference := range snapshot.References {
		if reference.Name != "f" {
			continue
		}
		if strings.Contains(reference.QualifiedName, "A::") {
			qualifiedReference = reference
			continue
		}
		localReferences = append(localReferences, reference)
	}
	if len(localReferences) != 2 {
		t.Fatalf("local references = %+v", localReferences)
	}
	for _, reference := range localReferences {
		if reference.ScopeID == "" || len(reference.Candidates) != 1 {
			t.Fatalf("scoped reference = %+v", reference)
		}
	}
	if qualifiedReference.ScopeID == "" || qualifiedReference.QualifiedName != "A::f" || len(qualifiedReference.Candidates) != 1 {
		t.Fatalf("qualified reference = %+v", qualifiedReference)
	}
	if snapshot.Symbols[qualifiedReference.Candidates[0]].QualifiedName != "A::f" {
		t.Fatalf("qualified candidate = %+v", snapshot.Symbols[qualifiedReference.Candidates[0]])
	}
}

func TestBuildReturnsAllActiveCandidatesInTheSameScope(t *testing.T) {
	source := "int overload(int value); double overload(double value); int use() { return overload(1); }"
	context := resolvedParseContext("file:///workspace/overloads.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	for _, reference := range snapshot.References {
		if reference.Name == "overload" && reference.QualifiedName == "overload" {
			if len(reference.Candidates) != 2 || reference.Status != ReferenceStatusAmbiguous {
				t.Fatalf("overload reference = %+v", reference)
			}
			return
		}
	}
	t.Fatal("overload reference not found")
}

func TestBuildResolvesMemberUseToReceiverField(t *testing.T) {
	source := "struct Point { int x; }; int use() { Point point; return point.x; }"
	context := resolvedParseContext("file:///workspace/member.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	xOffset := sourceOffset(source, "x", 1)
	var memberReference ReferenceRecord
	found := false
	for _, reference := range snapshot.References {
		if reference.Name == "x" && reference.Kind == "member-use" && reference.Range.Start == xOffset {
			memberReference = reference
			found = true
			break
		}
	}
	if !found || len(memberReference.Candidates) != 1 {
		t.Fatalf("member reference = %+v", memberReference)
	}
	field, ok := snapshot.Symbols[memberReference.Candidates[0]]
	if !ok || field.Kind != SymbolKindField || field.Name != "x" || field.TypeName != "int" {
		t.Fatalf("member field = %+v", field)
	}
	locations := snapshot.DefinitionAt(xOffset)
	if len(locations) != 1 || locations[0].URI != context.DocumentURI || source[locations[0].SelectionRange.Start:locations[0].SelectionRange.End] != "x" {
		t.Fatalf("member definition = %+v", locations)
	}
}

func TestBuildResolvesNamespacedMemberUseAfterAttribute(t *testing.T) {
	source := "#define EDGE_FLAG 1\nnamespace edge { struct Point { int x; }; [[deprecated]] int use(Point point) { return point.x; } }"
	context := resolvedParseContext("file:///workspace/edge.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	x := symbolByName(snapshot, "x")
	if x == nil {
		t.Fatal("field symbol not found")
	}
	if references := snapshot.ReferencesAt(x.NameRange.Start, false); len(references) != 1 {
		t.Fatalf("namespaced field references = %+v, all references = %+v", references, snapshot.References)
	}
}

func TestDefinitionAndReferencesFallbackToUnknownInVisibleScope(t *testing.T) {
	uri := "file:///workspace/scope-condition.tops"
	rootScope := ScopeRecord{ScopeID: "root", SourceURI: uri, Kind: ScopeTranslationUnit, Range: parser.SourceRange{Start: 0, End: 100}}
	outsideScope := ScopeRecord{ScopeID: "outside", SourceURI: uri, Kind: ScopeNamespace, ParentID: rootScope.ScopeID, Range: parser.SourceRange{Start: 0, End: 20}}
	currentScope := ScopeRecord{ScopeID: "current", SourceURI: uri, Kind: ScopeFunction, ParentID: rootScope.ScopeID, Range: parser.SourceRange{Start: 20, End: 100}}
	active := SymbolRecord{
		SymbolID:    "active-outside",
		Name:        "value",
		Kind:        SymbolKindVariable,
		ScopeID:     outsideScope.ScopeID,
		Condition:   ConditionalProvenance{SourceURI: uri, State: parser.ConditionalActive},
		NameRange:   parser.SourceRange{Start: 5, End: 10},
		Definitions: []SymbolLocation{{URI: uri, Range: parser.SourceRange{Start: 0, End: 10}, SelectionRange: parser.SourceRange{Start: 5, End: 10}, Role: SymbolLocationDefinition}},
	}
	unknown := SymbolRecord{
		SymbolID:    "unknown-current",
		Name:        "value",
		Kind:        SymbolKindVariable,
		ScopeID:     currentScope.ScopeID,
		Condition:   ConditionalProvenance{SourceURI: uri, State: parser.ConditionalUnknown},
		NameRange:   parser.SourceRange{Start: 45, End: 50},
		Definitions: []SymbolLocation{{URI: uri, Range: parser.SourceRange{Start: 40, End: 50}, SelectionRange: parser.SourceRange{Start: 45, End: 50}, Role: SymbolLocationDefinition}},
	}
	snapshot := IndexSnapshot{
		RootURI:     uri,
		IndexStatus: IndexStatusDegraded,
		Scopes:      map[string]ScopeRecord{rootScope.ScopeID: rootScope, outsideScope.ScopeID: outsideScope, currentScope.ScopeID: currentScope},
		Symbols:     map[string]SymbolRecord{active.SymbolID: active, unknown.SymbolID: unknown},
		References:  map[string]ReferenceRecord{},
	}
	candidates := candidateIDsForSnapshot(&snapshot, []string{active.SymbolID, unknown.SymbolID}, "value", "", currentScope.ScopeID, parser.ConditionalActive)
	if len(candidates) != 1 || candidates[0] != unknown.SymbolID {
		t.Fatalf("visible-scope candidates = %v", candidates)
	}
	reference := ReferenceRecord{ReferenceID: "value-reference", URI: uri, Range: parser.SourceRange{Start: 70, End: 75}, Name: "value", ScopeID: currentScope.ScopeID, Candidates: candidates, Status: ReferenceStatusResolved, Condition: ConditionalProvenance{SourceURI: uri, State: parser.ConditionalActive}}
	snapshot.References[reference.ReferenceID] = reference
	unknown.References = []string{reference.ReferenceID}
	snapshot.Symbols[unknown.SymbolID] = unknown

	definitions := snapshot.DefinitionAtURI(uri, reference.Range.Start)
	if len(definitions) != 1 || definitions[0].Range != unknown.Definitions[0].Range {
		t.Fatalf("visible-scope definition = %+v", definitions)
	}
	references := snapshot.ReferencesAtURI(uri, unknown.NameRange.Start, false)
	if len(references) != 1 || references[0].Range != reference.Range {
		t.Fatalf("visible-scope references = %+v", references)
	}
}

func TestMemberUseFallsBackToUnknownField(t *testing.T) {
	uri := "file:///workspace/member-condition.tops"
	snapshot := IndexSnapshot{Symbols: map[string]SymbolRecord{
		"unknown-field": {SymbolID: "unknown-field", Name: "x", Kind: SymbolKindField, Condition: ConditionalProvenance{SourceURI: uri, State: parser.ConditionalUnknown}},
	}}
	candidates := conditionalCandidates(&snapshot, []string{"unknown-field"}, parser.ConditionalActive)
	if len(candidates) != 1 || candidates[0] != "unknown-field" {
		t.Fatalf("unknown member candidates = %v", candidates)
	}
}
