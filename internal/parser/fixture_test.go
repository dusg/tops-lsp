package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

type fixtureManifest struct {
	Version  int           `json:"version"`
	Fixtures []fixtureSpec `json:"fixtures"`
}

type fixtureSpec struct {
	ID                  string              `json:"id"`
	Source              string              `json:"source"`
	LanguageStandard    string              `json:"language_standard"`
	DriverKind          string              `json:"driver_kind"`
	ContextStatus       string              `json:"compiler_context_status"`
	ArgumentProvenance  string              `json:"argument_provenance"`
	RawArguments        []string            `json:"raw_arguments"`
	NormalizedArguments []string            `json:"normalized_arguments"`
	PredefinedMacros    map[string]any      `json:"predefined_macros"`
	IncludeRoots        []string            `json:"include_roots"`
	TargetProfile       string              `json:"target_profile"`
	PassKind            string              `json:"pass_kind"`
	ExpectedDiagnostics []string            `json:"expected_diagnostics"`
	ExpectedAcceptance  *bool               `json:"expected_acceptance"`
	ExpectedNodes       []string            `json:"expected_nodes"`
	Expected            *fixtureExpectation `json:"expected"`
	Compare             []string            `json:"compare"`
}

type fixtureExpectation struct {
	Status            ParseStatus                    `json:"status"`
	RequiredNodes     []NodeKind                     `json:"required_nodes"`
	RequiredTokens    []string                       `json:"required_tokens"`
	Diagnostics       []fixtureDiagnosticExpectation `json:"diagnostics"`
	ConditionalStates []ConditionalState             `json:"conditional_states"`
	Context           fixtureContextExpectation      `json:"context"`
}

type fixtureDiagnosticExpectation struct {
	Code             string              `json:"code"`
	Severity         *DiagnosticSeverity `json:"severity,omitempty"`
	ConditionalState ConditionalState    `json:"conditional_state,omitempty"`
	Start            int                 `json:"start"`
	End              int                 `json:"end"`
	Recoverable      *bool               `json:"recoverable,omitempty"`
	Incomplete       *bool               `json:"incomplete,omitempty"`
	DocumentVersion  *int                `json:"document_version,omitempty"`
	ContextVersion   *int                `json:"context_version,omitempty"`
}

type fixtureContextExpectation struct {
	LanguageStandard   string `json:"language_standard"`
	DriverKind         string `json:"driver_kind"`
	ContextStatus      string `json:"context_status"`
	ArgumentProvenance string `json:"argument_provenance"`
	TargetProfile      string `json:"target_profile"`
	PassKind           string `json:"pass_kind"`
	DocumentVersion    int    `json:"document_version"`
	ContextVersion     int    `json:"context_version"`
}

type fixtureGoldenFile struct {
	Version  int             `json:"version"`
	Fixtures []fixtureGolden `json:"fixtures"`
}

type fixtureGolden struct {
	ID                 string                    `json:"id"`
	Status             ParseStatus               `json:"status"`
	Context            goldenContext             `json:"context"`
	SourceHash         string                    `json:"source_hash"`
	ContextHash        string                    `json:"context_hash"`
	ConsumedBytes      int                       `json:"consumed_bytes"`
	Tokens             []goldenToken             `json:"tokens"`
	Nodes              []goldenNode              `json:"nodes"`
	ConditionalRegions []goldenConditionalRegion `json:"conditional_regions"`
	Diagnostics        []goldenDiagnostic        `json:"diagnostics"`
}

type goldenToken struct {
	Kind             TokenKind        `json:"kind"`
	Spelling         string           `json:"spelling"`
	Start            int              `json:"start"`
	End              int              `json:"end"`
	ConditionalState ConditionalState `json:"conditional_state"`
	Unterminated     bool             `json:"unterminated"`
	LineStart        bool             `json:"line_start"`
}

type goldenNode struct {
	Kind             NodeKind         `json:"kind"`
	Name             string           `json:"name,omitempty"`
	Value            string           `json:"value,omitempty"`
	Callee           string           `json:"callee"`
	ConfigTokens     []goldenToken    `json:"config_tokens"`
	ArgumentTokens   []goldenToken    `json:"argument_tokens"`
	ConditionalState ConditionalState `json:"conditional_state"`
	Start            int              `json:"start"`
	End              int              `json:"end"`
	Incomplete       bool             `json:"incomplete,omitempty"`
	Depth            int              `json:"depth"`
	Path             []int            `json:"path"`
}

type goldenDiagnostic struct {
	Code             string             `json:"code"`
	Severity         DiagnosticSeverity `json:"severity"`
	Message          string             `json:"message"`
	ConditionalState ConditionalState   `json:"conditional_state"`
	Start            int                `json:"start"`
	End              int                `json:"end"`
	Recoverable      bool               `json:"recoverable"`
	Incomplete       bool               `json:"incomplete"`
	DocumentVersion  int                `json:"document_version"`
	ContextVersion   int                `json:"context_version"`
	Related          []goldenRelated    `json:"related,omitempty"`
}

type goldenRelated struct {
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Message string `json:"message"`
}

type goldenContext struct {
	LanguageStandard      string `json:"language_standard"`
	DriverKind            string `json:"driver_kind"`
	ContextStatus         string `json:"context_status"`
	ArgumentProvenance    string `json:"argument_provenance"`
	TargetProfile         string `json:"target_profile"`
	PassKind              string `json:"pass_kind"`
	IncludeRootsAvailable bool   `json:"include_roots_available"`
	DocumentVersion       int    `json:"document_version"`
	ContextVersion        int    `json:"context_version"`
}

type goldenConditionalRegion struct {
	ConditionStart       int                       `json:"condition_start"`
	ConditionEnd         int                       `json:"condition_end"`
	State                ConditionalState          `json:"state"`
	ParentID             int                       `json:"parent_id"`
	Branches             []goldenConditionalBranch `json:"branches"`
	DirectiveDiagnostics []goldenDiagnostic        `json:"directive_diagnostics"`
}

type goldenConditionalBranch struct {
	DirectiveStart int              `json:"directive_start"`
	DirectiveEnd   int              `json:"directive_end"`
	BodyStart      int              `json:"body_start"`
	BodyEnd        int              `json:"body_end"`
	State          ConditionalState `json:"state"`
	Condition      []goldenToken    `json:"condition"`
}

func TestFixtureManifestLoadsFromRepository(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.Version != 1 {
		t.Fatalf("manifest version = %d, want 1", manifest.Version)
	}
}

func TestFixtureManifestSourcesParse(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	fixtureRoot := filepath.Dir(manifestPath)
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			sourcePath := filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source))
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			result := Parse(string(source), ParseContext{DocumentURI: sourcePath, LanguageID: "tops", LanguageStandard: fixture.LanguageStandard, DriverKind: fixture.DriverKind, CompilerContextStatus: fixture.ContextStatus, ArgumentProvenance: fixture.ArgumentProvenance, PredefinedMacros: fixtureMacroValues(fixture.PredefinedMacros), TargetProfile: fixture.TargetProfile, PassKind: fixture.PassKind, IncludeRootsAvailable: fixture.ContextStatus == "resolved", DocumentVersion: 1, ContextVersion: 1})
			if result.Root == nil || result.ConsumedBytes != len(source) {
				t.Fatalf("result status=%s consumed=%d source=%d", result.Status, result.ConsumedBytes, len(source))
			}
		})
	}
}

func TestFixtureManifestExpectationsAreConsistent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	fixtureRoot := filepath.Dir(manifestPath)
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.DriverKind == "" || fixture.ContextStatus == "" || fixture.ArgumentProvenance == "" || fixture.RawArguments == nil || fixture.NormalizedArguments == nil {
				t.Fatal("fixture is missing context provenance")
			}
			if len(fixture.Compare) > 0 && fixture.ExpectedAcceptance == nil {
				t.Fatal("comparison fixture is missing expected_acceptance")
			}
			source, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source)))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			context := fixtureParseContext(fixture, fixture.Source)
			result := Parse(string(source), context)
			if fixture.Expected == nil {
				t.Fatal("fixture is missing independent expected data")
			}
			if fixture.Expected.Status != result.Status {
				t.Fatalf("expected status=%s, got=%s", fixture.Expected.Status, result.Status)
			}
			for _, nodeKind := range fixture.Expected.RequiredNodes {
				if !containsNodeKind(result.Root, nodeKind) {
					t.Fatalf("expected node %q is missing", nodeKind)
				}
			}
			for _, spelling := range fixture.Expected.RequiredTokens {
				if !hasTokenSpelling(result.Tokens, spelling) {
					t.Fatalf("expected token %q is missing", spelling)
				}
			}
			for _, expectedDiagnostic := range fixture.Expected.Diagnostics {
				if !hasDiagnosticAt(result.Diagnostics, expectedDiagnostic.Code, expectedDiagnostic.Start, expectedDiagnostic.End) {
					t.Fatalf("expected diagnostic %+v is missing from %+v", expectedDiagnostic, result.Diagnostics)
				}
			}
			for _, expectedState := range fixture.Expected.ConditionalStates {
				if !hasConditionalRegionState(result.ConditionalRegions, expectedState) {
					t.Fatalf("expected conditional state %q is missing", expectedState)
				}
			}
			if fixture.Expected.Context.LanguageStandard != context.LanguageStandard || fixture.Expected.Context.DriverKind != context.DriverKind || fixture.Expected.Context.ContextStatus != context.CompilerContextStatus || fixture.Expected.Context.ArgumentProvenance != context.ArgumentProvenance || fixture.Expected.Context.TargetProfile != context.TargetProfile || fixture.Expected.Context.PassKind != context.PassKind || fixture.Expected.Context.DocumentVersion != context.DocumentVersion || fixture.Expected.Context.ContextVersion != context.ContextVersion {
				t.Fatalf("expected context=%+v, got=%+v", fixture.Expected.Context, context)
			}
			for _, expectedCode := range fixture.ExpectedDiagnostics {
				if !hasDiagnostic(result.Diagnostics, expectedCode) {
					t.Fatalf("missing expected diagnostic %q in %+v", expectedCode, result.Diagnostics)
				}
			}
			for _, expectedNode := range fixture.ExpectedNodes {
				if !containsNodeKind(result.Root, NodeKind(expectedNode)) {
					t.Fatalf("missing expected node %q", expectedNode)
				}
			}
		})
	}
}

func TestFixtureManifestCoversAllParserFixtures(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	fixtureRoot := filepath.Dir(manifestPath)
	manifestSources := make(map[string]string, len(manifest.Fixtures))
	manifestIDs := make(map[string]struct{}, len(manifest.Fixtures))
	for _, fixture := range manifest.Fixtures {
		if fixture.ID == "" {
			t.Fatal("fixture has an empty id")
		}
		if _, exists := manifestIDs[fixture.ID]; exists {
			t.Fatalf("duplicate manifest id %q", fixture.ID)
		}
		manifestIDs[fixture.ID] = struct{}{}
		cleanSource := filepath.Clean(filepath.FromSlash(fixture.Source))
		if filepath.IsAbs(cleanSource) || cleanSource == ".." || strings.HasPrefix(cleanSource, ".."+string(filepath.Separator)) {
			t.Fatalf("fixture %q has non-repository source path %q", fixture.ID, fixture.Source)
		}
		if _, exists := manifestSources[fixture.Source]; exists {
			t.Fatalf("duplicate manifest source %q", fixture.Source)
		}
		manifestSources[fixture.Source] = fixture.ID
	}

	discovered := make(map[string]struct{})
	for _, directory := range []string{"positive", "negative", "incomplete", "macros"} {
		root := filepath.Join(fixtureRoot, directory)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			extension := filepath.Ext(entry.Name())
			if extension != ".cpp" && extension != ".tops" && extension != ".h" {
				return nil
			}
			relative, err := filepath.Rel(fixtureRoot, path)
			if err != nil {
				return err
			}
			discovered[filepath.ToSlash(relative)] = struct{}{}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", directory, err)
		}
	}
	for source := range discovered {
		if _, ok := manifestSources[source]; !ok {
			t.Errorf("fixture %q is not registered in manifest", source)
		}
	}
	for source, id := range manifestSources {
		if _, ok := discovered[source]; !ok {
			t.Errorf("manifest fixture %q (%s) does not exist in parser fixture directories", source, id)
		}
	}
}

func TestFixtureGoldensMatchParserOutput(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	fixtureRoot := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser")
	manifest := loadFixtureManifest(t, filepath.Join(fixtureRoot, "manifest.json"))
	goldenPath := filepath.Join(fixtureRoot, "expected", "parser_goldens.json")
	body, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read parser goldens: %v", err)
	}
	var goldens fixtureGoldenFile
	if err := json.Unmarshal(body, &goldens); err != nil {
		t.Fatalf("decode parser goldens: %v", err)
	}
	if goldens.Version != 1 {
		t.Fatalf("parser golden version = %d, want 1", goldens.Version)
	}
	expected := make(map[string]fixtureGolden, len(goldens.Fixtures))
	for _, golden := range goldens.Fixtures {
		if _, exists := expected[golden.ID]; exists {
			t.Fatalf("duplicate parser golden %q", golden.ID)
		}
		expected[golden.ID] = golden
	}
	if len(expected) != len(manifest.Fixtures) {
		t.Fatalf("golden fixture count = %d, manifest count = %d", len(expected), len(manifest.Fixtures))
	}
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			golden, ok := expected[fixture.ID]
			if !ok {
				t.Fatalf("missing parser golden for %q", fixture.ID)
			}
			source, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source)))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			actual := snapshotFixture(fixture.ID, string(source), fixtureParseContext(fixture, filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source))))
			if !reflect.DeepEqual(golden, actual) {
				t.Fatalf("fixture golden mismatch\nwant=%+v\n got=%+v", golden, actual)
			}
		})
	}
}

func TestDumpFixtureGoldens(t *testing.T) {
	if !goldenWriteEnabled() {
		t.Skip("golden writing requires TOPS_LSP_DUMP_GOLDEN=1 and TOPS_LSP_ALLOW_GOLDEN_WRITE=1")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	fixtureRoot := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser")
	manifest := loadFixtureManifest(t, filepath.Join(fixtureRoot, "manifest.json"))
	goldens := fixtureGoldenFile{Version: 1, Fixtures: make([]fixtureGolden, 0, len(manifest.Fixtures))}
	for _, fixture := range manifest.Fixtures {
		sourcePath := filepath.Join(fixtureRoot, filepath.FromSlash(fixture.Source))
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		goldens.Fixtures = append(goldens.Fixtures, snapshotFixture(fixture.ID, string(source), fixtureParseContext(fixture, sourcePath)))
	}
	body, err := json.MarshalIndent(goldens, "", "  ")
	if err != nil {
		t.Fatalf("marshal parser goldens: %v", err)
	}
	goldenPath := filepath.Join(fixtureRoot, "expected", "parser_goldens.json")
	if err := os.WriteFile(goldenPath, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write parser goldens: %v", err)
	}
	t.Logf("wrote parser goldens to %s", goldenPath)
}

func goldenWriteEnabled() bool {
	return os.Getenv("TOPS_LSP_DUMP_GOLDEN") == "1" && os.Getenv("TOPS_LSP_ALLOW_GOLDEN_WRITE") == "1"
}

func loadFixtureManifest(t *testing.T, path string) fixtureManifest {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest fixtureManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	return manifest
}

func fixtureParseContext(fixture fixtureSpec, documentURI string) ParseContext {
	return ParseContext{DocumentURI: documentURI, LanguageID: "tops", LanguageStandard: fixture.LanguageStandard, DriverKind: fixture.DriverKind, CompilerContextStatus: fixture.ContextStatus, ArgumentProvenance: fixture.ArgumentProvenance, PredefinedMacros: fixtureMacroValues(fixture.PredefinedMacros), TargetProfile: fixture.TargetProfile, PassKind: fixture.PassKind, IncludeRootsAvailable: fixture.ContextStatus == "resolved", DocumentVersion: 1, ContextVersion: 1}
}

func snapshotFixture(id, source string, context ParseContext) fixtureGolden {
	result := Parse(source, context)
	golden := fixtureGolden{ID: id, Status: result.Status, Context: goldenContext{LanguageStandard: context.LanguageStandard, DriverKind: context.DriverKind, ContextStatus: context.CompilerContextStatus, ArgumentProvenance: context.ArgumentProvenance, TargetProfile: context.TargetProfile, PassKind: context.PassKind, IncludeRootsAvailable: context.IncludeRootsAvailable, DocumentVersion: context.DocumentVersion, ContextVersion: context.ContextVersion}, SourceHash: sourceHash([]byte(source)), ContextHash: fixtureContextHash(context), ConsumedBytes: result.ConsumedBytes, Tokens: make([]goldenToken, 0, len(result.Tokens)), Nodes: make([]goldenNode, 0), ConditionalRegions: make([]goldenConditionalRegion, 0, len(result.ConditionalRegions)), Diagnostics: make([]goldenDiagnostic, 0, len(result.Diagnostics))}
	for _, token := range result.Tokens {
		golden.Tokens = append(golden.Tokens, snapshotGoldenToken(token))
	}
	appendGoldenNodes(&golden.Nodes, result.Root, nil)
	for _, region := range result.ConditionalRegions {
		goldenRegion := goldenConditionalRegion{ConditionStart: region.ConditionRange.Start, ConditionEnd: region.ConditionRange.End, State: region.State, ParentID: region.ParentID, Branches: make([]goldenConditionalBranch, 0, len(region.Branches)), DirectiveDiagnostics: diagnosticGoldenValues(region.DirectiveDiagnostics)}
		for _, branch := range region.Branches {
			goldenBranch := goldenConditionalBranch{DirectiveStart: branch.DirectiveRange.Start, DirectiveEnd: branch.DirectiveRange.End, BodyStart: branch.BodyRange.Start, BodyEnd: branch.BodyRange.End, State: branch.State, Condition: make([]goldenToken, 0, len(branch.Condition))}
			for _, token := range branch.Condition {
				goldenBranch.Condition = append(goldenBranch.Condition, snapshotGoldenToken(token))
			}
			goldenRegion.Branches = append(goldenRegion.Branches, goldenBranch)
		}
		golden.ConditionalRegions = append(golden.ConditionalRegions, goldenRegion)
	}
	for _, diagnostic := range result.Diagnostics {
		goldenDiagnosticValue := goldenDiagnostic{Code: diagnostic.Code, Severity: diagnostic.Severity, Message: diagnostic.Message, ConditionalState: diagnostic.ConditionalState, Start: diagnostic.Range.Start, End: diagnostic.Range.End, Recoverable: diagnostic.Recoverable, Incomplete: diagnostic.Incomplete, DocumentVersion: diagnostic.DocumentVersion, ContextVersion: diagnostic.ContextVersion}
		for _, related := range diagnostic.Related {
			goldenDiagnosticValue.Related = append(goldenDiagnosticValue.Related, goldenRelated{Start: related.Range.Start, End: related.Range.End, Message: related.Message})
		}
		golden.Diagnostics = append(golden.Diagnostics, goldenDiagnosticValue)
	}
	return golden
}

func fixtureContextHash(context ParseContext) string {
	value := struct {
		LanguageID            string
		LanguageStandard      string
		DriverKind            string
		CompilerContextStatus string
		ArgumentProvenance    string
		PredefinedMacros      map[string]MacroValue
		TargetProfile         string
		PassKind              string
		IncludeRootsAvailable bool
		DocumentVersion       int
		ContextVersion        int
	}{
		LanguageID:            context.LanguageID,
		LanguageStandard:      context.LanguageStandard,
		DriverKind:            context.DriverKind,
		CompilerContextStatus: context.CompilerContextStatus,
		ArgumentProvenance:    context.ArgumentProvenance,
		PredefinedMacros:      context.PredefinedMacros,
		TargetProfile:         context.TargetProfile,
		PassKind:              context.PassKind,
		IncludeRootsAvailable: context.IncludeRootsAvailable,
		DocumentVersion:       context.DocumentVersion,
		ContextVersion:        context.ContextVersion,
	}
	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return sourceHash(body)
}

func appendGoldenNodes(nodes *[]goldenNode, node *SyntaxNode, path []int) {
	if node == nil {
		return
	}
	goldenNodeValue := goldenNode{Kind: node.Kind, Name: node.Name, Value: node.Value, Callee: node.Callee, ConfigTokens: make([]goldenToken, 0, len(node.ConfigTokens)), ArgumentTokens: make([]goldenToken, 0, len(node.ArgumentTokens)), ConditionalState: node.ConditionalState, Start: node.Range.Start, End: node.Range.End, Incomplete: node.Incomplete, Depth: len(path), Path: append([]int(nil), path...)}
	for _, token := range node.ConfigTokens {
		goldenNodeValue.ConfigTokens = append(goldenNodeValue.ConfigTokens, snapshotGoldenToken(token))
	}
	for _, token := range node.ArgumentTokens {
		goldenNodeValue.ArgumentTokens = append(goldenNodeValue.ArgumentTokens, snapshotGoldenToken(token))
	}
	*nodes = append(*nodes, goldenNodeValue)
	for index, child := range node.Children {
		appendGoldenNodes(nodes, child, append(append([]int(nil), path...), index))
	}
}

func snapshotGoldenToken(token Token) goldenToken {
	return goldenToken{Kind: token.Kind, Spelling: token.Spelling, Start: token.Range.Start, End: token.Range.End, ConditionalState: token.ConditionalState, Unterminated: token.Unterminated, LineStart: token.LineStart}
}

func hasDiagnosticAt(diagnostics []ParserDiagnostic, code string, start, end int) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code && diagnostic.Range.Start == start && diagnostic.Range.End == end {
			return true
		}
	}
	return false
}

func hasConditionalRegionState(regions []ConditionalRegion, state ConditionalState) bool {
	for _, region := range regions {
		if region.State == state {
			return true
		}
		for _, branch := range region.Branches {
			if branch.State == state {
				return true
			}
		}
	}
	return false
}

func containsNodeKind(node *SyntaxNode, kind NodeKind) bool {
	if node == nil {
		return false
	}
	if node.Kind == kind {
		return true
	}
	for _, child := range node.Children {
		if containsNodeKind(child, kind) {
			return true
		}
	}
	return false
}

func fixtureMacroValues(values map[string]any) map[string]MacroValue {
	result := make(map[string]MacroValue, len(values))
	for name, raw := range values {
		switch value := raw.(type) {
		case float64:
			result[name] = MacroValue{Kind: MacroInteger, Value: int64(value)}
		case string:
			result[name] = MacroValue{Kind: MacroDefined}
		default:
			result[name] = MacroValue{Kind: MacroUnknown}
		}
	}
	return result
}
