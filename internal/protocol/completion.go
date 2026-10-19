package protocol

import "fmt"

type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

type CompletionItem struct {
	Label      string            `json:"label"`
	Kind       int               `json:"kind,omitempty"`
	Detail     string            `json:"detail,omitempty"`
	InsertText string            `json:"insertText,omitempty"`
	TextEdit   *TextEdit         `json:"textEdit,omitempty"`
	FilterText string            `json:"filterText,omitempty"`
	SortText   string            `json:"sortText,omitempty"`
	Data       map[string]string `json:"data,omitempty"`
}

// Completion item kinds, matching the LSP CompletionItemKind enumeration.
const (
	CompletionItemKindText      = 1
	CompletionItemKindMethod    = 2
	CompletionItemKindFunction  = 3
	CompletionItemKindField     = 5
	CompletionItemKindVariable  = 6
	CompletionItemKindClass     = 7
	CompletionItemKindInterface = 8
	CompletionItemKindModule    = 9
	CompletionItemKindProperty  = 10
	CompletionItemKindEnum      = 13
	CompletionItemKindKeyword   = 14
	CompletionItemKindConstant  = 21
	CompletionItemKindStruct    = 22
)

func (params *CompletionParams) UnmarshalJSON(data []byte) error {
	type alias CompletionParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return fmt.Errorf("completion params: %w", err)
	}
	if err := validateTextDocumentIdentifier(fields["textDocument"]); err != nil {
		return fmt.Errorf("completion params: %w", err)
	}
	if _, err := requireField(fields, "position"); err != nil {
		return fmt.Errorf("completion params: %w", err)
	}
	if err := validatePosition(fields["position"]); err != nil {
		return fmt.Errorf("completion params: %w", err)
	}
	*params = CompletionParams(value)
	return nil
}
