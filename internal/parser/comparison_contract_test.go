package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestComparisonRecordsContainReproducibleContextAndStages(t *testing.T) {
	root := standardFixtureRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "expected", "clang_comparison.json"))
	if err != nil {
		t.Fatalf("read comparison records: %v", err)
	}
	var document struct {
		Records []struct {
			RawArguments        []string                     `json:"raw_arguments"`
			NormalizedArguments []string                     `json:"normalized_arguments"`
			PredefinedMacros    map[string]json.RawMessage   `json:"predefined_macros"`
			IncludeRoots        []string                     `json:"include_roots"`
			TargetProfile       string                       `json:"target_profile"`
			PassKind            string                       `json:"pass_kind"`
			SourceHash          string                       `json:"source_hash"`
			ClangVersion        string                       `json:"clang_version"`
			Stages              map[string]string            `json:"stages"`
			Commands            map[string][]string          `json:"commands"`
			Differences         []map[string]json.RawMessage `json:"differences"`
			ClangStatus         string                       `json:"clang_status"`
		} `json:"records"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("decode comparison records: %v", err)
	}
	if len(document.Records) == 0 {
		t.Fatal("comparison records are empty")
	}
	for index, record := range document.Records {
		if record.RawArguments == nil || record.NormalizedArguments == nil || record.PredefinedMacros == nil || record.IncludeRoots == nil || record.SourceHash == "" || record.ClangVersion == "" || record.Commands == nil || record.Differences == nil {
			t.Fatalf("record %d lacks reproducible context: %+v", index, record)
		}
		if record.ClangStatus == "not-run" {
			t.Fatalf("record %d keeps Clang status at not-run", index)
		}
		for _, stage := range []string{"preprocess", "syntax", "ast", "ranges", "diagnostics"} {
			status, ok := record.Stages[stage]
			if !ok || status == "not-run" || status == "requested" {
				t.Fatalf("record %d stage %q is not reproducible: %q", index, stage, status)
			}
			if len(record.Commands[stage]) == 0 {
				t.Fatalf("record %d stage %q has no command provenance", index, stage)
			}
		}
	}
}

func TestComparisonRecordsSourceHashesMatchManifest(t *testing.T) {
	root := standardFixtureRoot(t)
	manifest := loadFixtureManifest(t, filepath.Join(root, "manifest.json"))
	fixtureSources := make(map[string]string)
	for _, fixture := range manifest.Fixtures {
		if len(fixture.Compare) > 0 {
			fixtureSources[fixture.ID] = fixture.Source
		}
	}
	body, err := os.ReadFile(filepath.Join(root, "expected", "clang_comparison.json"))
	if err != nil {
		t.Fatalf("read comparison records: %v", err)
	}
	var document struct {
		Records []struct {
			Fixture    string `json:"fixture"`
			SourceHash string `json:"source_hash"`
		} `json:"records"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("decode comparison records: %v", err)
	}
	for _, record := range document.Records {
		relativeSource, ok := fixtureSources[record.Fixture]
		if !ok {
			t.Fatalf("comparison record %q is not in manifest", record.Fixture)
		}
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativeSource)))
		if err != nil {
			t.Fatalf("read source %q: %v", relativeSource, err)
		}
		if record.SourceHash != sourceHash(source) {
			t.Fatalf("comparison record %q has stale source hash", record.Fixture)
		}
	}
}
