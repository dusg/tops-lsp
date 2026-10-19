package server

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
)

func (server *Server) handleHover(ctx context.Context, message protocol.Message) []byte {
	started := time.Now()
	var params protocol.HoverParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.observeRequestOutcome(ctx, "hover", "", 0, "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "hover params are invalid")
	}
	uri := params.TextDocument.URI
	state, ok := server.documents.Get(uri)
	if !ok {
		server.observeRequestOutcome(ctx, "hover", uri, 0, "empty", started)
		return server.successResponse(message.ID, nil)
	}
	mapper := position.New(state.Text)
	offset, err := mapper.OffsetAt(position.Position{Line: params.Position.Line, Character: params.Position.Character})
	if err != nil {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "hover position is invalid")
	}
	snapshot, availability := server.snapshotAvailability(uri)
	switch availability {
	case snapshotStale:
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "stale", started)
		return server.successResponse(message.ID, nil)
	case snapshotMissing:
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "empty", started)
		return server.successResponse(message.ID, nil)
	case snapshotAnalyzing:
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "analyzing", started)
		return server.successResponse(message.ID, server.analyzingHover(state.Text, offset, server.hoverMarkupKind()))
	}
	if err := ctx.Err(); err != nil {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	symbols := snapshot.SymbolsAtURI(uri, offset)
	if len(symbols) == 0 {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "empty", started)
		return server.successResponse(message.ID, nil)
	}
	symbol := symbols[0]
	nameRange, ok := hoverNameRange(snapshot, uri, offset, symbol)
	if !ok {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "empty", started)
		return server.successResponse(message.ID, nil)
	}
	rangeValue, err := mapper.Range(nameRange.Start, nameRange.End)
	if err != nil {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "range-encoding", started)
		return server.errorResponse(message.ID, protocol.InternalError, "hover range could not be encoded")
	}
	markupKind := server.hoverMarkupKind()
	hover := protocol.Hover{
		Range: protocolRange(rangeValue),
		Contents: protocol.MarkupContent{
			Kind:  markupKind,
			Value: server.renderHover(snapshot, uri, symbol, markupKind),
		},
	}
	// A version change while the hover was built invalidates the result: return
	// the agreed empty value instead of a stale description.
	if !server.hasFreshAnalysis(uri) {
		server.observeRequestOutcome(ctx, "hover", uri, state.Version, "stale", started)
		return server.successResponse(message.ID, nil)
	}
	server.observeRequestOutcome(ctx, "hover", uri, state.Version, "success", started)
	return server.successResponse(message.ID, hover)
}

func (server *Server) hoverMarkupKind() string {
	server.mu.RLock()
	formats := append([]string(nil), server.hoverContentFormat...)
	server.mu.RUnlock()
	if len(formats) == 0 {
		return protocol.MarkupKindMarkdown
	}
	for _, format := range formats {
		if format == protocol.MarkupKindMarkdown {
			return protocol.MarkupKindMarkdown
		}
	}
	return protocol.MarkupKindPlainText
}

// analyzingHover builds the "analysis in progress" result. The range rules live
// in contracts/lsp-core-language.md section 4.5.1.
func (server *Server) analyzingHover(text string, offset int, markupKind string) protocol.Hover {
	start := offset
	end := offset
	switch {
	case offset > 0 && offset < len(text):
		end = offset + 1
	case len(text) == 0:
		start, end = 0, 0
	case offset == 0:
		end = 1
	case offset >= len(text):
		start, end = len(text)-1, len(text)
	}
	mapper := position.New(text)
	rangeValue, err := mapper.Range(start, end)
	if err != nil {
		rangeValue = position.Range{}
	}
	return protocol.Hover{
		Range:    protocolRange(rangeValue),
		Contents: protocol.MarkupContent{Kind: markupKind, Value: "Analyzing… analysis is still in progress"},
	}
}

func hoverNameRange(snapshot index.IndexSnapshot, uri string, offset int, symbol index.SymbolRecord) (parser.SourceRange, bool) {
	if symbol.SourceVersion.URI == uri && symbol.NameRange.Start <= offset && offset < symbol.NameRange.End {
		return symbol.NameRange, true
	}
	for _, token := range snapshot.Tokens {
		if token.Range.Start <= offset && offset < token.Range.End {
			return token.Range, true
		}
	}
	if offset > 0 {
		for _, token := range snapshot.Tokens {
			if token.Range.Start <= offset-1 && offset-1 < token.Range.End {
				return token.Range, true
			}
		}
	}
	return parser.SourceRange{}, false
}

func (server *Server) renderHover(snapshot index.IndexSnapshot, uri string, symbol index.SymbolRecord, markupKind string) string {
	provenance := server.analysisProvenanceFor(uri, snapshot)
	kindLabel := hoverKindLabel(symbol)

	header := symbol.Name + " — " + kindLabel
	details := hoverDetails(symbol)
	declaration := hoverDeclarationLocation(snapshot, uri, symbol)
	sourceLabel := "standard C++"
	if isTopsSymbol(symbol) {
		sourceLabel = "Tops extension"
	}

	lines := make([]string, 0, 10)
	if markupKind == protocol.MarkupKindMarkdown {
		lines = append(lines, "**"+symbol.Name+"** — "+kindLabel)
		if details != "" {
			lines = append(lines, "", "```cpp", details, "```")
		}
	} else {
		lines = append(lines, header)
		if details != "" {
			lines = append(lines, details)
		}
	}
	if declaration != "" {
		lines = append(lines, "", "Declared at "+declaration)
	}
	lines = append(lines, "Source: "+sourceLabel)
	if symbol.Condition.State == parser.ConditionalUnknown {
		lines = append(lines, "Condition: unknown (degraded: conditional compilation is not decidable)")
	}
	if provenance.profileID != "" || provenance.passKind != "" {
		profile := provenance.profileID
		if profile == "" {
			profile = "unknown"
		}
		pass := provenance.passKind
		if pass == "" {
			pass = "not-applicable"
		}
		lines = append(lines, "Profile: "+profile+", Pass: "+pass)
	}
	lines = append(lines, "Context: "+hoverContextStatus(provenance.contextStatus))
	return strings.Join(lines, "\n")
}

func hoverDetails(symbol index.SymbolRecord) string {
	if symbol.Signature != "" {
		return symbol.Signature
	}
	if symbol.Detail != "" {
		return symbol.Detail
	}
	if symbol.TypeName != "" {
		return symbol.TypeName
	}
	return ""
}

func isTopsSymbol(symbol index.SymbolRecord) bool {
	if symbol.Kind == index.SymbolKindBuiltin || symbol.Kind == index.SymbolKindAttribute {
		return true
	}
	if symbol.Builtin.TargetProfile != "" || symbol.Builtin.PassKind != "" {
		return true
	}
	return len(symbol.ExecutionQualifiers) > 0
}

func hoverContextStatus(status string) string {
	if status == "" {
		return string(semantic.ContextStatusMissing) + " (degraded: Tops rules are not fully evaluated)"
	}
	if status == string(semantic.ContextStatusResolved) {
		return status
	}
	return status + " (degraded: Tops rules are not fully evaluated)"
}

func hoverDeclarationLocation(snapshot index.IndexSnapshot, uri string, symbol index.SymbolRecord) string {
	location, ok := hoverDeclarationLocationValue(symbol)
	if !ok {
		return ""
	}
	locationURI := location.URI
	if locationURI == "" {
		locationURI = uri
	}
	fileName := hoverFileBaseName(locationURI)
	text, hasText := snapshot.Sources[locationURI]
	if !hasText {
		return fileName
	}
	mapper := position.New(text)
	value, err := mapper.PositionAt(location.Range.Start)
	if err != nil {
		return fileName
	}
	return fmt.Sprintf("%s:%d:%d", fileName, value.Line+1, value.Character+1)
}

func hoverDeclarationLocationValue(symbol index.SymbolRecord) (index.SymbolLocation, bool) {
	if len(symbol.Definitions) > 0 {
		return symbol.Definitions[0], true
	}
	if len(symbol.Declarations) > 0 {
		return symbol.Declarations[0], true
	}
	if len(symbol.AttributeSources) > 0 {
		return symbol.AttributeSources[0], true
	}
	return index.SymbolLocation{}, false
}

func hoverFileBaseName(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Path != "" {
		return filepath.Base(parsed.Path)
	}
	return filepath.Base(uri)
}
