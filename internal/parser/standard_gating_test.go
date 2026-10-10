package parser

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLanguageStandardGatesCxxFeatures(t *testing.T) {
	tests := []struct {
		name            string
		standard        string
		source          string
		wantUnsupported bool
		unsupported     string
	}{
		{name: "generic-lambda-cxx11", standard: "c++11", source: "auto apply = [](auto value) { return value; };", wantUnsupported: true, unsupported: "auto"},
		{name: "generic-lambda-cxx14", standard: "c++14", source: "auto apply = [](auto value) { return value; };"},
		{name: "return-deduction-cxx11", standard: "c++11", source: "auto identity(int value) { return value; }", wantUnsupported: true, unsupported: "auto"},
		{name: "return-deduction-cxx14", standard: "c++14", source: "auto identity(int value) { return value; }"},
		{name: "if-constexpr-cxx14", standard: "c++14", source: "int choose(int value) { if constexpr (value) return 1; return 0; }", wantUnsupported: true, unsupported: "constexpr"},
		{name: "if-constexpr-cxx17", standard: "c++17", source: "int choose(int value) { if constexpr (value) return 1; return 0; }"},
		{name: "structured-binding-cxx14", standard: "c++14", source: "int read() { auto [first, second] = pair; return first + second; }", wantUnsupported: true, unsupported: "auto"},
		{name: "structured-binding-cxx17", standard: "c++17", source: "int read() { auto [first, second] = pair; return first + second; }"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := Parse(testCase.source, ParseContext{LanguageStandard: testCase.standard})
			unsupported := diagnosticsWithCode(result.Diagnostics, DiagnosticUnsupported)
			if testCase.wantUnsupported {
				if len(unsupported) != 1 {
					t.Fatalf("unsupported diagnostics = %+v", unsupported)
				}
				if got := testCase.source[unsupported[0].Range.Start:unsupported[0].Range.End]; got != testCase.unsupported {
					t.Fatalf("unsupported range = %+v, source text = %q", unsupported[0].Range, got)
				}
				return
			}
			if len(unsupported) != 0 {
				t.Fatalf("unexpected unsupported diagnostics = %+v", unsupported)
			}
		})
	}
}

func TestStandardFixturesKeepExpectedNodesTokensAndRecovery(t *testing.T) {
	root := standardFixtureRoot(t)
	tests := []struct {
		name       string
		standard   string
		relative   string
		wantNodes  []NodeKind
		wantTokens []string
		incomplete bool
	}{
		{name: "cxx11", standard: "c++11", relative: "positive/standard_cpp11.cpp", wantNodes: []NodeKind{NodeFunction, NodeStatement}, wantTokens: []string{"nullptr", "constexpr"}},
		{name: "cxx14", standard: "c++14", relative: "positive/standard_cpp14.cpp", wantNodes: []NodeKind{NodeFunction, NodeStatement}, wantTokens: []string{"auto"}},
		{name: "cxx17", standard: "c++17", relative: "positive/standard_cpp17.cpp", wantNodes: []NodeKind{NodeFunction, NodeStatement}, wantTokens: []string{"constexpr", "auto"}},
		{name: "cxx14-incomplete", standard: "c++14", relative: "incomplete/standard_cpp14.cpp", wantNodes: []NodeKind{NodeFunction, NodeMissingToken}, wantTokens: []string{"auto"}, incomplete: true},
		{name: "cxx17-incomplete", standard: "c++17", relative: "incomplete/standard_cpp17.cpp", wantNodes: []NodeKind{NodeFunction, NodeStatement, NodeMissingToken}, wantTokens: []string{"constexpr"}, incomplete: true},
		{name: "cxx11-negative", standard: "c++11", relative: "negative/standard_cpp11.cpp", wantNodes: []NodeKind{NodeFunction}, wantTokens: []string{"auto"}, incomplete: false},
		{name: "cxx14-negative", standard: "c++14", relative: "negative/standard_cpp14.cpp", wantNodes: []NodeKind{NodeFunction, NodeStatement}, wantTokens: []string{"auto"}, incomplete: false},
		{name: "cxx17-negative", standard: "c++17", relative: "negative/standard_cpp17.cpp", wantNodes: []NodeKind{NodeTemplate}, wantTokens: []string{"concept"}, incomplete: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			sourceBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(testCase.relative)))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			source := string(sourceBytes)
			result := Parse(source, ParseContext{LanguageStandard: testCase.standard, DocumentVersion: 4, ContextVersion: 2})
			if result.Root == nil || result.ConsumedBytes != len(source) {
				t.Fatalf("result = %+v", result)
			}
			for _, nodeKind := range testCase.wantNodes {
				if findNode(result.Root, nodeKind) == nil {
					t.Fatalf("missing node %q in %+v", nodeKind, result.Root)
				}
			}
			for _, spelling := range testCase.wantTokens {
				if !hasTokenSpelling(result.Tokens, spelling) {
					t.Fatalf("missing token %q", spelling)
				}
			}
			if testCase.incomplete {
				if result.Status != ParseStatusRecovered || !hasDiagnostic(result.Diagnostics, DiagnosticMissingToken) {
					t.Fatalf("incomplete result = %+v", result)
				}
			}
			if !testCase.incomplete && testCase.name == "cxx11-negative" && len(diagnosticsWithCode(result.Diagnostics, DiagnosticUnsupported)) == 0 {
				t.Fatalf("cxx11 negative fixture lacks unsupported diagnostic: %+v", result.Diagnostics)
			}
			if !testCase.incomplete && testCase.name == "cxx14-negative" && len(diagnosticsWithCode(result.Diagnostics, DiagnosticUnsupported)) == 0 {
				t.Fatalf("cxx14 negative fixture lacks unsupported diagnostic: %+v", result.Diagnostics)
			}
			if !testCase.incomplete && testCase.name == "cxx17-negative" && len(diagnosticsWithCode(result.Diagnostics, DiagnosticUnsupported)) == 0 {
				t.Fatalf("cxx17 negative fixture lacks unsupported diagnostic: %+v", result.Diagnostics)
			}
		})
	}
}

func TestStandardUnsupportedDiagnosticPreservesTokenRange(t *testing.T) {
	source := "auto identity(int value) { return value; }"
	result := Parse(source, ParseContext{LanguageStandard: "c++11", DocumentVersion: 7, ContextVersion: 3})
	unsupported := diagnosticsWithCode(result.Diagnostics, DiagnosticUnsupported)
	if len(unsupported) != 1 {
		t.Fatalf("unsupported diagnostics = %+v", result.Diagnostics)
	}
	if unsupported[0].Range != (SourceRange{Start: 0, End: 4}) {
		t.Fatalf("unsupported range = %+v", unsupported[0].Range)
	}
	if unsupported[0].DocumentVersion != 7 || unsupported[0].ContextVersion != 3 {
		t.Fatalf("diagnostic versions = %+v", unsupported[0])
	}
}

func diagnosticsWithCode(diagnostics []ParserDiagnostic, code string) []ParserDiagnostic {
	result := make([]ParserDiagnostic, 0)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			result = append(result, diagnostic)
		}
	}
	return result
}

func standardFixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser")
}
