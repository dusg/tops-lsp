package parser

import "testing"

func TestTokenizeUnterminatedCommentAndRawString(t *testing.T) {
	for _, source := range []string{"/* unfinished", `R"tag(unfinished`} {
		result := Tokenize(source, ParseContext{})
		if !hasDiagnostic(result.Diagnostics, DiagnosticUnterminated) {
			t.Fatalf("source %q diagnostics = %+v", source, result.Diagnostics)
		}
	}
}
