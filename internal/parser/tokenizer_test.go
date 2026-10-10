package parser

import (
	"strings"
	"testing"
)

func TestTokenizeStandardAndTopsOperators(t *testing.T) {
	result := Tokenize("int main() { return value<<<grid, block>>>(arg); }", ParseContext{})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}

	var spellings []string
	for _, token := range result.Tokens {
		if token.Kind != TokenComment && token.Kind != TokenNewline && token.Kind != TokenEOF {
			spellings = append(spellings, token.Spelling)
		}
	}
	want := []string{"int", "main", "(", ")", "{", "return", "value", "<<<", "grid", ",", "block", ">>>", "(", "arg", ")", ";", "}"}
	if len(spellings) != len(want) {
		t.Fatalf("spellings = %q, want %q", spellings, want)
	}
	for index := range want {
		if spellings[index] != want[index] {
			t.Fatalf("spellings[%d] = %q, want %q", index, spellings[index], want[index])
		}
	}
}

func TestTokenizeUnterminatedStringProducesDiagnostic(t *testing.T) {
	result := Tokenize("const char* value = \"unfinished", ParseContext{DocumentVersion: 4, ContextVersion: 2})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != DiagnosticUnterminated || !diagnostic.Incomplete || !diagnostic.Recoverable {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
	if diagnostic.Range.Start != strings.Index("const char* value = \"unfinished", "\"") || diagnostic.Range.End != len("const char* value = \"unfinished") {
		t.Fatalf("diagnostic range = %+v", diagnostic.Range)
	}
}

func TestTokenizePrefixedAndRawLiteralsAsSingleTokens(t *testing.T) {
	result := Tokenize(`u8"text" L'x' R"tag(raw)tag" u8R"tag(raw)tag" uR"tag(raw)tag" UR"tag(raw)tag" LR"tag(raw)tag"`, ParseContext{})
	var literals []string
	for _, token := range result.Tokens {
		if token.Kind == TokenLiteral {
			literals = append(literals, token.Spelling)
		}
	}
	want := []string{`u8"text"`, `L'x'`, `R"tag(raw)tag"`, `u8R"tag(raw)tag"`, `uR"tag(raw)tag"`, `UR"tag(raw)tag"`, `LR"tag(raw)tag"`}
	if len(literals) != len(want) {
		t.Fatalf("literals = %q, want %q", literals, want)
	}
	for index := range want {
		if literals[index] != want[index] {
			t.Fatalf("literal[%d] = %q, want %q", index, literals[index], want[index])
		}
	}
}

func TestTokenizeInvalidLiteralsProduceDiagnostics(t *testing.T) {
	result := Tokenize(`1e+ 0x 1.2.3 '' "bad\q"`, ParseContext{})
	invalid := 0
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticInvalidLiteral {
			invalid++
		}
	}
	if invalid != 5 {
		t.Fatalf("invalid literal diagnostics = %+v", result.Diagnostics)
	}
}

func TestTokenizeLiteralDiagnosticMetadataMatrix(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		code       string
		incomplete bool
	}{
		{name: "quoted eof", source: `"unfinished`, code: DiagnosticUnterminated, incomplete: true},
		{name: "quoted newline", source: "\"unfinished\n", code: DiagnosticInvalidLiteral, incomplete: false},
		{name: "raw eof", source: `R"tag(unfinished`, code: DiagnosticUnterminated, incomplete: true},
		{name: "raw delimiter", source: `R"bad delimiter(value)bad delimiter"`, code: DiagnosticInvalidLiteral, incomplete: false},
		{name: "number", source: `1e+`, code: DiagnosticInvalidLiteral, incomplete: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := Tokenize(testCase.source, ParseContext{})
			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != testCase.code || result.Diagnostics[0].Incomplete != testCase.incomplete {
				t.Fatalf("source %q diagnostics = %+v", testCase.source, result.Diagnostics)
			}
		})
	}
}

func TestTokenizeStandardVersionKeywords(t *testing.T) {
	result := Tokenize("char16_t char32_t thread_local consteval constinit and_eq", ParseContext{})
	for _, token := range result.Tokens {
		if token.Spelling == "" || token.Kind == TokenEOF {
			continue
		}
		if token.Kind != TokenKeyword {
			t.Fatalf("token %q kind = %q, want keyword", token.Spelling, token.Kind)
		}
	}
}
