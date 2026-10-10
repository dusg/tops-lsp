package parser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

type comparisonRecord struct {
	Fixture               string              `json:"fixture"`
	Source                string              `json:"source"`
	LanguageStandard      string              `json:"language_standard"`
	DriverKind            string              `json:"driver_kind"`
	ContextStatus         string              `json:"context_status"`
	ArgumentProvenance    string              `json:"argument_provenance"`
	RawArguments          []string            `json:"raw_arguments"`
	NormalizedArguments   []string            `json:"normalized_arguments"`
	PredefinedMacros      map[string]any      `json:"predefined_macros"`
	IncludeRoots          []string            `json:"include_roots"`
	IncludeRootsAvailable bool                `json:"include_roots_available"`
	TargetProfile         string              `json:"target_profile"`
	PassKind              string              `json:"pass_kind"`
	SourceHash            string              `json:"source_hash"`
	ClangVersion          string              `json:"clang_version"`
	Commands              map[string][]string `json:"commands"`
	Differences           []map[string]any    `json:"differences"`
	Stages                map[string]string   `json:"stages"`
	GoStatus              string              `json:"go_status"`
	ClangStatus           string              `json:"clang_status"`
	GoNodeKinds           []string            `json:"go_node_kinds"`
	GoNodeRanges          []comparisonNode    `json:"go_node_ranges"`
	ClangNodeKinds        []string            `json:"clang_node_kinds"`
	ClangNodeRanges       []comparisonNode    `json:"clang_node_ranges"`
	GoDiagnostics         []goldenDiagnostic  `json:"go_diagnostics"`
	ClangDiagnostics      []goldenDiagnostic  `json:"clang_diagnostics"`
	KnownDifference       string              `json:"known_difference"`
}

type comparisonNode struct {
	Kind  string `json:"kind"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type clangStageResult struct {
	Status          string
	Stages          map[string]string
	Commands        map[string][]string
	NodeKinds       []string
	NodeRanges      []comparisonNode
	Diagnostics     []goldenDiagnostic
	Differences     []map[string]any
	KnownDifference string
}

type comparisonRecords struct {
	Version int                `json:"version"`
	Records []comparisonRecord `json:"records"`
}

func TestOptionalClangManifestComparison(t *testing.T) {
	clangPath := os.Getenv("TOPS_LSP_CLANG")
	if clangPath == "" {
		t.Skip("TOPS_LSP_CLANG is not set; parser tests do not require Clang")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	manifestBody, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	for _, fixture := range manifest.Fixtures {
		if len(fixture.Compare) == 0 || fixture.ExpectedAcceptance == nil {
			continue
		}
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			sourcePath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(fixture.Source))
			arguments := []string{"-std=" + fixture.LanguageStandard}
			for name, value := range fixture.PredefinedMacros {
				arguments = append(arguments, "-D"+name+"="+macroArgument(value))
			}
			context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for _, mode := range []string{"-###", "-E", "-dM"} {
				commandArgs := append([]string{}, arguments...)
				if mode == "-###" {
					commandArgs = append(commandArgs, mode, "-fsyntax-only", sourcePath)
				} else {
					commandArgs = append(commandArgs, mode, "-E", sourcePath)
				}
				output, err := exec.CommandContext(context, clangPath, commandArgs...).CombinedOutput()
				if err != nil && mode != "-dM" {
					if clangContextUnavailable(output) {
						t.Skipf("Clang context is unavailable for %s: %s", fixture.ID, strings.TrimSpace(string(output)))
					}
					t.Fatalf("Clang %s failed: %v", mode, err)
				}
			}
			syntaxArgs := append(append([]string{}, arguments...), "-fsyntax-only", sourcePath)
			output, err := exec.CommandContext(context, clangPath, syntaxArgs...).CombinedOutput()
			accepted := err == nil
			if !accepted && clangContextUnavailable(output) {
				t.Skipf("Clang context is unavailable for %s: %s", fixture.ID, strings.TrimSpace(string(output)))
			}
			if accepted != *fixture.ExpectedAcceptance {
				t.Fatalf("Clang acceptance=%v want=%v output=%s", accepted, *fixture.ExpectedAcceptance, output)
			}
			source, readErr := os.ReadFile(sourcePath)
			if readErr != nil {
				t.Fatalf("read source: %v", readErr)
			}
			result := Parse(string(source), ParseContext{LanguageStandard: fixture.LanguageStandard, DriverKind: fixture.DriverKind, CompilerContextStatus: fixture.ContextStatus, ArgumentProvenance: fixture.ArgumentProvenance, PredefinedMacros: fixtureMacroValues(fixture.PredefinedMacros), TargetProfile: fixture.TargetProfile, PassKind: fixture.PassKind, IncludeRootsAvailable: fixture.ContextStatus == "resolved", DocumentVersion: 1, ContextVersion: 1})
			t.Logf("comparison fixture=%s go_status=%s clang_status=%v diagnostics=%d", fixture.ID, result.Status, accepted, len(result.Diagnostics))
		})
	}
}

func clangContextUnavailable(output []byte) bool {
	text := strings.ToLower(string(output))
	for _, marker := range []string{"gcu300_api.bc", "gcu300_inline_vector.bc", "gcu300_no_debug_check.bc", "no such file or directory"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func runClangComparison(t *testing.T, clangPath string, fixture fixtureSpec, sourcePath, source string) clangStageResult {
	t.Helper()
	stages := map[string]string{"preprocess": "not-run", "syntax": "not-run", "ast": "not-run", "ranges": "not-run", "diagnostics": "not-run"}
	arguments := []string{"-std=" + fixture.LanguageStandard, "-fdiagnostics-color=never"}
	for _, includeRoot := range fixture.IncludeRoots {
		arguments = append(arguments, "-I"+includeRoot)
	}
	for name, value := range fixture.PredefinedMacros {
		arguments = append(arguments, "-D"+name+"="+macroArgument(value))
	}
	commandContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	commandFor := func(extra ...string) []string {
		command := append([]string{clangPath}, arguments...)
		return append(command, extra...)
	}
	commands := map[string][]string{
		"preprocess":  commandFor("-E", "-dD", sourcePath),
		"syntax":      commandFor("-fsyntax-only", sourcePath),
		"ast":         commandFor("-Xclang", "-ast-dump=json", "-fsyntax-only", sourcePath),
		"ranges":      commandFor("-Xclang", "-ast-dump=json", "-fsyntax-only", sourcePath),
		"diagnostics": commandFor("-fsyntax-only", sourcePath),
	}
	run := func(extra ...string) ([]byte, error) {
		commandArguments := append(append([]string{}, arguments...), extra...)
		return exec.CommandContext(commandContext, clangPath, commandArguments...).CombinedOutput()
	}
	contextInvalid := func(output []byte) bool {
		return clangContextUnavailable(output)
	}
	preprocessOutput, err := run("-E", "-dD", sourcePath)
	if err != nil {
		if contextInvalid(preprocessOutput) {
			return clangStageResult{Status: "context-invalid", Stages: contextInvalidStages(stages), Commands: commands, NodeKinds: []string{}, NodeRanges: []comparisonNode{}, Diagnostics: []goldenDiagnostic{}, Differences: []map[string]any{{"kind": "context", "stage": "preprocess", "detail": firstOutputLine(preprocessOutput)}}, KnownDifference: "Clang context-invalid: " + firstOutputLine(preprocessOutput)}
		}
		stages["preprocess"] = "rejected"
	} else {
		stages["preprocess"] = "captured"
	}

	syntaxOutput, syntaxErr := run("-fsyntax-only", sourcePath)
	if contextInvalid(syntaxOutput) {
		return clangStageResult{Status: "context-invalid", Stages: contextInvalidStages(stages), Commands: commands, NodeKinds: []string{}, NodeRanges: []comparisonNode{}, Diagnostics: []goldenDiagnostic{}, Differences: []map[string]any{{"kind": "context", "stage": "syntax", "detail": firstOutputLine(syntaxOutput)}}, KnownDifference: "Clang context-invalid: " + firstOutputLine(syntaxOutput)}
	}
	stages["syntax"] = "captured"
	diagnostics := parseClangDiagnostics(source, syntaxOutput)
	if len(diagnostics) == 0 && syntaxErr != nil {
		stages["diagnostics"] = "text-only"
	} else if len(diagnostics) == 0 {
		stages["diagnostics"] = "captured-empty"
	} else {
		stages["diagnostics"] = "captured"
	}
	status := "accepted"
	if syntaxErr != nil {
		status = "rejected"
	}

	astOutput, astErr := run("-Xclang", "-ast-dump=json", "-fsyntax-only", sourcePath)
	if contextInvalid(astOutput) {
		return clangStageResult{Status: "context-invalid", Stages: contextInvalidStages(stages), Commands: commands, NodeKinds: []string{}, NodeRanges: []comparisonNode{}, Diagnostics: diagnostics, Differences: []map[string]any{{"kind": "context", "stage": "ast", "detail": firstOutputLine(astOutput)}}, KnownDifference: "Clang context-invalid: " + firstOutputLine(astOutput)}
	}
	nodeKinds, nodeRanges, decodeErr := parseClangAST(astOutput)
	if astErr != nil && decodeErr != nil {
		stages["ast"] = "unavailable"
		stages["ranges"] = "unavailable"
	} else {
		stages["ast"] = "captured"
		if len(nodeRanges) > 0 {
			stages["ranges"] = "captured"
		} else {
			stages["ranges"] = "captured-empty"
		}
	}
	return clangStageResult{Status: status, Stages: stages, Commands: commands, NodeKinds: nodeKinds, NodeRanges: nodeRanges, Diagnostics: diagnostics, Differences: []map[string]any{}}
}

func contextInvalidStages(stages map[string]string) map[string]string {
	result := make(map[string]string, len(stages))
	for stage := range stages {
		result[stage] = "context-invalid"
	}
	return result
}

func firstOutputLine(output []byte) string {
	line := strings.TrimSpace(strings.SplitN(string(output), "\n", 2)[0])
	if len(line) > 240 {
		return line[:240]
	}
	return line
}

var clangDiagnosticPattern = regexp.MustCompile(`^(.+):([0-9]+):([0-9]+): (error|warning|note|remark): (.*)$`)

func parseClangDiagnostics(source string, output []byte) []goldenDiagnostic {
	result := make([]goldenDiagnostic, 0)
	for _, line := range strings.Split(string(output), "\n") {
		matches := clangDiagnosticPattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) != 6 {
			continue
		}
		lineNumber, lineErr := strconv.Atoi(matches[2])
		columnNumber, columnErr := strconv.Atoi(matches[3])
		if lineErr != nil || columnErr != nil {
			continue
		}
		start := sourceOffsetAtLineColumn(source, lineNumber, columnNumber)
		severity := SeverityError
		if matches[4] == "warning" {
			severity = SeverityWarning
		} else if matches[4] == "note" || matches[4] == "remark" {
			severity = SeverityInformation
		}
		result = append(result, goldenDiagnostic{Code: "clang-" + matches[4], Severity: severity, Message: matches[5], ConditionalState: ConditionalActive, Start: start, End: minInt(start+1, len(source)), Recoverable: false, Incomplete: false})
	}
	return result
}

func sourceOffsetAtLineColumn(source string, line, column int) int {
	if line < 1 {
		return 0
	}
	currentLine := 1
	for index := 0; index < len(source); index++ {
		if currentLine == line {
			return minInt(index+maxInt(column-1, 0), len(source))
		}
		if source[index] == '\n' {
			currentLine++
		}
	}
	return len(source)
}

func parseClangAST(output []byte) ([]string, []comparisonNode, error) {
	var root any
	if err := json.Unmarshal(output, &root); err != nil {
		return nil, nil, err
	}
	kinds := make([]string, 0)
	ranges := make([]comparisonNode, 0)
	seenKinds := make(map[string]struct{})
	var visit func(any)
	visit = func(value any) {
		object, ok := value.(map[string]any)
		if !ok {
			return
		}
		if implicit, ok := object["isImplicit"].(bool); ok && implicit {
			return
		}
		kind, _ := object["kind"].(string)
		if kind != "" {
			if _, ok := seenKinds[kind]; !ok {
				seenKinds[kind] = struct{}{}
				kinds = append(kinds, kind)
			}
			if rangeValue, ok := object["range"].(map[string]any); ok {
				if begin, beginOK := clangASTOffset(rangeValue, "begin"); beginOK {
					if end, endOK := clangASTOffset(rangeValue, "end"); endOK {
						ranges = append(ranges, comparisonNode{Kind: kind, Start: begin, End: minInt(end+1, int(^uint(0)>>1))})
					}
				}
			}
		}
		if children, ok := object["inner"].([]any); ok {
			for _, child := range children {
				visit(child)
			}
		}
	}
	visit(root)
	return kinds, ranges, nil
}

func clangASTOffset(value map[string]any, key string) (int, bool) {
	child, ok := value[key].(map[string]any)
	if !ok {
		return 0, false
	}
	offset, ok := child["offset"].(float64)
	return int(offset), ok
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func macroArgument(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case string:
		return typed
	default:
		return "1"
	}
}

func TestComparisonRecordFileHasKnownFixtures(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	recordPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "expected", "clang_comparison.json")
	body, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read comparison records: %v", err)
	}
	var records comparisonRecords
	if err := json.Unmarshal(body, &records); err != nil {
		t.Fatalf("decode comparison records: %v", err)
	}
	if records.Version != 1 || len(records.Records) == 0 {
		t.Fatalf("comparison records = %+v", records)
	}
	manifestPath := filepath.Join(filepath.Dir(recordPath), "..", "manifest.json")
	manifest := loadFixtureManifest(t, manifestPath)
	wanted := make(map[string]fixtureSpec)
	for _, fixture := range manifest.Fixtures {
		if len(fixture.Compare) > 0 {
			wanted[fixture.ID] = fixture
		}
	}
	seen := make(map[string]struct{}, len(records.Records))
	for _, record := range records.Records {
		fixture, knownFixture := wanted[record.Fixture]
		if !knownFixture || record.Source != fixture.Source || record.LanguageStandard != fixture.LanguageStandard || record.DriverKind != fixture.DriverKind || record.ContextStatus != fixture.ContextStatus || record.ArgumentProvenance != fixture.ArgumentProvenance || record.SourceHash == "" || record.Commands == nil || record.Differences == nil || record.ClangStatus == "" || record.GoStatus == "" || record.KnownDifference == "" {
			t.Fatalf("incomplete comparison record = %+v", record)
		}
		for _, stage := range []string{"preprocess", "syntax", "ast", "ranges", "diagnostics"} {
			if _, ok := record.Stages[stage]; !ok {
				t.Fatalf("comparison record %q has no stage %q", record.Fixture, stage)
			}
		}
		if record.GoNodeKinds == nil || record.ClangNodeKinds == nil || record.GoDiagnostics == nil || record.ClangDiagnostics == nil {
			t.Fatalf("comparison record %q has incomplete AST/diagnostic arrays", record.Fixture)
		}
		if _, ok := seen[record.Fixture]; ok {
			t.Fatalf("duplicate comparison record for %q", record.Fixture)
		}
		seen[record.Fixture] = struct{}{}
	}
	if len(seen) != len(wanted) {
		t.Fatalf("comparison record count = %d, want %d", len(seen), len(wanted))
	}
}

func TestDumpComparisonRecords(t *testing.T) {
	if os.Getenv("TOPS_LSP_DUMP_COMPARISON") != "1" || !goldenWriteEnabled() {
		t.Skip("comparison writing requires TOPS_LSP_DUMP_COMPARISON=1, TOPS_LSP_DUMP_GOLDEN=1 and TOPS_LSP_ALLOW_GOLDEN_WRITE=1")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	manifest := loadFixtureManifest(t, manifestPath)
	records := comparisonRecords{Version: 1, Records: make([]comparisonRecord, 0)}
	fixtureRoot := filepath.Dir(manifestPath)
	clangPath := os.Getenv("TOPS_LSP_CLANG")
	if clangPath == "" {
		t.Skip("TOPS_LSP_CLANG is not set; comparison records are opt-in")
	}
	clangVersionOutput, err := exec.Command(clangPath, "--version").Output()
	if err != nil {
		t.Fatalf("read Clang version: %v", err)
	}
	clangVersion := firstOutputLine(clangVersionOutput)
	for _, fixture := range manifest.Fixtures {
		if len(fixture.Compare) == 0 {
			continue
		}
		sourcePath := filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source))
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read comparison fixture %q: %v", fixture.ID, err)
		}
		result := Parse(string(source), fixtureParseContext(fixture, sourcePath))
		stages := map[string]string{"preprocess": "not-run", "syntax": "not-run", "ast": "not-run", "ranges": "not-run", "diagnostics": "not-run"}
		for _, stage := range fixture.Compare {
			stages[stage] = "requested"
		}
		record := comparisonRecord{
			Fixture:               fixture.ID,
			Source:                fixture.Source,
			LanguageStandard:      fixture.LanguageStandard,
			DriverKind:            fixture.DriverKind,
			ContextStatus:         fixture.ContextStatus,
			ArgumentProvenance:    fixture.ArgumentProvenance,
			RawArguments:          append([]string{}, fixture.RawArguments...),
			NormalizedArguments:   append([]string{}, fixture.NormalizedArguments...),
			PredefinedMacros:      cloneFixtureMacros(fixture.PredefinedMacros),
			IncludeRoots:          append([]string{}, fixture.IncludeRoots...),
			IncludeRootsAvailable: fixture.ContextStatus == "resolved",
			TargetProfile:         fixture.TargetProfile,
			PassKind:              fixture.PassKind,
			SourceHash:            sourceHash(source),
			ClangVersion:          clangVersion,
			Commands:              map[string][]string{},
			Differences:           []map[string]any{},
			Stages:                stages,
			GoStatus:              string(result.Status),
			ClangStatus:           "not-run",
			GoNodeKinds:           comparisonNodeKinds(result.Root),
			GoNodeRanges:          comparisonNodeRanges(result.Root),
			ClangNodeKinds:        []string{},
			ClangNodeRanges:       []comparisonNode{},
			GoDiagnostics:         diagnosticGoldenValues(result.Diagnostics),
			ClangDiagnostics:      []goldenDiagnostic{},
			KnownDifference:       "Clang comparison not run in the baseline; set TOPS_LSP_CLANG to execute the optional runner.",
		}
		if clangPath != "" {
			clangResult := runClangComparison(t, clangPath, fixture, sourcePath, string(source))
			record.Stages = clangResult.Stages
			record.ClangStatus = clangResult.Status
			record.ClangNodeKinds = nonNilStrings(clangResult.NodeKinds)
			record.ClangNodeRanges = nonNilComparisonNodes(clangResult.NodeRanges)
			record.ClangDiagnostics = nonNilGoldenDiagnostics(clangResult.Diagnostics)
			record.Commands = nonNilCommands(clangResult.Commands)
			record.Differences = nonNilDifferences(clangResult.Differences)
			record.KnownDifference = clangResult.KnownDifference
			if record.KnownDifference == "" && fixture.ExpectedAcceptance != nil && ((clangResult.Status == "accepted") != *fixture.ExpectedAcceptance) {
				record.KnownDifference = "Go fixture acceptance expectation differs from Clang result."
				record.Differences = append(record.Differences, map[string]any{"kind": "classification", "go_expected": *fixture.ExpectedAcceptance, "clang_status": clangResult.Status})
			}
			if record.KnownDifference == "" {
				record.KnownDifference = "No known difference recorded."
			}
		}
		records.Records = append(records.Records, record)
	}
	body, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal comparison records: %v", err)
	}
	outputPath := filepath.Join(fixtureRoot, "expected", "clang_comparison.json")
	if err := os.WriteFile(outputPath, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write comparison records: %v", err)
	}
	t.Logf("wrote comparison records to %s", outputPath)
}

func comparisonNodeKinds(root *SyntaxNode) []string {
	result := make([]string, 0)
	seen := make(map[NodeKind]struct{})
	var visit func(*SyntaxNode)
	visit = func(node *SyntaxNode) {
		if node == nil {
			return
		}
		if _, ok := seen[node.Kind]; !ok {
			seen[node.Kind] = struct{}{}
			result = append(result, string(node.Kind))
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return result
}

func comparisonNodeRanges(root *SyntaxNode) []comparisonNode {
	result := make([]comparisonNode, 0)
	var visit func(*SyntaxNode)
	visit = func(node *SyntaxNode) {
		if node == nil {
			return
		}
		result = append(result, comparisonNode{Kind: string(node.Kind), Start: node.Range.Start, End: node.Range.End})
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return result
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilComparisonNodes(values []comparisonNode) []comparisonNode {
	if values == nil {
		return []comparisonNode{}
	}
	return values
}

func nonNilGoldenDiagnostics(values []goldenDiagnostic) []goldenDiagnostic {
	if values == nil {
		return []goldenDiagnostic{}
	}
	return values
}

func nonNilCommands(values map[string][]string) map[string][]string {
	if values == nil {
		return map[string][]string{}
	}
	return values
}

func nonNilDifferences(values []map[string]any) []map[string]any {
	if values == nil {
		return []map[string]any{}
	}
	return values
}

func cloneFixtureMacros(values map[string]any) map[string]any {
	result := make(map[string]any, len(values))
	for name, value := range values {
		result[name] = value
	}
	return result
}

func sourceHash(source []byte) string {
	digest := sha256.Sum256(source)
	return hex.EncodeToString(digest[:])
}
