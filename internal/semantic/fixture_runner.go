package semantic

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

// RunFixtureManifest executes the fixture manifest from the repository root.
func RunFixtureManifest(path string) (FixtureManifestReport, error) {
	manifest, err := LoadFixtureManifest(path)
	if err != nil {
		return FixtureManifestReport{}, err
	}
	report := ValidateFixtureManifest(manifest)
	if report.Failed != 0 {
		return report, nil
	}
	root, err := fixtureRepositoryRoot(path, manifest)
	if err != nil {
		return FixtureManifestReport{}, err
	}
	fileValidation := ValidateFixtureManifestFiles(manifest, root)
	if fileValidation.Failed != 0 {
		return fileValidation, nil
	}
	return ExecuteFixtureManifest(manifest, root), nil
}

// ExecuteFixtureManifest executes every validated fixture using parser, index and semantic layers.
func ExecuteFixtureManifest(manifest FixtureManifest, repositoryRoot string) FixtureManifestReport {
	validation := ValidateFixtureManifest(manifest)
	if validation.Failed != 0 {
		return validation
	}
	report := FixtureManifestReport{ByCapabilityScenario: make(map[string]FixtureGroupReport)}
	for _, fixture := range manifest.Cases {
		recordFixtureOutcome(&report, fixture, executeFixtureCase(fixture, repositoryRoot))
	}
	sortFixtureFailures(&report)
	return report
}

func executeFixtureCase(fixture FixtureCase, repositoryRoot string) []string {
	texts := make(map[string]string, len(fixture.SourceFiles))
	uris := make(map[string]string, len(fixture.SourceFiles))
	for _, sourceFile := range fixture.SourceFiles {
		path, err := fixtureSourcePath(repositoryRoot, sourceFile)
		if err != nil {
			return []string{fmt.Sprintf("source %q: %v", sourceFile, err)}
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return []string{fmt.Sprintf("source %q: %v", sourceFile, err)}
		}
		uri, err := fixtureSourceURI(repositoryRoot, sourceFile)
		if err != nil {
			return []string{fmt.Sprintf("source %q: %v", sourceFile, err)}
		}
		texts[uri] = string(body)
		uris[sourceFile] = uri
	}
	if len(fixture.SourceFiles) == 0 {
		return []string{"source_files is empty"}
	}
	rootURI := uris[fixture.SourceFiles[0]]
	rootText := texts[rootURI]
	provider := index.NewLocalSourceProvider(textToSourceDocuments(texts), []string{repositoryRoot})
	context := fixtureSemanticContext(fixture, rootURI, repositoryRoot)
	parseContext := fixtureParseContext(context, fixtureMacroValues(fixture))
	parsed := parser.Parse(rootText, parseContext)
	indexed := index.BuildWithProvider(rootText, parsed, parseContext, provider)
	context.ParseStatus = parsed.Status
	context.IndexStatus = string(indexed.IndexStatus)
	if context.Status == ContextStatusResolved && fixture.Scenario == "context-boundary" {
		context.Status = ContextStatusPartial
		context.PassViews = NewPassViews(context.Command.PassKind, context.Status)
	}
	if context.IncompleteRange.Start == context.IncompleteRange.End {
		for _, diagnostic := range parsed.Diagnostics {
			if diagnostic.Incomplete || (diagnostic.Recoverable && diagnostic.Code != parser.DiagnosticUnknownCondition) {
				context.IncompleteRange = diagnostic.Range
				if context.IncompleteRange.Start == context.IncompleteRange.End && len(rootText) > 0 {
					context.IncompleteRange = parser.SourceRange{Start: 0, End: 1}
				}
				break
			}
		}
	}
	context.SourceVersions = []SourceVersionEntry{{URI: rootURI, Version: index.SourceDigest(rootText)}}
	context.HeaderVersions = nil
	for _, dependency := range indexed.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		entry := SourceVersionEntry{URI: dependency.TargetURI, Version: dependency.SourceVersion.SourceVersion}
		if entry.Version == "" {
			switch dependency.Status {
			case index.HeaderDependencyMissing:
				entry.Status = SourceVersionStatusMissing
			case index.HeaderDependencyUnreadable:
				entry.Status = SourceVersionStatusUnreadable
			case index.HeaderDependencyStale:
				entry.Status = SourceVersionStatusStale
			default:
				entry.Status = SourceVersionStatusUnresolved
			}
		}
		context.SourceVersions = append(context.SourceVersions, entry)
		context.HeaderVersions = append(context.HeaderVersions, entry)
	}
	context.SourceVersionFingerprint, _ = AggregateSourceVersion(rootURI, context.SourceVersions)
	snapshot, err := BuildSemanticSnapshot(indexed, context)
	if err != nil {
		return []string{err.Error()}
	}
	reasons := make([]string, 0)
	reasons = append(reasons, validateFixturePipeline(fixture, parsed, indexed, context, snapshot)...)
	if snapshot.ProfileID != fixture.ProfileID {
		reasons = append(reasons, fmt.Sprintf("profile_id: got %q, want %q", snapshot.ProfileID, fixture.ProfileID))
	}
	if status := fixtureSnapshotStatus(fixture, snapshot); status != fixture.ExpectedStatus {
		reasons = append(reasons, fmt.Sprintf("expected_status: got %q, want %q", status, fixture.ExpectedStatus))
	}
	reasons = append(reasons, validateFixtureFacts(fixture, snapshot)...)
	reasons = append(reasons, compareFixtureDiagnostics(fixture, snapshot.Diagnostics)...)
	return reasons
}

func fixtureSemanticContext(fixture FixtureCase, documentURI, repositoryRoot string) SemanticContext {
	passKind := PassKindHostDevice
	if fixture.PassKind != "" && fixture.PassKind != "not-applicable" {
		passKind = PassKind(fixture.PassKind)
	}
	toolchainMacros := map[string]string{"__GCU_ARCH__": "400"}
	workspaceMacros := map[string]string{"SEMANTIC_FIXTURE": "1"}
	input := ResolverInput{
		DocumentURI:   documentURI,
		WorkspaceRoot: repositoryRoot,
		Mode:          "tops",
		LanguageID:    "tops",
		Toolchain: ToolchainProvenance{
			Identity:        "topscc-fixture",
			Version:         "fixture",
			IncludeRoots:    []string{repositoryRoot},
			Macros:          toolchainMacros,
			MacroVersion:    "fixture",
			NormalizedRoots: []string{documentURI},
		},
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{repositoryRoot},
			Macros:       workspaceMacros,
			Version:      "fixture",
		},
	}
	if fixture.PassKind != "" && fixture.PassKind != "not-applicable" && fixture.PassKind != string(PassKindHostDevice) {
		command := CompilationCommand{Source: "compile_commands", SourceURI: documentURI, WorkingDirectory: repositoryRoot, RawArguments: []string{"topscc", "-arch", "gcu400"}, NormalizedArguments: []string{"topscc", "-arch", "gcu400"}, Arch: "gcu400", Driver: "topscc", Language: "TOPS", LanguageStandard: "c++17", PassKind: passKind, ToolchainIncludeRoots: []string{repositoryRoot}, WorkspaceIncludeRoots: []string{repositoryRoot}, ToolchainMacros: toolchainMacros, WorkspaceMacros: workspaceMacros}
		input.ExistingCommand = &command
		profile := DefaultSemanticProfile(true)
		input.ExistingProfile = &profile
		input.PassAvailability = map[PassKind]PassAvailability{passKind: PassAvailabilityAvailable}
		input.PassProvenance = map[PassKind]map[string]string{passKind: {"source": "fixture", "pass_kind": string(passKind)}}
	}
	context := ResolveSemanticContext(input)
	context.ConditionalMacros = make(map[string]string)
	for name, value := range fixtureMacroStrings(fixture) {
		context.ConditionalMacros[name] = value
	}
	context.Provenance["fixture"] = fixture.CaseID
	if fixture.Capability == "out-of-scope" {
		context.Provenance["out_of_scope_kind"] = fixture.OutOfScopeKind
	}
	return context
}

func fixtureParseContext(context SemanticContext, macros map[string]parser.MacroValue) parser.ParseContext {
	allMacros := make(map[string]parser.MacroValue, len(macros))
	for name, value := range macros {
		allMacros[name] = value
	}
	for name, value := range context.Command.ToolchainMacros {
		if _, exists := allMacros[name]; !exists {
			allMacros[name] = fixtureMacroValue(value)
		}
	}
	for name, value := range context.Command.WorkspaceMacros {
		if _, exists := allMacros[name]; !exists {
			allMacros[name] = fixtureMacroValue(value)
		}
	}
	includeRoots := append([]string(nil), context.Command.ToolchainIncludeRoots...)
	includeRoots = append(includeRoots, context.Command.WorkspaceIncludeRoots...)
	return parser.ParseContext{DocumentURI: context.DocumentURI, LanguageID: "tops", LanguageStandard: context.Command.LanguageStandard, DriverKind: context.Command.Driver, CompilerContextStatus: string(context.Status), ArgumentProvenance: context.Command.Source, PredefinedMacros: allMacros, TargetProfile: context.Profile.TargetArch, PassKind: string(context.Command.PassKind), IncludeRoots: includeRoots, IncludeRootsAvailable: context.Status == ContextStatusResolved, DocumentVersion: 1, ContextVersion: 1}
}

func fixtureMacroStrings(fixture FixtureCase) map[string]string {
	result := make(map[string]string)
	for _, condition := range fixtureConditions(fixture) {
		if condition.ExpectedState != "active" {
			continue
		}
		for name, value := range condition.Macros {
			result[name] = fmt.Sprint(value)
		}
	}
	return result
}

func fixtureMacroValues(fixture FixtureCase) map[string]parser.MacroValue {
	result := make(map[string]parser.MacroValue)
	for _, condition := range fixtureConditions(fixture) {
		for name, value := range condition.Macros {
			if condition.ExpectedState == "unknown" {
				continue
			}
			if condition.ExpectedState == "inactive" {
				result[name] = parser.MacroValue{Kind: parser.MacroInteger}
				continue
			}
			result[name] = fixtureMacroValue(fmt.Sprint(value))
		}
	}
	return result
}

func fixtureMacroValue(value string) parser.MacroValue {
	if parsed, err := strconv.ParseInt(value, 0, 64); err == nil {
		return parser.MacroValue{Kind: parser.MacroInteger, Value: parsed}
	}
	return parser.MacroValue{Kind: parser.MacroDefined}
}

func fixtureSnapshotStatus(_ FixtureCase, snapshot SemanticSnapshot) string {
	if snapshot.UnsupportedReason != "" {
		return "unsupported"
	}
	if snapshot.InvalidReason != "" {
		return "invalid"
	}
	if snapshot.Status == SnapshotStatusDegraded {
		return "degraded"
	}
	return "complete"
}

func validateFixturePipeline(fixture FixtureCase, parsed parser.ParseResult, indexed index.IndexSnapshot, context SemanticContext, snapshot SemanticSnapshot) []string {
	reasons := make([]string, 0)
	switch fixture.ExpectedStatus {
	case "complete":
		if parsed.Status != parser.ParseStatusComplete {
			reasons = append(reasons, fmt.Sprintf("parser status: got %q, want %q", parsed.Status, parser.ParseStatusComplete))
		}
		if indexed.IndexStatus != index.IndexStatusComplete {
			reasons = append(reasons, fmt.Sprintf("index status: got %q, want %q", indexed.IndexStatus, index.IndexStatusComplete))
		}
		if context.Status != ContextStatusResolved {
			reasons = append(reasons, fmt.Sprintf("context status: got %q, want %q", context.Status, ContextStatusResolved))
		}
		if snapshot.Status != SnapshotStatusComplete {
			reasons = append(reasons, fmt.Sprintf("snapshot status: got %q, want %q", snapshot.Status, SnapshotStatusComplete))
		}
	case "invalid":
		if snapshot.Status != SnapshotStatusDegraded || snapshot.InvalidReason == "" {
			reasons = append(reasons, fmt.Sprintf("invalid snapshot status/reason: got %q/%q", snapshot.Status, snapshot.InvalidReason))
		}
	case "degraded":
		if snapshot.Status != SnapshotStatusDegraded {
			reasons = append(reasons, fmt.Sprintf("degraded snapshot status: got %q", snapshot.Status))
		}
	case "unsupported":
		if snapshot.Status != SnapshotStatusDegraded || snapshot.UnsupportedReason == "" {
			reasons = append(reasons, fmt.Sprintf("unsupported snapshot status/reason: got %q/%q", snapshot.Status, snapshot.UnsupportedReason))
		}
	}
	return reasons
}

func validateFixtureFacts(fixture FixtureCase, snapshot SemanticSnapshot) []string {
	expected := fixture.ExpectedFacts
	if expected == nil {
		fallback := defaultFixtureFactExpectation(fixture)
		expected = &fallback
	}
	reasons := make([]string, 0)
	if len(snapshot.Declarations) < expected.MinDeclarations {
		reasons = append(reasons, fmt.Sprintf("declaration facts: got %d, want at least %d", len(snapshot.Declarations), expected.MinDeclarations))
	}
	if len(snapshot.Scopes) < expected.MinScopes {
		reasons = append(reasons, fmt.Sprintf("scope facts: got %d, want at least %d", len(snapshot.Scopes), expected.MinScopes))
	}
	if len(snapshot.Types) < expected.MinTypes {
		reasons = append(reasons, fmt.Sprintf("type facts: got %d, want at least %d", len(snapshot.Types), expected.MinTypes))
	}
	if len(snapshot.Callables) < expected.MinCallables {
		reasons = append(reasons, fmt.Sprintf("callable facts: got %d, want at least %d", len(snapshot.Callables), expected.MinCallables))
	}
	if len(snapshot.Calls) < expected.MinCalls {
		reasons = append(reasons, fmt.Sprintf("call facts: got %d, want at least %d", len(snapshot.Calls), expected.MinCalls))
	}
	if expected.RequireVectorType {
		found := false
		for _, typeValue := range snapshot.Types {
			if typeValue.VectorLanes > 0 && typeValue.VectorElement != "" {
				found = true
				break
			}
		}
		if !found {
			for _, callable := range snapshot.Callables {
				if callable.ReturnType.VectorLanes > 0 && callable.ReturnType.VectorElement != "" {
					found = true
					break
				}
				for _, parameter := range callable.Parameters {
					if parameter.Type.VectorLanes > 0 && parameter.Type.VectorElement != "" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if !found {
			reasons = append(reasons, "vector fact was not found")
		}
	}
	if expected.RequireAddressSpaceType {
		found := false
		for _, typeValue := range snapshot.Types {
			if typeValue.AddressSpace != "" {
				found = true
				break
			}
		}
		if !found {
			for _, callable := range snapshot.Callables {
				if callable.ReturnType.AddressSpace != "" {
					found = true
					break
				}
				for _, parameter := range callable.Parameters {
					if parameter.Type.AddressSpace != "" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if !found {
			reasons = append(reasons, "address-space fact was not found")
		}
	}
	for _, wanted := range expected.ConversionStatuses {
		found := false
		for _, call := range snapshot.Calls {
			for _, conversion := range call.Conversions {
				if string(conversion.Status) == wanted {
					found = true
					break
				}
			}
		}
		if !found {
			reasons = append(reasons, fmt.Sprintf("conversion status %q was not found", wanted))
		}
	}
	for _, wanted := range expected.CandidateStatuses {
		found := false
		for _, call := range snapshot.Calls {
			if string(call.CandidateStatus) == wanted {
				found = true
				break
			}
		}
		if !found {
			reasons = append(reasons, fmt.Sprintf("candidate status %q was not found", wanted))
		}
	}
	for _, wanted := range expected.PassKinds {
		found := false
		for _, call := range snapshot.Calls {
			for _, passFact := range call.PassFacts {
				if string(passFact.PassKind) == wanted {
					found = true
					break
				}
			}
		}
		if !found {
			reasons = append(reasons, fmt.Sprintf("pass fact %q was not found", wanted))
		}
	}
	if fixture.Capability == "callables" {
		for _, callable := range snapshot.Callables {
			if callable.ExecutionKind == fixture.CallableKind && callable.DefaultExecutionSpace == fixture.DefaultExecutionSpace {
				return reasons
			}
		}
		reasons = append(reasons, fmt.Sprintf("callable fact %q with default_execution_space=%v was not found", fixture.CallableKind, fixture.DefaultExecutionSpace))
	}
	if fixture.Capability == "calls" {
		if len(expected.CandidateStatuses) > 0 {
			return reasons
		}
		for _, call := range snapshot.Calls {
			if call.CallerKind == fixture.CallerKind && call.CalleeKind == fixture.CalleeKind {
				return reasons
			}
		}
		reasons = append(reasons, fmt.Sprintf("call fact caller=%q callee=%q was not found", fixture.CallerKind, fixture.CalleeKind))
	}
	return reasons
}

func defaultFixtureFactExpectation(fixture FixtureCase) FixtureFactExpectation {
	expected := FixtureFactExpectation{}
	switch fixture.Capability {
	case "declarations":
		expected.MinDeclarations = 1
		expected.MinScopes = 1
	case "types":
		expected.MinTypes = 1
	case "callables":
		expected.MinCallables = 1
	case "calls":
		expected.MinCalls = 1
		if fixture.PassKind != "" && fixture.PassKind != "not-applicable" {
			expected.PassKinds = []string{fixture.PassKind}
		}
	case "address-space":
		expected.MinTypes = 1
		expected.RequireAddressSpaceType = fixture.Scenario == "invalid"
	case "vectors":
		expected.MinTypes = 1
		expected.RequireVectorType = fixture.Scenario != "incomplete"
	}
	if fixture.Scenario == "incomplete" {
		expected.MinDeclarations = 0
		expected.MinScopes = 0
		expected.MinTypes = 0
		expected.MinCallables = 0
		expected.MinCalls = 0
		expected.RequireVectorType = false
		expected.RequireAddressSpaceType = false
		expected.PassKinds = nil
	}
	return expected
}

func compareFixtureDiagnostics(fixture FixtureCase, actual []SemanticDiagnostic) []string {
	actualPublished := make(map[string]int)
	for _, diagnostic := range actual {
		key := fixtureDiagnosticKey(diagnostic.Code, string(diagnostic.Severity), diagnostic.PassKind, diagnostic.Range.Start, diagnostic.Range.End)
		actualPublished[key]++
	}
	expectedPublished := make(map[string]int)
	reasons := make([]string, 0)
	for _, expected := range fixture.ExpectedDiagnostics {
		if expected.Status != "published" {
			for key := range actualPublished {
				if strings.HasPrefix(key, expected.Code+"\x00") {
					reasons = append(reasons, fmt.Sprintf("suppressed diagnostic %q was published", expected.Code))
				}
			}
			continue
		}
		if expected.Range == nil {
			continue
		}
		key := fixtureDiagnosticKey(expected.Code, expected.Severity, expected.PassKind, expected.Range.Start, expected.Range.End)
		expectedPublished[key]++
	}
	for key, count := range expectedPublished {
		if actualPublished[key] != count {
			reasons = append(reasons, fmt.Sprintf("diagnostic %q: got %d, want %d", key, actualPublished[key], count))
		}
	}
	for key, count := range actualPublished {
		if expectedPublished[key] != count {
			reasons = append(reasons, fmt.Sprintf("unexpected diagnostic %q: got %d", key, count))
		}
	}
	sort.Strings(reasons)
	return reasons
}

func fixtureDiagnosticKey(code, severity, passKind string, start, end int) string {
	return strings.Join([]string{code, severity, passKind, strconv.Itoa(start), strconv.Itoa(end)}, "\x00")
}

func textToSourceDocuments(texts map[string]string) map[string]index.SourceDocument {
	result := make(map[string]index.SourceDocument, len(texts))
	for uri, text := range texts {
		result[uri] = index.SourceDocument{URI: uri, Text: text, Version: 1, SourceVersion: index.SourceDigest(text)}
	}
	return result
}

func fixtureRepositoryRoot(manifestPath string, manifest FixtureManifest) (string, error) {
	if len(manifest.Cases) == 0 || len(manifest.Cases[0].SourceFiles) == 0 {
		return "", fmt.Errorf("manifest has no source files")
	}
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", err
	}
	current := filepath.Dir(abs)
	for {
		path, pathErr := fixtureSourcePath(current, manifest.Cases[0].SourceFiles[0])
		if pathErr == nil {
			if _, statErr := os.Stat(path); statErr == nil {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("cannot find repository root for fixture manifest %q", manifestPath)
}

func fixtureSourcePath(repositoryRoot, source string) (string, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "file" {
		return filepath.Clean(parsed.Path), nil
	}
	if parsed.Scheme != "" {
		return "", fmt.Errorf("unsupported source URI %q", source)
	}
	return filepath.Clean(filepath.Join(repositoryRoot, filepath.FromSlash(source))), nil
}

func fixtureSourceURI(repositoryRoot, source string) (string, error) {
	path, err := fixtureSourcePath(repositoryRoot, source)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return index.NormalizeSourceURI((&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()), nil
}
