package server

import (
	"context"
	"errors"
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
	uri := params.TextDocument.URI
	snapshot, offset, response := server.navigationPrelude(ctx, message, "definition", uri, "definition document or position is invalid", params.Position, nil, started)
	if response != nil {
		return response
	}
	locations, err := server.protocolLocations(snapshot, snapshot.DefinitionAtURI(uri, offset))
	if err != nil {
		server.observeNavigationError(ctx, "definition", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "definition range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "definition", uri, &snapshot, "definition-query", started)
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
	uri := params.TextDocument.URI
	snapshot, offset, response := server.navigationPrelude(ctx, message, "references", uri, "references document or position is invalid", params.Position, []protocol.Location{}, started)
	if response != nil {
		return response
	}
	locations, err := server.protocolLocations(snapshot, snapshot.ReferencesAtURI(uri, offset, params.Context.IncludeDeclaration))
	if err != nil {
		server.observeNavigationError(ctx, "references", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "reference range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "references", uri, &snapshot, "references-query", started)
		return server.successResponse(message.ID, []protocol.Location{})
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
	uri := params.TextDocument.URI
	snapshot, _, response := server.navigationPrelude(ctx, message, "documentSymbol", uri, "document symbol document is invalid", protocol.Position{Line: 0, Character: 0}, []protocol.DocumentSymbol{}, started)
	if response != nil {
		return response
	}
	state, ok := server.documents.Get(uri)
	if !ok {
		server.observeNavigationError(ctx, "documentSymbol", uri, nil, 0, "none", "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "document symbol document is invalid")
	}
	symbols, err := server.protocolDocumentSymbols(state.Text, snapshot.DocumentSymbols())
	if err != nil {
		server.observeNavigationError(ctx, "documentSymbol", snapshot.RootURI, &snapshot, 0, "none", "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "document symbol range could not be encoded")
	}
	if !server.navigationSnapshotCurrent(snapshot) {
		server.observeStaleResultOnce(ctx, "documentSymbol", uri, &snapshot, "document-symbol-query", started)
		return server.successResponse(message.ID, []protocol.DocumentSymbol{})
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, "documentSymbol", snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.observeNavigation(ctx, "documentSymbol", snapshot, len(symbols), started)
	return server.successResponse(message.ID, symbols)
}

// navigationPrelude runs the shared validation, waiting and freshness steps for
// the navigation handlers. A non-nil response means the request already has its
// final answer and the caller must return it unchanged.
func (server *Server) navigationPrelude(ctx context.Context, message protocol.Message, operation, uri, invalidParamsMessage string, requested protocol.Position, emptyResult any, started time.Time) (index.IndexSnapshot, int, []byte) {
	if _, open := server.documents.Get(uri); !open {
		server.observeNavigationError(ctx, operation, uri, nil, 0, "none", "empty", started)
		return index.IndexSnapshot{}, 0, server.successResponse(message.ID, emptyResult)
	}
	_, snapshot, offset, ok := server.navigationInput(uri, requested)
	if !ok {
		server.observeNavigationError(ctx, operation, uri, nil, 0, "none", "invalid-params", started)
		return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.InvalidParams, invalidParamsMessage)
	}
	if err := ctx.Err(); err != nil {
		server.observeNavigationError(ctx, operation, uri, &snapshot, 0, "none", "request-cancelled", started)
		return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	snapshot, offset, ok, waitErr := server.awaitNavigationSnapshot(ctx, uri, requested, snapshot, offset, started.Add(analysisWaitTimeout))
	if waitErr != nil {
		if errors.Is(waitErr, errAnalysisTimeout) {
			server.observeNavigationError(ctx, operation, uri, nil, 0, "navigation-timeout", "navigation-timeout", started)
			return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.ContentModified, "content was modified; retry the request")
		}
		server.observeNavigationError(ctx, operation, uri, &snapshot, 0, "none", "request-cancelled", started)
		return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	if !ok {
		server.observeNavigationError(ctx, operation, uri, nil, 0, "none", "invalid-params", started)
		return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.InvalidParams, invalidParamsMessage)
	}
	if snapshot.RootURI == "" {
		server.observeNavigationError(ctx, operation, uri, nil, 0, "stale_result", "no-fresh-snapshot", started)
		return index.IndexSnapshot{}, 0, server.successResponse(message.ID, emptyResult)
	}
	if waitErr := server.waitNavigation(ctx, operation, started.Add(analysisWaitTimeout)); waitErr != nil {
		if errors.Is(waitErr, errAnalysisTimeout) {
			server.observeNavigationError(ctx, operation, snapshot.RootURI, &snapshot, 0, "navigation-timeout", "navigation-timeout", started)
			return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.ContentModified, "content was modified; retry the request")
		}
		server.observeNavigationError(ctx, operation, snapshot.RootURI, &snapshot, 0, "none", "request-cancelled", started)
		return index.IndexSnapshot{}, 0, server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	server.bindNavigationFreshness(ctx, operation, snapshot)
	return snapshot, offset, nil
}

func (server *Server) waitNavigation(ctx context.Context, operation string, deadline time.Time) error {
	server.mu.RLock()
	gate := server.navigationGate
	server.mu.RUnlock()
	if gate == nil {
		return ctx.Err()
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return errAnalysisTimeout
	}
	gateContext, cancel := context.WithTimeout(ctx, remaining)
	defer cancel()
	if err := gate(gateContext, operation); err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return errAnalysisTimeout
		}
		return err
	}
	return ctx.Err()
}

// awaitNavigationSnapshot waits until deadline when no snapshot exists yet and
// analysis is still running. Existing stale snapshots are returned immediately
// so callers can emit the empty result without waiting.
func (server *Server) awaitNavigationSnapshot(ctx context.Context, uri string, requested protocol.Position, snapshot index.IndexSnapshot, offset int, deadline time.Time) (index.IndexSnapshot, int, bool, error) {
	if _, availability := server.snapshotAvailability(uri); availability != snapshotAnalyzing {
		return snapshot, offset, true, nil
	}
	if err := server.waitForAnalysis(ctx, uri, deadline); err != nil {
		return snapshot, offset, true, err
	}
	_, current, currentOffset, ok := server.navigationInput(uri, requested)
	return current, currentOffset, ok, nil
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
	snapshot, fresh := server.currentFreshSnapshot(uri)
	if !fresh {
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
		result = append(result, protocol.Location{URI: location.URI, Range: protocolRange(rangeValue)})
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
			Range:          protocolRange(rangeValue),
			SelectionRange: protocolRange(selectionRange),
			Children:       children,
		})
	}
	return result, nil
}
