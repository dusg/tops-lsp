package server

import (
	"context"
	"testing"

	"tops-lsp/internal/semantic"
)

func TestUS1SemanticSnapshotContainsFactsAndDiagnostics(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{CompilationContext: &CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "test",
		TargetProfile:         "gcu400",
		PassKind:              "host+device",
	}})
	state, err := server.openDocument("file:///workspace/us1.tops", "tops", 1, "int known; int use() { return missing_name; }")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	snapshot, ok := server.semanticStore.Get(state.URI)
	if !ok || len(snapshot.Declarations) == 0 || len(snapshot.Scopes) == 0 || len(snapshot.Callables) == 0 || len(snapshot.References) == 0 {
		t.Fatalf("semantic snapshot = %+v, ok = %v", snapshot, ok)
	}
	if snapshot.ProfileID == "" || snapshot.ContextStatus == "" || snapshot.SourceVersionFingerprint == "" {
		t.Fatalf("snapshot metadata = %+v", snapshot)
	}
	foundUnknown := false
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Code == "tops-semantic-unknown-name" && diagnostic.Severity == semantic.SeverityError {
			foundUnknown = true
		}
	}
	if !foundUnknown {
		t.Fatalf("snapshot diagnostics = %+v", snapshot.Diagnostics)
	}
}

func TestUS1OrdinaryCppUsesCppDefaultWithoutTopsRules(t *testing.T) {
	server := New(nil)
	state, err := server.openDocument("file:///workspace/us1.cpp", "cpp", 1, "int known; int use() { return known; }")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	context, ok := server.CurrentSemanticContext(state.URI)
	if !ok || context.Profile.ProfileID != "cpp-default" || context.Profile.TargetRules || context.PassViews[semantic.PassKindDevice].Availability != semantic.PassAvailabilityNotApplicable {
		t.Fatalf("ordinary C++ context = %+v, ok = %v", context, ok)
	}
}
