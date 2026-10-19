package semantic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolverLanguageSignalPriorityMatrix(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		mode     string
		language string
		profile  string
		status   ContextStatus
	}{
		{name: "tops mode wins", uri: "file:///workspace/main.cpp", mode: "tops", language: "cpp", profile: "gcu400-default", status: ContextStatusPartial},
		{name: "cpp mode wins", uri: "file:///workspace/main.tops", mode: "cpp", language: "tops", profile: "cpp-default", status: ContextStatusResolved},
		{name: "tops language id", uri: "file:///workspace/main.cc", language: "tops", profile: "gcu400-default", status: ContextStatusPartial},
		{name: "tops extension", uri: "file:///workspace/main.tops", profile: "gcu400-default", status: ContextStatusPartial},
		{name: "cpp extension", uri: "file:///workspace/main.cc", profile: "cpp-default", status: ContextStatusResolved},
		{name: "unknown language", uri: "file:///workspace/main.unknown", profile: "cpp-default", status: ContextStatusMissing},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := ResolveSemanticContext(ResolverInput{DocumentURI: test.uri, Mode: test.mode, LanguageID: test.language})
			if context.Profile.ProfileID != test.profile || context.Status != test.status {
				t.Fatalf("context = %+v", context)
			}
		})
	}
}

func TestResolverDropsUnusableResponseFileCandidateAndUsesDefault(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI:   "file:///workspace/main.tops",
		WorkspaceRoot: "/workspace",
		LanguageID:    "tops",
		Candidates: []CommandCandidate{{
			SourceKind:       "compile_commands",
			SourceURI:        "file:///workspace/main.tops",
			WorkingDirectory: "/workspace",
			RawArguments:     []string{"@does-not-exist.rsp"},
		}},
	})
	if !context.Command.Fallback || context.Command.Arch != "gcu400" {
		t.Fatalf("unusable candidate context = %+v", context)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "args.rsp"), []byte("topscc -arch gcu400"), 0o644); err != nil {
		t.Fatal(err)
	}
	valid := ResolveSemanticContext(ResolverInput{DocumentURI: "file:///workspace/main.tops", WorkspaceRoot: root, LanguageID: "tops", Candidates: []CommandCandidate{{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: root, RawArguments: []string{"@args.rsp"}, Language: "TOPS", LanguageStandard: "c++17", PassKind: PassKindHostDevice}}})
	if valid.Command.Fallback || valid.Status == ContextStatusInvalid {
		t.Fatalf("response-file candidate context = %+v", valid)
	}
}

func TestResolverWorkspaceFallbackDoesNotFillTargetFields(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{DocumentURI: "file:///workspace/main.tops", LanguageID: "tops", Toolchain: ToolchainProvenance{IncludeRoots: []string{"/tops/include"}, Macros: map[string]string{"BUILTIN": "1"}}, WorkspaceFallback: WorkspaceFallbackFields{IncludeRoots: []string{"/workspace/include"}, Macros: map[string]string{"PROJECT": "1"}}})
	if context.Command.Arch != "gcu400" || context.Command.Driver != "topscc" || context.Command.Language != "TOPS" || context.Command.LanguageStandard != "c++17" || context.Command.PassKind != PassKindHostDevice {
		t.Fatalf("fallback command = %+v", context.Command)
	}
	if context.Command.FieldSource["workspace_include_roots"] != "workspace" || context.Command.FieldSource["workspace_macros"] != "workspace" {
		t.Fatalf("workspace field sources = %+v", context.Command.FieldSource)
	}
	if context.Command.FieldSource["arch"] != "fallback_default" || context.Command.FieldSource["driver"] != "fallback_default" {
		t.Fatalf("target field sources = %+v", context.Command.FieldSource)
	}
}

func TestResolverConflictFieldSourcesAndValuesStayAligned(t *testing.T) {
	first := CommandCandidate{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "-arch", "gcu400"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	second := first
	second.NormalizedArguments = []string{"topscc", "-arch", "gcu500"}
	second.Arch = "gcu500"
	context := ResolveSemanticContext(ResolverInput{DocumentURI: first.SourceURI, Candidates: []CommandCandidate{first, second}})
	conflict := context.Command.FieldConflict["arch"]
	if context.Status != ContextStatusInvalid || conflict == nil || len(conflict.Sources) != len(conflict.Values) || len(conflict.Sources) < 2 || context.Command.FieldResolution["arch"] != "invalid" {
		t.Fatalf("conflict command = %+v", context.Command)
	}
}

func TestResolverRetainsEquivalentCandidateProvenanceAndStableRepresentative(t *testing.T) {
	compile := CommandCandidate{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "main.tops"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	workspace := compile
	workspace.SourceKind = "workspace"
	workspace.Version = "workspace-v1"
	left := ResolveSemanticContext(ResolverInput{DocumentURI: compile.SourceURI, Candidates: []CommandCandidate{workspace, compile}})
	right := ResolveSemanticContext(ResolverInput{DocumentURI: compile.SourceURI, Candidates: []CommandCandidate{compile, workspace}})
	if left.Status == ContextStatusInvalid || right.Status == ContextStatusInvalid || left.Command.Source != "compile_commands" || right.Command.Source != "compile_commands" {
		t.Fatalf("equivalent contexts = %+v / %+v", left, right)
	}
	if left.Provenance["candidate_instances"] == "" || left.Provenance["candidate_instances"] != right.Provenance["candidate_instances"] {
		t.Fatalf("candidate provenance = %#v / %#v", left.Provenance, right.Provenance)
	}
}

func TestResolverMarksExplicitUnresolvedSourceFieldsPartial(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI: "file:///workspace/main.tops",
		LanguageID:  "tops",
		Toolchain: ToolchainProvenance{
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"BUILTIN": "1"},
			FieldResolution: map[string]string{
				"toolchain_include_roots": "unresolved",
				"toolchain_macros":        "resolved",
			},
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{"/workspace/include"},
			Macros:       map[string]string{"PROJECT": "1"},
			FieldResolution: map[string]string{
				"workspace_macros": "unresolved",
			},
		},
	})
	if context.Status != ContextStatusPartial || context.Profile.ProfileID != "gcu400-default" {
		t.Fatalf("unresolved source context = %+v", context)
	}
}
