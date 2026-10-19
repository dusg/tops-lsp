package server

import (
	"context"
	"testing"

	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
)

func TestServerUsesResolvedSemanticContextForParserAndIndex(t *testing.T) {
	var parsedContext parser.ParseContext
	server := NewWithOptions(nil, ServerOptions{
		CommandCandidateProvider: newSemanticTestProvider(semanticTestProviderOptions{
			Macros: map[string]string{"PROJECT_FLAG": "1"},
			PassAvailability: map[semantic.PassKind]semantic.PassAvailability{
				semantic.PassKindHost:   semantic.PassAvailabilityAvailable,
				semantic.PassKindDevice: semantic.PassAvailabilityAvailable,
			},
		}),
		Parse: func(text string, context parser.ParseContext) parser.ParseResult {
			parsedContext = context
			return parser.Parse(text, context)
		},
	})
	state, err := server.openDocument("file:///workspace/default.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	resolved, ok := server.CurrentSemanticContext(state.URI)
	if !ok || resolved.Status != semantic.ContextStatusResolved || resolved.Command.Arch != "gcu400" || resolved.Command.Driver != "topscc" || resolved.Command.LanguageStandard != "c++17" || resolved.Command.PassKind != semantic.PassKindHostDevice {
		t.Fatalf("resolved context = %+v, ok = %v", resolved, ok)
	}
	if parsedContext.TargetProfile != "gcu400" || parsedContext.DriverKind != "topscc" || parsedContext.LanguageStandard != "c++17" || parsedContext.PassKind != string(semantic.PassKindHostDevice) || parsedContext.CompilerContextStatus != string(semantic.ContextStatusResolved) {
		t.Fatalf("parser context = %+v", parsedContext)
	}
	if !parsedContext.IncludeRootsAvailable || !containsString(parsedContext.IncludeRoots, "/tops/include") || !containsString(parsedContext.IncludeRoots, "/workspace/include") || parsedContext.PredefinedMacros["__GCU_ARCH__"].Value != 400 || parsedContext.PredefinedMacros["PROJECT_FLAG"].Value != 1 {
		t.Fatalf("parser provenance = %+v", parsedContext)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestServerResolvesSemanticContextAlongsideIndex(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{SourceProvider: testSourceProvider{roots: []string{"/workspace/include"}}})
	state, err := server.openDocument("file:///workspace/main.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	resolved, ok := server.CurrentSemanticContext(state.URI)
	if !ok || resolved.Profile.ProfileID != "gcu400-default" || resolved.Command.Language != "TOPS" || resolved.Command.PassKind != semantic.PassKindHostDevice {
		t.Fatalf("semantic context = %+v, ok = %v", resolved, ok)
	}
	if resolved.Status != semantic.ContextStatusPartial || resolved.SemanticContextVersion != server.currentContextVersion() {
		t.Fatalf("semantic context status/version = %q/%d", resolved.Status, resolved.SemanticContextVersion)
	}
	if _, ok := server.CurrentIndexSnapshot(state.URI); !ok {
		t.Fatal("index snapshot was not committed")
	}

	updated := CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "compile_commands",
		TargetProfile:         "gcu400",
		PassKind:              "device",
	}
	server.UpdateCompilationContext(updated)
	server.WaitForAnalysis()
	updatedContext, ok := server.CurrentSemanticContext(state.URI)
	if !ok || updatedContext.SemanticContextVersion != server.currentContextVersion() || updatedContext.Command.PassKind != semantic.PassKindDevice {
		t.Fatalf("updated semantic context = %+v, ok = %v", updatedContext, ok)
	}
}

func TestServerPublishesSemanticUnknownNameWithWireMetadata(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{CompilationContext: &CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "topscc",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "test",
		TargetProfile:         "gcu400",
		PassKind:              "host",
	}})
	state, err := server.openDocument("file:///workspace/unknown.tops", "tops", 1, "int use() { return missing_name; }")
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingDiagnosticPublisher{}
	server.parseAndPublish(context.Background(), publisher, state)
	for _, params := range publisher.params {
		for _, diagnostic := range params.Diagnostics {
			if diagnostic.Code != "tops-semantic-unknown-name" {
				continue
			}
			if diagnostic.Severity != protocol.DiagnosticSeverityError || diagnostic.Data == nil || diagnostic.Data.ProfileID == "" || diagnostic.Data.ContextStatus == "" || diagnostic.Data.SemanticKind != "unknown-name" || diagnostic.Data.DocumentVersion != 1 {
				t.Fatalf("semantic diagnostic = %+v", diagnostic)
			}
			return
		}
	}
	t.Fatal("semantic unknown-name diagnostic was not published")
}

func TestServerPreservesExplicitCompilationLanguageForTopsDocument(t *testing.T) {
	var parsedContext parser.ParseContext
	server := NewWithOptions(nil, ServerOptions{
		CompilationContext: &CompilationContext{
			Language:              "C++",
			Mode:                  "tops",
			LanguageStandard:      "c++17",
			DriverKind:            "clang++",
			CompilerContextStatus: "resolved",
			ArgumentProvenance:    "compile_commands",
			PassKind:              "host",
		},
		Parse: func(text string, context parser.ParseContext) parser.ParseResult {
			parsedContext = context
			return parser.Parse(text, context)
		},
	})
	state, err := server.openDocument("file:///workspace/explicit-cpp.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	resolved, ok := server.CurrentSemanticContext(state.URI)
	if !ok {
		t.Fatal("semantic context was not published")
	}
	if resolved.Command.Language != "C++" || resolved.Profile.ProfileID != "cpp-default" || resolved.Command.PassKind != semantic.PassKindHost {
		t.Fatalf("semantic context = %+v", resolved)
	}
	if resolved.PassViews[semantic.PassKindHost].Availability != semantic.PassAvailabilityAvailable || resolved.PassViews[semantic.PassKindDevice].Availability != semantic.PassAvailabilityNotApplicable {
		t.Fatalf("semantic pass views = %+v", resolved.PassViews)
	}
	if parsedContext.LanguageID != "tops" || parsedContext.DriverKind != "clang++" || parsedContext.PassKind != string(semantic.PassKindHost) {
		t.Fatalf("parser context = %+v", parsedContext)
	}
}

func TestServerPreservesExplicitCompilationModeForTopsDocument(t *testing.T) {
	server := NewWithOptions(nil, ServerOptions{CompilationContext: &CompilationContext{
		Mode:                  "cpp",
		LanguageStandard:      "c++17",
		DriverKind:            "clang++",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "compile_commands",
		PassKind:              "host",
	}})
	state, err := server.openDocument("file:///workspace/explicit-mode.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	resolved, ok := server.CurrentSemanticContext(state.URI)
	if !ok || resolved.Command.Language != "C++" || resolved.Profile.ProfileID != "cpp-default" || resolved.Command.PassKind != semantic.PassKindHost {
		t.Fatalf("semantic context = %+v, ok = %v", resolved, ok)
	}
}
