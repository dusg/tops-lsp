package semantic

import "testing"

func TestSemanticContextContractStates(t *testing.T) {
	tests := []struct {
		name           string
		status         ContextStatus
		fallback       bool
		profileID      string
		wantTarget     bool
		wantProvenance string
	}{
		{name: "resolved", status: ContextStatusResolved, profileID: "gcu400", wantTarget: true, wantProvenance: "compile_commands"},
		{name: "partial fallback", status: ContextStatusPartial, fallback: true, profileID: "gcu400-default", wantTarget: true, wantProvenance: "fallback_default"},
		{name: "missing cpp", status: ContextStatusMissing, profileID: "cpp-default", wantTarget: false, wantProvenance: "unresolved"},
		{name: "invalid target conflict", status: ContextStatusInvalid, fallback: true, profileID: "gcu400-default", wantTarget: false, wantProvenance: "conflict"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			context := SemanticContext{
				DocumentURI: "file:///workspace/main.tops",
				Command: CompilationCommand{
					PassKind: PassKindHostDevice,
					Fallback: test.fallback,
				},
				Profile: SemanticProfile{
					ProfileID:   test.profileID,
					TargetArch:  "gcu400",
					Fallback:    test.fallback,
					TargetRules: test.wantTarget,
				},
				Status: test.status,
				Provenance: map[string]string{
					"command": test.wantProvenance,
				},
			}
			if err := context.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			if context.Profile.ProfileID == "" {
				t.Fatal("profile ID is empty")
			}
			if context.Provenance["command"] != test.wantProvenance {
				t.Fatalf("command provenance = %q", context.Provenance["command"])
			}
			if test.wantTarget && !context.Profile.TargetRules {
				t.Fatal("target rules were unexpectedly disabled")
			}
			if !test.wantTarget && context.Profile.TargetRules {
				t.Fatal("target rules were unexpectedly enabled")
			}
		})
	}
}

func TestSemanticContextRequiresProfileAndValidStatus(t *testing.T) {
	missingProfile := SemanticContext{DocumentURI: "file:///workspace/main.tops", Status: ContextStatusResolved}
	if err := missingProfile.Validate(); err == nil {
		t.Fatal("Validate() accepted an empty profile")
	}
	invalidStatus := SemanticContext{DocumentURI: "file:///workspace/main.tops", Profile: SemanticProfile{ProfileID: "cpp-default"}, Status: ContextStatus("unknown")}
	if err := invalidStatus.Validate(); err == nil {
		t.Fatal("Validate() accepted an invalid status")
	}
}

func TestPassViewsKeepHostAndDeviceIndependent(t *testing.T) {
	context := SemanticContext{
		DocumentURI: "file:///workspace/main.tops",
		Command:     CompilationCommand{PassKind: PassKindHostDevice},
		Profile:     SemanticProfile{ProfileID: "gcu400-default", TargetArch: "gcu400", TargetRules: true},
		Status:      ContextStatusPartial,
		PassViews: map[PassKind]PassView{
			PassKindHost:   {PassKind: PassKindHost, Availability: PassAvailabilityAvailable, Status: ContextStatusPartial},
			PassKindDevice: {PassKind: PassKindDevice, Availability: PassAvailabilityMissing, Status: ContextStatusPartial},
		},
	}
	if err := context.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if context.PassViews[PassKindHost].Availability != PassAvailabilityAvailable || context.PassViews[PassKindDevice].Availability != PassAvailabilityMissing {
		t.Fatalf("pass views = %+v", context.PassViews)
	}
}

func TestResolverAppliesProviderPassAvailabilityAndProvenance(t *testing.T) {
	context := ResolveSemanticContext(ResolverInput{
		DocumentURI: "file:///workspace/main.tops",
		LanguageID:  "tops",
		Toolchain:   ToolchainProvenance{IncludeRoots: []string{"/tops/include"}, Macros: map[string]string{"TOPS": "1"}},
		PassAvailability: map[PassKind]PassAvailability{
			PassKindHost:   PassAvailabilityAvailable,
			PassKindDevice: PassAvailabilityMissing,
		},
		PassProvenance: map[PassKind]map[string]string{
			PassKindHost: {"source": "toolchain"},
		},
	})
	if context.PassViews[PassKindHost].Availability != PassAvailabilityAvailable || context.PassViews[PassKindDevice].Availability != PassAvailabilityMissing || context.PassViews[PassKindHost].Provenance["source"] != "toolchain" {
		t.Fatalf("pass inputs = %+v", context.PassViews)
	}
}

func TestResolverSynchronizesPassViewsAcrossCommandSources(t *testing.T) {
	base := ResolverInput{
		DocumentURI:   "file:///workspace/main.tops",
		WorkspaceRoot: "/workspace",
		LanguageID:    "tops",
		Toolchain: ToolchainProvenance{
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"__GCU_ARCH__": "400"},
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{"/workspace/include"},
			Macros:       map[string]string{"PROJECT": "1"},
		},
		PassAvailability: map[PassKind]PassAvailability{
			PassKindHost:   PassAvailabilityAvailable,
			PassKindDevice: PassAvailabilityMissing,
		},
		PassProvenance: map[PassKind]map[string]string{
			PassKindHost:   {"source": "host-provider"},
			PassKindDevice: {"source": "device-provider"},
		},
	}
	cases := []struct {
		name  string
		input ResolverInput
	}{
		{name: "default", input: base},
		{
			name: "candidate",
			input: func() ResolverInput {
				input := base
				input.Candidates = []CommandCandidate{{
					SourceKind:       "compile_commands",
					SourceURI:        input.DocumentURI,
					WorkingDirectory: input.WorkspaceRoot,
					RawArguments:     []string{"topscc", "-arch", "gcu400"},
					Language:         "TOPS",
					Arch:             "gcu400",
					Driver:           "topscc",
					LanguageStandard: "c++17",
					PassKind:         PassKindHostDevice,
				}}
				return input
			}(),
		},
		{
			name: "existing",
			input: func() ResolverInput {
				input := base
				input.ExistingCommand = &CompilationCommand{
					Source:           "compile_commands",
					SourceURI:        input.DocumentURI,
					WorkingDirectory: input.WorkspaceRoot,
					Language:         "TOPS",
					Arch:             "gcu400",
					Driver:           "topscc",
					LanguageStandard: "c++17",
					PassKind:         PassKindHostDevice,
				}
				return input
			}(),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			context := ResolveSemanticContext(test.input)
			if context.Status != ContextStatusPartial {
				t.Fatalf("context status = %q, want partial", context.Status)
			}
			if len(context.Command.ToolchainIncludeRoots) != 1 || context.Command.ToolchainIncludeRoots[0] != "/tops/include" || len(context.Command.WorkspaceIncludeRoots) != 1 || context.Command.WorkspaceIncludeRoots[0] != "/workspace/include" {
				t.Fatalf("command roots = %+v", context.Command)
			}
			for _, key := range []string{"toolchain_include_roots", "toolchain_macros"} {
				if context.Command.FieldSource[key] != "toolchain_default" || context.Command.FieldResolution[key] != "resolved" {
					t.Fatalf("toolchain field %q = source %q resolution %q", key, context.Command.FieldSource[key], context.Command.FieldResolution[key])
				}
			}
			for _, key := range []string{"workspace_include_roots", "workspace_macros"} {
				if context.Command.FieldSource[key] != "workspace" || context.Command.FieldResolution[key] != "resolved" {
					t.Fatalf("workspace field %q = source %q resolution %q", key, context.Command.FieldSource[key], context.Command.FieldResolution[key])
				}
			}
			for _, pass := range []PassKind{PassKindHost, PassKindDevice} {
				view := context.PassViews[pass]
				if view.Status != ContextStatusPartial || len(view.RulesAvailable) != 0 {
					t.Fatalf("%s view = %+v", pass, view)
				}
				if view.Provenance["profile"] != context.Profile.ProfileID || view.Provenance["command"] != context.Command.Source || view.Provenance["source"] == "" {
					t.Fatalf("%s provenance = %+v", pass, view.Provenance)
				}
			}
		})
	}
}

func TestResolverExposesFourPassViewCombinations(t *testing.T) {
	base := ResolverInput{
		DocumentURI:   "file:///workspace/main.tops",
		WorkspaceRoot: "/workspace",
		LanguageID:    "tops",
		Toolchain: ToolchainProvenance{
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"__GCU_ARCH__": "400"},
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{"/workspace/include"},
			Macros:       map[string]string{"PROJECT": "1"},
		},
	}
	cases := []struct {
		name          string
		commandPass   PassKind
		availability  map[PassKind]PassAvailability
		wantStatus    ContextStatus
		wantHost      PassAvailability
		wantDevice    PassAvailability
		wantHostRules bool
		wantDevRules  bool
	}{
		{name: "host-only", commandPass: PassKindHost, availability: map[PassKind]PassAvailability{PassKindHost: PassAvailabilityAvailable}, wantStatus: ContextStatusResolved, wantHost: PassAvailabilityAvailable, wantDevice: PassAvailabilityNotApplicable, wantHostRules: true},
		{name: "device-only", commandPass: PassKindDevice, availability: map[PassKind]PassAvailability{PassKindDevice: PassAvailabilityAvailable}, wantStatus: ContextStatusResolved, wantHost: PassAvailabilityNotApplicable, wantDevice: PassAvailabilityAvailable, wantDevRules: true},
		{name: "host-and-device", commandPass: PassKindHostDevice, availability: map[PassKind]PassAvailability{PassKindHost: PassAvailabilityAvailable, PassKindDevice: PassAvailabilityAvailable}, wantStatus: ContextStatusResolved, wantHost: PassAvailabilityAvailable, wantDevice: PassAvailabilityAvailable, wantHostRules: true, wantDevRules: true},
		{name: "device-missing", commandPass: PassKindHostDevice, availability: map[PassKind]PassAvailability{PassKindHost: PassAvailabilityAvailable, PassKindDevice: PassAvailabilityMissing}, wantStatus: ContextStatusPartial, wantHost: PassAvailabilityAvailable, wantDevice: PassAvailabilityMissing},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.ExistingCommand = &CompilationCommand{Source: "compile_commands", SourceURI: input.DocumentURI, WorkingDirectory: "/workspace", Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: test.commandPass}
			input.PassAvailability = test.availability
			input.PassProvenance = map[PassKind]map[string]string{PassKindHost: {"source": "host"}, PassKindDevice: {"source": "device"}}
			context := ResolveSemanticContext(input)
			if context.Status != test.wantStatus || context.PassViews[PassKindHost].Availability != test.wantHost || context.PassViews[PassKindDevice].Availability != test.wantDevice {
				t.Fatalf("context = %+v", context)
			}
			for _, pass := range []PassKind{PassKindHost, PassKindDevice} {
				view := context.PassViews[pass]
				if view.Status != test.wantStatus || view.Provenance["profile"] != context.Profile.ProfileID || view.Provenance["source"] == "" {
					t.Fatalf("%s view = %+v", pass, view)
				}
			}
			if hasPassRule(context.PassViews[PassKindHost].RulesAvailable, "host-device-boundary") != test.wantHostRules || hasPassRule(context.PassViews[PassKindDevice].RulesAvailable, "vector-constraint") != test.wantDevRules {
				t.Fatalf("pass rules = %+v", context.PassViews)
			}
		})
	}
}

func TestResolverPreservesUnresolvedSourceResolutionAcrossCommandSources(t *testing.T) {
	base := ResolverInput{
		DocumentURI:   "file:///workspace/main.tops",
		WorkspaceRoot: "/workspace",
		LanguageID:    "tops",
		Toolchain: ToolchainProvenance{
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"__GCU_ARCH__": "400"},
			FieldResolution: map[string]string{
				"toolchain_include_roots": "unresolved",
				"toolchain_macros":        "unresolved",
			},
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{"/workspace/include"},
			Macros:       map[string]string{"PROJECT": "1"},
			FieldResolution: map[string]string{
				"workspace_include_roots": "unresolved",
				"workspace_macros":        "unresolved",
			},
		},
	}
	inputs := []struct {
		name  string
		input ResolverInput
	}{
		{name: "default", input: base},
		{
			name: "candidate",
			input: func() ResolverInput {
				input := base
				input.Candidates = []CommandCandidate{{SourceKind: "compile_commands", SourceURI: input.DocumentURI, WorkingDirectory: input.WorkspaceRoot, RawArguments: []string{"topscc", "-arch", "gcu400"}, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}}
				return input
			}(),
		},
		{
			name: "existing",
			input: func() ResolverInput {
				input := base
				input.ExistingCommand = &CompilationCommand{Source: "compile_commands", SourceURI: input.DocumentURI, WorkingDirectory: input.WorkspaceRoot, Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: PassKindHostDevice}
				return input
			}(),
		},
	}
	for _, test := range inputs {
		t.Run(test.name, func(t *testing.T) {
			context := ResolveSemanticContext(test.input)
			if context.Status != ContextStatusPartial {
				t.Fatalf("context status = %q", context.Status)
			}
			for _, key := range []string{"toolchain_include_roots", "toolchain_macros", "workspace_include_roots", "workspace_macros"} {
				if context.Command.FieldResolution[key] != "unresolved" {
					t.Fatalf("field %q resolution = %q, command = %+v", key, context.Command.FieldResolution[key], context.Command)
				}
			}
		})
	}
}

func hasPassRule(rules []string, expected string) bool {
	for _, rule := range rules {
		if rule == expected {
			return true
		}
	}
	return false
}
