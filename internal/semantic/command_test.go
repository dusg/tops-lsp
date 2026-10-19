package semantic

import "testing"

func TestResolveSemanticContextUsesDefaultTopsCommand(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI:   "file:///workspace/main.tops",
		WorkspaceRoot: "/workspace",
		LanguageID:    "tops",
		Toolchain: ToolchainProvenance{
			Identity:     "topscc",
			Version:      "1",
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"__GCU_ARCH__": "400"},
			MacroVersion: "1",
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{"/workspace/include"},
			Macros:       map[string]string{"PROJECT": "1"},
			Version:      "workspace-v1",
		},
	})
	if context.Status != ContextStatusResolved || context.Profile.ProfileID != "gcu400-default" {
		t.Fatalf("context status/profile = %q/%q", context.Status, context.Profile.ProfileID)
	}
	if context.Command.Arch != "gcu400" || context.Command.Driver != "topscc" || context.Command.Language != "TOPS" || context.Command.LanguageStandard != "c++17" || context.Command.PassKind != PassKindHostDevice || !context.Command.Fallback {
		t.Fatalf("default command = %+v", context.Command)
	}
	if context.PassViews[PassKindHost].Availability != PassAvailabilityAvailable || context.PassViews[PassKindDevice].Availability != PassAvailabilityAvailable {
		t.Fatalf("pass views = %+v", context.PassViews)
	}
	for _, key := range CommandFieldKeys() {
		if _, ok := context.Command.FieldSource[key]; !ok {
			t.Fatalf("field source missing key %q", key)
		}
		if _, ok := context.Command.FieldResolution[key]; !ok {
			t.Fatalf("field resolution missing key %q", key)
		}
		if _, ok := context.Command.FieldConflict[key]; !ok {
			t.Fatalf("field conflict missing key %q", key)
		}
	}
}

func TestConflictSourceInstancesUseCanonicalGoldenVectors(t *testing.T) {
	compile, err := commandCandidateInstance(CommandCandidate{
		SourceKind:          "compile_commands",
		SourceURI:           "file:///workspace/main.tops",
		WorkingDirectory:    "/workspace",
		NormalizedArguments: []string{"topscc", "-arch", "gcu400"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if compile.Instance != "f61c05edb65ed20e049627773f3b6d117000392cb09b6b339d89c9d452f6a445" {
		t.Fatalf("compile instance = %q", compile.Instance)
	}
	workspace, err := commandCandidateInstance(CommandCandidate{
		SourceKind:          "workspace",
		SourceURI:           "file:///workspace/main.tops",
		Version:             "w1",
		WorkingDirectory:    "/workspace",
		NormalizedArguments: []string{"topscc", "-arch", "gcu400"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if workspace.Instance != "f46a965ae8153e3544c8ef226740dff3e62794b4f1872e2a73c9d9a7b831e63d" {
		t.Fatalf("workspace instance = %q", workspace.Instance)
	}
	toolchain, err := ToolchainConflictSource(ToolchainProvenance{Identity: "topscc", Version: "1", NormalizedRoots: []string{"file:///tops/include"}, MacroVersion: "m1"})
	if err != nil {
		t.Fatal(err)
	}
	if toolchain.Instance != "2d533f73f8fa271fcb43b3ade3e78cb08a820ceb32ea2d7125dabb2364833edf" {
		t.Fatalf("toolchain instance = %q", toolchain.Instance)
	}
	if fallback := FallbackConflictSource(); fallback != (ConflictSource{Kind: "fallback_default", Instance: "gcu400-default"}) {
		t.Fatalf("fallback instance = %+v", fallback)
	}
}

func TestResolveSemanticContextKeepsOrdinaryCppOutOfTopsRules(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{DocumentURI: "file:///workspace/main.cpp", LanguageID: "cpp"})
	if context.Profile.ProfileID != "cpp-default" || context.Profile.TargetRules || context.Command.Language == "TOPS" {
		t.Fatalf("ordinary C++ context = %+v", context)
	}
	if context.Command.PassKind != PassKindHost || context.PassViews[PassKindDevice].Availability != PassAvailabilityNotApplicable {
		t.Fatalf("ordinary C++ pass views = %+v", context.PassViews)
	}
}

func TestResolveSemanticContextFiltersURIAndDeduplicatesEquivalentCandidates(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI: "file:///workspace/main.tops",
		Candidates: []CommandCandidate{
			{SourceKind: "workspace", SourceURI: "main.tops", Version: "workspace-v1", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "main.tops"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice},
			{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "main.tops"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice},
			{SourceKind: "compile_commands", SourceURI: "file:///other/main.tops", WorkingDirectory: "/other", NormalizedArguments: []string{"topscc", "other.tops"}, Language: "C++", Arch: "gcu500", Driver: "clang", LanguageStandard: "c++20", PassKind: PassKindHost},
		},
	})
	if context.Status == ContextStatusInvalid || context.Command.Source != "compile_commands" || context.Command.Arch != "gcu400" {
		t.Fatalf("deduplicated context = %+v", context)
	}
}

func TestResolveSemanticContextRetainsStableConflictInstances(t *testing.T) {
	first := CommandCandidate{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "-arch", "gcu400", "main.tops"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	second := CommandCandidate{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "-arch", "gcu500", "main.tops"}, Language: "TOPS", Arch: "gcu500", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	left := ResolveSemanticContext(ResolverInput{DocumentURI: first.SourceURI, Candidates: []CommandCandidate{first, second}})
	right := ResolveSemanticContext(ResolverInput{DocumentURI: first.SourceURI, Candidates: []CommandCandidate{second, first}})
	if left.Status != ContextStatusInvalid || right.Status != ContextStatusInvalid {
		t.Fatalf("conflict statuses = %q/%q", left.Status, right.Status)
	}
	if len(left.Conflicts) == 0 || len(right.Conflicts) == 0 || left.Conflicts[0] != right.Conflicts[0] {
		t.Fatalf("conflicts = %#v / %#v", left.Conflicts, right.Conflicts)
	}
}

func TestResolveSemanticContextDoesNotUseDefaultForExplicitLanguageConflict(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI: "file:///workspace/main.tops",
		Candidates: []CommandCandidate{
			{SourceKind: "compile_commands", SourceURI: "file:///workspace/main.tops", WorkingDirectory: "/workspace", NormalizedArguments: []string{"topscc", "main.tops"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice},
			{SourceKind: "workspace", SourceURI: "file:///workspace/main.tops", Version: "workspace-v1", WorkingDirectory: "/workspace", NormalizedArguments: []string{"clang++", "main.tops"}, Language: "C++", Arch: "", Driver: "clang++", LanguageStandard: "c++17", PassKind: PassKindHost},
		},
	})
	if context.Status != ContextStatusInvalid || context.Command.Fallback {
		t.Fatalf("language conflict context = %+v", context)
	}
}

func TestResolveSemanticContextOnlyReplacesEmptyProfile(t *testing.T) {
	command := CompilationCommand{Source: "compile_commands", Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	context := ResolveSemanticContext(ResolverInput{DocumentURI: "file:///workspace/main.tops", ExistingCommand: &command})
	if context.Command.Source != command.Source || context.Command.Arch != command.Arch || context.Profile.ProfileID != "gcu400" || context.Profile.Fallback {
		t.Fatalf("existing command context = %+v", context)
	}
}

func TestExistingCommandUsesTargetProfileAndPreservesUnresolvedSources(t *testing.T) {
	command := CompilationCommand{Source: "compile_commands", Language: "TOPS", Arch: "gcu500", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
	context := ResolveSemanticContext(ResolverInput{DocumentURI: "file:///workspace/main.tops", ExistingCommand: &command})
	if context.Profile.ProfileID != "gcu500" || context.Profile.TargetArch != "gcu500" || context.Profile.Fallback || context.Status != ContextStatusPartial {
		t.Fatalf("existing target context = %+v", context)
	}
	for _, key := range []string{"toolchain_include_roots", "toolchain_macros", "workspace_include_roots", "workspace_macros"} {
		if context.Command.FieldResolution[key] != "unresolved" {
			t.Fatalf("field %q resolution = %q, command = %+v", key, context.Command.FieldResolution[key], context.Command)
		}
	}
}

func TestExistingCommandAndProfileWinWhenCandidateIsAlsoPresent(t *testing.T) {
	documentURI := "file:///workspace/main.tops"
	command := CompilationCommand{Source: "compile_commands", SourceURI: documentURI, Language: "TOPS", Arch: "gcu500", Driver: "topscc", LanguageStandard: "c++20", PassKind: PassKindDevice}
	profile := DefaultSemanticProfile(true)
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI:     documentURI,
		LanguageID:      "tops",
		ExistingCommand: &command,
		ExistingProfile: &profile,
		Candidates:      []CommandCandidate{{SourceKind: "workspace", SourceURI: documentURI, WorkingDirectory: "/workspace", NormalizedArguments: []string{"clang++", "main.tops"}, Language: "C++", Driver: "clang++", LanguageStandard: "c++17", PassKind: PassKindHost}},
	})
	if context.Command.Source != "compile_commands" || context.Command.Arch != "gcu500" || context.Command.PassKind != PassKindDevice || context.Profile.ProfileID != profile.ProfileID {
		t.Fatalf("existing command/profile were ignored: %+v", context)
	}
}
