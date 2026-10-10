package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type ccKernelManifest struct {
	Root     string            `json:"root"`
	Fixtures []ccKernelFixture `json:"fixtures"`
}

type ccKernelFixture struct {
	ID            string               `json:"id"`
	Source        string               `json:"source"`
	ContextStatus string               `json:"context_status"`
	Structures    []string             `json:"structures"`
	Expected      *ccKernelExpectation `json:"expected"`
}

type ccKernelExpectation struct {
	Partial         bool       `json:"partial"`
	RequiredNodes   []NodeKind `json:"required_nodes"`
	RequiredTokens  []string   `json:"required_tokens"`
	DiagnosticCodes []string   `json:"diagnostic_codes"`
}

func TestCCKernelManifestSourcesAreReadableAndParseable(t *testing.T) {
	manifest := loadCCKernelManifest(t)
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			sourcePath := filepath.Join(manifest.Root, filepath.FromSlash(fixture.Source))
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			result := Parse(string(source), ParseContext{LanguageID: "tops", LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: fixture.ContextStatus, ArgumentProvenance: "cc-kernel-manifest", TargetProfile: "gcu400", PassKind: "device", DocumentVersion: 1, ContextVersion: 1})
			if result.Root == nil || result.ConsumedBytes != len(source) {
				t.Fatalf("result status=%s consumed=%d source=%d", result.Status, result.ConsumedBytes, len(source))
			}
			if fixture.ContextStatus != "partial" || result.Status == ParseStatusComplete {
				t.Fatalf("context status=%q result status=%q", fixture.ContextStatus, result.Status)
			}
			assertCCKernelStructures(t, string(source), result)
		})
	}
	if len(manifest.Fixtures) < 10 {
		t.Fatalf("corpus fixture count = %d, want at least 10", len(manifest.Fixtures))
	}
	required := map[string]bool{"template": false, "if": false, "global": false, "host": false, "vector": false, "dte": false, "kernel-launch": false, "attribute": false}
	for _, fixture := range manifest.Fixtures {
		for _, structure := range fixture.Structures {
			if _, ok := required[structure]; ok {
				required[structure] = true
			}
		}
	}
	for structure, present := range required {
		if !present {
			t.Errorf("corpus is missing structure %q", structure)
		}
	}
}

func TestCCKernelManifestExpectedStructuresMatchParserOutput(t *testing.T) {
	manifest := loadCCKernelManifest(t)
	for _, fixture := range manifest.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			if fixture.Expected == nil {
				t.Fatal("corpus fixture is missing independent expected data")
			}
			sourcePath := filepath.Join(manifest.Root, filepath.FromSlash(fixture.Source))
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}
			result := Parse(string(source), ParseContext{LanguageID: "tops", LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: fixture.ContextStatus, ArgumentProvenance: "cc-kernel-manifest", TargetProfile: "gcu400", PassKind: "device", DocumentVersion: 1, ContextVersion: 1})
			if fixture.Expected.Partial != (result.Status != ParseStatusComplete) {
				t.Fatalf("partial expectation=%v result status=%s", fixture.Expected.Partial, result.Status)
			}
			for _, nodeKind := range fixture.Expected.RequiredNodes {
				node := findNode(result.Root, nodeKind)
				if node == nil || node.Range.Start < 0 || node.Range.End < node.Range.Start || node.Range.End > len(source) {
					t.Fatalf("expected node %q has no valid parsed range: %+v", nodeKind, node)
				}
			}
			for _, spelling := range fixture.Expected.RequiredTokens {
				found := false
				for _, token := range result.Tokens {
					if token.Spelling != spelling {
						continue
					}
					if token.Range.Start < 0 || token.Range.End < token.Range.Start || token.Range.End > len(source) || string(source[token.Range.Start:token.Range.End]) != token.Spelling {
						t.Fatalf("token %q has invalid source range: %+v", spelling, token)
					}
					found = true
					break
				}
				if !found {
					t.Fatalf("expected token %q is missing", spelling)
				}
			}
			if got := corpusDiagnosticCodes(result.Diagnostics); !sameStrings(got, fixture.Expected.DiagnosticCodes) {
				t.Fatalf("diagnostic codes = %v, want %v", got, fixture.Expected.DiagnosticCodes)
			}
			assertMappedCCKernelStructures(t, sourcePath, string(source), result, fixture.Structures)
		})
	}
}

func loadCCKernelManifest(t *testing.T) ccKernelManifest {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	manifestPath := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "topsop", "cc_kernel_manifest.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest ccKernelManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	root := os.Getenv("TOPS_LSP_CC_KERNEL_ROOT")
	if root == "" {
		t.Skip("TOPS_LSP_CC_KERNEL_ROOT is not set; cc_kernel corpus tests are opt-in")
	}
	if !filepath.IsAbs(root) {
		t.Fatalf("TOPS_LSP_CC_KERNEL_ROOT must be absolute: %q", root)
	}
	manifest.Root = root
	return manifest
}

func assertMappedCCKernelStructures(t *testing.T, sourcePath, source string, result ParseResult, structures []string) {
	t.Helper()
	for _, structure := range structures {
		switch structure {
		case "template":
			if findNode(result.Root, NodeTemplate) == nil {
				t.Fatalf("%s: template structure is not parsed", sourcePath)
			}
		case "namespace":
			if findNode(result.Root, NodeNamespace) == nil && !hasTokenSpelling(result.Tokens, "KERNEL_NAMESPACE_BEGIN") {
				t.Fatalf("%s: namespace structure is not represented", sourcePath)
			}
		case "if":
			if len(result.ConditionalRegions) == 0 {
				t.Fatalf("%s: conditional region is not parsed", sourcePath)
			}
		case "global":
			if !containsQualifier(result.Root, "__global__") {
				t.Fatalf("%s: global qualifier is not parsed", sourcePath)
			}
		case "local":
			if !containsQualifier(result.Root, "__local__") {
				t.Fatalf("%s: local qualifier is not parsed", sourcePath)
			}
		case "valigned":
			if !containsQualifier(result.Root, "__valigned__") {
				t.Fatalf("%s: valigned qualifier is not parsed", sourcePath)
			}
		case "dte":
			if !hasTokenContaining(result.Tokens, "dte") {
				t.Fatalf("%s: DTE token is not parsed", sourcePath)
			}
		case "host":
			if !containsQualifier(result.Root, "__host__") {
				t.Fatalf("%s: host qualifier is not parsed", sourcePath)
			}
		case "switch", "cast":
			if !hasTokenSpelling(result.Tokens, map[string]string{"switch": "switch", "cast": "reinterpret_cast"}[structure]) {
				t.Fatalf("%s: %s token is not parsed", sourcePath, structure)
			}
		case "kernel-launch":
			if findNode(result.Root, NodeTopsLaunch) == nil {
				t.Fatalf("%s: kernel launch is not parsed", sourcePath)
			}
		case "vector", "bf16":
			if !hasTokenSpelling(result.Tokens, map[string]string{"vector": "__vector", "bf16": "__bf16"}[structure]) {
				t.Fatalf("%s: %s token is not parsed", sourcePath, structure)
			}
		case "device", "forceinline", "restrict":
			spelling := map[string]string{"device": "__device__", "forceinline": "__forceinline__", "restrict": "__restrict__"}[structure]
			if !containsQualifier(result.Root, spelling) {
				t.Fatalf("%s: %s qualifier is not parsed", sourcePath, structure)
			}
		case "attribute":
			if findNode(result.Root, NodeAttribute) == nil {
				t.Fatalf("%s: attribute node is not parsed", sourcePath)
			}
		case "thread-dims":
			if !containsQualifier(result.Root, "__thread_dims__") {
				t.Fatalf("%s: thread dims attribute is not parsed", sourcePath)
			}
		case "corpus":
			if findNode(result.Root, NodeDeclaration) == nil && findNode(result.Root, NodeFunction) == nil && findNode(result.Root, NodeTemplate) == nil {
				t.Fatalf("%s: corpus declaration is not parsed", sourcePath)
			}
		case "":
		default:
			if !strings.Contains(source, structure) {
				t.Fatalf("%s: unsupported manifest structure %q is absent from source", sourcePath, structure)
			}
		}
	}
}

func corpusDiagnosticCodes(diagnostics []ParserDiagnostic) []string {
	result := make([]string, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, ok := seen[diagnostic.Code]; ok {
			continue
		}
		seen[diagnostic.Code] = struct{}{}
		result = append(result, diagnostic.Code)
	}
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func assertCCKernelStructures(t *testing.T, source string, result ParseResult) {
	t.Helper()
	if findNode(result.Root, NodeDeclaration) == nil && findNode(result.Root, NodeFunction) == nil && findNode(result.Root, NodeTemplate) == nil {
		t.Fatalf("no declaration-like AST node in corpus result")
	}
	if strings.Contains(source, "#if") && len(result.ConditionalRegions) == 0 {
		t.Fatalf("source contains conditional directive but result has no regions")
	}
	if strings.Contains(source, "template") && findNode(result.Root, NodeTemplate) == nil {
		t.Fatalf("source contains template but template node is missing")
	}
	if strings.Contains(source, "<<<") && findNode(result.Root, NodeTopsLaunch) == nil {
		t.Fatalf("source contains kernel launch but launch node is missing")
	}
	if strings.Contains(source, "[[") && findNode(result.Root, NodeAttribute) == nil {
		t.Fatalf("source contains standard attribute but attribute node is missing")
	}
	if strings.Contains(source, "__attribute__") && findNode(result.Root, NodeAttribute) == nil {
		t.Fatalf("source contains GNU attribute but attribute node is missing")
	}
	for _, qualifier := range []string{"__global__", "__host__", "__device__", "__local__", "__valigned__", "__vector", "__vector2", "__vector4", "__vector8"} {
		if strings.Contains(source, qualifier) && !containsQualifier(result.Root, qualifier) {
			t.Fatalf("source contains %q but qualifier node is missing", qualifier)
		}
	}
	if strings.Contains(source, "__thread_dims__") && !containsQualifier(result.Root, "__thread_dims__") {
		t.Fatalf("source contains thread dims attribute but qualifier node is missing")
	}
	for _, spelling := range []string{"switch", "static_cast", "__thread_dims__"} {
		if strings.Contains(source, spelling) && !hasTokenSpelling(result.Tokens, spelling) {
			t.Fatalf("source contains %q but token is missing", spelling)
		}
	}
	if hasDTEIdentifier(source) && !hasTokenContaining(result.Tokens, "dte") {
		t.Fatalf("source contains a DTE identifier but no DTE token is present")
	}
	lastStart := -1
	seenDiagnostics := make(map[string]struct{}, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "" || diagnostic.Range.Start < lastStart || diagnostic.Range.Start < 0 || diagnostic.Range.End < diagnostic.Range.Start || diagnostic.Range.End > len(source) {
			t.Fatalf("invalid corpus diagnostic = %+v", diagnostic)
		}
		key := diagnostic.Code + ":" + strconv.Itoa(diagnostic.Range.Start) + ":" + strconv.Itoa(diagnostic.Range.End)
		if _, exists := seenDiagnostics[key]; exists {
			t.Fatalf("duplicate corpus diagnostic = %+v", diagnostic)
		}
		seenDiagnostics[key] = struct{}{}
		lastStart = diagnostic.Range.Start
	}
}

func containsQualifier(node *SyntaxNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Kind == NodeTopsQualifier && node.Name == name {
		return true
	}
	for _, child := range node.Children {
		if containsQualifier(child, name) {
			return true
		}
	}
	return false
}

func hasTokenSpelling(tokens []Token, spelling string) bool {
	for _, token := range tokens {
		if token.Spelling == spelling {
			return true
		}
	}
	return false
}

func hasDTEIdentifier(source string) bool {
	return strings.Contains(source, "tops_dte_") || strings.Contains(source, "private_dte") || strings.Contains(source, "cdte_") || strings.Contains(source, "sdte_")
}

func hasTokenContaining(tokens []Token, fragment string) bool {
	for _, token := range tokens {
		if strings.Contains(token.Spelling, fragment) {
			return true
		}
	}
	return false
}
