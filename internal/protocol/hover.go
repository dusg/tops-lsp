package protocol

import "fmt"

type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Hover struct {
	Range    Range         `json:"range"`
	Contents MarkupContent `json:"contents"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

const (
	MarkupKindMarkdown  = "markdown"
	MarkupKindPlainText = "plaintext"
)

func (params *HoverParams) UnmarshalJSON(data []byte) error {
	type alias HoverParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return fmt.Errorf("hover params: %w", err)
	}
	if err := validateTextDocumentIdentifier(fields["textDocument"]); err != nil {
		return fmt.Errorf("hover params: %w", err)
	}
	if _, err := requireField(fields, "position"); err != nil {
		return fmt.Errorf("hover params: %w", err)
	}
	if err := validatePosition(fields["position"]); err != nil {
		return fmt.Errorf("hover params: %w", err)
	}
	*params = HoverParams(value)
	return nil
}
