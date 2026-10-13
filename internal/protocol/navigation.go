package protocol

import (
	"encoding/json"
	"fmt"
)

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type SymbolKind uint32

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           SymbolKind       `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

func (params *TextDocumentPositionParams) UnmarshalJSON(data []byte) error {
	type alias TextDocumentPositionParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return err
	}
	if err := validateTextDocumentIdentifier(fields["textDocument"]); err != nil {
		return err
	}
	if _, err := requireField(fields, "position"); err != nil {
		return err
	}
	if err := validatePosition(fields["position"]); err != nil {
		return err
	}
	*params = TextDocumentPositionParams(value)
	return nil
}

func (params *ReferenceParams) UnmarshalJSON(data []byte) error {
	type alias ReferenceParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	for _, field := range []string{"textDocument", "position", "context"} {
		if _, err := requireField(fields, field); err != nil {
			return fmt.Errorf("reference params: %w", err)
		}
	}
	if err := validateTextDocumentIdentifier(fields["textDocument"]); err != nil {
		return fmt.Errorf("reference params: %w", err)
	}
	if err := validatePosition(fields["position"]); err != nil {
		return fmt.Errorf("reference params: %w", err)
	}
	if err := validateReferenceContext(fields["context"]); err != nil {
		return fmt.Errorf("reference params: %w", err)
	}
	*params = ReferenceParams(value)
	return nil
}

func (params *DocumentSymbolParams) UnmarshalJSON(data []byte) error {
	type alias DocumentSymbolParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return err
	}
	if err := validateTextDocumentIdentifier(fields["textDocument"]); err != nil {
		return err
	}
	*params = DocumentSymbolParams(value)
	return nil
}

func validateTextDocumentIdentifier(data []byte) error {
	var value TextDocumentIdentifier
	fields, err := decodeObject(data, &value)
	if err != nil {
		return fmt.Errorf("textDocument: %w", err)
	}
	uri, err := requireField(fields, "uri")
	if err != nil {
		return fmt.Errorf("textDocument: %w", err)
	}
	if err := requireNonEmptyString(uri, "uri"); err != nil {
		return fmt.Errorf("textDocument: %w", err)
	}
	return nil
}

func validatePosition(data []byte) error {
	var value Position
	fields, err := decodeObject(data, &value)
	if err != nil {
		return fmt.Errorf("position: %w", err)
	}
	line, err := requireField(fields, "line")
	if err != nil {
		return fmt.Errorf("position: %w", err)
	}
	if err := requireNonNegativeInt(line, "line"); err != nil {
		return fmt.Errorf("position: %w", err)
	}
	character, err := requireField(fields, "character")
	if err != nil {
		return fmt.Errorf("position: %w", err)
	}
	if err := requireNonNegativeInt(character, "character"); err != nil {
		return fmt.Errorf("position: %w", err)
	}
	return nil
}

func validateReferenceContext(data []byte) error {
	var value ReferenceContext
	fields, err := decodeObject(data, &value)
	if err != nil {
		return fmt.Errorf("context: %w", err)
	}
	includeDeclaration, err := requireField(fields, "includeDeclaration")
	if err != nil {
		return fmt.Errorf("context: %w", err)
	}
	var boolean bool
	if err := json.Unmarshal(includeDeclaration, &boolean); err != nil {
		return fmt.Errorf("context.includeDeclaration must be boolean")
	}
	return nil
}
