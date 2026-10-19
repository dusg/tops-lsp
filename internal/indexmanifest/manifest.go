package indexmanifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	indexpkg "tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
)

type FixtureCase struct {
	CaseID                string                  `json:"case_id"`
	Category              string                  `json:"category"`
	ValidationMode        string                  `json:"validation_mode,omitempty"`
	SkipReason            string                  `json:"skip_reason,omitempty"`
	Context               FixtureContext          `json:"context"`
	SourceURIs            []string                `json:"source_uris,omitempty"`
	SourceFiles           []string                `json:"source_files,omitempty"`
	IncludeRoots          []string                `json:"include_roots,omitempty"`
	ExpectedFile          string                  `json:"expected_file,omitempty"`
	ExpectedSymbolCount   int                     `json:"expected_symbol_count"`
	ExpectedLocationCount int                     `json:"expected_location_count"`
	ExpectedStatus        string                  `json:"expected_status"`
	LSPOperations         []string                `json:"lsp_operations,omitempty"`
	ExpectedDiagnostics   []DiagnosticExpectation `json:"expected_diagnostics,omitempty"`
	DocumentVersions      []int                   `json:"document_versions,omitempty"`
	VersionTexts          []string                `json:"version_texts,omitempty"`
	ExpectedFinalVersion  *int                    `json:"expected_final_version,omitempty"`
	ExpectedSymbols       []SymbolExpectation     `json:"expected_symbols,omitempty"`
	ExpectedReferences    []ReferenceExpectation  `json:"expected_references,omitempty"`
	ExpectedLocations     []LocationExpectation   `json:"expected_locations,omitempty"`
	ExpectedDependencies  []string                `json:"expected_dependencies,omitempty"`
	Queries               []ManifestQuery         `json:"queries,omitempty"`
}

type FixtureContext struct {
	LanguageStandard      string                       `json:"language_standard,omitempty"`
	DriverKind            string                       `json:"driver_kind,omitempty"`
	CompilerContextStatus string                       `json:"compiler_context_status,omitempty"`
	ArgumentProvenance    string                       `json:"argument_provenance,omitempty"`
	PredefinedMacros      map[string]parser.MacroValue `json:"predefined_macros,omitempty"`
	TargetProfile         string                       `json:"target_profile,omitempty"`
	PassKind              string                       `json:"pass_kind,omitempty"`
	ContextVersion        int                          `json:"context_version"`
}

func (context FixtureContext) parseContext(documentURI, languageID string, includeRoots []string, includeRootsAvailable bool, documentVersion int) parser.ParseContext {
	return parser.ParseContext{
		DocumentURI:           documentURI,
		LanguageID:            languageID,
		LanguageStandard:      context.LanguageStandard,
		DriverKind:            context.DriverKind,
		CompilerContextStatus: context.CompilerContextStatus,
		ArgumentProvenance:    context.ArgumentProvenance,
		PredefinedMacros:      cloneManifestMacros(context.PredefinedMacros),
		TargetProfile:         context.TargetProfile,
		PassKind:              context.PassKind,
		IncludeRoots:          append([]string(nil), includeRoots...),
		IncludeRootsAvailable: includeRootsAvailable,
		DocumentVersion:       documentVersion,
		ContextVersion:        context.ContextVersion,
	}
}

func cloneManifestMacros(macros map[string]parser.MacroValue) map[string]parser.MacroValue {
	if macros == nil {
		return map[string]parser.MacroValue{}
	}
	result := make(map[string]parser.MacroValue, len(macros))
	for name, value := range macros {
		result[name] = value
	}
	return result
}

type SymbolExpectation struct {
	Name             string `json:"name"`
	QualifiedName    string `json:"qualified_name,omitempty"`
	Kind             string `json:"kind,omitempty"`
	TypeKind         string `json:"type_kind,omitempty"`
	Status           string `json:"status,omitempty"`
	Condition        string `json:"condition,omitempty"`
	Origin           string `json:"origin,omitempty"`
	DeclarationCount *int   `json:"declaration_count,omitempty"`
	DefinitionCount  *int   `json:"definition_count,omitempty"`
}

type ReferenceExpectation struct {
	Name           string `json:"name"`
	URI            string `json:"uri,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Status         string `json:"status,omitempty"`
	CandidateCount *int   `json:"candidate_count,omitempty"`
}

type LocationExpectation struct {
	URI   string `json:"uri"`
	Start *int   `json:"start,omitempty"`
	End   *int   `json:"end,omitempty"`
	Role  string `json:"role,omitempty"`
}

type DiagnosticExpectation struct {
	Code                  string                            `json:"code"`
	SourceURI             string                            `json:"source_uri"`
	Range                 LocationExpectation               `json:"range"`
	ConditionalProvenance *ConditionalProvenanceExpectation `json:"conditional_provenance"`
}

type ConditionalProvenanceExpectation struct {
	SourceURI        string                             `json:"source_uri"`
	RegionID         *int                               `json:"region_id,omitempty"`
	BranchIndex      *int                               `json:"branch_index,omitempty"`
	ExpressionRange  *LocationExpectation               `json:"expression_range,omitempty"`
	State            string                             `json:"state"`
	ParentRegionID   *int                               `json:"parent_region_id,omitempty"`
	ParentConditions []ConditionalProvenanceExpectation `json:"parent_conditions,omitempty"`
}

type ManifestQuery struct {
	Operation          string                      `json:"operation"`
	Name               string                      `json:"name,omitempty"`
	SourceURI          string                      `json:"source_uri,omitempty"`
	Occurrence         int                         `json:"occurrence,omitempty"`
	IncludeDeclaration *bool                       `json:"include_declaration,omitempty"`
	ExpectedCount      *int                        `json:"expected_count,omitempty"`
	ExpectedURIs       []string                    `json:"expected_uris,omitempty"`
	ExpectedNull       bool                        `json:"expected_null,omitempty"`
	Malformed          bool                        `json:"malformed,omitempty"`
	ExpectedErrorCode  *int                        `json:"expected_error_code,omitempty"`
	ExpectedRanges     []LocationExpectation       `json:"expected_ranges,omitempty"`
	ExpectedSymbols    []DocumentSymbolExpectation `json:"expected_symbols,omitempty"`
}

type DocumentSymbolExpectation struct {
	Name           string                      `json:"name"`
	Kind           int                         `json:"kind"`
	Range          LocationExpectation         `json:"range"`
	SelectionRange LocationExpectation         `json:"selection_range"`
	Children       []DocumentSymbolExpectation `json:"children,omitempty"`
}

type ManifestQueryExecutor func(root string, fixture FixtureCase, snapshot indexpkg.IndexSnapshot, query ManifestQuery) ([]byte, error)

type ManifestQueryMetricsExecutor func(root string, fixture FixtureCase, snapshot indexpkg.IndexSnapshot, query ManifestQuery) ([]byte, int, error)

type Manifest struct {
	Cases []FixtureCase `json:"cases"`
}

type CategoryReport struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type ManifestReport struct {
	Total                    int                       `json:"total"`
	Passed                   int                       `json:"passed"`
	Failed                   int                       `json:"failed"`
	Skipped                  int                       `json:"skipped"`
	OldVersionOverwriteCount int                       `json:"old_version_overwrite_count"`
	ByCategory               map[string]CategoryReport `json:"by_category"`
	FailureIDs               []string                  `json:"failure_ids,omitempty"`
	FailureReasons           map[string]string         `json:"failure_reasons,omitempty"`
}

type ManifestValidationFailure struct {
	Err                      error
	OldVersionOverwriteCount int
}

func (failure *ManifestValidationFailure) Error() string {
	if failure == nil || failure.Err == nil {
		return "manifest validation failed"
	}
	return failure.Err.Error()
}

func (failure *ManifestValidationFailure) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.Err
}

var manifestCategories = map[string]struct{}{
	"core-symbol": {}, "multi-file": {}, "missing-context": {}, "unknown-condition": {},
	"incomplete": {}, "version-race": {}, "header-change": {}, "out-of-scope": {},
}

func LoadManifest(path string) (Manifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	return manifest, nil
}

func ValidateManifest(manifest Manifest) (ManifestReport, error) {
	return validateManifestWithMetricsExecutor(manifest, "", nil, nil)
}

func ValidateManifestAt(root string, manifest Manifest) (ManifestReport, error) {
	return validateManifestWithMetricsExecutor(manifest, root, nil, nil)
}

func ValidateManifestAtWithExecutor(root string, manifest Manifest, executor ManifestQueryExecutor) (ManifestReport, error) {
	return validateManifestWithMetricsExecutor(manifest, root, executor, nil)
}

func ValidateManifestAtWithMetricsExecutor(root string, manifest Manifest, executor ManifestQueryMetricsExecutor) (ManifestReport, error) {
	return validateManifestWithMetricsExecutor(manifest, root, nil, executor)
}

func validateManifestWithMetricsExecutor(manifest Manifest, root string, executor ManifestQueryExecutor, metricsExecutor ManifestQueryMetricsExecutor) (ManifestReport, error) {
	report := ManifestReport{ByCategory: make(map[string]CategoryReport), FailureReasons: make(map[string]string)}
	for _, fixture := range manifest.Cases {
		report.Total++
		category := report.ByCategory[fixture.Category]
		category.Total++
		valid := fixture.CaseID != "" && fixture.ExpectedStatus != "" && fixture.ExpectedSymbolCount >= 0 && fixture.ExpectedLocationCount >= 0
		if _, ok := manifestCategories[fixture.Category]; !ok {
			valid = false
		}
		mode := fixture.ValidationMode
		if mode == "" {
			mode = "execute"
		}
		if valid && mode == "schema-only" {
			if fixture.SkipReason == "" {
				valid = false
				report.FailureReasons[fixture.CaseID] = fmt.Sprintf("case %s: schema-only validation requires skip_reason", fixture.CaseID)
			} else {
				report.Skipped++
				category.Skipped++
				report.ByCategory[fixture.Category] = category
				continue
			}
		}
		if valid && mode != "execute" {
			valid = false
			report.FailureReasons[fixture.CaseID] = fmt.Sprintf("case %s: unknown validation_mode %q", fixture.CaseID, mode)
		}
		if valid {
			if err := validateManifestSchema(fixture); err != nil {
				valid = false
				report.FailureReasons[fixture.CaseID] = err.Error()
			}
		}
		if valid && root != "" {
			if err := validateFixtureCase(root, fixture, executor, metricsExecutor, &report.OldVersionOverwriteCount); err != nil {
				valid = false
				var validationFailure *ManifestValidationFailure
				if errors.As(err, &validationFailure) {
					report.OldVersionOverwriteCount += validationFailure.OldVersionOverwriteCount
				}
				failureID := fixture.CaseID
				if failureID == "" {
					failureID = "<missing-case-id>"
				}
				report.FailureReasons[failureID] = err.Error()
			}
		}
		if valid {
			report.Passed++
			category.Passed++
		} else {
			report.Failed++
			category.Failed++
			failureID := fixture.CaseID
			if failureID == "" {
				failureID = "<missing-case-id>"
			}
			report.FailureIDs = append(report.FailureIDs, failureID)
		}
		report.ByCategory[fixture.Category] = category
	}
	return report, nil
}

func validateManifestSchema(fixture FixtureCase) error {
	if fixture.Context.ContextVersion <= 0 {
		return fmt.Errorf("case %s: context.context_version must be positive", fixture.CaseID)
	}
	if len(fixture.SourceFiles) == 0 {
		return fmt.Errorf("case %s: source_files are required for execute validation", fixture.CaseID)
	}
	if len(fixture.SourceURIs) == 0 {
		return fmt.Errorf("case %s: source_uris are required for execute validation", fixture.CaseID)
	}
	if len(fixture.SourceFiles) != len(fixture.SourceURIs) {
		return fmt.Errorf("case %s: source_files/source_uris count mismatch", fixture.CaseID)
	}
	versions := fixture.DocumentVersions
	if len(versions) == 0 {
		versions = []int{1}
	}
	if len(fixture.VersionTexts) > 0 && len(fixture.VersionTexts) != len(versions) {
		return fmt.Errorf("case %s: version_texts count %d, want %d", fixture.CaseID, len(fixture.VersionTexts), len(versions))
	}
	for index, expected := range fixture.ExpectedDiagnostics {
		if err := validateDiagnosticExpectationSchema(expected); err != nil {
			return fmt.Errorf("case %s: expected_diagnostics[%d]: %w", fixture.CaseID, index, err)
		}
	}
	return nil
}

func validateDiagnosticExpectationSchema(expected DiagnosticExpectation) error {
	if expected.Code == "" || expected.SourceURI == "" {
		return fmt.Errorf("code and source_uri are required")
	}
	if expected.Range.URI == "" || expected.Range.Start == nil || expected.Range.End == nil {
		return fmt.Errorf("range requires URI, start and end")
	}
	if expected.ConditionalProvenance == nil {
		return fmt.Errorf("conditional_provenance is required")
	}
	if expected.ConditionalProvenance.SourceURI == "" || expected.ConditionalProvenance.State == "" {
		return fmt.Errorf("conditional_provenance requires source_uri and state")
	}
	return nil
}

func validateFixtureCase(root string, fixture FixtureCase, executor ManifestQueryExecutor, metricsExecutor ManifestQueryMetricsExecutor, overwriteCount *int) error {
	opened, primaryURI, primaryText, loadedURIs, err := loadFixtureSources(root, fixture.SourceFiles)
	if err != nil {
		return err
	}
	seenSourceURIs := make(map[string]struct{}, len(fixture.SourceURIs))
	for index, sourceURI := range fixture.SourceURIs {
		canonicalURI := manifestURI(root, sourceURI)
		if canonicalURI == "" {
			return fmt.Errorf("case %s: source_uris contains an empty URI", fixture.CaseID)
		}
		if _, ok := seenSourceURIs[canonicalURI]; ok {
			return fmt.Errorf("case %s: duplicate source URI %q", fixture.CaseID, canonicalURI)
		}
		seenSourceURIs[canonicalURI] = struct{}{}
		if canonicalURI != loadedURIs[index] {
			return fmt.Errorf("case %s: source URI %q does not match source file %q", fixture.CaseID, canonicalURI, fixture.SourceFiles[index])
		}
	}
	roots := make([]string, 0, len(fixture.IncludeRoots))
	for _, includeRoot := range fixture.IncludeRoots {
		if filepath.IsAbs(includeRoot) {
			roots = append(roots, includeRoot)
		} else {
			roots = append(roots, filepath.Join(root, includeRoot))
		}
	}
	provider := indexpkg.NewLocalSourceProvider(opened, roots)
	_, includeRootsAvailable := provider.IncludeRoots()
	versions := append([]int(nil), fixture.DocumentVersions...)
	if len(versions) == 0 {
		versions = []int{1}
	}
	if len(fixture.VersionTexts) > 0 && len(fixture.VersionTexts) != len(versions) {
		return fmt.Errorf("case %s: version_texts count %d, want %d", fixture.CaseID, len(fixture.VersionTexts), len(versions))
	}
	if fixture.Category == "version-race" && len(versions) < 2 {
		return fmt.Errorf("case %s: version-race requires at least two document versions", fixture.CaseID)
	}
	store := indexpkg.NewStore()
	var snapshot indexpkg.IndexSnapshot
	for index, version := range versions {
		versionText := primaryText
		if len(fixture.VersionTexts) > 0 {
			versionText = fixture.VersionTexts[index]
		}
		parseContext := fixture.Context.parseContext(primaryURI, "tops", roots, includeRootsAvailable, version)
		result := parser.Parse(versionText, parseContext)
		snapshot = indexpkg.BuildWithProvider(versionText, result, parseContext, provider)
		snapshot.Generation = uint64(version)
		snapshot.SourceVersion.Generation = uint64(version)
		if !store.Replace(snapshot) {
			continue
		}
	}
	if current, ok := store.Get(primaryURI); ok {
		snapshot = current
	} else {
		return fmt.Errorf("case %s: no current snapshot", fixture.CaseID)
	}
	if fixture.ExpectedFinalVersion != nil && snapshot.SourceVersion.DocumentVersion != *fixture.ExpectedFinalVersion {
		err := fmt.Errorf("case %s: final document version %d, want %d", fixture.CaseID, snapshot.SourceVersion.DocumentVersion, *fixture.ExpectedFinalVersion)
		if fixture.Category == "version-race" {
			return &ManifestValidationFailure{Err: err, OldVersionOverwriteCount: 1}
		}
		return err
	}
	if fixture.Category == "header-change" {
		if err := validateHeaderChange(root, fixture, opened, roots, primaryURI, primaryText, store, snapshot); err != nil {
			return err
		}
		if current, ok := store.Get(primaryURI); ok {
			snapshot = current
		}
	}
	if len(snapshot.Symbols) != fixture.ExpectedSymbolCount {
		names := make([]string, 0, len(snapshot.Symbols))
		for _, symbol := range snapshot.Symbols {
			names = append(names, symbol.QualifiedName)
		}
		sort.Strings(names)
		return fmt.Errorf("case %s: symbol count %d, want %d (%s)", fixture.CaseID, len(snapshot.Symbols), fixture.ExpectedSymbolCount, strings.Join(names, ", "))
	}
	if string(snapshot.IndexStatus) != fixture.ExpectedStatus {
		return fmt.Errorf("case %s: status %q, want %q", fixture.CaseID, snapshot.IndexStatus, fixture.ExpectedStatus)
	}
	actualLocations := 0
	for _, symbol := range snapshot.Symbols {
		actualLocations += len(symbol.Declarations) + len(symbol.Definitions)
	}
	if actualLocations != fixture.ExpectedLocationCount {
		return fmt.Errorf("case %s: location count %d, want %d", fixture.CaseID, actualLocations, fixture.ExpectedLocationCount)
	}
	if err := validateExpectedDiagnostics(root, snapshot, fixture.ExpectedDiagnostics); err != nil {
		return fmt.Errorf("case %s: %w", fixture.CaseID, err)
	}
	if (fixture.Category == "multi-file" || fixture.Category == "header-change") && len(snapshot.Dependencies) == 0 {
		return fmt.Errorf("case %s: expected header dependency", fixture.CaseID)
	}
	for _, dependency := range snapshot.Dependencies {
		if dependency.Status == indexpkg.HeaderDependencyResolved && dependency.SourceVersion.SourceVersion == "" {
			return fmt.Errorf("case %s: resolved dependency %q has no source version", fixture.CaseID, dependency.TargetURI)
		}
	}
	if err := validateSnapshotSources(snapshot, loadedURIs); err != nil {
		return fmt.Errorf("case %s: %w", fixture.CaseID, err)
	}
	if err := validateExpectedSymbols(snapshot, fixture.ExpectedSymbols); err != nil {
		return err
	}
	if err := validateExpectedReferences(snapshot, fixture.ExpectedReferences); err != nil {
		return err
	}
	if err := validateExpectedLocations(root, snapshot, fixture.ExpectedLocations); err != nil {
		return err
	}
	if err := validateExpectedDependencies(snapshot, fixture.ExpectedDependencies); err != nil {
		return err
	}
	if err := validateExpectedFile(root, fixture, snapshot); err != nil {
		return err
	}
	if err := validateManifestQueries(root, snapshot, fixture, executor, metricsExecutor, overwriteCount); err != nil {
		return err
	}
	return nil
}

func validateExpectedFile(root string, fixture FixtureCase, snapshot indexpkg.IndexSnapshot) error {
	if fixture.ExpectedFile == "" {
		return nil
	}
	path := fixture.ExpectedFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("case %s: read expected file: %w", fixture.CaseID, err)
	}
	var expected FixtureCase
	if err := json.Unmarshal(body, &expected); err != nil {
		return fmt.Errorf("case %s: decode expected file: %w", fixture.CaseID, err)
	}
	if expected.CaseID != fixture.CaseID || (expected.Category != "" && expected.Category != fixture.Category) {
		return fmt.Errorf("case %s: expected file identity mismatch", fixture.CaseID)
	}
	if string(snapshot.IndexStatus) != expected.ExpectedStatus {
		return fmt.Errorf("case %s: expected file status %q, got %q", fixture.CaseID, expected.ExpectedStatus, snapshot.IndexStatus)
	}
	if len(snapshot.Symbols) != expected.ExpectedSymbolCount {
		return fmt.Errorf("case %s: expected file symbol count %d, got %d", fixture.CaseID, expected.ExpectedSymbolCount, len(snapshot.Symbols))
	}
	actualLocations := 0
	for _, symbol := range snapshot.Symbols {
		actualLocations += len(symbol.Declarations) + len(symbol.Definitions)
	}
	if actualLocations != expected.ExpectedLocationCount {
		return fmt.Errorf("case %s: expected file location count %d, got %d", fixture.CaseID, expected.ExpectedLocationCount, actualLocations)
	}
	if err := validateExpectedSymbols(snapshot, expected.ExpectedSymbols); err != nil {
		return fmt.Errorf("case %s: expected file: %w", fixture.CaseID, err)
	}
	if err := validateExpectedReferences(snapshot, expected.ExpectedReferences); err != nil {
		return fmt.Errorf("case %s: expected file: %w", fixture.CaseID, err)
	}
	if err := validateExpectedDiagnostics(root, snapshot, expected.ExpectedDiagnostics); err != nil {
		return fmt.Errorf("case %s: expected file: %w", fixture.CaseID, err)
	}
	if err := validateExpectedLocations(root, snapshot, expected.ExpectedLocations); err != nil {
		return fmt.Errorf("case %s: expected file: %w", fixture.CaseID, err)
	}
	return nil
}

func loadFixtureSources(root string, paths []string) (map[string]indexpkg.SourceDocument, string, string, []string, error) {
	opened := make(map[string]indexpkg.SourceDocument, len(paths))
	loadedURIs := make([]string, 0, len(paths))
	primaryURI := ""
	primaryText := ""
	for index, sourcePath := range paths {
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(root, sourcePath)
		}
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, "", "", nil, err
		}
		absolutePath, err := filepath.Abs(sourcePath)
		if err != nil {
			return nil, "", "", nil, err
		}
		uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}).String()
		document := indexpkg.SourceDocument{URI: uri, Text: string(body), Version: 1, SourceVersion: indexpkg.SourceDigest(string(body))}
		opened[uri] = document
		loadedURIs = append(loadedURIs, uri)
		if index == 0 {
			primaryURI = uri
			primaryText = document.Text
		}
	}
	return opened, primaryURI, primaryText, loadedURIs, nil
}

func validateSnapshotSources(snapshot indexpkg.IndexSnapshot, loadedURIs []string) error {
	for _, uri := range loadedURIs {
		if _, ok := snapshot.Sources[uri]; !ok {
			return fmt.Errorf("source %q was not present in the snapshot", uri)
		}
	}
	for _, symbol := range snapshot.Symbols {
		locations := append(append([]indexpkg.SymbolLocation{}, symbol.Declarations...), symbol.Definitions...)
		for _, location := range locations {
			if err := validateSnapshotLocation(snapshot, location); err != nil {
				return err
			}
		}
	}
	for _, reference := range snapshot.References {
		text, ok := snapshot.Sources[reference.URI]
		if !ok || !validSourceRange(text, reference.Range) {
			return fmt.Errorf("reference %q has an invalid source range", reference.ReferenceID)
		}
	}
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI != "" {
			if _, ok := snapshot.Sources[dependency.TargetURI]; !ok {
				return fmt.Errorf("dependency target %q was not loaded", dependency.TargetURI)
			}
		}
		text, ok := snapshot.Sources[dependency.RequesterURI]
		if !ok || !validSourceRange(text, dependency.IncludeRange) {
			return fmt.Errorf("dependency %q has an invalid include range", dependency.IncludeSpelling)
		}
	}
	return nil
}

func validateSnapshotLocation(snapshot indexpkg.IndexSnapshot, location indexpkg.SymbolLocation) error {
	text, ok := snapshot.Sources[location.URI]
	if !ok || !validSourceRange(text, location.Range) || !validSourceRange(text, location.SelectionRange) {
		return fmt.Errorf("location %q has an invalid source range", location.URI)
	}
	return nil
}

func validSourceRange(text string, sourceRange parser.SourceRange) bool {
	return sourceRange.Start >= 0 && sourceRange.Start <= sourceRange.End && sourceRange.End <= len(text)
}

func validateHeaderChange(root string, fixture FixtureCase, opened map[string]indexpkg.SourceDocument, roots []string, primaryURI, primaryText string, store *indexpkg.Store, previous indexpkg.IndexSnapshot) error {
	var headerURI string
	var header indexpkg.SourceDocument
	var previousDependency indexpkg.HeaderDependency
	for _, dependency := range previous.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		if document, ok := opened[dependency.TargetURI]; ok && strings.HasSuffix(strings.ToLower(dependency.TargetURI), ".h") {
			headerURI = dependency.TargetURI
			header = document
			previousDependency = dependency
			break
		}
	}
	if headerURI == "" {
		return fmt.Errorf("case %s: header-change requires a header source", fixture.CaseID)
	}
	changedOpened := make(map[string]indexpkg.SourceDocument, len(opened))
	for uri, document := range opened {
		changedOpened[uri] = document
	}
	header.Text += "\nint header_version_marker;\n"
	header.Version++
	header.SourceVersion = indexpkg.SourceDigest(header.Text)
	changedOpened[headerURI] = header
	if len(previous.Dependencies) == 0 {
		return fmt.Errorf("case %s: header-change has no dependency", fixture.CaseID)
	}
	stale := store.MarkHeaderStale(headerURI)
	if len(stale) != 1 || stale[0] != primaryURI {
		return fmt.Errorf("case %s: header target %q did not stale requester %q", fixture.CaseID, headerURI, primaryURI)
	}
	provider := indexpkg.NewLocalSourceProvider(changedOpened, roots)
	_, includeRootsAvailable := provider.IncludeRoots()
	parseContext := fixture.Context.parseContext(primaryURI, "tops", roots, includeRootsAvailable, previous.SourceVersion.DocumentVersion)
	result := parser.Parse(primaryText, parseContext)
	changed := indexpkg.BuildWithProvider(primaryText, result, parseContext, provider)
	changed.Generation = previous.Generation + 1
	changed.SourceVersion.Generation = changed.Generation
	if !store.Replace(changed) {
		return fmt.Errorf("case %s: changed header snapshot was rejected", fixture.CaseID)
	}
	current, ok := store.Get(primaryURI)
	if !ok || current.SourceVersion.SourceVersion == previous.SourceVersion.SourceVersion {
		return fmt.Errorf("case %s: header change did not change root source version", fixture.CaseID)
	}
	changedDependency := indexpkg.HeaderDependency{}
	for _, dependency := range current.Dependencies {
		if dependency.TargetURI == headerURI {
			changedDependency = dependency
			break
		}
	}
	if changedDependency.TargetURI == "" || changedDependency.SourceVersion.SourceVersion == previousDependency.SourceVersion.SourceVersion {
		return fmt.Errorf("case %s: dependency target %q source version did not change", fixture.CaseID, headerURI)
	}
	var marker indexpkg.SymbolRecord
	foundMarker := false
	for _, symbol := range current.Symbols {
		if symbol.Name == "header_version_marker" && symbol.SourceVersion.URI == headerURI {
			marker = symbol
			foundMarker = true
			break
		}
	}
	if !foundMarker {
		return fmt.Errorf("case %s: changed dependency did not produce header_version_marker", fixture.CaseID)
	}
	if definitions := current.DefinitionAtURI(headerURI, marker.NameRange.Start); len(definitions) != 1 || definitions[0].URI != headerURI {
		return fmt.Errorf("case %s: changed header marker definition = %+v", fixture.CaseID, definitions)
	}
	if references := current.ReferencesAtURI(headerURI, marker.NameRange.Start, true); len(references) != 1 || references[0].URI != headerURI {
		return fmt.Errorf("case %s: changed header marker references = %+v", fixture.CaseID, references)
	}
	return nil
}

func validateExpectedSymbols(snapshot indexpkg.IndexSnapshot, expectations []SymbolExpectation) error {
	for _, expected := range expectations {
		var match *indexpkg.SymbolRecord
		for _, symbol := range snapshot.Symbols {
			if symbol.Name != expected.Name || (expected.QualifiedName != "" && symbol.QualifiedName != expected.QualifiedName) || (expected.Kind != "" && string(symbol.Kind) != expected.Kind) {
				continue
			}
			copy := symbol
			match = &copy
			break
		}
		if match == nil {
			return fmt.Errorf("symbol %q not found", expected.Name)
		}
		if expected.TypeKind != "" && match.TypeKind != expected.TypeKind {
			return fmt.Errorf("symbol %q type kind %q, want %q", expected.Name, match.TypeKind, expected.TypeKind)
		}
		if expected.Status != "" && string(match.Status) != expected.Status {
			return fmt.Errorf("symbol %q status %q, want %q", expected.Name, match.Status, expected.Status)
		}
		if expected.Condition != "" && string(match.Condition.State) != expected.Condition {
			return fmt.Errorf("symbol %q condition %q, want %q", expected.Name, match.Condition.State, expected.Condition)
		}
		if expected.Origin != "" && match.Origin != expected.Origin && match.OriginKind != expected.Origin {
			return fmt.Errorf("symbol %q origin %q/%q, want %q", expected.Name, match.Origin, match.OriginKind, expected.Origin)
		}
		if expected.DeclarationCount != nil && len(match.Declarations) != *expected.DeclarationCount {
			return fmt.Errorf("symbol %q declaration count %d, want %d", expected.Name, len(match.Declarations), *expected.DeclarationCount)
		}
		if expected.DefinitionCount != nil && len(match.Definitions) != *expected.DefinitionCount {
			return fmt.Errorf("symbol %q definition count %d, want %d", expected.Name, len(match.Definitions), *expected.DefinitionCount)
		}
	}
	return nil
}

func validateExpectedReferences(snapshot indexpkg.IndexSnapshot, expectations []ReferenceExpectation) error {
	for _, expected := range expectations {
		found := false
		for _, reference := range snapshot.References {
			if reference.Name != expected.Name || (expected.URI != "" && reference.URI != expected.URI) || (expected.Kind != "" && reference.Kind != expected.Kind) || (expected.Status != "" && string(reference.Status) != expected.Status) {
				continue
			}
			if expected.CandidateCount != nil && len(reference.Candidates) != *expected.CandidateCount {
				continue
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("reference %q not found", expected.Name)
		}
	}
	return nil
}

func validateExpectedLocations(root string, snapshot indexpkg.IndexSnapshot, expectations []LocationExpectation) error {
	for _, expected := range expectations {
		uri := manifestURI(root, expected.URI)
		found := false
		for _, symbol := range snapshot.Symbols {
			locations := append(append([]indexpkg.SymbolLocation{}, symbol.Declarations...), symbol.Definitions...)
			for _, location := range locations {
				if locationMatches(location, uri, expected) {
					found = true
				}
			}
		}
		for _, reference := range snapshot.References {
			location := indexpkg.SymbolLocation{URI: reference.URI, Range: reference.Range, Role: indexpkg.SymbolLocationReference}
			if locationMatches(location, uri, expected) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("location %q not found", expected.URI)
		}
	}
	return nil
}

func locationMatches(location indexpkg.SymbolLocation, uri string, expected LocationExpectation) bool {
	if location.URI != uri || (expected.Role != "" && string(location.Role) != expected.Role) {
		return false
	}
	return (expected.Start == nil || location.Range.Start == *expected.Start) && (expected.End == nil || location.Range.End == *expected.End)
}

func validateExpectedDependencies(snapshot indexpkg.IndexSnapshot, expectations []string) error {
	for _, expected := range expectations {
		found := false
		for _, dependency := range snapshot.Dependencies {
			if string(dependency.Status) == expected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("dependency status %q not found", expected)
		}
	}
	return nil
}

func validateExpectedDiagnostics(root string, snapshot indexpkg.IndexSnapshot, expectations []DiagnosticExpectation) error {
	used := make([]bool, len(snapshot.Diagnostics))
	for index, expected := range expectations {
		matched := false
		for actualIndex, actual := range snapshot.Diagnostics {
			if used[actualIndex] || actual.Code != expected.Code || actual.SourceURI != manifestURI(root, expected.SourceURI) {
				continue
			}
			if err := validateDiagnosticRange(root, snapshot, actual, expected.Range); err != nil {
				continue
			}
			if err := validateConditionalProvenance(root, snapshot, actual.Condition, *expected.ConditionalProvenance); err != nil {
				continue
			}
			used[actualIndex] = true
			matched = true
			break
		}
		if !matched {
			return fmt.Errorf("diagnostic %d was not found: code=%q source_uri=%q", index, expected.Code, expected.SourceURI)
		}
	}
	return nil
}

func validateDiagnosticRange(root string, snapshot indexpkg.IndexSnapshot, actual indexpkg.IndexDiagnostic, expected LocationExpectation) error {
	if actual.SourceURI != manifestURI(root, expected.URI) {
		return fmt.Errorf("diagnostic source URI %q, want %q", actual.SourceURI, expected.URI)
	}
	if expected.Start == nil || expected.End == nil {
		return fmt.Errorf("diagnostic range requires start and end")
	}
	text, ok := snapshot.Sources[actual.SourceURI]
	if !ok || !validSourceRange(text, actual.Range) {
		return fmt.Errorf("diagnostic range source %q is not loaded", actual.SourceURI)
	}
	if actual.Range.Start != *expected.Start || actual.Range.End != *expected.End {
		return fmt.Errorf("diagnostic range [%d,%d], want [%d,%d]", actual.Range.Start, actual.Range.End, *expected.Start, *expected.End)
	}
	return nil
}

func validateConditionalProvenance(root string, snapshot indexpkg.IndexSnapshot, actual indexpkg.ConditionalProvenance, expected ConditionalProvenanceExpectation) error {
	if actual.SourceURI != manifestURI(root, expected.SourceURI) || actual.State != parser.ConditionalState(expected.State) {
		return fmt.Errorf("conditional provenance source/state mismatch")
	}
	if expected.RegionID != nil && actual.RegionID != *expected.RegionID {
		return fmt.Errorf("conditional region id %d, want %d", actual.RegionID, *expected.RegionID)
	}
	if expected.BranchIndex != nil && actual.BranchIndex != *expected.BranchIndex {
		return fmt.Errorf("conditional branch index %d, want %d", actual.BranchIndex, *expected.BranchIndex)
	}
	if expected.ParentRegionID != nil && actual.ParentRegionID != *expected.ParentRegionID {
		return fmt.Errorf("conditional parent region id %d, want %d", actual.ParentRegionID, *expected.ParentRegionID)
	}
	if expected.ExpressionRange == nil {
		if actual.ExpressionRange.Start != actual.ExpressionRange.End {
			return fmt.Errorf("conditional expression range was omitted")
		}
	} else {
		if err := validateDiagnosticRange(root, snapshot, indexpkg.IndexDiagnostic{SourceURI: actual.SourceURI, Range: actual.ExpressionRange}, *expected.ExpressionRange); err != nil {
			return err
		}
	}
	if len(actual.ParentConditions) != len(expected.ParentConditions) {
		return fmt.Errorf("parent condition count %d, want %d", len(actual.ParentConditions), len(expected.ParentConditions))
	}
	for index := range actual.ParentConditions {
		if err := validateConditionalProvenance(root, snapshot, actual.ParentConditions[index], expected.ParentConditions[index]); err != nil {
			return fmt.Errorf("parent condition %d: %w", index, err)
		}
	}
	return nil
}

func validateManifestQueries(root string, snapshot indexpkg.IndexSnapshot, fixture FixtureCase, executor ManifestQueryExecutor, metricsExecutor ManifestQueryMetricsExecutor, overwriteCount *int) error {
	queries := append([]ManifestQuery(nil), fixture.Queries...)
	for _, operation := range fixture.LSPOperations {
		found := false
		for _, query := range queries {
			if query.Operation == operation {
				found = true
				break
			}
		}
		if !found {
			queries = append(queries, ManifestQuery{Operation: operation})
		}
	}
	for _, query := range queries {
		if err := validateManifestQueryOracle(query); err != nil {
			return err
		}
		if metricsExecutor != nil {
			body, count, err := metricsExecutor(root, fixture, snapshot, query)
			if overwriteCount != nil {
				*overwriteCount += count
			}
			if err != nil {
				return fmt.Errorf("query %s executor: %w", query.Operation, err)
			}
			if err := validateManifestQueryResponse(root, snapshot, query, body); err != nil {
				return err
			}
			continue
		}
		if executor != nil {
			body, err := executor(root, fixture, snapshot, query)
			if err != nil {
				return fmt.Errorf("query %s executor: %w", query.Operation, err)
			}
			if err := validateManifestQueryResponse(root, snapshot, query, body); err != nil {
				return err
			}
			continue
		}
		if err := validateManifestQueryRequest(snapshot, query); err != nil {
			return err
		}
		if query.ExpectedErrorCode != nil {
			continue
		}
		switch query.Operation {
		case "documentSymbol":
			if query.ExpectedNull {
				staleSnapshot := snapshot
				staleSnapshot.IndexStatus = indexpkg.IndexStatusStale
				result, err := encodeManifestDocumentSymbolsFor(staleSnapshot, staleSnapshot.DocumentSymbols())
				body, marshalErr := json.Marshal(result)
				if err != nil || marshalErr != nil || string(body) != "null" {
					return fmt.Errorf("documentSymbol null oracle is invalid")
				}
				continue
			}
			result, err := encodeManifestDocumentSymbolsFor(snapshot, snapshot.DocumentSymbols())
			if err != nil {
				return err
			}
			body, err := json.Marshal(result)
			if err != nil {
				return err
			}
			var decoded []protocol.DocumentSymbol
			if err := json.Unmarshal(body, &decoded); err != nil {
				return fmt.Errorf("documentSymbol result is not standard LSP JSON: %w", err)
			}
			if query.ExpectedCount != nil && len(decoded) != *query.ExpectedCount {
				return fmt.Errorf("query %s count %d, want %d", query.Operation, len(decoded), *query.ExpectedCount)
			}
			if query.ExpectedSymbols == nil {
				return fmt.Errorf("query %s requires expected_symbols oracle", query.Operation)
			}
			if err := validateDocumentSymbolRanges(snapshot, decoded); err != nil {
				return err
			}
			if err := validateExpectedDocumentSymbols(snapshot, decoded, query.ExpectedSymbols); err != nil {
				return err
			}
		case "definition", "references":
			if query.ExpectedNull {
				queryURI, queryOffset, err := manifestQueryOffset(snapshot, query)
				if err != nil {
					return err
				}
				staleSnapshot := snapshot
				staleSnapshot.IndexStatus = indexpkg.IndexStatusStale
				var locations []indexpkg.SymbolLocation
				if query.Operation == "definition" {
					locations = staleSnapshot.DefinitionAtURI(queryURI, queryOffset)
				} else {
					includeDeclaration := true
					if query.IncludeDeclaration != nil {
						includeDeclaration = *query.IncludeDeclaration
					}
					locations = staleSnapshot.ReferencesAtURI(queryURI, queryOffset, includeDeclaration)
				}
				result, err := encodeManifestLocations(staleSnapshot, locations)
				body, marshalErr := json.Marshal(result)
				if err != nil || marshalErr != nil || string(body) != "null" {
					return fmt.Errorf("%s null oracle is invalid", query.Operation)
				}
				continue
			}
			locations := make([]indexpkg.SymbolLocation, 0)
			if query.Name != "" {
				queryURI, queryOffset, err := manifestQueryOffset(snapshot, query)
				if err != nil {
					return err
				}
				if query.Operation == "definition" {
					locations = snapshot.DefinitionAtURI(queryURI, queryOffset)
				} else {
					includeDeclaration := true
					if query.IncludeDeclaration != nil {
						includeDeclaration = *query.IncludeDeclaration
					}
					locations = snapshot.ReferencesAtURI(queryURI, queryOffset, includeDeclaration)
				}
			}
			result, err := encodeManifestLocations(snapshot, locations)
			if err != nil {
				return err
			}
			body, err := json.Marshal(result)
			if err != nil {
				return err
			}
			var decoded []protocol.Location
			if err := json.Unmarshal(body, &decoded); err != nil {
				return fmt.Errorf("%s result is not standard LSP JSON: %w", query.Operation, err)
			}
			if query.ExpectedCount != nil && len(decoded) != *query.ExpectedCount {
				return fmt.Errorf("query %s count %d, want %d", query.Operation, len(decoded), *query.ExpectedCount)
			}
			if query.ExpectedRanges == nil {
				return fmt.Errorf("query %s requires expected_ranges oracle", query.Operation)
			}
			if err := validateExpectedQueryRanges(root, snapshot, decoded, query.ExpectedRanges); err != nil {
				return err
			}
			if len(query.ExpectedURIs) > 0 {
				actual := make([]string, 0, len(decoded))
				for _, location := range decoded {
					actual = append(actual, location.URI)
				}
				sort.Strings(actual)
				expected := append([]string(nil), query.ExpectedURIs...)
				for index := range expected {
					expected[index] = manifestURI(root, expected[index])
				}
				sort.Strings(expected)
				if strings.Join(actual, "\x00") != strings.Join(expected, "\x00") {
					return fmt.Errorf("query %s URIs %v, want %v", query.Operation, actual, expected)
				}
			}
		default:
			return fmt.Errorf("query %q is unsupported", query.Operation)
		}
	}
	return nil
}

func validateManifestQueryResponse(root string, snapshot indexpkg.IndexSnapshot, query ManifestQuery, body []byte) error {
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("query %s response is not JSON-RPC: %w", query.Operation, err)
	}
	if query.ExpectedErrorCode != nil {
		if envelope.Error == nil || envelope.Error.Code != *query.ExpectedErrorCode {
			return fmt.Errorf("query %s error code mismatch", query.Operation)
		}
		return nil
	}
	if envelope.Error != nil {
		return fmt.Errorf("query %s returned error code %d", query.Operation, envelope.Error.Code)
	}
	result := bytes.TrimSpace(envelope.Result)
	if query.ExpectedNull {
		if !bytes.Equal(result, []byte("null")) {
			return fmt.Errorf("query %s result %s, want null", query.Operation, result)
		}
		return nil
	}
	if bytes.Equal(result, []byte("null")) || len(result) == 0 {
		return fmt.Errorf("query %s returned null without expected_null", query.Operation)
	}
	switch query.Operation {
	case "documentSymbol":
		var symbols []protocol.DocumentSymbol
		if err := json.Unmarshal(result, &symbols); err != nil {
			return fmt.Errorf("documentSymbol result is not standard LSP JSON: %w", err)
		}
		if query.ExpectedCount != nil && len(symbols) != *query.ExpectedCount {
			return fmt.Errorf("query %s count %d, want %d", query.Operation, len(symbols), *query.ExpectedCount)
		}
		if query.ExpectedSymbols == nil {
			return fmt.Errorf("query %s requires expected_symbols oracle", query.Operation)
		}
		if err := validateDocumentSymbolRanges(snapshot, symbols); err != nil {
			return err
		}
		if err := validateExpectedDocumentSymbols(snapshot, symbols, query.ExpectedSymbols); err != nil {
			return err
		}
	case "definition", "references":
		var locations []protocol.Location
		if err := json.Unmarshal(result, &locations); err != nil {
			return fmt.Errorf("%s result is not standard LSP JSON: %w", query.Operation, err)
		}
		if query.ExpectedCount != nil && len(locations) != *query.ExpectedCount {
			return fmt.Errorf("query %s count %d, want %d", query.Operation, len(locations), *query.ExpectedCount)
		}
		if query.ExpectedRanges == nil {
			return fmt.Errorf("query %s requires expected_ranges oracle", query.Operation)
		}
		if len(query.ExpectedURIs) > 0 {
			actual := make([]string, 0, len(locations))
			for _, location := range locations {
				actual = append(actual, location.URI)
			}
			sort.Strings(actual)
			expected := append([]string(nil), query.ExpectedURIs...)
			for index := range expected {
				expected[index] = manifestURI(root, expected[index])
			}
			sort.Strings(expected)
			if strings.Join(actual, "\x00") != strings.Join(expected, "\x00") {
				return fmt.Errorf("query %s URIs %v, want %v", query.Operation, actual, expected)
			}
		}
		if err := validateProtocolLocations(snapshot, locations); err != nil {
			return fmt.Errorf("query %s: %w", query.Operation, err)
		}
		if err := validateExpectedQueryRanges(root, snapshot, locations, query.ExpectedRanges); err != nil {
			return err
		}
	default:
		return fmt.Errorf("query %q is unsupported", query.Operation)
	}
	return nil
}

func validateProtocolLocations(snapshot indexpkg.IndexSnapshot, locations []protocol.Location) error {
	for _, location := range locations {
		text, ok := snapshot.Sources[location.URI]
		if !ok {
			return fmt.Errorf("location source %q is not loaded", location.URI)
		}
		mapper := position.New(text)
		start, err := mapper.OffsetAt(position.Position{Line: location.Range.Start.Line, Character: location.Range.Start.Character})
		if err != nil {
			return fmt.Errorf("location %q has invalid start range (%+v): %w", location.URI, location.Range.Start, err)
		}
		end, err := mapper.OffsetAt(position.Position{Line: location.Range.End.Line, Character: location.Range.End.Character})
		if err != nil || end < start {
			return fmt.Errorf("location %q has invalid end range (%+v), start=%d source_length=%d: %v", location.URI, location.Range.End, start, len(text), err)
		}
	}
	return nil
}

func validateDocumentSymbolRanges(snapshot indexpkg.IndexSnapshot, symbols []protocol.DocumentSymbol) error {
	text, ok := snapshot.Sources[snapshot.RootURI]
	if !ok {
		return fmt.Errorf("document symbol source %q is not loaded", snapshot.RootURI)
	}
	mapper := position.New(text)
	for _, symbol := range symbols {
		start, err := mapper.OffsetAt(position.Position{Line: symbol.Range.Start.Line, Character: symbol.Range.Start.Character})
		if err != nil {
			return fmt.Errorf("document symbol %q has invalid range: %w", symbol.Name, err)
		}
		end, err := mapper.OffsetAt(position.Position{Line: symbol.Range.End.Line, Character: symbol.Range.End.Character})
		if err != nil || end < start {
			return fmt.Errorf("document symbol %q has invalid range", symbol.Name)
		}
		selectionStart, err := mapper.OffsetAt(position.Position{Line: symbol.SelectionRange.Start.Line, Character: symbol.SelectionRange.Start.Character})
		if err != nil {
			return fmt.Errorf("document symbol %q has invalid selectionRange: %w", symbol.Name, err)
		}
		selectionEnd, err := mapper.OffsetAt(position.Position{Line: symbol.SelectionRange.End.Line, Character: symbol.SelectionRange.End.Character})
		if err != nil || selectionStart < start || selectionEnd > end || selectionEnd < selectionStart {
			return fmt.Errorf("document symbol %q has selectionRange outside range", symbol.Name)
		}
		if err := validateDocumentSymbolRanges(snapshot, symbol.Children); err != nil {
			return err
		}
	}
	return nil
}

func validateExpectedQueryRanges(root string, snapshot indexpkg.IndexSnapshot, locations []protocol.Location, expectations []LocationExpectation) error {
	if len(expectations) == 0 {
		return nil
	}
	if len(expectations) != len(locations) {
		return fmt.Errorf("query ranges count %d, want %d", len(locations), len(expectations))
	}
	for index, expected := range expectations {
		location := locations[index]
		if location.URI != manifestURI(root, expected.URI) {
			return fmt.Errorf("query range %d URI %q, want %q", index, location.URI, expected.URI)
		}
		text, ok := snapshot.Sources[location.URI]
		if !ok {
			return fmt.Errorf("query range source %q is not loaded", location.URI)
		}
		mapper := position.New(text)
		start, err := mapper.OffsetAt(position.Position{Line: location.Range.Start.Line, Character: location.Range.Start.Character})
		if err != nil || (expected.Start != nil && start != *expected.Start) {
			return fmt.Errorf("query range %d start mismatch", index)
		}
		end, err := mapper.OffsetAt(position.Position{Line: location.Range.End.Line, Character: location.Range.End.Character})
		if err != nil || (expected.End != nil && end != *expected.End) {
			return fmt.Errorf("query range %d end mismatch", index)
		}
	}
	return nil
}

func validateExpectedDocumentSymbols(snapshot indexpkg.IndexSnapshot, actual []protocol.DocumentSymbol, expected []DocumentSymbolExpectation) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("document symbol oracle count %d, want %d", len(actual), len(expected))
	}
	for index, want := range expected {
		got := actual[index]
		if got.Name != want.Name {
			return fmt.Errorf("document symbol %d name %q, want %q", index, got.Name, want.Name)
		}
		if want.Kind <= 0 || int(got.Kind) != want.Kind {
			return fmt.Errorf("document symbol %d kind %d, want %d", index, got.Kind, want.Kind)
		}
		if err := validateDocumentSymbolLocation(snapshot, got.Range, want.Range); err != nil {
			return err
		}
		if err := validateDocumentSymbolLocation(snapshot, got.SelectionRange, want.SelectionRange); err != nil {
			return err
		}
		if err := validateExpectedDocumentSymbols(snapshot, got.Children, want.Children); err != nil {
			return err
		}
	}
	return nil
}

func validateDocumentSymbolLocation(snapshot indexpkg.IndexSnapshot, actual protocol.Range, expected LocationExpectation) error {
	if expected.URI != "" && manifestSourceURI(snapshot, expected.URI) != snapshot.RootURI {
		return fmt.Errorf("document symbol URI %q, want %q", snapshot.RootURI, expected.URI)
	}
	if expected.Start == nil || expected.End == nil {
		return fmt.Errorf("document symbol range oracle requires start and end")
	}
	text, ok := snapshot.Sources[snapshot.RootURI]
	if !ok {
		return fmt.Errorf("document symbol source %q is not loaded", snapshot.RootURI)
	}
	mapper := position.New(text)
	start, err := mapper.OffsetAt(position.Position{Line: actual.Start.Line, Character: actual.Start.Character})
	if err != nil || start != *expected.Start {
		return fmt.Errorf("document symbol range start %d, want %d", start, *expected.Start)
	}
	end, err := mapper.OffsetAt(position.Position{Line: actual.End.Line, Character: actual.End.Character})
	if err != nil || end != *expected.End {
		return fmt.Errorf("document symbol range end %d, want %d", end, *expected.End)
	}
	return nil
}

func validateManifestQueryRequest(snapshot indexpkg.IndexSnapshot, query ManifestQuery) error {
	if query.Operation != "documentSymbol" && query.Operation != "definition" && query.Operation != "references" {
		return fmt.Errorf("query %q is unsupported", query.Operation)
	}
	if query.SourceURI == "" {
		return fmt.Errorf("query %s requires source_uri", query.Operation)
	}
	if query.ExpectedErrorCode != nil {
		if !query.Malformed || *query.ExpectedErrorCode != int(protocol.InvalidParams) {
			return fmt.Errorf("query %s has unsupported error oracle", query.Operation)
		}
		var err error
		switch query.Operation {
		case "documentSymbol":
			var params protocol.DocumentSymbolParams
			err = json.Unmarshal([]byte(`{"textDocument":{}}`), &params)
		case "definition":
			var params protocol.TextDocumentPositionParams
			err = json.Unmarshal([]byte(`{"textDocument":{},"position":{}}`), &params)
		case "references":
			var params protocol.ReferenceParams
			err = json.Unmarshal([]byte(`{"textDocument":{},"position":{},"context":{}}`), &params)
		}
		if err == nil {
			return fmt.Errorf("query %s malformed params were accepted", query.Operation)
		}
		return nil
	}
	if query.Operation == "documentSymbol" {
		params := protocol.DocumentSymbolParams{TextDocument: protocol.TextDocumentIdentifier{URI: snapshot.RootURI}}
		body, err := json.Marshal(params)
		if err != nil {
			return err
		}
		var decoded protocol.DocumentSymbolParams
		if err := json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("documentSymbol params are not standard LSP JSON: %w", err)
		}
		return nil
	}
	if query.Name == "" {
		return fmt.Errorf("query %s requires a symbol name", query.Operation)
	}
	queryURI, positionValue, err := ManifestQueryLocation(snapshot, query)
	if err != nil {
		return err
	}
	if query.Operation == "definition" {
		params := protocol.TextDocumentPositionParams{TextDocument: protocol.TextDocumentIdentifier{URI: queryURI}, Position: protocol.Position{Line: positionValue.Line, Character: positionValue.Character}}
		body, err := json.Marshal(params)
		if err != nil {
			return err
		}
		var decoded protocol.TextDocumentPositionParams
		if err := json.Unmarshal(body, &decoded); err != nil {
			return fmt.Errorf("definition params are not standard LSP JSON: %w", err)
		}
		return nil
	}
	includeDeclaration := true
	if query.IncludeDeclaration != nil {
		includeDeclaration = *query.IncludeDeclaration
	}
	params := protocol.ReferenceParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: queryURI},
		Position:     protocol.Position{Line: positionValue.Line, Character: positionValue.Character},
		Context:      protocol.ReferenceContext{IncludeDeclaration: includeDeclaration},
	}
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var decoded protocol.ReferenceParams
	if err := json.Unmarshal(body, &decoded); err != nil {
		return fmt.Errorf("references params are not standard LSP JSON: %w", err)
	}
	return nil
}

func validateManifestQueryOracle(query ManifestQuery) error {
	if query.Operation != "documentSymbol" && query.Operation != "definition" && query.Operation != "references" {
		return fmt.Errorf("query %q is unsupported", query.Operation)
	}
	if query.SourceURI == "" {
		return fmt.Errorf("query %s requires source_uri", query.Operation)
	}
	if query.ExpectedErrorCode != nil {
		return nil
	}
	if query.ExpectedCount == nil {
		return fmt.Errorf("query %s requires expected_count", query.Operation)
	}
	if *query.ExpectedCount < 0 {
		return fmt.Errorf("query %s expected_count must not be negative", query.Operation)
	}
	if query.ExpectedNull && *query.ExpectedCount != 0 {
		return fmt.Errorf("query %s expected_null requires expected_count 0", query.Operation)
	}
	if query.Operation == "documentSymbol" {
		if query.ExpectedSymbols == nil {
			return fmt.Errorf("query %s requires expected_symbols oracle", query.Operation)
		}
		if len(query.ExpectedSymbols) != *query.ExpectedCount {
			return fmt.Errorf("query %s symbol oracle count %d, want %d", query.Operation, len(query.ExpectedSymbols), *query.ExpectedCount)
		}
		return nil
	}
	if query.ExpectedRanges == nil {
		return fmt.Errorf("query %s requires expected_ranges oracle", query.Operation)
	}
	if len(query.ExpectedRanges) != *query.ExpectedCount {
		return fmt.Errorf("query %s range oracle count %d, want %d", query.Operation, len(query.ExpectedRanges), *query.ExpectedCount)
	}
	for index, expected := range query.ExpectedRanges {
		if expected.URI == "" || expected.Start == nil || expected.End == nil {
			return fmt.Errorf("query %s range oracle %d requires URI, start and end", query.Operation, index)
		}
	}
	if query.ExpectedURIs != nil {
		if len(query.ExpectedURIs) != *query.ExpectedCount {
			return fmt.Errorf("query %s URI oracle count %d, want %d", query.Operation, len(query.ExpectedURIs), *query.ExpectedCount)
		}
		for index, uri := range query.ExpectedURIs {
			if uri == "" {
				return fmt.Errorf("query %s URI oracle %d is empty", query.Operation, index)
			}
		}
	}
	return nil
}

func encodeManifestLocations(snapshot indexpkg.IndexSnapshot, locations []indexpkg.SymbolLocation) ([]protocol.Location, error) {
	if locations == nil {
		return nil, nil
	}
	result := make([]protocol.Location, 0, len(locations))
	for _, location := range locations {
		text, ok := snapshot.Sources[location.URI]
		if !ok {
			return nil, fmt.Errorf("location source %q is not loaded", location.URI)
		}
		mapper := position.New(text)
		rangeValue, err := mapper.Range(location.Range.Start, location.Range.End)
		if err != nil {
			return nil, err
		}
		result = append(result, protocol.Location{
			URI: location.URI,
			Range: protocol.Range{
				Start: protocol.Position{Line: rangeValue.Start.Line, Character: rangeValue.Start.Character},
				End:   protocol.Position{Line: rangeValue.End.Line, Character: rangeValue.End.Character},
			},
		})
	}
	return result, nil
}

func encodeManifestDocumentSymbolsFor(snapshot indexpkg.IndexSnapshot, symbols []indexpkg.DocumentSymbol) ([]protocol.DocumentSymbol, error) {
	if symbols == nil {
		return nil, nil
	}
	result := make([]protocol.DocumentSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		text, ok := snapshot.Sources[snapshot.RootURI]
		if !ok {
			return nil, fmt.Errorf("document symbol source %q is not loaded", snapshot.RootURI)
		}
		mapper := position.New(text)
		rangeValue, err := mapper.Range(symbol.Range.Start, symbol.Range.End)
		if err != nil {
			return nil, err
		}
		selectionRange, err := mapper.Range(symbol.SelectionRange.Start, symbol.SelectionRange.End)
		if err != nil {
			return nil, err
		}
		children, err := encodeManifestDocumentSymbolsFor(snapshot, symbol.Children)
		if err != nil {
			return nil, err
		}
		result = append(result, protocol.DocumentSymbol{
			Name:           symbol.Name,
			Detail:         symbol.Detail,
			Kind:           manifestSymbolKind(symbol.Kind, symbol.TypeKind),
			Range:          protocol.Range{Start: protocol.Position{Line: rangeValue.Start.Line, Character: rangeValue.Start.Character}, End: protocol.Position{Line: rangeValue.End.Line, Character: rangeValue.End.Character}},
			SelectionRange: protocol.Range{Start: protocol.Position{Line: selectionRange.Start.Line, Character: selectionRange.Start.Character}, End: protocol.Position{Line: selectionRange.End.Line, Character: selectionRange.End.Character}},
			Children:       children,
		})
	}
	return result, nil
}

func manifestSymbolKind(kind indexpkg.SymbolKind, typeKind string) protocol.SymbolKind {
	switch kind {
	case indexpkg.SymbolKindNamespace:
		return protocol.SymbolKindNamespace
	case indexpkg.SymbolKindFunction:
		return protocol.SymbolKindFunction
	case indexpkg.SymbolKindType:
		switch typeKind {
		case "enum":
			return protocol.SymbolKindEnum
		case "struct", "union":
			return protocol.SymbolKindStruct
		}
		return protocol.SymbolKindClass
	case indexpkg.SymbolKindObject:
		return protocol.SymbolKindObject
	case indexpkg.SymbolKindField:
		return protocol.SymbolKindField
	case indexpkg.SymbolKindAttribute:
		return protocol.SymbolKindProperty
	case indexpkg.SymbolKindBuiltin, indexpkg.SymbolKindVariable:
		return protocol.SymbolKindVariable
	case indexpkg.SymbolKindMacro:
		return protocol.SymbolKindConstant
	default:
		return protocol.SymbolKindObject
	}
}

func ManifestQueryPosition(snapshot indexpkg.IndexSnapshot, query ManifestQuery) (protocol.Position, error) {
	_, value, err := ManifestQueryLocation(snapshot, query)
	return value, err
}

func ManifestQueryLocation(snapshot indexpkg.IndexSnapshot, query ManifestQuery) (string, protocol.Position, error) {
	uri, offset, err := manifestQueryOffset(snapshot, query)
	if err != nil {
		return "", protocol.Position{}, err
	}
	text, ok := snapshot.Sources[uri]
	if !ok {
		return "", protocol.Position{}, fmt.Errorf("query source %q is not loaded", uri)
	}
	value, err := position.New(text).PositionAt(offset)
	if err != nil {
		return "", protocol.Position{}, err
	}
	return uri, protocol.Position{Line: value.Line, Character: value.Character}, nil
}

type manifestQueryOccurrence struct {
	URI    string
	Offset int
}

func manifestQueryOffset(snapshot indexpkg.IndexSnapshot, query ManifestQuery) (string, int, error) {
	queryURI := snapshot.RootURI
	if query.SourceURI != "" {
		queryURI = manifestSourceURI(snapshot, query.SourceURI)
		if queryURI == "" {
			return "", 0, fmt.Errorf("query source %q is not loaded", query.SourceURI)
		}
	}
	occurrences := manifestQueryOccurrences(snapshot, query.Name, queryURI)
	if len(occurrences) == 0 && query.SourceURI == "" {
		occurrences = manifestQueryOccurrences(snapshot, query.Name, "")
	}
	if query.Occurrence < 0 || query.Occurrence >= len(occurrences) {
		return "", 0, fmt.Errorf("query %s symbol %q not found", query.Operation, query.Name)
	}
	occurrence := occurrences[query.Occurrence]
	return occurrence.URI, occurrence.Offset, nil
}

func manifestQueryOccurrences(snapshot indexpkg.IndexSnapshot, name, uri string) []manifestQueryOccurrence {
	occurrences := make([]manifestQueryOccurrence, 0)
	for _, reference := range snapshot.References {
		if reference.Name == name && (uri == "" || reference.URI == uri) {
			occurrences = append(occurrences, manifestQueryOccurrence{URI: reference.URI, Offset: reference.Range.Start})
		}
	}
	for _, symbol := range snapshot.Symbols {
		if name != "" && symbol.Name != name {
			continue
		}
		appendManifestLocation := func(location indexpkg.SymbolLocation) {
			if uri == "" || location.URI == uri {
				selection := location.SelectionRange
				if selection.Start == selection.End {
					selection = location.Range
				}
				occurrences = append(occurrences, manifestQueryOccurrence{URI: location.URI, Offset: selection.Start})
			}
		}
		if symbol.Name == name && (uri == "" || symbol.SourceVersion.URI == uri) {
			selection := symbol.NameRange
			if selection.Start != selection.End {
				occurrences = append(occurrences, manifestQueryOccurrence{URI: symbol.SourceVersion.URI, Offset: selection.Start})
			}
		}
		for _, location := range symbol.Declarations {
			appendManifestLocation(location)
		}
		for _, location := range symbol.Definitions {
			appendManifestLocation(location)
		}
		for _, location := range symbol.AttributeUsages {
			appendManifestLocation(location)
		}
	}
	sort.SliceStable(occurrences, func(left, right int) bool {
		if occurrences[left].URI != occurrences[right].URI {
			return occurrences[left].URI < occurrences[right].URI
		}
		return occurrences[left].Offset < occurrences[right].Offset
	})
	result := occurrences[:0]
	seen := make(map[string]struct{}, len(occurrences))
	for _, occurrence := range occurrences {
		key := occurrence.URI + ":" + strconv.Itoa(occurrence.Offset)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, occurrence)
	}
	return result
}

func manifestSourceURI(snapshot indexpkg.IndexSnapshot, value string) string {
	if _, ok := snapshot.Sources[value]; ok {
		return value
	}
	value = strings.TrimPrefix(value, "file://")
	candidates := make([]string, 0, 1)
	for uri := range snapshot.Sources {
		if strings.HasSuffix(strings.TrimPrefix(uri, "file://"), "/"+strings.TrimPrefix(value, "/")) {
			candidates = append(candidates, uri)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 1 {
		return candidates[0]
	}
	return ""
}

func manifestURI(root, value string) string {
	if value == "" || strings.HasPrefix(value, "file://") {
		return value
	}
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return value
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(absPath)}).String()
}
