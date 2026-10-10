package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"tops-lsp/internal/position"
)

func TestUnterminatedDiagnosticRangeUsesOriginalBytesAndUTF16Mapping(t *testing.T) {
	source := "int 𐐀 = \"unfinished"
	result := Tokenize(source, ParseContext{})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	mapper := position.New(source)
	rangeValue, err := mapper.Range(diagnostic.Range.Start, diagnostic.Range.End)
	if err != nil {
		t.Fatalf("Range() error = %v", err)
	}
	if rangeValue.Start.Character != 9 || rangeValue.End.Character != 20 {
		t.Fatalf("range = %+v", rangeValue)
	}
}

func TestDiagnosticRangeCorpusHasTwentyRecoverableCases(t *testing.T) {
	for _, testCase := range diagnosticRangeCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			result := Parse(testCase.source, testCase.context)
			if len(result.Diagnostics) == 0 {
				t.Fatal("expected at least one diagnostic")
			}
			mapper := position.New(testCase.source)
			lastStart := -1
			seen := make(map[string]struct{})
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Range.Start < lastStart {
					t.Fatalf("diagnostics are not sorted: %+v", result.Diagnostics)
				}
				lastStart = diagnostic.Range.Start
				if _, err := mapper.Range(diagnostic.Range.Start, diagnostic.Range.End); err != nil {
					t.Fatalf("range %+v is not mappable: %v", diagnostic.Range, err)
				}
				key := diagnostic.Code + ":" + string(rune(diagnostic.Range.Start)) + ":" + string(rune(diagnostic.Range.End))
				if _, ok := seen[key]; ok {
					t.Fatalf("duplicate diagnostic: %+v", diagnostic)
				}
				seen[key] = struct{}{}
			}
		})
	}
}

type diagnosticRangeCase struct {
	name    string
	source  string
	context ParseContext
}

type diagnosticRangeGoldenFile struct {
	Version int                         `json:"version"`
	Cases   []diagnosticRangeGoldenCase `json:"cases"`
}

type diagnosticRangeGoldenCase struct {
	Name        string             `json:"name"`
	Diagnostics []goldenDiagnostic `json:"diagnostics"`
}

func diagnosticRangeCases() []diagnosticRangeCase {
	return []diagnosticRangeCase{
		{name: "unterminated-string", source: `int value = "`, context: ParseContext{}},
		{name: "unterminated-comment", source: "/*", context: ParseContext{}},
		{name: "unterminated-raw", source: `R"tag(value`, context: ParseContext{}},
		{name: "missing-semicolon", source: "int value", context: ParseContext{}},
		{name: "missing-function-brace", source: "int main() {", context: ParseContext{}},
		{name: "missing-return-semicolon", source: "int main() { return 1", context: ParseContext{}},
		{name: "missing-close-paren", source: "int main( { return 1; }", context: ParseContext{}},
		{name: "missing-close-bracket", source: "int value[2;", context: ParseContext{}},
		{name: "empty-initializer", source: "int value = ;", context: ParseContext{}},
		{name: "missing-endif", source: "#if FLAG\nint value;\n", context: ParseContext{}},
		{name: "unexpected-endif", source: "#endif\n", context: ParseContext{}},
		{name: "bad-conditional", source: "#if (\n", context: ParseContext{}},
		{name: "empty-tops-attribute", source: "__maxnreg__() void f() {}", context: ParseContext{}},
		{name: "incomplete-launch", source: "kernel<<<grid,", context: ParseContext{}},
		{name: "unknown-character", source: "int value = @;", context: ParseContext{}},
		{name: "unsupported-concept", source: "concept C = true;", context: ParseContext{LanguageStandard: "c++17"}},
		{name: "incomplete-template", source: "template <typename T", context: ParseContext{}},
		{name: "incomplete-attribute", source: "[[deprecated(", context: ParseContext{}},
		{name: "incomplete-parenthesized", source: "int value = (1;", context: ParseContext{}},
		{name: "utf16-crlf", source: "int 𐐀 = \"unfinished\r\n", context: ParseContext{}},
	}
}

func TestDiagnosticRangeGoldensMatchParserOutput(t *testing.T) {
	goldens := loadDiagnosticRangeGoldens(t)
	for _, testCase := range diagnosticRangeCases() {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			want, ok := goldens[testCase.name]
			if !ok {
				t.Fatalf("missing diagnostic golden for %q", testCase.name)
			}
			result := Parse(testCase.source, testCase.context)
			got := diagnosticGoldenValues(result.Diagnostics)
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("diagnostic golden mismatch\nwant=%+v\n got=%+v", want, got)
			}
		})
	}
}

func TestDumpDiagnosticRangeGoldens(t *testing.T) {
	if !goldenWriteEnabled() {
		t.Skip("golden writing requires TOPS_LSP_DUMP_GOLDEN=1 and TOPS_LSP_ALLOW_GOLDEN_WRITE=1")
	}
	goldens := diagnosticRangeGoldenFile{Version: 1, Cases: make([]diagnosticRangeGoldenCase, 0, len(diagnosticRangeCases()))}
	for _, testCase := range diagnosticRangeCases() {
		result := Parse(testCase.source, testCase.context)
		goldens.Cases = append(goldens.Cases, diagnosticRangeGoldenCase{Name: testCase.name, Diagnostics: diagnosticGoldenValues(result.Diagnostics)})
	}
	body, err := json.MarshalIndent(goldens, "", "  ")
	if err != nil {
		t.Fatalf("marshal diagnostic goldens: %v", err)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "expected", "diagnostic_ranges.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatalf("write diagnostic goldens: %v", err)
	}
	t.Logf("wrote diagnostic goldens to %s", path)
}

func loadDiagnosticRangeGoldens(t *testing.T) map[string][]goldenDiagnostic {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "parser", "expected", "diagnostic_ranges.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read diagnostic goldens: %v", err)
	}
	var fileValue diagnosticRangeGoldenFile
	if err := json.Unmarshal(body, &fileValue); err != nil {
		t.Fatalf("decode diagnostic goldens: %v", err)
	}
	if fileValue.Version != 1 {
		t.Fatalf("diagnostic golden version = %d, want 1", fileValue.Version)
	}
	result := make(map[string][]goldenDiagnostic, len(fileValue.Cases))
	for _, golden := range fileValue.Cases {
		if _, exists := result[golden.Name]; exists {
			t.Fatalf("duplicate diagnostic golden %q", golden.Name)
		}
		result[golden.Name] = golden.Diagnostics
	}
	if len(result) != len(diagnosticRangeCases()) {
		t.Fatalf("diagnostic golden count = %d, want %d", len(result), len(diagnosticRangeCases()))
	}
	return result
}

func diagnosticGoldenValues(diagnostics []ParserDiagnostic) []goldenDiagnostic {
	result := make([]goldenDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		golden := goldenDiagnostic{Code: diagnostic.Code, Severity: diagnostic.Severity, ConditionalState: diagnostic.ConditionalState, Start: diagnostic.Range.Start, End: diagnostic.Range.End}
		for _, related := range diagnostic.Related {
			golden.Related = append(golden.Related, goldenRelated{Start: related.Range.Start, End: related.Range.End, Message: related.Message})
		}
		result = append(result, golden)
	}
	return result
}
