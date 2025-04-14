package lsp

import "fmt"

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
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
