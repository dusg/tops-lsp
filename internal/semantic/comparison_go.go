package semantic

import (
	"fmt"
	"sort"
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func buildComparisonGoResults(input ComparisonCase, sourceTexts map[string]string, repositoryRoot string) ([]ComparisonPassResult, []ComparisonDiagnostic, error) {
	rootURI := normalizeComparisonURI(input.RootSourceURI)
	if rootURI == "" {
		return nil, nil, fmt.Errorf("root source URI is empty")
	}
	rootText, ok := sourceTexts[rootURI]
	if !ok {
		return nil, nil, fmt.Errorf("root source %q is not readable", rootURI)
	}
	passKind := PassKind(input.PassKind)
	if passKind != PassKindHost && passKind != PassKindDevice && passKind != PassKindHostDevice {
		return nil, nil, fmt.Errorf("invalid pass kind %q", input.PassKind)
	}
	language := "TOPS"
	arch := "gcu400"
	driver := "topscc"
	profileID := input.ProfileID
	if profileID == "cpp-default" {
		language = "C++"
		arch = ""
		driver = "clang++"
	}
	workingDirectory := normalizedWorkspaceRoot(repositoryRoot)
	if input.WorkingDirectory != "" {
		resolvedWorkingDirectory, err := normalizeWorkingDirectory(input.WorkingDirectory, repositoryRoot)
		if err != nil {
			return nil, nil, err
		}
		workingDirectory = resolvedWorkingDirectory
	}
	command := CompilationCommand{
		Source:                "compile_commands",
		SourceURI:             rootURI,
		WorkingDirectory:      workingDirectory,
		RawArguments:          append([]string(nil), input.RawArguments...),
		NormalizedArguments:   append([]string(nil), input.NormalizedArguments...),
		Arch:                  arch,
		Driver:                driver,
		Language:              language,
		LanguageStandard:      "c++17",
		PassKind:              passKind,
		ToolchainIncludeRoots: []string{repositoryRoot},
		WorkspaceIncludeRoots: []string{repositoryRoot},
		ToolchainMacros:       map[string]string{"__GCU_ARCH__": "400"},
		WorkspaceMacros:       map[string]string{"SEMANTIC_COMPARISON": "1"},
	}
	resolverInput := ResolverInput{
		DocumentURI:     rootURI,
		WorkspaceRoot:   repositoryRoot,
		Mode:            strings.ToLower(language),
		LanguageID:      strings.ToLower(language),
		ExistingCommand: &command,
		Toolchain: ToolchainProvenance{
			Identity:        "comparison-toolchain",
			Version:         input.CompilerToolVersion,
			IncludeRoots:    []string{repositoryRoot},
			Macros:          map[string]string{"__GCU_ARCH__": "400"},
			MacroVersion:    input.CompilerToolVersion,
			NormalizedRoots: []string{rootURI},
			FieldResolution: resolvedSourceFields("toolchain_include_roots", "toolchain_macros"),
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots:    []string{repositoryRoot},
			Macros:          map[string]string{"SEMANTIC_COMPARISON": "1"},
			Version:         input.CompilerToolVersion,
			FieldResolution: resolvedSourceFields("workspace_include_roots", "workspace_macros"),
		},
		DocumentVersion: 1,
		ContextVersion:  1,
	}
	profile := DefaultSemanticProfile(false)
	switch profileID {
	case "gcu400-default":
		profile = DefaultSemanticProfile(true)
	case "cpp-default":
		profile = DefaultSemanticProfile(false)
	default:
		profile = profileForTarget(profileID, false)
	}
	resolverInput.ExistingProfile = &profile
	for _, pass := range comparisonPassKinds(passKind) {
		resolverInput.PassAvailability = appendPassAvailability(resolverInput.PassAvailability, pass, PassAvailabilityAvailable)
		if resolverInput.PassProvenance == nil {
			resolverInput.PassProvenance = make(map[PassKind]map[string]string)
		}
		resolverInput.PassProvenance[pass] = map[string]string{"source": "comparison"}
	}
	context := ResolveSemanticContext(resolverInput)
	parseContext := comparisonParseContext(context)
	parsed := parser.Parse(rootText, parseContext)
	provider := index.NewLocalSourceProvider(textToSourceDocuments(sourceTexts), []string{repositoryRoot})
	indexed := index.BuildWithProvider(rootText, parsed, parseContext, provider)
	context.ParseStatus = parsed.Status
	context.IndexStatus = string(indexed.IndexStatus)
	context.SourceVersions = []SourceVersionEntry{{URI: rootURI, Version: index.SourceDigest(rootText)}}
	context.HeaderVersions = nil
	for _, dependency := range indexed.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		entry := SourceVersionEntry{URI: dependency.TargetURI, Version: dependency.SourceVersion.SourceVersion}
		if entry.Version == "" {
			entry.Status = sourceVersionStatusForDependency(dependency.Status)
		}
		context.SourceVersions = append(context.SourceVersions, entry)
		context.HeaderVersions = append(context.HeaderVersions, entry)
	}
	if fingerprint, err := AggregateSourceVersion(rootURI, context.SourceVersions); err != nil {
		return nil, nil, err
	} else {
		context.SourceVersionFingerprint = fingerprint
	}
	snapshot, err := BuildSemanticSnapshot(indexed, context)
	if err != nil {
		return nil, nil, err
	}
	if profileID != "" && snapshot.ProfileID != profileID {
		return nil, nil, fmt.Errorf("profile mismatch: got %q, want %q", snapshot.ProfileID, profileID)
	}
	neutral := comparisonDiagnosticsForPass(snapshot, PassKind("not-applicable"))
	results := make([]ComparisonPassResult, 0, len(comparisonPassKinds(passKind)))
	for _, pass := range comparisonPassKinds(passKind) {
		clangInput := input
		clangInput.WorkingDirectory = workingDirectory
		clangStatus, clangCategories := comparisonClangInput(clangInput, pass, rootURI)
		results = append(results, ComparisonPassResult{
			PassKind:        string(pass),
			GoStatus:        comparisonSnapshotStatus(snapshot),
			GoDiagnostics:   comparisonDiagnosticsForPass(snapshot, pass),
			ClangStatus:     clangStatus,
			ClangCategories: clangCategories,
		})
	}
	return results, neutral, nil
}

func comparisonParseContext(context SemanticContext) parser.ParseContext {
	macros := make(map[string]parser.MacroValue)
	for name, value := range context.Command.ToolchainMacros {
		macros[name] = fixtureMacroValue(value)
	}
	for name, value := range context.Command.WorkspaceMacros {
		if _, exists := macros[name]; !exists {
			macros[name] = fixtureMacroValue(value)
		}
	}
	languageID := "cpp"
	if context.Command.Language == "TOPS" {
		languageID = "tops"
	}
	includeRoots := append([]string(nil), context.Command.ToolchainIncludeRoots...)
	includeRoots = append(includeRoots, context.Command.WorkspaceIncludeRoots...)
	return parser.ParseContext{
		DocumentURI:           context.DocumentURI,
		LanguageID:            languageID,
		LanguageStandard:      context.Command.LanguageStandard,
		DriverKind:            context.Command.Driver,
		CompilerContextStatus: string(context.Status),
		ArgumentProvenance:    context.Command.Source,
		PredefinedMacros:      macros,
		TargetProfile:         context.Command.Arch,
		PassKind:              string(context.Command.PassKind),
		IncludeRoots:          includeRoots,
		IncludeRootsAvailable: context.Status == ContextStatusResolved,
		DocumentVersion:       context.DocumentVersion,
		ContextVersion:        context.SemanticContextVersion,
	}
}

func resolvedSourceFields(keys ...string) map[string]string {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = "resolved"
	}
	return result
}

func appendPassAvailability(values map[PassKind]PassAvailability, pass PassKind, availability PassAvailability) map[PassKind]PassAvailability {
	if values == nil {
		values = make(map[PassKind]PassAvailability)
	}
	values[pass] = availability
	return values
}

func comparisonPassKinds(passKind PassKind) []PassKind {
	if passKind == PassKindHostDevice {
		return []PassKind{PassKindHost, PassKindDevice}
	}
	return []PassKind{passKind}
}

func sourceVersionStatusForDependency(status index.HeaderDependencyStatus) SourceVersionStatus {
	switch status {
	case index.HeaderDependencyMissing:
		return SourceVersionStatusMissing
	case index.HeaderDependencyUnreadable:
		return SourceVersionStatusUnreadable
	case index.HeaderDependencyStale:
		return SourceVersionStatusStale
	default:
		return SourceVersionStatusUnresolved
	}
}

func comparisonSnapshotStatus(snapshot SemanticSnapshot) string {
	if snapshot.UnsupportedReason != "" {
		return "unsupported"
	}
	if snapshot.ContextStatus == ContextStatusInvalid {
		return "invalid"
	}
	if snapshot.Status == SnapshotStatusDegraded {
		return "degraded"
	}
	return string(SnapshotStatusComplete)
}

func comparisonDiagnosticsForPass(snapshot SemanticSnapshot, pass PassKind) []ComparisonDiagnostic {
	result := make([]ComparisonDiagnostic, 0)
	for _, diagnostic := range snapshot.Diagnostics {
		if pass == PassKind("not-applicable") {
			if diagnostic.PassKind != "not-applicable" {
				continue
			}
		} else if diagnostic.PassKind != string(pass) {
			continue
		}
		if diagnostic.Range.End <= diagnostic.Range.Start || diagnostic.Range.Start < 0 {
			continue
		}
		result = append(result, ComparisonDiagnostic{
			Code:     diagnostic.Code,
			Severity: string(diagnostic.Severity),
			PassKind: diagnostic.PassKind,
			Range: FixtureSourceRange{
				SourceURI: snapshot.Context.DocumentURI,
				Start:     diagnostic.Range.Start,
				End:       diagnostic.Range.End,
			},
		})
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Code != result[right].Code {
			return result[left].Code < result[right].Code
		}
		if result[left].Range.Start != result[right].Range.Start {
			return result[left].Range.Start < result[right].Range.Start
		}
		return result[left].Range.End < result[right].Range.End
	})
	return result
}
