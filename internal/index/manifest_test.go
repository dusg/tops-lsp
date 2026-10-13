package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tops-lsp/internal/parser"
)

func TestValidateManifestReportsCategoryCounts(t *testing.T) {
	manifest := Manifest{Cases: []FixtureCase{
		{CaseID: "core", Category: "core-symbol", Context: manifestTestContext(), SourceFiles: []string{"main.tops"}, SourceURIs: []string{"main.tops"}, ExpectedStatus: "complete", ExpectedSymbolCount: 2},
		{CaseID: "missing", Category: "missing-context", Context: manifestTestContext(), SourceFiles: []string{"missing.tops"}, SourceURIs: []string{"missing.tops"}, ExpectedStatus: "degraded"},
	}}
	report, err := ValidateManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.Passed != 2 || report.Failed != 0 || report.ByCategory["core-symbol"].Total != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestReportsInvalidCaseIDAndCategory(t *testing.T) {
	manifest := Manifest{Cases: []FixtureCase{{Category: "unknown", ExpectedStatus: "complete"}}}
	report, err := ValidateManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 1 || len(report.FailureIDs) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestAtBuildsFixtureAndChecksExpectedStatus(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "main.tops")
	if err := os.WriteFile(fixturePath, []byte("int value();\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Cases: []FixtureCase{{
		CaseID:                "actual",
		Category:              "core-symbol",
		Context:               manifestTestContext(),
		SourceURIs:            []string{"main.tops"},
		SourceFiles:           []string{"main.tops"},
		ExpectedSymbolCount:   1,
		ExpectedLocationCount: 1,
		ExpectedStatus:        "complete",
	}}}
	report, err := ValidateManifestAt(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 0 || report.Passed != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestAtRejectsExecutableCaseWithoutSourceFiles(t *testing.T) {
	root := t.TempDir()
	report, err := ValidateManifestAt(root, Manifest{Cases: []FixtureCase{{
		CaseID:              "missing-material",
		Category:            "core-symbol",
		SourceURIs:          []string{"file:///workspace/main.tops"},
		ExpectedStatus:      "complete",
		ExpectedSymbolCount: 0,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 1 || len(report.FailureIDs) != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestAtReportsSchemaOnlyCaseAsSkipped(t *testing.T) {
	root := t.TempDir()
	report, err := ValidateManifestAt(root, Manifest{Cases: []FixtureCase{{
		CaseID:         "out-of-scope",
		Category:       "out-of-scope",
		ValidationMode: "schema-only",
		SkipReason:     "not an index execution case",
		ExpectedStatus: "out-of-scope",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 0 || report.Skipped != 1 || report.ByCategory["out-of-scope"].Skipped != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateHeaderChangeUsesDependencyTarget(t *testing.T) {
	root := t.TempDir()
	primaryURI := fileURI(filepath.Join(root, "main.tops"))
	dependencyURI := fileURI(filepath.Join(root, "api.h"))
	unrelatedURI := fileURI(filepath.Join(root, "unrelated.h"))
	primaryText := "#include \"api.h\"\nint use() { return api_value; }\n"
	opened := map[string]SourceDocument{
		primaryURI:    {URI: primaryURI, Text: primaryText, Version: 1, SourceVersion: SourceDigest(primaryText)},
		dependencyURI: {URI: dependencyURI, Text: "int api_value;\n", Version: 1, SourceVersion: SourceDigest("int api_value;\n")},
		unrelatedURI:  {URI: unrelatedURI, Text: "int unrelated;\n", Version: 1, SourceVersion: SourceDigest("int unrelated;\n")},
	}
	parseContext := parser.ParseContext{DocumentURI: primaryURI, LanguageID: "tops", IncludeRoots: []string{root}, IncludeRootsAvailable: true, CompilerContextStatus: "resolved", DocumentVersion: 1, ContextVersion: 1}
	previous := BuildWithProvider(primaryText, parser.Parse(primaryText, parseContext), parseContext, NewLocalSourceProvider(opened, []string{root}))
	store := NewStore()
	previous.Generation = 1
	previous.SourceVersion.Generation = 1
	if !store.Replace(previous) {
		t.Fatal("previous snapshot was not stored")
	}
	if err := validateHeaderChange(root, FixtureCase{CaseID: "dependency-target", Context: manifestTestContext()}, opened, []string{root}, primaryURI, primaryText, store, previous); err != nil {
		t.Fatal(err)
	}
	current, ok := store.Get(primaryURI)
	if !ok || len(current.Dependencies) == 0 || current.Dependencies[0].TargetURI != dependencyURI {
		t.Fatalf("changed dependencies = %+v, ok = %v", current.Dependencies, ok)
	}
}

func TestValidateManifestSchemaRejectsMissingSourcesWithoutRoot(t *testing.T) {
	report, err := ValidateManifest(Manifest{Cases: []FixtureCase{{CaseID: "missing-schema", Category: "core-symbol", ExpectedStatus: "complete"}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestAtRejectsMismatchedCanonicalSourceURI(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateManifestAt(root, Manifest{Cases: []FixtureCase{{
		CaseID:                "mismatch",
		Category:              "core-symbol",
		SourceFiles:           []string{"main.tops"},
		SourceURIs:            []string{"other.tops"},
		ExpectedSymbolCount:   1,
		ExpectedLocationCount: 1,
		ExpectedStatus:        "complete",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestRejectsOperationWithoutOracle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateManifestAt(root, Manifest{Cases: []FixtureCase{{
		CaseID:                "missing-oracle",
		Category:              "core-symbol",
		SourceFiles:           []string{"main.tops"},
		SourceURIs:            []string{"main.tops"},
		ExpectedSymbolCount:   1,
		ExpectedLocationCount: 1,
		ExpectedStatus:        "complete",
		LSPOperations:         []string{"definition"},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateManifestRejectsCountOnlyNavigationOracle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Cases: []FixtureCase{
		{
			CaseID: "count-only-definition", Category: "core-symbol", Context: manifestTestContext(), SourceFiles: []string{"main.tops"}, SourceURIs: []string{"main.tops"}, ExpectedSymbolCount: 1, ExpectedLocationCount: 1, ExpectedStatus: "complete",
			Queries: []ManifestQuery{{Operation: "definition", Name: "value", SourceURI: "main.tops", ExpectedCount: intPointer(1)}},
		},
		{
			CaseID: "count-only-symbols", Category: "core-symbol", Context: manifestTestContext(), SourceFiles: []string{"main.tops"}, SourceURIs: []string{"main.tops"}, ExpectedSymbolCount: 1, ExpectedLocationCount: 1, ExpectedStatus: "complete",
			Queries: []ManifestQuery{{Operation: "documentSymbol", SourceURI: "main.tops", ExpectedCount: intPointer(1)}},
		},
	}}
	report, err := ValidateManifestAt(root, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 0 || report.Failed != 2 {
		t.Fatalf("count-only oracle report = %+v", report)
	}
}

func TestValidateManifestQueriesRejectsDocumentSymbolSelectionOutsideRange(t *testing.T) {
	root := t.TempDir()
	uri := fileURI(filepath.Join(root, "main.tops"))
	snapshot := IndexSnapshot{
		RootURI:     uri,
		IndexStatus: IndexStatusComplete,
		Sources:     map[string]string{uri: "int value;\n"},
		RootSymbols: []string{"value"},
		Symbols: map[string]SymbolRecord{
			"value": {
				Name:      "value",
				Kind:      SymbolKindVariable,
				NameRange: parser.SourceRange{Start: 4, End: 9},
				Definitions: []SymbolLocation{{
					URI:            uri,
					Range:          parser.SourceRange{Start: 0, End: 4},
					SelectionRange: parser.SourceRange{Start: 4, End: 9},
					Role:           SymbolLocationDefinition,
				}},
			},
		},
	}
	err := validateManifestQueries(root, snapshot, FixtureCase{Queries: []ManifestQuery{{
		Operation:     "documentSymbol",
		SourceURI:     "main.tops",
		ExpectedCount: intPointer(1),
		ExpectedSymbols: []DocumentSymbolExpectation{{
			Name:           "value",
			Kind:           int(manifestSymbolKind(SymbolKindVariable, "")),
			Range:          LocationExpectation{URI: "main.tops", Start: intPointer(0), End: intPointer(4)},
			SelectionRange: LocationExpectation{URI: "main.tops", Start: intPointer(4), End: intPointer(9)},
		}},
	}}}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected invalid document symbol selection range to be rejected")
	}
}

func TestValidateManifestAtWithExecutorRunsStandardQueryOracle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	report, err := ValidateManifestAtWithExecutor(root, Manifest{Cases: []FixtureCase{{
		CaseID:                "executor-query",
		Category:              "core-symbol",
		Context:               manifestTestContext(),
		SourceFiles:           []string{"main.tops"},
		SourceURIs:            []string{"main.tops"},
		ExpectedSymbolCount:   1,
		ExpectedLocationCount: 1,
		ExpectedStatus:        "complete",
		Queries: []ManifestQuery{{
			Operation:      "definition",
			Name:           "value",
			SourceURI:      "main.tops",
			ExpectedCount:  intPointer(1),
			ExpectedRanges: []LocationExpectation{{URI: "main.tops", Start: intPointer(0), End: intPointer(10)}},
		}},
	}}}, func(root string, _ FixtureCase, snapshot IndexSnapshot, query ManifestQuery) ([]byte, error) {
		called = true
		positionValue, err := ManifestQueryPosition(snapshot, query)
		if err != nil || positionValue.Line != 0 || positionValue.Character != 4 {
			t.Fatalf("query position = %+v, err = %v", positionValue, err)
		}
		return []byte(`{"jsonrpc":"2.0","id":1,"result":[{"uri":"` + fileURI(filepath.Join(root, "main.tops")) + `","range":{"start":{"line":0,"character":0},"end":{"line":0,"character":10}}}]}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called || report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("executor report = %+v, called = %v", report, called)
	}
}

func TestValidateManifestReportsOldVersionOverwriteCount(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Cases: []FixtureCase{{
		CaseID: "overwrite", Category: "version-race", Context: manifestTestContext(), SourceFiles: []string{"main.tops"}, SourceURIs: []string{"main.tops"}, DocumentVersions: []int{1, 2}, VersionTexts: []string{"int old;\n", "int value;\n"}, ExpectedSymbolCount: 1, ExpectedLocationCount: 1, ExpectedStatus: "complete",
		Queries: []ManifestQuery{{Operation: "definition", Name: "value", SourceURI: "main.tops", ExpectedCount: intPointer(1), ExpectedRanges: []LocationExpectation{{URI: "main.tops", Start: intPointer(4), End: intPointer(9)}}}},
	}}}
	report, err := ValidateManifestAtWithExecutor(root, manifest, func(string, FixtureCase, IndexSnapshot, ManifestQuery) ([]byte, error) {
		return nil, &ManifestValidationFailure{Err: errors.New("stale result"), OldVersionOverwriteCount: 2}
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Failed != 1 || report.OldVersionOverwriteCount != 2 {
		t.Fatalf("overwrite report = %+v", report)
	}
}

func TestValidateManifestAtLoadsIndependentExpectedFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte("int value;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "expected.json"), []byte(`{"case_id":"independent","category":"core-symbol","expected_symbol_count":1,"expected_location_count":1,"expected_status":"complete","expected_symbols":[{"name":"value","kind":"variable"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ValidateManifestAt(root, Manifest{Cases: []FixtureCase{{
		CaseID:                "independent",
		Category:              "core-symbol",
		Context:               manifestTestContext(),
		SourceFiles:           []string{"main.tops"},
		SourceURIs:            []string{"main.tops"},
		ExpectedFile:          "expected.json",
		ExpectedSymbolCount:   1,
		ExpectedLocationCount: 1,
		ExpectedStatus:        "complete",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed != 1 || report.Failed != 0 {
		t.Fatalf("independent expected report = %+v", report)
	}
}

func intPointer(value int) *int {
	return &value
}

func manifestTestContext() FixtureContext {
	return FixtureContext{
		LanguageStandard:      "c++17",
		DriverKind:            "manifest",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "manifest-test",
		TargetProfile:         "gcu400",
		PassKind:              "host",
		ContextVersion:        1,
	}
}
