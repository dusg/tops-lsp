package semantic

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"tops-lsp/internal/position"
)

const fixtureManifestVersion = 1

var fixtureCapabilities = map[string]struct{}{
	"declarations":  {},
	"types":         {},
	"callables":     {},
	"calls":         {},
	"address-space": {},
	"vectors":       {},
	"out-of-scope":  {},
}

var fixtureScenarios = map[string]struct{}{
	"valid":            {},
	"invalid":          {},
	"incomplete":       {},
	"context-boundary": {},
	"unsupported":      {},
}

var fixtureCallableKinds = map[string]struct{}{
	"host":         {},
	"device":       {},
	"host-device":  {},
	"kernel-entry": {},
}

var fixtureOutOfScopeKinds = map[string]struct{}{
	"templates":           {},
	"overload-resolution": {},
	"constant-evaluation": {},
	"access-control":      {},
	"linkage":             {},
	"code-generation":     {},
}

var fixturePassKinds = map[string]struct{}{
	"host":           {},
	"device":         {},
	"host+device":    {},
	"not-applicable": {},
}

var fixtureDiagnosticCodes = map[string]string{
	"tops-semantic-unknown-name":         "Error",
	"tops-semantic-ambiguous-name":       "Warning",
	"tops-semantic-type-mismatch":        "Error",
	"tops-semantic-invalid-call":         "Error",
	"tops-semantic-host-device-boundary": "Error",
	"tops-semantic-address-space":        "Error",
	"tops-semantic-vector-constraint":    "Error",
	"tops-semantic-context-missing":      "Information",
	"tops-semantic-context-degraded":     "Information",
	"tops-semantic-incomplete-input":     "Hint",
}

var fixturePassSensitiveCodes = map[string]struct{}{
	"tops-semantic-invalid-call":         {},
	"tops-semantic-host-device-boundary": {},
	"tops-semantic-address-space":        {},
	"tops-semantic-vector-constraint":    {},
}

// FixtureManifest is the schema used by the semantic fixture checker.
type FixtureManifest struct {
	Version int           `json:"version"`
	Cases   []FixtureCase `json:"cases"`
}

// FixtureCase describes one semantic capability and scenario.
type FixtureCase struct {
	CaseID                string                  `json:"case_id"`
	Capability            string                  `json:"capability"`
	Scenario              string                  `json:"scenario"`
	OutOfScopeKind        string                  `json:"out_of_scope_kind,omitempty"`
	SourceFiles           []string                `json:"source_files"`
	ProfileID             string                  `json:"profile_id"`
	Provenance            map[string]any          `json:"provenance"`
	CallableKind          string                  `json:"callable_kind"`
	CallerKind            string                  `json:"caller_kind"`
	CalleeKind            string                  `json:"callee_kind"`
	PassKind              string                  `json:"pass_kind"`
	DefaultExecutionSpace any                     `json:"default_execution_space"`
	ExpectedStatus        string                  `json:"expected_status"`
	ExpectedDiagnostics   []FixtureDiagnostic     `json:"expected_diagnostics"`
	ExpectedFacts         *FixtureFactExpectation `json:"expected_facts,omitempty"`
	Condition             *FixtureCondition       `json:"condition,omitempty"`
	Conditions            []FixtureCondition      `json:"conditions,omitempty"`
}

type FixtureFactExpectation struct {
	MinDeclarations         int      `json:"min_declarations,omitempty"`
	MinScopes               int      `json:"min_scopes,omitempty"`
	MinTypes                int      `json:"min_types,omitempty"`
	MinCallables            int      `json:"min_callables,omitempty"`
	MinCalls                int      `json:"min_calls,omitempty"`
	RequireVectorType       bool     `json:"require_vector_type,omitempty"`
	RequireAddressSpaceType bool     `json:"require_address_space_type,omitempty"`
	ConversionStatuses      []string `json:"conversion_statuses,omitempty"`
	CandidateStatuses       []string `json:"candidate_statuses,omitempty"`
	PassKinds               []string `json:"pass_kinds,omitempty"`
}

// FixtureCondition records conditional compilation provenance for a fixture.
type FixtureCondition struct {
	ConditionID      int                `json:"condition_id"`
	Macros           map[string]any     `json:"macros"`
	ParentConditions []int              `json:"parent_conditions"`
	ExpressionRange  FixtureSourceRange `json:"expression_range"`
	ExpectedState    string             `json:"expected_state"`
}

// FixtureSourceRange is a byte-offset, half-open source range.
type FixtureSourceRange struct {
	SourceURI string `json:"source_uri"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
}

// FixtureDiagnostic is the normalized diagnostic expectation in a fixture.
type FixtureDiagnostic struct {
	Code     string              `json:"code"`
	Severity string              `json:"severity,omitempty"`
	PassKind string              `json:"passKind,omitempty"`
	Range    *FixtureSourceRange `json:"range,omitempty"`
	Status   string              `json:"status"`
	Reason   string              `json:"reason,omitempty"`
}

// FixtureFailure describes one invalid case and the reasons it failed.
type FixtureFailure struct {
	CaseID  string   `json:"case_id"`
	Reasons []string `json:"reasons"`
}

// FixtureGroupReport summarizes a capability/scenario group.
type FixtureGroupReport struct {
	Total          int      `json:"total"`
	Passed         int      `json:"passed"`
	Failed         int      `json:"failed"`
	FailureCaseIDs []string `json:"failure_case_ids,omitempty"`
}

// FixtureManifestReport is the command's stable JSON report.
type FixtureManifestReport struct {
	Total                int                           `json:"total"`
	Passed               int                           `json:"passed"`
	Failed               int                           `json:"failed"`
	ByCapabilityScenario map[string]FixtureGroupReport `json:"by_capability_scenario"`
	Failures             []FixtureFailure              `json:"failures,omitempty"`
}

// LoadFixtureManifest reads and decodes a semantic fixture manifest.
func LoadFixtureManifest(path string) (FixtureManifest, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return FixtureManifest{}, err
	}
	var manifest FixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return FixtureManifest{}, fmt.Errorf("decode semantic manifest: %w", err)
	}
	return manifest, nil
}

// recordFixtureOutcome folds one case result into the report and its
// capability/scenario group.
func recordFixtureOutcome(report *FixtureManifestReport, fixture FixtureCase, reasons []string) {
	groupKey := fixture.Capability + "/" + fixture.Scenario
	group := report.ByCapabilityScenario[groupKey]
	group.Total++
	if len(reasons) == 0 {
		report.Passed++
		group.Passed++
	} else {
		report.Failed++
		group.Failed++
		group.FailureCaseIDs = append(group.FailureCaseIDs, fixture.CaseID)
		report.Failures = append(report.Failures, FixtureFailure{CaseID: fixture.CaseID, Reasons: reasons})
	}
	report.Total++
	report.ByCapabilityScenario[groupKey] = group
}

// sortFixtureFailures makes the failure list order independent of case order.
func sortFixtureFailures(report *FixtureManifestReport) {
	sort.Slice(report.Failures, func(left, right int) bool { return report.Failures[left].CaseID < report.Failures[right].CaseID })
}

// ValidateFixtureManifest validates schema and cross-case registration rules.
// Fixture source files are intentionally not opened here; Phase 1 permits rows
// to reference fixture paths that are created by later implementation tasks.
func ValidateFixtureManifest(manifest FixtureManifest) FixtureManifestReport {
	report := FixtureManifestReport{
		ByCapabilityScenario: make(map[string]FixtureGroupReport),
	}

	caseIDs := make(map[string]int, len(manifest.Cases))
	conditions := make(map[int]string)
	conditionCounts := make(map[int]int)
	conditionParents := make(map[int][]int)
	for index, fixture := range manifest.Cases {
		if fixture.CaseID != "" {
			caseIDs[fixture.CaseID]++
		}
		for _, condition := range fixtureConditions(fixture) {
			if condition.ConditionID != 0 {
				conditions[condition.ConditionID] = fixture.CaseID
				conditionCounts[condition.ConditionID]++
				conditionParents[condition.ConditionID] = append([]int(nil), condition.ParentConditions...)
			}
		}
		_ = index
	}

	if manifest.Version != fixtureManifestVersion {
		addManifestFailure(&report, fmt.Sprintf("version must be %d, got %d", fixtureManifestVersion, manifest.Version))
	}
	if len(manifest.Cases) == 0 {
		addManifestFailure(&report, "cases must not be empty")
	}

	for _, fixture := range manifest.Cases {
		recordFixtureOutcome(&report, fixture, validateFixtureCase(fixture, caseIDs))
	}

	for caseID, count := range caseIDs {
		if count > 1 {
			addManifestFailure(&report, fmt.Sprintf("duplicate case_id %q", caseID))
		}
	}
	for conditionID, count := range conditionCounts {
		if count > 1 {
			addManifestFailure(&report, fmt.Sprintf("duplicate condition_id %d", conditionID))
		}
	}

	for _, fixture := range manifest.Cases {
		for _, condition := range fixtureConditions(fixture) {
			for _, parentID := range condition.ParentConditions {
				parentCaseID, ok := conditions[parentID]
				if !ok {
					addManifestFailure(&report, fmt.Sprintf("case %s: condition %d references missing parent condition %d", fixture.CaseID, condition.ConditionID, parentID))
					continue
				}
				if parentCaseID != fixture.CaseID {
					addManifestFailure(&report, fmt.Sprintf("case %s: condition %d references condition %d from case %s", fixture.CaseID, condition.ConditionID, parentID, parentCaseID))
				}
			}
		}
	}
	for conditionID := range conditions {
		if cycle := conditionCycle(conditionID, conditionParents, make(map[int]bool), make(map[int]bool)); cycle {
			addManifestFailure(&report, fmt.Sprintf("condition %d participates in a parent_conditions cycle", conditionID))
		}
	}

	validateFixtureCoverage(manifest, &report)
	sortFixtureFailures(&report)
	return report
}

func validateFixtureCase(fixture FixtureCase, caseIDs map[string]int) []string {
	var reasons []string
	require := func(condition bool, message string) {
		if !condition {
			reasons = append(reasons, message)
		}
	}

	require(fixture.CaseID != "", "case_id is required")
	if fixture.CaseID != "" && caseIDs[fixture.CaseID] > 1 {
		reasons = append(reasons, "case_id must be unique")
	}
	_, capabilityOK := fixtureCapabilities[fixture.Capability]
	require(capabilityOK, fmt.Sprintf("invalid capability %q", fixture.Capability))
	_, scenarioOK := fixtureScenarios[fixture.Scenario]
	require(scenarioOK, fmt.Sprintf("invalid scenario %q", fixture.Scenario))
	require(fixture.ProfileID != "", "profile_id is required")
	require(len(fixture.SourceFiles) > 0, "source_files must not be empty")
	require(len(fixture.Provenance) > 0, "provenance is required")
	if fixture.ExpectedFacts != nil {
		for name, value := range map[string]int{
			"min_declarations": fixture.ExpectedFacts.MinDeclarations,
			"min_scopes":       fixture.ExpectedFacts.MinScopes,
			"min_types":        fixture.ExpectedFacts.MinTypes,
			"min_callables":    fixture.ExpectedFacts.MinCallables,
			"min_calls":        fixture.ExpectedFacts.MinCalls,
		} {
			require(value >= 0, fmt.Sprintf("expected_facts.%s must not be negative", name))
		}
		for _, passKind := range fixture.ExpectedFacts.PassKinds {
			require(fixturePassKindValidForExecution(passKind), fmt.Sprintf("expected_facts.pass_kinds has invalid pass kind %q", passKind))
		}
		for _, status := range fixture.ExpectedFacts.ConversionStatuses {
			require(status == string(ConversionStatusSupported) || status == string(ConversionStatusUnsupported) || status == string(ConversionStatusUnresolved), fmt.Sprintf("expected_facts.conversion_statuses has invalid status %q", status))
		}
		for _, status := range fixture.ExpectedFacts.CandidateStatuses {
			require(status == string(CandidateStatusResolved) || status == string(CandidateStatusAmbiguous) || status == string(CandidateStatusUnresolved), fmt.Sprintf("expected_facts.candidate_statuses has invalid status %q", status))
		}
	}
	if fixture.Provenance != nil {
		sourceKind, ok := fixture.Provenance["source_kind"].(string)
		require(ok && sourceKind != "", "provenance.source_kind is required")
		sourceVersion, ok := fixture.Provenance["source_version"].(string)
		require(ok && sourceVersion != "", "provenance.source_version is required")
	}

	seenSources := make(map[string]struct{}, len(fixture.SourceFiles))
	for _, source := range fixture.SourceFiles {
		require(source != "", "source_files entries must not be empty")
		if _, exists := seenSources[source]; exists {
			reasons = append(reasons, fmt.Sprintf("source_files contains duplicate %q", source))
		}
		seenSources[source] = struct{}{}
	}

	if fixture.Capability == "out-of-scope" {
		require(fixture.Scenario == "unsupported", "out-of-scope scenario must be unsupported")
		_, ok := fixtureOutOfScopeKinds[fixture.OutOfScopeKind]
		require(ok, fmt.Sprintf("invalid out_of_scope_kind %q", fixture.OutOfScopeKind))
		require(fixture.ExpectedStatus == "unsupported" || fixture.ExpectedStatus == "degraded", "out-of-scope expected_status must be unsupported or degraded")
	} else {
		require(fixture.OutOfScopeKind == "", "out_of_scope_kind is only valid for out-of-scope cases")
		expectedStatus := map[string]string{"valid": "complete", "invalid": "invalid", "incomplete": "degraded", "context-boundary": "degraded"}[fixture.Scenario]
		require(fixture.ExpectedStatus == expectedStatus, fmt.Sprintf("scenario %q requires expected_status %q", fixture.Scenario, expectedStatus))
	}
	if _, ok := map[string]struct{}{"complete": {}, "degraded": {}, "unsupported": {}, "invalid": {}}[fixture.ExpectedStatus]; !ok {
		reasons = append(reasons, fmt.Sprintf("invalid expected_status %q", fixture.ExpectedStatus))
	}

	if fixture.Capability == "callables" {
		require(fixtureCallableKindValid(fixture.CallableKind), fmt.Sprintf("invalid callable_kind %q", fixture.CallableKind))
		require(fixture.CallerKind == "not-applicable", "callables caller_kind must be not-applicable")
		require(fixture.CalleeKind == "not-applicable", "callables callee_kind must be not-applicable")
		require(fixturePassKindValidForExecution(fixture.PassKind), fmt.Sprintf("invalid callables pass_kind %q", fixture.PassKind))
		defaultExecutionSpace, ok := fixture.DefaultExecutionSpace.(bool)
		require(ok, "callables default_execution_space must be boolean")
		if ok && defaultExecutionSpace {
			require(fixture.CallableKind == "host" && fixture.Scenario == "valid" && fixture.ExpectedStatus == "complete" && fixture.ProfileID == "gcu400-default", "default_execution_space=true is only valid for resolved Tops default-host")
		}
	} else if fixture.Capability == "calls" {
		require(fixtureCallableKindValid(fixture.CallableKind), fmt.Sprintf("invalid calls callable_kind %q", fixture.CallableKind))
		require(fixtureCallableKindValid(fixture.CallerKind), fmt.Sprintf("invalid calls caller_kind %q", fixture.CallerKind))
		require(fixtureCallableKindValid(fixture.CalleeKind), fmt.Sprintf("invalid calls callee_kind %q", fixture.CalleeKind))
		require(fixture.DefaultExecutionSpace == "not-applicable", "calls default_execution_space must be not-applicable")
		require(fixturePassKindValidForExecution(fixture.PassKind), fmt.Sprintf("invalid calls pass_kind %q", fixture.PassKind))
	} else {
		require(fixture.CallableKind == "not-applicable", fmt.Sprintf("%s callable_kind must be not-applicable", fixture.Capability))
		require(fixture.CallerKind == "not-applicable", fmt.Sprintf("%s caller_kind must be not-applicable", fixture.Capability))
		require(fixture.CalleeKind == "not-applicable", fmt.Sprintf("%s callee_kind must be not-applicable", fixture.Capability))
		require(fixture.DefaultExecutionSpace == "not-applicable", fmt.Sprintf("%s default_execution_space must be not-applicable", fixture.Capability))
		if fixture.Capability == "address-space" || fixture.Capability == "vectors" {
			require(fixturePassKindValidForExecution(fixture.PassKind), fmt.Sprintf("invalid %s pass_kind %q", fixture.Capability, fixture.PassKind))
		} else {
			require(fixture.PassKind == "not-applicable", fmt.Sprintf("%s pass_kind must be not-applicable", fixture.Capability))
		}
	}

	for index, diagnostic := range fixture.ExpectedDiagnostics {
		prefix := fmt.Sprintf("expected_diagnostics[%d]", index)
		expectedSeverity, codeOK := fixtureDiagnosticCodes[diagnostic.Code]
		require(codeOK, fmt.Sprintf("%s has invalid code %q", prefix, diagnostic.Code))
		require(diagnostic.Status == "published" || diagnostic.Status == "suppressed", fmt.Sprintf("%s has invalid status %q", prefix, diagnostic.Status))
		if diagnostic.Status == "published" {
			require(diagnostic.Severity == expectedSeverity, fmt.Sprintf("%s severity must be %q", prefix, expectedSeverity))
			require(fixturePassKindValid(diagnostic.PassKind), fmt.Sprintf("%s has invalid passKind %q", prefix, diagnostic.PassKind))
			require(diagnostic.Range != nil, fmt.Sprintf("%s range is required", prefix))
			if diagnostic.Range != nil {
				require(diagnostic.Range.Start >= 0 && diagnostic.Range.End > diagnostic.Range.Start, fmt.Sprintf("%s range must be a non-empty half-open range", prefix))
				_, sourceOK := seenSources[diagnostic.Range.SourceURI]
				require(sourceOK, fmt.Sprintf("%s range source_uri %q is not registered", prefix, diagnostic.Range.SourceURI))
			}
			_, passSensitive := fixturePassSensitiveCodes[diagnostic.Code]
			if passSensitive {
				require(diagnostic.PassKind == "host" || diagnostic.PassKind == "device", fmt.Sprintf("%s passKind must be host or device", prefix))
			} else {
				require(diagnostic.PassKind == "not-applicable", fmt.Sprintf("%s passKind must be not-applicable", prefix))
			}
		} else {
			require(diagnostic.Reason != "", fmt.Sprintf("%s reason is required when suppressed", prefix))
		}
	}

	for _, condition := range fixtureConditions(fixture) {
		prefix := fmt.Sprintf("condition %d", condition.ConditionID)
		require(condition.ConditionID > 0, prefix+" condition_id must be positive")
		require(condition.Macros != nil, prefix+" macros is required")
		require(condition.ExpectedState == "active" || condition.ExpectedState == "inactive" || condition.ExpectedState == "unknown", prefix+" has invalid expected_state")
		require(condition.ExpressionRange.Start >= 0 && condition.ExpressionRange.End > condition.ExpressionRange.Start, prefix+" expression_range must be a non-empty half-open range")
		_, sourceOK := seenSources[condition.ExpressionRange.SourceURI]
		require(sourceOK, prefix+" expression_range source_uri is not registered")
	}

	return reasons
}

// ValidateFixtureManifestFiles validates ranges against the actual fixture bytes.
func ValidateFixtureManifestFiles(manifest FixtureManifest, repositoryRoot string) FixtureManifestReport {
	report := ValidateFixtureManifest(manifest)
	if report.Failed != 0 {
		return report
	}
	for _, fixture := range manifest.Cases {
		texts := make(map[string]string, len(fixture.SourceFiles))
		for _, sourceFile := range fixture.SourceFiles {
			path, err := fixtureSourcePath(repositoryRoot, sourceFile)
			if err != nil {
				addManifestFailure(&report, fmt.Sprintf("case %s: source %q: %v", fixture.CaseID, sourceFile, err))
				continue
			}
			body, err := os.ReadFile(path)
			if err != nil {
				addManifestFailure(&report, fmt.Sprintf("case %s: source %q: %v", fixture.CaseID, sourceFile, err))
				continue
			}
			texts[sourceFile] = string(body)
		}
		checkRange := func(label string, value FixtureSourceRange, requireNonEmpty bool) {
			text, ok := texts[value.SourceURI]
			if !ok {
				addManifestFailure(&report, fmt.Sprintf("case %s: %s source_uri %q is not readable", fixture.CaseID, label, value.SourceURI))
				return
			}
			if value.Start < 0 || value.End < value.Start || value.End > len(text) || (requireNonEmpty && value.Start == value.End) {
				addManifestFailure(&report, fmt.Sprintf("case %s: %s range [%d,%d) is outside source bytes", fixture.CaseID, label, value.Start, value.End))
				return
			}
			if _, err := position.New(text).Range(value.Start, value.End); err != nil {
				addManifestFailure(&report, fmt.Sprintf("case %s: %s range cannot map to UTF-16: %v", fixture.CaseID, label, err))
			}
		}
		for index, diagnostic := range fixture.ExpectedDiagnostics {
			if diagnostic.Range != nil {
				checkRange(fmt.Sprintf("expected_diagnostics[%d]", index), *diagnostic.Range, diagnostic.Status == "published")
			}
		}
		for _, condition := range fixtureConditions(fixture) {
			checkRange(fmt.Sprintf("condition %d expression_range", condition.ConditionID), condition.ExpressionRange, true)
		}
	}
	return report
}

func validateFixtureCoverage(manifest FixtureManifest, report *FixtureManifestReport) {
	regularCapabilities := []string{"declarations", "types", "callables", "calls", "address-space", "vectors"}
	requiredScenarios := []string{"valid", "invalid", "incomplete", "context-boundary"}
	seen := make(map[string]struct{}, len(manifest.Cases))
	callableGrid := make(map[string]struct{})
	outOfScope := make(map[string]struct{})
	callableCount := 0
	for _, fixture := range manifest.Cases {
		seen[fixture.Capability+"/"+fixture.Scenario] = struct{}{}
		if fixture.Capability == "callables" {
			callableCount++
			callableGrid[fixture.CallableKind+"/"+fixture.Scenario] = struct{}{}
		}
		if fixture.Capability == "out-of-scope" {
			outOfScope[fixture.OutOfScopeKind] = struct{}{}
		}
	}
	for _, capability := range regularCapabilities {
		for _, scenario := range requiredScenarios {
			if _, ok := seen[capability+"/"+scenario]; !ok {
				addManifestFailure(report, fmt.Sprintf("missing capability/scenario registration %s/%s", capability, scenario))
			}
		}
	}
	if callableCount != len(fixtureCallableKinds)*len(requiredScenarios) {
		addManifestFailure(report, fmt.Sprintf("callables must register exactly %d callable cases, got %d", len(fixtureCallableKinds)*len(requiredScenarios), callableCount))
	}
	for callableKind := range fixtureCallableKinds {
		for _, scenario := range requiredScenarios {
			if _, ok := callableGrid[callableKind+"/"+scenario]; !ok {
				addManifestFailure(report, fmt.Sprintf("missing callable registration %s/%s", callableKind, scenario))
			}
		}
	}
	for outOfScopeKind := range fixtureOutOfScopeKinds {
		if _, ok := outOfScope[outOfScopeKind]; !ok {
			addManifestFailure(report, fmt.Sprintf("missing out_of_scope_kind registration %s", outOfScopeKind))
		}
	}
}

func addManifestFailure(report *FixtureManifestReport, reason string) {
	report.Total++
	report.Failed++
	report.Failures = append(report.Failures, FixtureFailure{CaseID: "<manifest>", Reasons: []string{reason}})
	group := report.ByCapabilityScenario["<manifest>"]
	group.Total++
	group.Failed++
	group.FailureCaseIDs = append(group.FailureCaseIDs, "<manifest>")
	report.ByCapabilityScenario["<manifest>"] = group
}

func fixtureConditions(fixture FixtureCase) []FixtureCondition {
	conditions := make([]FixtureCondition, 0, len(fixture.Conditions)+1)
	if fixture.Condition != nil {
		conditions = append(conditions, *fixture.Condition)
	}
	conditions = append(conditions, fixture.Conditions...)
	return conditions
}

func conditionCycle(conditionID int, parents map[int][]int, visiting, visited map[int]bool) bool {
	if visiting[conditionID] {
		return true
	}
	if visited[conditionID] {
		return false
	}
	visiting[conditionID] = true
	for _, parentID := range parents[conditionID] {
		if _, ok := parents[parentID]; ok && conditionCycle(parentID, parents, visiting, visited) {
			return true
		}
	}
	delete(visiting, conditionID)
	visited[conditionID] = true
	return false
}

func fixtureCallableKindValid(kind string) bool {
	_, ok := fixtureCallableKinds[kind]
	return ok
}

func fixturePassKindValid(passKind string) bool {
	_, ok := fixturePassKinds[passKind]
	return ok
}

func fixturePassKindValidForExecution(passKind string) bool {
	return passKind == "host" || passKind == "device" || passKind == "host+device"
}
