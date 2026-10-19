package server

import (
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
)

// protocolPosition converts a source position into its protocol form.
func protocolPosition(value position.Position) protocol.Position {
	return protocol.Position{Line: value.Line, Character: value.Character}
}

// protocolRange converts a source range into its protocol form.
func protocolRange(value position.Range) protocol.Range {
	return protocol.Range{Start: protocolPosition(value.Start), End: protocolPosition(value.End)}
}
