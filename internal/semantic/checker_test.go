package semantic

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestBuildSemanticSnapshotUsesRealCallerAndTargetFacts(t *testing.T) {
	root := repositoryRootForCheckerTest(t)
	callSnapshot := buildFixtureSemanticSnapshot(t, root, "calls/host-device/invalid.tops")
	var foundCall bool
	for _, call := range callSnapshot.Calls {
		if call.CalleeName != "device_only" {
			continue
		}
		foundCall = true
		if call.CallerKind != "host" || call.CalleeKind != "device" || call.CallerSymbolID == "" || call.CalleeSymbolID == "" {
			t.Fatalf("call facts = %+v", call)
		}
		if len(call.PassFacts) != 2 || call.PassKind == PassKindHostDevice {
			t.Fatalf("independent pass facts = %+v", call.PassFacts)
		}
		for _, passFact := range call.PassFacts {
			if passFact.Availability != PassAvailabilityAvailable || passFact.CallerKind != "host" || passFact.CalleeKind != "device" {
				t.Fatalf("pass fact = %+v", passFact)
			}
		}
	}
	if !foundCall || !hasSemanticDiagnostic(callSnapshot.Diagnostics, "tops-semantic-host-device-boundary", SeverityError) {
		t.Fatalf("call snapshot = %+v", callSnapshot)
	}

	addressSnapshot := buildFixtureSemanticSnapshot(t, root, "address-space/invalid.tops")
	if !hasSemanticDiagnostic(addressSnapshot.Diagnostics, "tops-semantic-address-space", SeverityError) {
		t.Fatalf("address-space diagnostics = %+v", addressSnapshot.Diagnostics)
	}
	vectorSnapshot := buildFixtureSemanticSnapshot(t, root, "vectors/invalid.tops")
	if !hasSemanticDiagnostic(vectorSnapshot.Diagnostics, "tops-semantic-vector-constraint", SeverityError) {
		t.Fatalf("vector diagnostics = %+v", vectorSnapshot.Diagnostics)
	}
	validVectorSnapshot := buildFixtureSemanticSnapshot(t, root, "vectors/valid.tops")
	if hasSemanticDiagnostic(validVectorSnapshot.Diagnostics, "tops-semantic-vector-constraint", SeverityError) {
		t.Fatalf("valid vector diagnostics = %+v", validVectorSnapshot.Diagnostics)
	}
}

func TestKernelAndConditionalCallFixturesExposeCallableFacts(t *testing.T) {
	root := repositoryRootForCheckerTest(t)
	kernel := buildFixtureSemanticSnapshot(t, root, "callables/kernel-entry/invalid.tops")
	if len(kernel.Calls) != 1 || kernel.Calls[0].CallerKind != "kernel-entry" || kernel.Calls[0].CalleeKind != "device" || !hasSemanticDiagnostic(kernel.Diagnostics, "tops-semantic-invalid-call", SeverityError) {
		t.Fatalf("kernel facts = %+v diagnostics = %+v", kernel.Calls, kernel.Diagnostics)
	}
	conditional := buildFixtureSemanticSnapshot(t, root, "calls/basic/context-boundary.tops")
	if len(conditional.Calls) < 2 {
		t.Fatalf("conditional calls = %+v", conditional.Calls)
	}
	foundBoundaryCall := false
	for _, call := range conditional.Calls {
		if call.CalleeKind == "kernel-entry" {
			foundBoundaryCall = call.CallerKind == "host" && call.CandidateStatus == CandidateStatusUnresolved
		}
	}
	if !foundBoundaryCall {
		t.Fatalf("conditional calls = %+v", conditional.Calls)
	}
}

func TestSemanticDiagnosticsIncludeCandidateRelatedInformation(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.References["ambiguous-related"] = index.ReferenceRecord{ReferenceID: "ambiguous-related", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 50, End: 57}, Name: "overload", Candidates: []string{"add", "Point"}, Kind: "name", Status: index.ReferenceStatusAmbiguous, SourceVersion: snapshot.SourceVersion}
	result, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusResolved))
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "tops-semantic-ambiguous-name" {
			if len(diagnostic.Related) == 0 {
				t.Fatalf("ambiguous diagnostic has no related candidates: %+v", diagnostic)
			}
			return
		}
	}
	t.Fatalf("ambiguous diagnostic not found: %+v", result.Diagnostics)
}

func repositoryRootForCheckerTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "semantic")
}

func buildFixtureSemanticSnapshot(t *testing.T, fixtureRoot, relativePath string) SemanticSnapshot {
	t.Helper()
	path := filepath.Join(fixtureRoot, filepath.FromSlash(relativePath))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	uri := "file:///workspace/" + filepath.ToSlash(relativePath)
	parseContext := parser.ParseContext{DocumentURI: uri, LanguageID: "tops", LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "resolved", ArgumentProvenance: "fixture", TargetProfile: "gcu400", PassKind: "host+device", DocumentVersion: 1, ContextVersion: 1}
	parsed := parser.Parse(string(body), parseContext)
	indexed := index.Build(string(body), parsed, parseContext)
	context := semanticTestContext(ContextStatusResolved)
	context.DocumentURI = uri
	context.DocumentVersion = 1
	context.SourceVersions = []SourceVersionEntry{{URI: uri, Version: index.SourceDigest(string(body))}}
	context.SourceVersionFingerprint, err = AggregateSourceVersion(uri, context.SourceVersions)
	if err != nil {
		t.Fatal(err)
	}
	result, err := BuildSemanticSnapshot(indexed, context)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBuildSemanticSnapshotReportsDeterministicCallAndTypeErrors(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.References["invalid-call"] = index.ReferenceRecord{ReferenceID: "invalid-call", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 20, End: 24}, Name: "value", Candidates: []string{"value"}, Kind: "call", Status: index.ReferenceStatusResolved, SourceVersion: snapshot.SourceVersion}
	snapshot.References["type-mismatch"] = index.ReferenceRecord{ReferenceID: "type-mismatch", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 25, End: 30}, Name: "value", Kind: "type-mismatch", Status: index.ReferenceStatusResolved, SourceVersion: snapshot.SourceVersion}
	result, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusResolved))
	if err != nil {
		t.Fatal(err)
	}
	if !hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-invalid-call", SeverityError) || !hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-type-mismatch", SeverityError) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestBuildSemanticSnapshotDoesNotPublishDeterministicErrorsForUnsupportedConversion(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.References["unsupported-conversion"] = index.ReferenceRecord{ReferenceID: "unsupported-conversion", URI: snapshot.RootURI, Range: parser.SourceRange{Start: 31, End: 39}, Name: "value", Kind: "unsupported-conversion", Status: index.ReferenceStatusResolved, SourceVersion: snapshot.SourceVersion}
	result, err := BuildSemanticSnapshot(snapshot, semanticTestContext(ContextStatusResolved))
	if err != nil {
		t.Fatal(err)
	}
	if hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-type-mismatch", SeverityError) || hasSemanticDiagnostic(result.Diagnostics, "tops-semantic-invalid-call", SeverityError) {
		t.Fatalf("unsupported diagnostics = %+v", result.Diagnostics)
	}
}

func TestInferArgumentTypeMarksNamedSymbolConversionUnsupported(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.Symbols["meters"] = index.SymbolRecord{SymbolID: "meters", Name: "meters", ScopeID: "root", TypeName: "Meters"}
	argument := inferArgumentType(snapshot, "root", []parser.Token{{Spelling: "meters"}})
	conversion := CheckConversion(argument, ParseSemanticType("Seconds"))
	if conversion.Status != ConversionStatusUnsupported || conversion.Kind != ConversionKindUserDefined {
		t.Fatalf("named conversion = %+v", conversion)
	}
}
