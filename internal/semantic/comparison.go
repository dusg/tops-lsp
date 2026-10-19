package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ComparisonManifest struct {
	Version int              `json:"version"`
	Cases   []ComparisonCase `json:"cases"`
}

type ComparisonSourceFile struct {
	URI         string `json:"uri"`
	ContentHash string `json:"content_hash"`
}

type ComparisonDiagnostic struct {
	Code     string             `json:"code"`
	Severity string             `json:"severity"`
	Range    FixtureSourceRange `json:"range"`
	PassKind string             `json:"passKind"`
}

type ComparisonPassResult struct {
	PassKind        string                 `json:"pass_kind"`
	GoStatus        string                 `json:"go_status"`
	GoDiagnostics   []ComparisonDiagnostic `json:"go_diagnostics"`
	ClangStatus     string                 `json:"clang_status"`
	ClangCategories []string               `json:"clang_categories"`
}

type ComparisonCase struct {
	FixtureCaseID            string                 `json:"fixture_case_id"`
	DocumentURI              string                 `json:"document_uri"`
	RootSourceURI            string                 `json:"root_source_uri"`
	SourceFiles              []ComparisonSourceFile `json:"source_files"`
	SourceHash               string                 `json:"source_hash"`
	WorkingDirectory         string                 `json:"working_directory"`
	RawArguments             []string               `json:"raw_arguments"`
	NormalizedArguments      []string               `json:"normalized_arguments"`
	ProfileID                string                 `json:"profile_id"`
	PassKind                 string                 `json:"pass_kind"`
	IncludeProvenance        map[string]any         `json:"include_provenance"`
	MacroProvenance          map[string]any         `json:"macro_provenance"`
	HeaderVersions           []SourceVersionEntry   `json:"header_versions"`
	CompilerToolVersion      string                 `json:"compiler_tool_version"`
	SourceVersionFingerprint string                 `json:"source_version_fingerprint"`
	CompareStatus            string                 `json:"compare_status"`
	MissingFields            []string               `json:"missing_fields"`
	KnownDifference          []string               `json:"known_difference"`
	NeutralDiagnostics       []ComparisonDiagnostic `json:"neutral_diagnostics"`
	PassResults              []ComparisonPassResult `json:"pass_results"`
}

type ComparisonRecord struct {
	FixtureCaseID            string                 `json:"fixture_case_id"`
	DocumentURI              string                 `json:"document_uri"`
	RootSourceURI            string                 `json:"root_source_uri"`
	SourceHash               string                 `json:"source_hash"`
	SourceVersionFingerprint string                 `json:"source_version_fingerprint"`
	WorkingDirectory         string                 `json:"working_directory"`
	RawArguments             []string               `json:"raw_arguments"`
	NormalizedArguments      []string               `json:"normalized_arguments"`
	ProfileID                string                 `json:"profile_id"`
	PassKind                 string                 `json:"pass_kind"`
	IncludeProvenance        map[string]any         `json:"include_provenance"`
	MacroProvenance          map[string]any         `json:"macro_provenance"`
	HeaderVersions           []SourceVersionEntry   `json:"header_versions"`
	CompilerToolVersion      string                 `json:"compiler_tool_version"`
	CompareStatus            string                 `json:"compare_status"`
	MissingFields            []string               `json:"missing_fields"`
	KnownDifference          []string               `json:"known_difference"`
	NeutralDiagnostics       []ComparisonDiagnostic `json:"neutral_diagnostics"`
	PassResults              []ComparisonPassResult `json:"pass_results"`
}

func LoadComparisonManifest(path string) (ComparisonManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ComparisonManifest{}, err
	}
	var manifest ComparisonManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return ComparisonManifest{}, fmt.Errorf("decode semantic comparison manifest: %w", err)
	}
	return manifest, nil
}

func ValidateComparisonManifest(manifest ComparisonManifest, fixtureCaseIDs map[string]struct{}) []string {
	reasons := make([]string, 0)
	add := func(caseID, field string) {
		reasons = append(reasons, fmt.Sprintf("case %s: %s", caseID, field))
	}
	if manifest.Version != 1 {
		add("<manifest>", "version must be 1")
	}
	if len(manifest.Cases) == 0 {
		add("<manifest>", "cases must not be empty")
	}
	for _, comparisonCase := range manifest.Cases {
		caseID := comparisonCase.FixtureCaseID
		if caseID == "" {
			add(caseID, "fixture_case_id is required")
		} else if _, ok := fixtureCaseIDs[caseID]; !ok {
			add(caseID, "fixture_case_id is not registered")
		}
		if comparisonCase.DocumentURI == "" {
			add(caseID, "document_uri is required")
		}
		if comparisonCase.RootSourceURI == "" {
			add(caseID, "root_source_uri is required")
		}
		if len(comparisonCase.SourceFiles) == 0 {
			add(caseID, "source_files is required")
		}
		documentRoot := normalizeComparisonURI(comparisonCase.DocumentURI)
		rootSource := normalizeComparisonURI(comparisonCase.RootSourceURI)
		if documentRoot == "" || rootSource == "" || documentRoot != rootSource {
			add(caseID, "root_source_uri must match normalized document_uri")
		}
		rootFound := false
		for _, source := range comparisonCase.SourceFiles {
			if source.URI == "" || source.ContentHash == "" {
				add(caseID, "source_files uri/content_hash is required")
			}
			if normalizeComparisonURI(source.URI) == documentRoot {
				rootFound = true
			}
		}
		if !rootFound {
			add(caseID, "root_source_uri must be present in source_files")
		}
		for _, header := range comparisonCase.HeaderVersions {
			if header.URI == "" || (header.Version == "" && header.Status == "") || (header.Version != "" && header.Status != "") {
				add(caseID, "header_versions entries must contain exactly one version or status")
			}
		}
		if comparisonCase.SourceHash == "" {
			add(caseID, "source_hash is required")
		}
		if comparisonCase.WorkingDirectory == "" {
			add(caseID, "working_directory is required")
		}
		if len(comparisonCase.RawArguments) == 0 || len(comparisonCase.NormalizedArguments) == 0 {
			add(caseID, "raw_arguments and normalized_arguments are required")
		}
		if comparisonCase.ProfileID == "" {
			add(caseID, "profile_id is required")
		}
		if comparisonCase.PassKind != string(PassKindHost) && comparisonCase.PassKind != string(PassKindDevice) && comparisonCase.PassKind != string(PassKindHostDevice) {
			add(caseID, "invalid pass_kind")
		}
		if comparisonCase.IncludeProvenance == nil || comparisonCase.MacroProvenance == nil {
			add(caseID, "include_provenance and macro_provenance are required")
		}
		if comparisonCase.CompilerToolVersion == "" {
			add(caseID, "compiler_tool_version is required")
		}
		if comparisonCase.SourceVersionFingerprint == "" {
			add(caseID, "source_version_fingerprint is required")
		}
		validateComparisonPasses(comparisonCase.PassKind, comparisonCase.PassResults, func(field string) { add(caseID, field) })
		for _, diagnostic := range comparisonCase.NeutralDiagnostics {
			if diagnostic.PassKind != "not-applicable" {
				add(caseID, "neutral_diagnostics.passKind")
			}
			validateComparisonDiagnostic(diagnostic, "not-applicable", func(field string) { add(caseID, "neutral_diagnostics."+field) })
		}
	}
	return reasons
}

func GenerateComparisonRecords(comparisonManifestPath, fixtureManifestPath string) ([]ComparisonRecord, error) {
	manifest, err := LoadComparisonManifest(comparisonManifestPath)
	if err != nil {
		return nil, err
	}
	fixtureManifest, err := LoadFixtureManifest(fixtureManifestPath)
	if err != nil {
		return nil, err
	}
	fixtureIDs := make(map[string]struct{}, len(fixtureManifest.Cases))
	for _, fixture := range fixtureManifest.Cases {
		fixtureIDs[fixture.CaseID] = struct{}{}
	}
	if reasons := ValidateComparisonManifest(manifest, fixtureIDs); len(reasons) > 0 {
		return nil, fmt.Errorf("validate semantic comparison manifest: %s", strings.Join(reasons, "; "))
	}
	repositoryRoot, err := comparisonRepositoryRoot(comparisonManifestPath, manifest)
	if err != nil {
		return nil, err
	}
	records := make([]ComparisonRecord, 0, len(manifest.Cases))
	for _, comparisonCase := range manifest.Cases {
		records = append(records, buildComparisonRecordAtRoot(comparisonCase, fixtureIDs, repositoryRoot))
	}
	return records, nil
}

func buildComparisonRecordAtRoot(input ComparisonCase, fixtureCaseIDs map[string]struct{}, repositoryRoot string) ComparisonRecord {
	prepared := input
	preparedDocumentURI := comparisonURIAtRoot(repositoryRoot, input.DocumentURI)
	preparedRootURI := comparisonURIAtRoot(repositoryRoot, input.RootSourceURI)
	prepared.DocumentURI = preparedDocumentURI
	prepared.RootSourceURI = preparedRootURI
	prepared.SourceFiles = make([]ComparisonSourceFile, 0, len(input.SourceFiles))
	missingSourceFields := make([]string, 0)
	sourceTexts := make(map[string]string, len(input.SourceFiles))
	for _, source := range input.SourceFiles {
		preparedSource := source
		preparedSource.URI = comparisonURIAtRoot(repositoryRoot, source.URI)
		path, err := fixtureSourcePath(repositoryRoot, source.URI)
		if err != nil {
			missingSourceFields = append(missingSourceFields, "source_files:"+source.URI)
			prepared.SourceFiles = append(prepared.SourceFiles, preparedSource)
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			missingSourceFields = append(missingSourceFields, "source_files:"+source.URI)
			prepared.SourceFiles = append(prepared.SourceFiles, preparedSource)
			continue
		}
		actualHash := comparisonContentHash(body)
		if source.ContentHash == "" {
			missingSourceFields = append(missingSourceFields, "source_files.content_hash")
		} else if source.ContentHash != actualHash {
			missingSourceFields = append(missingSourceFields, "source_files.content_hash_match:"+source.URI)
		}
		preparedSource.ContentHash = actualHash
		prepared.SourceFiles = append(prepared.SourceFiles, preparedSource)
		sourceTexts[preparedSource.URI] = string(body)
	}
	if passResults, neutralDiagnostics, err := buildComparisonGoResults(prepared, sourceTexts, repositoryRoot); err != nil {
		missingSourceFields = append(missingSourceFields, "go_semantic_results:"+err.Error())
	} else {
		prepared.PassResults = passResults
		prepared.NeutralDiagnostics = neutralDiagnostics
	}
	record := BuildComparisonRecord(prepared, fixtureCaseIDs)
	record.DocumentURI = input.DocumentURI
	record.RootSourceURI = preparedRootURI
	for _, field := range missingSourceFields {
		markComparisonNotComparable(&record, field)
	}
	return record
}

func comparisonRepositoryRoot(manifestPath string, manifest ComparisonManifest) (string, error) {
	if len(manifest.Cases) == 0 || len(manifest.Cases[0].SourceFiles) == 0 {
		return "", fmt.Errorf("comparison manifest has no source files")
	}
	abs, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", err
	}
	current := filepath.Dir(abs)
	for {
		path, pathErr := fixtureSourcePath(current, manifest.Cases[0].SourceFiles[0].URI)
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
	return "", fmt.Errorf("cannot find repository root for comparison manifest %q", manifestPath)
}

func comparisonURIAtRoot(repositoryRoot, value string) string {
	if value == "" {
		return ""
	}
	if uri, err := fixtureSourceURI(repositoryRoot, value); err == nil {
		return uri
	}
	return normalizeComparisonURI(value)
}

func comparisonContentHash(body []byte) string {
	digest := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func markComparisonNotComparable(record *ComparisonRecord, field string) {
	if record == nil {
		return
	}
	for _, existing := range record.MissingFields {
		if existing == field {
			record.CompareStatus = "not-comparable"
			return
		}
	}
	record.MissingFields = append(record.MissingFields, field)
	record.CompareStatus = "not-comparable"
}

func WriteComparisonRecords(path string, records []ComparisonRecord) error {
	body, err := json.MarshalIndent(struct {
		Version int                `json:"version"`
		Records []ComparisonRecord `json:"records"`
	}{Version: 1, Records: records}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func BuildComparisonRecord(input ComparisonCase, fixtureCaseIDs map[string]struct{}) ComparisonRecord {
	record := ComparisonRecord{
		FixtureCaseID:       input.FixtureCaseID,
		DocumentURI:         input.DocumentURI,
		RootSourceURI:       input.RootSourceURI,
		WorkingDirectory:    input.WorkingDirectory,
		RawArguments:        append([]string(nil), input.RawArguments...),
		NormalizedArguments: append([]string(nil), input.NormalizedArguments...),
		ProfileID:           input.ProfileID,
		PassKind:            input.PassKind,
		IncludeProvenance:   cloneAnyMap(input.IncludeProvenance),
		MacroProvenance:     cloneAnyMap(input.MacroProvenance),
		HeaderVersions:      append([]SourceVersionEntry(nil), input.HeaderVersions...),
		CompilerToolVersion: input.CompilerToolVersion,
		CompareStatus:       "comparable",
		MissingFields:       append([]string(nil), input.MissingFields...),
		KnownDifference:     append([]string(nil), input.KnownDifference...),
	}
	addMissing := func(field string) {
		for _, existing := range record.MissingFields {
			if existing == field {
				return
			}
		}
		record.MissingFields = append(record.MissingFields, field)
		record.CompareStatus = "not-comparable"
	}
	if _, ok := fixtureCaseIDs[input.FixtureCaseID]; !ok || input.FixtureCaseID == "" {
		addMissing("fixture_case_id")
	}
	root := normalizeComparisonURI(input.DocumentURI)
	derivedRoot := root
	if input.RootSourceURI == "" || normalizeComparisonURI(input.RootSourceURI) != derivedRoot {
		addMissing("root_source_uri")
	}
	record.RootSourceURI = derivedRoot
	sources := make([]ComparisonSourceFile, 0, len(input.SourceFiles))
	entries := make([]SourceVersionEntry, 0, len(input.SourceFiles)+len(input.HeaderVersions))
	rootFound := false
	for _, source := range input.SourceFiles {
		uri := normalizeComparisonURI(source.URI)
		if uri == "" || source.ContentHash == "" {
			addMissing("source_files")
			continue
		}
		sources = append(sources, ComparisonSourceFile{URI: uri, ContentHash: source.ContentHash})
		entries = append(entries, SourceVersionEntry{URI: uri, Version: source.ContentHash})
		if uri == derivedRoot {
			rootFound = true
		}
	}
	if !rootFound {
		addMissing("root_source_uri in source_files")
	}
	entries = append(entries, input.HeaderVersions...)
	sort.Slice(sources, func(left, right int) bool { return sources[left].URI < sources[right].URI })
	record.SourceHash = comparisonSourceHash(sources)
	if input.SourceHash == "" {
		addMissing("source_hash")
	} else if input.SourceHash != record.SourceHash {
		addMissing("source_hash_match")
	}
	if fingerprint, err := AggregateSourceVersion(input.DocumentURI, entries); err != nil {
		addMissing("source_version_fingerprint")
	} else {
		record.SourceVersionFingerprint = fingerprint
		if input.SourceVersionFingerprint == "" {
			addMissing("source_version_fingerprint")
		} else if input.SourceVersionFingerprint != fingerprint {
			addMissing("source_version_fingerprint_match")
		}
	}
	if input.ProfileID == "" {
		addMissing("profile_id")
	}
	if len(input.RawArguments) == 0 {
		addMissing("raw_arguments")
	}
	if len(input.NormalizedArguments) == 0 {
		addMissing("normalized_arguments")
	}
	if input.PassKind != string(PassKindHost) && input.PassKind != string(PassKindDevice) && input.PassKind != string(PassKindHostDevice) {
		addMissing("pass_kind")
	}
	if input.IncludeProvenance == nil {
		addMissing("include_provenance")
	}
	if input.MacroProvenance == nil {
		addMissing("macro_provenance")
	}
	if input.WorkingDirectory == "" {
		addMissing("working_directory")
	}
	if input.CompilerToolVersion == "" {
		addMissing("compiler_tool_version")
	}
	record.NeutralDiagnostics = append([]ComparisonDiagnostic(nil), input.NeutralDiagnostics...)
	for _, diagnostic := range record.NeutralDiagnostics {
		validateComparisonDiagnostic(diagnostic, "not-applicable", addMissing)
	}
	record.PassResults = append([]ComparisonPassResult(nil), input.PassResults...)
	validateComparisonPasses(input.PassKind, record.PassResults, addMissing)
	if record.CompareStatus == "not-comparable" {
		record.NeutralDiagnostics = nil
		for index := range record.PassResults {
			record.PassResults[index].GoDiagnostics = nil
		}
	}
	return record
}

func validateComparisonPasses(passKind string, passes []ComparisonPassResult, addMissing func(string)) {
	required := []string{passKind}
	if passKind == string(PassKindHostDevice) {
		required = []string{string(PassKindHost), string(PassKindDevice)}
	}
	seen := make(map[string]struct{}, len(passes))
	for _, pass := range passes {
		if pass.PassKind != string(PassKindHost) && pass.PassKind != string(PassKindDevice) {
			addMissing("pass_results.pass_kind")
		}
		if pass.GoStatus != string(SnapshotStatusComplete) && pass.GoStatus != string(SnapshotStatusDegraded) && pass.GoStatus != "unsupported" && pass.GoStatus != "invalid" {
			addMissing("pass_results.go_status")
		}
		if pass.ClangStatus != "available" && pass.ClangStatus != "skipped" && pass.ClangStatus != "failed" && pass.ClangStatus != "not-comparable" {
			addMissing("pass_results.clang_status")
		}
		if _, duplicate := seen[pass.PassKind]; duplicate {
			addMissing("pass_results.duplicate_pass_kind")
		}
		seen[pass.PassKind] = struct{}{}
		for _, diagnostic := range pass.GoDiagnostics {
			validateComparisonDiagnostic(diagnostic, pass.PassKind, addMissing)
		}
	}
	for _, requiredPass := range required {
		if _, ok := seen[requiredPass]; !ok {
			addMissing("pass_results." + requiredPass)
		}
	}
	if passKind == string(PassKindHostDevice) && len(passes) != 2 {
		addMissing("pass_results.host_device_cardinality")
	}
}

func validateComparisonDiagnostic(diagnostic ComparisonDiagnostic, passKind string, addMissing func(string)) {
	spec, known := SemanticDiagnosticSpecForCode(diagnostic.Code)
	if !known || diagnostic.Code == "" || diagnostic.Severity == "" {
		addMissing("diagnostic.code_or_severity")
	} else if diagnostic.Severity != string(spec.Severity) {
		addMissing("diagnostic.severity")
	}
	if diagnostic.PassKind != passKind {
		addMissing("diagnostic.passKind")
	}
	if diagnostic.Range.End <= diagnostic.Range.Start || diagnostic.Range.Start < 0 {
		addMissing("diagnostic.range")
	}
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func comparisonSourceHash(sources []ComparisonSourceFile) string {
	var builder strings.Builder
	for _, source := range sources {
		builder.WriteString(source.URI)
		builder.WriteByte(0)
		builder.WriteString(source.ContentHash)
		builder.WriteByte('\n')
	}
	digest := sha256.Sum256([]byte(builder.String()))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func normalizeComparisonURI(uri string) string {
	return normalizeCacheURI(uri)
}
