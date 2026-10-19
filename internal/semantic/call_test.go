package semantic

import (
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestBuildSemanticSnapshotKeepsCallCandidatesAndDegradedOverloads(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.References["ambiguous-call"] = index.ReferenceRecord{ReferenceID: "ambiguous-call", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 50, End: 57}, Name: "overload", Candidates: []string{"add", "other"}, Kind: "call", Status: index.ReferenceStatusAmbiguous, SourceVersion: snapshot.SourceVersion}
	snapshot.References["unresolved-call"] = index.ReferenceRecord{ReferenceID: "unresolved-call", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 58, End: 66}, Name: "missing", Kind: "call", Status: index.ReferenceStatusUnresolved, SourceVersion: snapshot.SourceVersion}
	result, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusResolved))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Calls) != 3 {
		t.Fatalf("calls = %+v", result.Calls)
	}
	var ambiguous, unresolved bool
	for _, call := range result.Calls {
		if call.CallID == "ambiguous-call" {
			ambiguous = call.CandidateStatus == CandidateStatusAmbiguous && len(call.Candidates) == 2
		}
		if call.CallID == "unresolved-call" {
			unresolved = call.CandidateStatus == CandidateStatusUnresolved
		}
	}
	if !ambiguous || !unresolved {
		t.Fatalf("calls = %+v", result.Calls)
	}
	unsupported := CheckConversion(SemanticType{Spelling: "Meters", Kind: TypeKindUserDefined, UserDefined: true}, SemanticType{Spelling: "Seconds", Kind: TypeKindUserDefined, UserDefined: true})
	if unsupported.Status != ConversionStatusUnsupported || hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-type-mismatch", SeverityError) {
		t.Fatalf("unsupported conversion = %+v diagnostics = %+v", unsupported, result.Diagnostics)
	}
}
