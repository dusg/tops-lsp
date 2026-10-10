package parser

import (
	"testing"

	"tops-lsp/internal/position"
)

func TestDiagnosticsAreSortedAndDeduplicated(t *testing.T) {
	result := Parse("int value = ;", ParseContext{})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if result.Diagnostics[0].Range.Start >= result.Diagnostics[0].Range.End {
		t.Fatalf("unexpected diagnostic range = %+v", result.Diagnostics[0].Range)
	}
}

func TestParserDiagnosticRangeMapsToLSPPosition(t *testing.T) {
	source := "int 𐐀 = \"unfinished"
	result := Tokenize(source, ParseContext{})
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	rangeValue, err := position.New(source).Range(result.Diagnostics[0].Range.Start, result.Diagnostics[0].Range.End)
	if err != nil {
		t.Fatalf("Range() error = %v", err)
	}
	if rangeValue.Start.Character != 9 || rangeValue.End.Character != 20 {
		t.Fatalf("range = %+v", rangeValue)
	}
}
