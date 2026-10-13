package protocol

import (
	"encoding/json"
	"testing"
)

func TestNavigationParamsRoundTrip(t *testing.T) {
	var position TextDocumentPositionParams
	if err := json.Unmarshal([]byte(`{"textDocument":{"uri":"file:///workspace/main.tops"},"position":{"line":3,"character":5}}`), &position); err != nil {
		t.Fatal(err)
	}
	if position.TextDocument.URI != "file:///workspace/main.tops" || position.Position.Line != 3 || position.Position.Character != 5 {
		t.Fatalf("position = %+v", position)
	}

	var references ReferenceParams
	if err := json.Unmarshal([]byte(`{"textDocument":{"uri":"file:///workspace/main.tops"},"position":{"line":3,"character":5},"context":{"includeDeclaration":true}}`), &references); err != nil {
		t.Fatal(err)
	}
	if !references.Context.IncludeDeclaration {
		t.Fatalf("reference context = %+v", references.Context)
	}
}

func TestDocumentSymbolJSONUsesNestedStandardShape(t *testing.T) {
	symbol := DocumentSymbol{
		Name:           "add",
		Kind:           SymbolKindFunction,
		Range:          Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 3, Character: 1}},
		SelectionRange: Range{Start: Position{Line: 1, Character: 4}, End: Position{Line: 1, Character: 7}},
		Children: []DocumentSymbol{{
			Name:           "value",
			Kind:           SymbolKindVariable,
			Range:          Range{Start: Position{Line: 2}, End: Position{Line: 2, Character: 10}},
			SelectionRange: Range{Start: Position{Line: 2, Character: 4}, End: Position{Line: 2, Character: 9}},
		}},
	}
	body, err := json.Marshal(symbol)
	if err != nil {
		t.Fatal(err)
	}
	var decoded DocumentSymbol
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != symbol.Name || len(decoded.Children) != 1 || decoded.Children[0].Name != "value" {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestInitializeAdvertisesNavigationCapabilities(t *testing.T) {
	result := InitializeResult{Capabilities: ServerCapabilities{
		DefinitionProvider:     true,
		ReferencesProvider:     true,
		DocumentSymbolProvider: true,
	}}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"definitionProvider", "referencesProvider", "documentSymbolProvider"} {
		if !containsJSONField(body, field) {
			t.Fatalf("capability %q missing from %s", field, body)
		}
	}
}

func TestNavigationParamsRequireNestedFields(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		decode func([]byte) error
	}{
		{name: "position line", data: `{"textDocument":{"uri":"file:///workspace/main.tops"},"position":{"character":1}}`, decode: func(data []byte) error {
			var value TextDocumentPositionParams
			return json.Unmarshal(data, &value)
		}},
		{name: "document uri", data: `{"textDocument":{},"position":{"line":0,"character":1}}`, decode: func(data []byte) error {
			var value TextDocumentPositionParams
			return json.Unmarshal(data, &value)
		}},
		{name: "reference context", data: `{"textDocument":{"uri":"file:///workspace/main.tops"},"position":{"line":0,"character":1},"context":{}}`, decode: func(data []byte) error {
			var value ReferenceParams
			return json.Unmarshal(data, &value)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.decode([]byte(test.data)); err == nil {
				t.Fatal("malformed navigation params were accepted")
			}
		})
	}
}

func containsJSONField(body []byte, field string) bool {
	var value map[string]any
	if json.Unmarshal(body, &value) != nil {
		return false
	}
	capabilities, ok := value["capabilities"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = capabilities[field]
	return ok
}
