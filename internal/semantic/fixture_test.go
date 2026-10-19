package semantic

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tops-lsp/internal/parser"
)

func TestRepositoryFixtureManifest(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	report := ValidateFixtureManifest(manifest)
	if report.Total != 43 || report.Passed != 43 || report.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}
	for _, key := range []string{
		"declarations/valid",
		"declarations/invalid",
		"declarations/incomplete",
		"declarations/context-boundary",
		"types/valid",
		"types/invalid",
		"types/incomplete",
		"types/context-boundary",
		"calls/valid",
		"calls/invalid",
		"calls/incomplete",
		"calls/context-boundary",
		"address-space/valid",
		"address-space/invalid",
		"address-space/incomplete",
		"address-space/context-boundary",
		"vectors/valid",
		"vectors/invalid",
		"vectors/incomplete",
		"vectors/context-boundary",
	} {
		group, ok := report.ByCapabilityScenario[key]
		if !ok || group.Total == 0 || group.Failed != 0 {
			t.Fatalf("group %q = %+v", key, group)
		}
	}
}

func TestRepositorySemanticFixturesExecuteExpectedResults(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	repositoryRoot := filepath.Join(filepath.Dir(file), "..", "..")
	report := ExecuteFixtureManifest(manifest, repositoryRoot)
	if report.Failed != 0 {
		for _, failure := range report.Failures {
			t.Errorf("fixture %s: %s", failure.CaseID, strings.Join(failure.Reasons, "; "))
		}
	}
	if report.Passed != report.Total || report.Total != len(manifest.Cases) {
		t.Fatalf("fixture execution report = %+v", report)
	}
}

func TestFixtureRunnerChecksOutOfScopeDiagnostics(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	var fixture FixtureCase
	for _, candidate := range manifest.Cases {
		if candidate.CaseID == "out-of-scope-templates" {
			fixture = candidate
			break
		}
	}
	if fixture.CaseID == "" {
		t.Fatal("out-of-scope fixture is missing")
	}
	fixture.SourceFiles = []string{"testdata/semantic/declarations/invalid.tops"}
	fixture.ExpectedDiagnostics = []FixtureDiagnostic{{
		Code:   "tops-semantic-unknown-name",
		Status: "suppressed",
		Reason: "out-of-scope",
	}}
	reasons := compareFixtureDiagnostics(fixture, []SemanticDiagnostic{{
		Code:     "tops-semantic-unknown-name",
		Severity: SeverityError,
		PassKind: "not-applicable",
		Range:    parser.SourceRange{Start: 1, End: 2},
	}})
	if len(reasons) == 0 || !strings.Contains(strings.Join(reasons, "\n"), "suppressed diagnostic") {
		t.Fatalf("out-of-scope diagnostics were not compared: %v", reasons)
	}
}

func TestRepositorySemanticFixtureSourcesAreReadable(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	seen := make(map[string]struct{})
	for _, fixture := range manifest.Cases {
		for _, sourceFile := range fixture.SourceFiles {
			if _, exists := seen[sourceFile]; exists {
				continue
			}
			seen[sourceFile] = struct{}{}
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourceFile)))
			if err != nil {
				t.Fatalf("read fixture %q: %v", sourceFile, err)
			}
			result := parser.Parse(string(body), parser.ParseContext{DocumentURI: sourceFile, LanguageID: "tops", LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "partial", TargetProfile: "gcu400", PassKind: "host+device", DocumentVersion: 1, ContextVersion: 1})
			if result.Root == nil {
				t.Fatalf("fixture %q produced no parse root", sourceFile)
			}
		}
	}
}

func TestValidateFixtureManifestRejectsInvalidCaseContracts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FixtureCase)
		reason string
	}{
		{
			name: "missing required fields",
			mutate: func(fixture *FixtureCase) {
				fixture.CaseID = ""
				fixture.SourceFiles = nil
				fixture.ProfileID = ""
				fixture.Provenance = nil
			},
			reason: "case_id is required",
		},
		{
			name: "ordinary cpp default execution space",
			mutate: func(fixture *FixtureCase) {
				fixture.Capability = "callables"
				fixture.Scenario = "valid"
				fixture.CallableKind = "host"
				fixture.PassKind = "host"
				fixture.DefaultExecutionSpace = true
				fixture.ProfileID = "cpp-default"
			},
			reason: "default_execution_space=true is only valid",
		},
		{
			name: "non boolean callable default",
			mutate: func(fixture *FixtureCase) {
				fixture.Capability = "callables"
				fixture.CallableKind = "host"
				fixture.PassKind = "host"
				fixture.DefaultExecutionSpace = "true"
			},
			reason: "must be boolean",
		},
		{
			name: "published diagnostic requires fields",
			mutate: func(fixture *FixtureCase) {
				fixture.ExpectedDiagnostics = []FixtureDiagnostic{{Code: "tops-semantic-context-degraded", Status: "published"}}
			},
			reason: "range is required",
		},
		{
			name: "suppressed diagnostic requires reason",
			mutate: func(fixture *FixtureCase) {
				fixture.ExpectedDiagnostics = []FixtureDiagnostic{{Code: "tops-semantic-context-degraded", Status: "suppressed"}}
			},
			reason: "reason is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := validFixtureCase()
			test.mutate(&fixture)
			report := ValidateFixtureManifest(FixtureManifest{Version: fixtureManifestVersion, Cases: []FixtureCase{fixture}})
			if report.Failed == 0 || !reportContainsReason(report, test.reason) {
				t.Fatalf("report = %+v, want reason containing %q", report, test.reason)
			}
		})
	}
}

func TestValidateFixtureManifestRejectsConditionCyclesAndDanglingParents(t *testing.T) {
	fixture := validFixtureCase()
	fixture.CaseID = "conditions"
	fixture.Conditions = []FixtureCondition{
		{
			ConditionID:      1,
			Macros:           map[string]any{"A": "1"},
			ParentConditions: []int{2},
			ExpressionRange:  FixtureSourceRange{SourceURI: fixture.SourceFiles[0], Start: 0, End: 1},
			ExpectedState:    "unknown",
		},
		{
			ConditionID:      2,
			Macros:           map[string]any{"B": "1"},
			ParentConditions: []int{1, 3},
			ExpressionRange:  FixtureSourceRange{SourceURI: fixture.SourceFiles[0], Start: 1, End: 2},
			ExpectedState:    "active",
		},
	}
	report := ValidateFixtureManifest(FixtureManifest{Version: fixtureManifestVersion, Cases: []FixtureCase{fixture}})
	if report.Failed == 0 || !reportContainsReason(report, "parent_conditions cycle") || !reportContainsReason(report, "missing parent condition 3") {
		t.Fatalf("report = %+v", report)
	}
}

func TestValidateFixtureManifestFilesRejectsOutOfBoundsRangesAndDuplicateConditions(t *testing.T) {
	fixture := validFixtureCase()
	fixture.CaseID = "range-check"
	fixture.Condition = &FixtureCondition{
		ConditionID:     1,
		Macros:          map[string]any{"A": "1"},
		ExpressionRange: FixtureSourceRange{SourceURI: fixture.SourceFiles[0], Start: 0, End: 100000},
		ExpectedState:   "active",
	}
	fixture.Conditions = []FixtureCondition{{ConditionID: 1, Macros: map[string]any{"B": "1"}, ExpressionRange: FixtureSourceRange{SourceURI: fixture.SourceFiles[0], Start: 0, End: 1}, ExpectedState: "active"}}
	duplicateReport := ValidateFixtureManifest(FixtureManifest{Version: fixtureManifestVersion, Cases: []FixtureCase{fixture}})
	if duplicateReport.Failed == 0 || !reportContainsReason(duplicateReport, "duplicate condition_id") {
		t.Fatalf("duplicate condition report = %+v", duplicateReport)
	}
	manifest := loadRepositoryFixtureManifest(t)
	for index := range manifest.Cases {
		if manifest.Cases[index].CaseID != "types-valid" {
			continue
		}
		manifest.Cases[index].Condition = &FixtureCondition{ConditionID: 999, Macros: map[string]any{"A": "1"}, ExpressionRange: FixtureSourceRange{SourceURI: manifest.Cases[index].SourceFiles[0], Start: 0, End: 100000}, ExpectedState: "active"}
		break
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	report := ValidateFixtureManifestFiles(manifest, root)
	if report.Failed == 0 || !reportContainsReason(report, "outside source bytes") {
		t.Fatalf("range/condition report = %+v", report)
	}
}

func loadRepositoryFixtureManifest(t *testing.T) FixtureManifest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "semantic", "manifest.json")
	manifest, err := LoadFixtureManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func validFixtureCase() FixtureCase {
	return FixtureCase{
		CaseID:                "valid-case",
		Capability:            "types",
		Scenario:              "valid",
		SourceFiles:           []string{"testdata/semantic/types/valid.tops"},
		ProfileID:             "gcu400-default",
		Provenance:            map[string]any{"source_kind": "fixture", "source_version": "test"},
		CallableKind:          "not-applicable",
		CallerKind:            "not-applicable",
		CalleeKind:            "not-applicable",
		PassKind:              "not-applicable",
		DefaultExecutionSpace: "not-applicable",
		ExpectedStatus:        "complete",
	}
}

func reportContainsReason(report FixtureManifestReport, fragment string) bool {
	for _, failure := range report.Failures {
		for _, reason := range failure.Reasons {
			if strings.Contains(reason, fragment) {
				return true
			}
		}
	}
	return false
}
