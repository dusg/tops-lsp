package semantic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSemanticComparisonManifest(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	manifest, err := LoadComparisonManifest(filepath.Join(root, "testdata", "semantic", "comparison_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Cases) != 1 {
		t.Fatalf("comparison cases = %d", len(manifest.Cases))
	}
	fixtureIDs := map[string]struct{}{"declarations-valid": {}}
	directCase := manifest.Cases[0]
	normalizedSources := make([]ComparisonSourceFile, 0, len(directCase.SourceFiles))
	entries := make([]SourceVersionEntry, 0, len(directCase.SourceFiles))
	for _, source := range directCase.SourceFiles {
		normalizedURI := normalizeComparisonURI(source.URI)
		normalizedSources = append(normalizedSources, ComparisonSourceFile{URI: normalizedURI, ContentHash: source.ContentHash})
		entries = append(entries, SourceVersionEntry{URI: normalizedURI, Version: source.ContentHash})
	}
	directCase.SourceHash = comparisonSourceHash(normalizedSources)
	directCase.SourceVersionFingerprint, _ = AggregateSourceVersion(directCase.DocumentURI, entries)
	record := BuildComparisonRecord(directCase, fixtureIDs)
	if record.CompareStatus != "comparable" || len(record.PassResults) != 2 || record.SourceVersionFingerprint == "" || record.SourceHash == "" {
		t.Fatalf("comparison record = %+v", record)
	}
	invalid := manifest.Cases[0]
	invalid.RootSourceURI = "file:///workspace/other.tops"
	invalidRecord := BuildComparisonRecord(invalid, fixtureIDs)
	if invalidRecord.CompareStatus != "not-comparable" || len(invalidRecord.PassResults[0].GoDiagnostics) != 0 {
		t.Fatalf("invalid comparison record = %+v", invalidRecord)
	}
	records, err := GenerateComparisonRecords(filepath.Join(root, "testdata", "semantic", "comparison_manifest.json"), filepath.Join(root, "testdata", "semantic", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	goldenPath := filepath.Join(root, "testdata", "semantic", "expected", "semantic_comparison.json")
	if os.Getenv("TOPS_LSP_ALLOW_GOLDEN_WRITE") == "1" {
		if err := WriteComparisonRecords(goldenPath, records); err != nil {
			t.Fatal(err)
		}
	}
	body, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	var golden struct {
		Version int                `json:"version"`
		Records []ComparisonRecord `json:"records"`
	}
	if err := json.Unmarshal(body, &golden); err != nil {
		t.Fatal(err)
	}
	if golden.Version != 1 || !reflect.DeepEqual(records, golden.Records) {
		t.Fatalf("generated records differ from golden: generated=%+v golden=%+v", records, golden.Records)
	}
}

func TestSemanticComparisonBuildsGoPassResultsFromSource(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	records, err := GenerateComparisonRecords(filepath.Join(root, "testdata", "semantic", "comparison_manifest.json"), filepath.Join(root, "testdata", "semantic", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("comparison records = %d", len(records))
	}
	record := records[0]
	if record.CompareStatus != "comparable" {
		t.Fatalf("comparison record is not comparable: %+v", record)
	}
	if len(record.PassResults) != 2 {
		t.Fatalf("pass results = %+v", record.PassResults)
	}
	for _, pass := range record.PassResults {
		if pass.GoStatus != string(SnapshotStatusComplete) || len(pass.GoDiagnostics) != 0 {
			t.Fatalf("Go %s result = %+v", pass.PassKind, pass)
		}
	}
	if record.SourceHash == "" || record.SourceVersionFingerprint == "" {
		t.Fatalf("source provenance = %+v", record)
	}
}

func TestSemanticComparisonRejectsMissingProvenanceAndPassMismatches(t *testing.T) {
	base := ComparisonCase{
		FixtureCaseID:       "declarations-valid",
		DocumentURI:         "file:///workspace/main.tops",
		RootSourceURI:       "file:///workspace/main.tops",
		SourceFiles:         []ComparisonSourceFile{{URI: "file:///workspace/main.tops", ContentHash: "root-v1"}},
		WorkingDirectory:    "/workspace",
		RawArguments:        []string{"topscc", "main.tops"},
		NormalizedArguments: []string{"topscc", "main.tops"},
		ProfileID:           "gcu400-default",
		PassKind:            string(PassKindHostDevice),
		IncludeProvenance:   map[string]any{"toolchain": "default"},
		MacroProvenance:     map[string]any{"toolchain": "default"},
		CompilerToolVersion: "topscc-test",
		PassResults: []ComparisonPassResult{
			{PassKind: "host", GoStatus: "complete", ClangStatus: "skipped"},
			{PassKind: "device", GoStatus: "complete", ClangStatus: "skipped"},
		},
	}
	base.SourceHash = comparisonSourceHash(base.SourceFiles)
	base.SourceVersionFingerprint, _ = AggregateSourceVersion(base.DocumentURI, []SourceVersionEntry{{URI: base.DocumentURI, Version: "root-v1"}})
	record := BuildComparisonRecord(base, map[string]struct{}{"declarations-valid": {}})
	if record.CompareStatus != "comparable" {
		t.Fatalf("base record = %+v", record)
	}
	missingProvenance := base
	missingProvenance.SourceHash = ""
	missingProvenance.SourceVersionFingerprint = ""
	missingRecord := BuildComparisonRecord(missingProvenance, map[string]struct{}{"declarations-valid": {}})
	if missingRecord.CompareStatus != "not-comparable" || !containsComparisonField(missingRecord.MissingFields, "source_hash") || !containsComparisonField(missingRecord.MissingFields, "source_version_fingerprint") {
		t.Fatalf("missing comparison provenance = %+v", missingRecord)
	}
	base.RootSourceURI = "file:///workspace/other.tops"
	base.NeutralDiagnostics = []ComparisonDiagnostic{{Code: "tops-semantic-context-degraded", PassKind: "host"}}
	base.PassResults[0].GoDiagnostics = []ComparisonDiagnostic{{Code: "tops-semantic-unknown-name", PassKind: "device"}}
	invalid := BuildComparisonRecord(base, map[string]struct{}{"declarations-valid": {}})
	if invalid.CompareStatus != "not-comparable" || len(invalid.NeutralDiagnostics) != 0 || len(invalid.PassResults[0].GoDiagnostics) != 0 {
		t.Fatalf("invalid record = %+v", invalid)
	}
	missingFixture := BuildComparisonRecord(ComparisonCase{DocumentURI: base.DocumentURI, RootSourceURI: base.DocumentURI}, map[string]struct{}{})
	if missingFixture.CompareStatus != "not-comparable" || len(missingFixture.MissingFields) == 0 {
		t.Fatalf("missing fixture record = %+v", missingFixture)
	}
}

func TestValidateComparisonManifestRequiresProvenanceFields(t *testing.T) {
	manifest := ComparisonManifest{
		Version: 1,
		Cases: []ComparisonCase{{
			FixtureCaseID:       "declarations-valid",
			DocumentURI:         "file:///workspace/main.tops",
			RootSourceURI:       "file:///workspace/main.tops",
			SourceFiles:         []ComparisonSourceFile{{URI: "file:///workspace/main.tops", ContentHash: "root-v1"}},
			WorkingDirectory:    "/workspace",
			RawArguments:        []string{"topscc"},
			NormalizedArguments: []string{"topscc"},
			ProfileID:           "gcu400-default",
			PassKind:            string(PassKindHost),
			IncludeProvenance:   map[string]any{},
			MacroProvenance:     map[string]any{},
			CompilerToolVersion: "topscc-test",
			PassResults:         []ComparisonPassResult{{PassKind: "host", GoStatus: "complete", ClangStatus: "skipped"}},
		}},
	}
	reasons := ValidateComparisonManifest(manifest, map[string]struct{}{"declarations-valid": {}})
	if !containsComparisonField(reasons, "case declarations-valid: source_hash is required") || !containsComparisonField(reasons, "case declarations-valid: source_version_fingerprint is required") {
		t.Fatalf("comparison validation reasons = %v", reasons)
	}
}

func TestComparisonManifestMutationsBecomeNotComparable(t *testing.T) {
	manifest, err := LoadComparisonManifest(filepath.Join(repositoryRootForComparisonTest(t), "testdata", "semantic", "comparison_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtureIDs := map[string]struct{}{"declarations-valid": {}}
	mutations := []struct {
		name   string
		mutate func(*ComparisonCase)
		want   string
	}{
		{name: "root mismatch", mutate: func(value *ComparisonCase) { value.RootSourceURI = "file:///workspace/other.tops" }, want: "root_source_uri must match normalized document_uri"},
		{name: "source hash missing", mutate: func(value *ComparisonCase) { value.SourceHash = "" }, want: "source_hash is required"},
		{name: "fingerprint missing", mutate: func(value *ComparisonCase) { value.SourceVersionFingerprint = "" }, want: "source_version_fingerprint is required"},
		{name: "arguments missing", mutate: func(value *ComparisonCase) { value.RawArguments = nil }, want: "raw_arguments and normalized_arguments are required"},
		{name: "profile missing", mutate: func(value *ComparisonCase) { value.ProfileID = "" }, want: "profile_id is required"},
		{name: "pass invalid", mutate: func(value *ComparisonCase) { value.PassKind = "invalid" }, want: "invalid pass_kind"},
		{name: "include provenance missing", mutate: func(value *ComparisonCase) { value.IncludeProvenance = nil }, want: "include_provenance and macro_provenance are required"},
		{name: "compiler version missing", mutate: func(value *ComparisonCase) { value.CompilerToolVersion = "" }, want: "compiler_tool_version is required"},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			value := manifest.Cases[0]
			mutation.mutate(&value)
			reasons := ValidateComparisonManifest(ComparisonManifest{Version: 1, Cases: []ComparisonCase{value}}, fixtureIDs)
			if !containsComparisonField(reasons, "case declarations-valid: "+mutation.want) {
				t.Fatalf("mutation reasons = %v", reasons)
			}
		})
	}
}

func repositoryRootForComparisonTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func TestSemanticComparisonRejectsSourceContentHashMismatch(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	manifest, err := LoadComparisonManifest(filepath.Join(root, "testdata", "semantic", "comparison_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest.Cases[0].SourceFiles[0].ContentHash = "sha256:wrong"
	record := buildComparisonRecordAtRoot(manifest.Cases[0], map[string]struct{}{"declarations-valid": {}}, root)
	if record.CompareStatus != "not-comparable" || !containsComparisonField(record.MissingFields, "source_files.content_hash_match:testdata/semantic/declarations/valid.tops") {
		t.Fatalf("mismatched source record = %+v", record)
	}
}

func TestComparisonClangInputPreservesWorkingDirectoryAndProvenance(t *testing.T) {
	root := t.TempDir()
	argsPath := filepath.Join(root, "clang-args.txt")
	clangPath := filepath.Join(root, "clang-wrapper.sh")
	script := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$TOPS_LSP_TEST_CLANG_ARGS\"\n"
	if err := os.WriteFile(clangPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOPS_LSP_CLANG", clangPath)
	t.Setenv("TOPS_LSP_TEST_CLANG_ARGS", argsPath)
	input := ComparisonCase{
		WorkingDirectory:    root,
		NormalizedArguments: []string{"topscc", "-std=c++17", "-I", "include", "-DVALUE=1"},
		ProfileID:           "gcu400-default",
		PassKind:            string(PassKindHostDevice),
		IncludeProvenance:   map[string]any{"include_roots": []any{"/tops/include"}},
		MacroProvenance:     map[string]any{"macros": map[string]any{"PROJECT": "1"}},
		CompilerToolVersion: "topscc-test",
	}
	status, _ := comparisonClangInput(input, PassKindDevice, "file:///workspace/main.tops")
	if status != "available" {
		t.Fatalf("clang status = %q", status)
	}
	body, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	output := string(body)
	for _, expected := range []string{root, "-std=c++17", "-I", "include", "-DVALUE=1", "-I", "/tops/include", "-DPROJECT", "-D__TOPS_LSP_PROFILE_ID=gcu400-default", "-D__TOPS_LSP_PASS_KIND=device", "-D__TOPS_LSP_COMPILER_TOOL_VERSION=topscc-test"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("clang invocation missing %q: %s", expected, output)
		}
	}
}

func containsComparisonField(fields []string, expected string) bool {
	for _, field := range fields {
		if field == expected {
			return true
		}
	}
	return false
}
