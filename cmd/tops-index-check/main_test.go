package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestManifestExecutorReplaysVersionAndHeaderLifecycles(t *testing.T) {
	root := filepath.Join("..", "..")
	manifest, err := index.LoadManifest(filepath.Join(root, "testdata", "index", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := index.ValidateManifestAtWithMetricsExecutor(root, manifest, executeManifestQueryWithMetrics)
	if err != nil {
		t.Fatal(err)
	}
	executableCases := 0
	schemaOnlyCases := 0
	for _, fixture := range manifest.Cases {
		if fixture.ValidationMode == "schema-only" {
			schemaOnlyCases++
		} else {
			executableCases++
		}
	}
	if report.Failed != 0 || report.Passed != executableCases || report.Skipped != schemaOnlyCases || report.OldVersionOverwriteCount != 0 {
		t.Fatalf("lifecycle replay report = %+v", report)
	}
}

func TestManifestExecutorRunsInFlightRaceForAllNavigationOperations(t *testing.T) {
	root := t.TempDir()
	oldText := "int old_name() { return 1; }\n"
	newText := "int new_name() { return 2; }\n"
	if err := os.WriteFile(filepath.Join(root, "main.tops"), []byte(oldText), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(root, "main.tops"))}).String()
	parseContext := parser.ParseContext{DocumentURI: uri, LanguageID: "tops", CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "host", IncludeRoots: []string{root}, IncludeRootsAvailable: true, DocumentVersion: 2, ContextVersion: 1, PredefinedMacros: map[string]parser.MacroValue{}}
	snapshot := index.Build(newText, parser.Parse(newText, parseContext), parseContext)
	fixture := index.FixtureCase{CaseID: "in-flight-navigation", Category: "version-race", SourceFiles: []string{"main.tops"}, SourceURIs: []string{"main.tops"}, DocumentVersions: []int{1, 2}, VersionTexts: []string{oldText, newText}}
	for _, operation := range []string{"definition", "references", "documentSymbol"} {
		query := index.ManifestQuery{Operation: operation, Name: "new_name", SourceURI: "main.tops"}
		body, overwriteCount, err := executeManifestQueryWithMetrics(root, fixture, snapshot, query)
		if err != nil {
			t.Fatalf("operation %s: %v", operation, err)
		}
		if overwriteCount != 0 {
			t.Fatalf("operation %s old version overwrite count = %d", operation, overwriteCount)
		}
		var response struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("operation %s response: %v", operation, err)
		}
		if string(response.Result) == "null" {
			t.Fatalf("operation %s final response remained null: %s", operation, body)
		}
	}
}
