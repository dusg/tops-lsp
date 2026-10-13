package index

import (
	"testing"

	"tops-lsp/internal/parser"
)

func TestQueryReturnsDefinitionReferencesAndDocumentSymbols(t *testing.T) {
	source := `struct Point { int x; }; int add(int left, int right) { return left + right; } int use() { Point point; return add(point.x, 2); }`
	context := resolvedParseContext("file:///workspace/main.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)

	callOffset := sourceOffset(source, "add", 1)
	definitions := snapshot.DefinitionAt(callOffset)
	if len(definitions) != 1 || definitions[0].Role != SymbolLocationDefinition {
		t.Fatalf("definitions = %+v", definitions)
	}
	if got := source[definitions[0].SelectionRange.Start:definitions[0].SelectionRange.End]; got != "add" {
		t.Fatalf("definition selection = %q", got)
	}

	references := snapshot.ReferencesAt(callOffset, true)
	if len(references) < 2 {
		t.Fatalf("references = %+v", references)
	}
	withoutDeclaration := snapshot.ReferencesAt(callOffset, false)
	if len(withoutDeclaration) >= len(references) {
		t.Fatalf("references without declaration = %+v, all = %+v", withoutDeclaration, references)
	}

	symbols := snapshot.DocumentSymbols()
	if !containsDocumentSymbol(symbols, "Point") || !containsDocumentSymbol(symbols, "add") || !containsDocumentSymbol(symbols, "use") {
		t.Fatalf("document symbols = %+v", symbols)
	}
}

func TestQueryReturnsEmptyForUnknownName(t *testing.T) {
	source := "int use() { return missing_name; }"
	context := resolvedParseContext("file:///workspace/main.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)
	if definitions := snapshot.DefinitionAt(sourceOffset(source, "missing_name", 0)); len(definitions) != 0 {
		t.Fatalf("definitions = %+v", definitions)
	}
}

func containsDocumentSymbol(symbols []DocumentSymbol, name string) bool {
	for _, symbol := range symbols {
		if symbol.Name == name || containsDocumentSymbol(symbol.Children, name) {
			return true
		}
	}
	return false
}
