package server

import (
	"context"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
)

func (server *Server) handleDefinition(ctx context.Context, message protocol.Message) []byte {
	started := time.Now()
	var params protocol.TextDocumentPositionParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.observeNavigationError(ctx, "definition", "", nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "definition params are invalid")
	}
	_, snapshot, offset, ok := server.navigationInput(params.TextDocument.URI, params.Position)
	if !ok {
		server.observeNavigationError(ctx, "definition", params.TextDocument.URI, nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "definition document or position is invalid")
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "definition", params.TextDocument.URI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	if snapshot.RootURI == "" {
		server.observeNavigationError(ctx, "definition", params.TextDocument.URI, nil, 0, "stale_result", "no-fresh-snapshot", started)
		return server.successResponse(message.ID, nil)
	}
	if !server.waitNavigation(ctx, "definition") {
		server.observeNavigationError(ctx, "definition", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.bindNavigationFreshness(ctx, "definition", snapshot)
	locations, err := server.protocolLocations(snapshot, snapshot.DefinitionAtURI(params.TextDocument.URI, offset))
	if err != nil {
		server.observeNavigationError(ctx, "definition", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "definition range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "definition", params.TextDocument.URI, &snapshot, "definition-query", started)
		return server.successResponse(message.ID, nil)
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "definition", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.observeNavigation(ctx, "definition", snapshot, len(locations), started)
	return server.successResponse(message.ID, locations)
}

func (server *Server) handleReferences(ctx context.Context, message protocol.Message) []byte {
	started := time.Now()
	var params protocol.ReferenceParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.observeNavigationError(ctx, "references", "", nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "references params are invalid")
	}
	_, snapshot, offset, ok := server.navigationInput(params.TextDocument.URI, params.Position)
	if !ok {
		server.observeNavigationError(ctx, "references", params.TextDocument.URI, nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "references document or position is invalid")
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "references", params.TextDocument.URI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	if snapshot.RootURI == "" {
		server.observeNavigationError(ctx, "references", params.TextDocument.URI, nil, 0, "stale_result", "no-fresh-snapshot", started)
		return server.successResponse(message.ID, nil)
	}
	if !server.waitNavigation(ctx, "references") {
		server.observeNavigationError(ctx, "references", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.bindNavigationFreshness(ctx, "references", snapshot)
	locations, err := server.protocolLocations(snapshot, snapshot.ReferencesAtURI(params.TextDocument.URI, offset, params.Context.IncludeDeclaration))
	if err != nil {
		server.observeNavigationError(ctx, "references", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "reference range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "references", params.TextDocument.URI, &snapshot, "references-query", started)
		return server.successResponse(message.ID, nil)
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "references", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.observeNavigation(ctx, "references", snapshot, len(locations), started)
	return server.successResponse(message.ID, locations)
}

func (server *Server) handleDocumentSymbol(ctx context.Context, message protocol.Message) []byte {
	started := time.Now()
	var params protocol.DocumentSymbolParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.observeNavigationError(ctx, "documentSymbol", "", nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "document symbol params are invalid")
	}
	state, snapshot, _, ok := server.navigationInput(params.TextDocument.URI, protocol.Position{Line: 0, Character: 0})
	if !ok {
		server.observeNavigationError(ctx, "documentSymbol", params.TextDocument.URI, nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "document symbol document is invalid")
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "documentSymbol", params.TextDocument.URI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	if snapshot.RootURI == "" {
		server.observeNavigationError(ctx, "documentSymbol", params.TextDocument.URI, nil, 0, "stale_result", "no-fresh-snapshot", started)
		return server.successResponse(message.ID, nil)
	}
	if !server.waitNavigation(ctx, "documentSymbol") {
		server.observeNavigationError(ctx, "documentSymbol", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.bindNavigationFreshness(ctx, "documentSymbol", snapshot)
	symbols, err := server.protocolDocumentSymbols(state.Text, snapshot.DocumentSymbols())
	if err != nil {
		server.observeNavigationError(ctx, "documentSymbol", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "document symbol range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "documentSymbol", params.TextDocument.URI, &snapshot, "document-symbol-query", started)
		return server.successResponse(message.ID, nil)
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "documentSymbol", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.observeNavigation(ctx, "documentSymbol", snapshot, len(symbols), started)
	return server.successResponse(message.ID, symbols)
}

func (server *Server) waitNavigation(ctx context.Context, operation string) bool {
	server.mu.RLock()
	gate := server.navigationGate
	server.mu.RUnlock()
	if gate != nil {
		if err := gate(ctx, operation); err != nil {
			return false
		}
	}
	return ctx.Err() == nil
}

func (server *Server) bindNavigationFreshness(ctx context.Context, operation string, snapshot index.IndexSnapshot) {
	state, ok := ctx.Value(requestStateKey{}).(*requestState)
	if !ok || state == nil {
		return
	}
	requestURI := snapshot.RootURI
	if metadata, ok := ctx.Value(requestMetadataKey{}).(requestMetadata); ok && metadata.URI != "" {
		requestURI = metadata.URI
	}
	state.SetFreshnessCheck(func() bool {
		return server.navigationSnapshotCurrent(snapshot)
	})
	state.SetStaleObserver(func() {
		server.observeStaleResult(ctx, operation, requestURI, &snapshot, "request-freshness-check", time.Now())
	})
}

func (server *Server) navigationInput(uri string, requested protocol.Position) (document.DocumentState, index.IndexSnapshot, int, bool) {
	state, ok := server.documents.Get(uri)
	if !ok || uri == "" {
		return document.DocumentState{}, index.IndexSnapshot{}, 0, false
	}
	offset, err := position.New(state.Text).OffsetAt(position.Position{Line: requested.Line, Character: requested.Character})
	if err != nil {
		return document.DocumentState{}, index.IndexSnapshot{}, 0, false
	}
	snapshot, ok := server.currentIndexSnapshot(uri)
	if !ok || snapshot.SourceVersion.DocumentVersion != state.Version || snapshot.ContextVersion != server.currentContextVersion() || snapshot.IndexStatus == index.IndexStatusStale {
		return state, index.IndexSnapshot{}, offset, true
	}
	return state, snapshot, offset, true
}

func (server *Server) navigationSnapshotCurrent(snapshot index.IndexSnapshot) bool {
	state, ok := server.documents.Get(snapshot.RootURI)
	if !ok {
		return false
	}
	current, ok := server.currentIndexSnapshot(snapshot.RootURI)
	if !ok || current.SourceVersion.DocumentVersion != state.Version || current.ContextVersion != server.currentContextVersion() {
		return false
	}
	if snapshot.SourceVersion.Generation != 0 && current.SourceVersion.Generation != snapshot.SourceVersion.Generation {
		return false
	}
	return current.SourceVersion.SourceVersion == snapshot.SourceVersion.SourceVersion
}

func (server *Server) protocolLocations(snapshot index.IndexSnapshot, locations []index.SymbolLocation) ([]protocol.Location, error) {
	result := make([]protocol.Location, 0, len(locations))
	for _, location := range locations {
		text, ok := snapshot.Sources[location.URI]
		if !ok {
			return nil, position.ErrInvalidRange
		}
		mapper := position.New(text)
		rangeValue, err := mapper.Range(location.Range.Start, location.Range.End)
		if err != nil {
			return nil, err
		}
		result = append(result, protocol.Location{URI: location.URI, Range: protocol.Range{Start: protocol.Position{Line: rangeValue.Start.Line, Character: rangeValue.Start.Character}, End: protocol.Position{Line: rangeValue.End.Line, Character: rangeValue.End.Character}}})
	}
	return result, nil
}

func (server *Server) protocolDocumentSymbols(text string, symbols []index.DocumentSymbol) ([]protocol.DocumentSymbol, error) {
	mapper := position.New(text)
	result := make([]protocol.DocumentSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		rangeValue, err := mapper.Range(symbol.Range.Start, symbol.Range.End)
		if err != nil {
			return nil, err
		}
		selectionRange, err := mapper.Range(symbol.SelectionRange.Start, symbol.SelectionRange.End)
		if err != nil {
			return nil, err
		}
		children, err := server.protocolDocumentSymbols(text, symbol.Children)
		if err != nil {
			return nil, err
		}
		result = append(result, protocol.DocumentSymbol{
			Name:           symbol.Name,
			Detail:         symbol.Detail,
			Kind:           protocolSymbolKind(symbol.Kind, symbol.TypeKind),
			Range:          protocol.Range{Start: protocol.Position{Line: rangeValue.Start.Line, Character: rangeValue.Start.Character}, End: protocol.Position{Line: rangeValue.End.Line, Character: rangeValue.End.Character}},
			SelectionRange: protocol.Range{Start: protocol.Position{Line: selectionRange.Start.Line, Character: selectionRange.Start.Character}, End: protocol.Position{Line: selectionRange.End.Line, Character: selectionRange.End.Character}},
			Children:       children,
		})
	}
	return result, nil
}

func protocolSymbolKind(kind index.SymbolKind, typeKind string) protocol.SymbolKind {
	switch kind {
	case index.SymbolKindNamespace:
		return protocol.SymbolKindNamespace
	case index.SymbolKindFunction:
		return protocol.SymbolKindFunction
	case index.SymbolKindType:
		switch typeKind {
		case "enum":
			return protocol.SymbolKindEnum
		case "struct", "union":
			return protocol.SymbolKindStruct
		}
		return protocol.SymbolKindClass
	case index.SymbolKindObject:
		return protocol.SymbolKindObject
	case index.SymbolKindField:
		return protocol.SymbolKindField
	case index.SymbolKindAttribute:
		return protocol.SymbolKindProperty
	case index.SymbolKindBuiltin, index.SymbolKindVariable:
		return protocol.SymbolKindVariable
	case index.SymbolKindMacro:
		return protocol.SymbolKindConstant
	default:
		return protocol.SymbolKindObject
	}
}
