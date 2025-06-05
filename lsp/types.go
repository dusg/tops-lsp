package lsp

import "fmt"

type Position struct {
	Line      uint32 `json:"line"`
	Character uint32 `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}
type SourcePosition struct {
	Position Position
	File     *string
}

func (p SourcePosition) String() string {
	if p.File == nil {
		return fmt.Sprintf("%d:%d", p.Position.Line, p.Position.Character)
	}
	return fmt.Sprintf("%s:%d:%d", *p.File, p.Position.Line, p.Position.Character)
}

type TextDocumentIdentifier struct {
	Uri string `json:"uri"`
}

type TextDocumentPositionParams struct {
	/**
	 * The text document.
	 */
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	/**
	 * The position inside the text document.
	 */
	Position Position `json:"position"`
}

type Location struct {
	Uri   string `json:"uri"`
	Range Range  `json:"range"`
}
