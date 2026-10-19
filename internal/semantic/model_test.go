package semantic

import (
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestBuildSemanticSnapshotCreatesUS1Facts(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	context := semanticTestContext(ContextStatusResolved)
	result, err := BuildSemanticSnapshot(snapshot, context)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProfileID != "gcu400-default" || result.ContextStatus != ContextStatusResolved || result.Status != SnapshotStatusComplete {
		t.Fatalf("snapshot metadata = %+v", result)
	}
	if len(result.Declarations) != 4 || len(result.Scopes) != 1 || len(result.Types) < 3 || len(result.Callables) != 1 || len(result.References) != 1 || len(result.Calls) != 1 {
		t.Fatalf("fact counts = declarations:%d scopes:%d types:%d callables:%d references:%d calls:%d", len(result.Declarations), len(result.Scopes), len(result.Types), len(result.Callables), len(result.References), len(result.Calls))
	}
	callable := result.Callables[0]
	if callable.Name != "add" || len(callable.Parameters) != 2 || callable.ReturnType.Spelling != "int" || callable.DeclarationOnly {
		t.Fatalf("callable = %+v", callable)
	}
	if result.Calls[0].CandidateStatus != CandidateStatusResolved || len(result.Calls[0].Candidates) != 1 {
		t.Fatalf("call = %+v", result.Calls[0])
	}
}

func TestBuildSemanticSnapshotEmitsStableNameDiagnostics(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.References["unknown"] = index.ReferenceRecord{ReferenceID: "unknown", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 30, End: 37}, Name: "missing", Status: index.ReferenceStatusUnresolved, Kind: "name", SourceVersion: snapshot.SourceVersion}
	snapshot.References["ambiguous"] = index.ReferenceRecord{ReferenceID: "ambiguous", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 40, End: 47}, Name: "overload", Candidates: []string{"f1", "f2"}, Status: index.ReferenceStatusAmbiguous, Kind: "name", SourceVersion: snapshot.SourceVersion}
	result, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusResolved))
	if err != nil {
		t.Fatal(err)
	}
	if !hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-unknown-name", SeverityError) || !hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-ambiguous-name", SeverityWarning) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	degraded, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusPartial))
	if err != nil {
		t.Fatal(err)
	}
	if hasSemanticDiagnostic(degraded.Diagnostics, "tops-semantic-unknown-name", SeverityError) || !hasSemanticDiagnostic(degraded.Diagnostics, "tops-semantic-context-degraded", SeverityInformation) {
		t.Fatalf("degraded diagnostics = %+v", degraded.Diagnostics)
	}
}

func TestBuildSemanticSnapshotPublishesIncompleteInputHintWithRange(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	context := semanticTestContext(ContextStatusPartial)
	context.ParseStatus = parser.ParseStatusRecovered
	context.IncompleteRange = parser.SourceRange{}
	result, err := BuildSemanticSnapshot(snapshot, context)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "tops-semantic-incomplete-input" {
			found = diagnostic.Severity == SeverityHint && diagnostic.Incomplete && diagnostic.Range.End > diagnostic.Range.Start && diagnostic.PassKind == "not-applicable"
		}
	}
	if !found {
		t.Fatalf("incomplete diagnostics = %+v", result.Diagnostics)
	}
}

func semanticTestIndexSnapshot() index.IndexSnapshot {
	uri := "file:///workspace/main.tops"
	sourceVersion := index.SourceVersion{URI: uri, DocumentVersion: 1, ContextVersion: 1, SourceVersion: "root-v1"}
	return index.IndexSnapshot{
		RootURI:        uri,
		SourceVersion:  sourceVersion,
		ContextVersion: 1,
		Sources:        map[string]string{uri: "int add(int left, int right) { return left + right; } add(1, 2);"},
		Scopes: map[string]index.ScopeRecord{
			"root": {ScopeID: "root", SourceURI: uri, Kind: index.ScopeTranslationUnit, QualifiedName: "", Range: parser.SourceRange{Start: 0, End: 65}, Symbols: []string{"add", "value", "Point", "field"}},
		},
		Symbols: map[string]index.SymbolRecord{
			"add":   {SymbolID: "add", Name: "add", QualifiedName: "add", Kind: index.SymbolKindFunction, ScopeID: "root", NameRange: parser.SourceRange{Start: 4, End: 7}, ParameterSummary: "int left , int right", ReturnTypeSummary: "int", Status: index.SymbolStatusActive, SourceVersion: sourceVersion},
			"value": {SymbolID: "value", Name: "value", QualifiedName: "value", Kind: index.SymbolKindVariable, ScopeID: "root", NameRange: parser.SourceRange{Start: 8, End: 13}, TypeName: "int", Status: index.SymbolStatusActive, SourceVersion: sourceVersion},
			"Point": {SymbolID: "Point", Name: "Point", QualifiedName: "Point", Kind: index.SymbolKindType, ScopeID: "root", NameRange: parser.SourceRange{Start: 14, End: 19}, TypeKind: "struct", TypeName: "Point", Status: index.SymbolStatusActive, SourceVersion: sourceVersion},
			"field": {SymbolID: "field", Name: "field", QualifiedName: "Point::field", Kind: index.SymbolKindField, ScopeID: "root", NameRange: parser.SourceRange{Start: 20, End: 25}, TypeName: "int", Status: index.SymbolStatusActive, SourceVersion: sourceVersion},
		},
		References: map[string]index.ReferenceRecord{
			"call": {ReferenceID: "call", URI: uri, Range: parser.SourceRange{Start: 60, End: 63}, Name: "add", QualifiedName: "add", ScopeID: "root", Candidates: []string{"add"}, Kind: "call", Status: index.ReferenceStatusResolved, SourceVersion: sourceVersion},
		},
	}
}

func semanticTestContext(status ContextStatus) SemanticContext {
	documentURI := "file:///workspace/main.tops"
	sourceVersions := []SourceVersionEntry{{URI: documentURI, Version: "root-v1"}}
	fingerprint, _ := AggregateSourceVersion(documentURI, sourceVersions)
	return SemanticContext{
		DocumentURI:              documentURI,
		Command:                  CompilationCommand{Source: "default", Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice},
		Profile:                  DefaultSemanticProfile(true),
		Status:                   status,
		SemanticContextVersion:   1,
		PassViews:                NewPassViews(PassKindHostDevice, status),
		SourceVersions:           sourceVersions,
		SourceVersionFingerprint: fingerprint,
	}
}

func hasSemanticDiagnostic(diagnostics []SemanticDiagnostic, code string, severity SemanticSeverity) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Severity == severity {
			return true
		}
	}
	return false
}
