package server

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
)

type completionCandidate struct {
	item      protocol.CompletionItem
	group     int
	kindOrder int
	address   string
}

type analysisProvenance struct {
	profileID     string
	passKind      string
	contextStatus string
}

func (server *Server) analysisProvenanceFor(uri string, snapshot index.IndexSnapshot) analysisProvenance {
	result := analysisProvenance{
		profileID:     snapshot.TargetProfile,
		passKind:      snapshot.PassKind,
		contextStatus: string(semantic.ContextStatusMissing),
	}
	if server.semanticStore == nil {
		return result
	}
	semanticSnapshot, ok := server.semanticStore.Get(uri)
	if !ok {
		return result
	}
	if semanticSnapshot.ProfileID != "" {
		result.profileID = semanticSnapshot.ProfileID
	}
	if semanticSnapshot.ContextStatus != "" {
		result.contextStatus = string(semanticSnapshot.ContextStatus)
	}
	if semanticSnapshot.Context.Command.PassKind != "" {
		result.passKind = string(semanticSnapshot.Context.Command.PassKind)
	}
	return result
}

func (server *Server) handleCompletion(ctx context.Context, message protocol.Message) []byte {
	started := time.Now()
	var params protocol.CompletionParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.observeRequestOutcome(ctx, "completion", "", 0, "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "completion params are invalid")
	}
	uri := params.TextDocument.URI
	state, ok := server.documents.Get(uri)
	if !ok {
		server.observeRequestOutcome(ctx, "completion", uri, 0, "empty", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{})
	}
	mapper := position.New(state.Text)
	offset, err := mapper.OffsetAt(position.Position{Line: params.Position.Line, Character: params.Position.Character})
	if err != nil {
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "invalid-params", started)
		return server.errorResponse(message.ID, protocol.InvalidParams, "completion position is invalid")
	}
	snapshot, availability := server.snapshotAvailability(uri)
	switch availability {
	case snapshotStale:
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "stale", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{})
	case snapshotMissing:
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "empty", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{})
	case snapshotAnalyzing:
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "analyzing", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{server.analyzingCompletionItem(state.Text, offset)})
	}
	if err := ctx.Err(); err != nil {
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "cancelled", started)
		return server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
	}
	region := classifyCompletionRegion(snapshot, uri, offset)
	if region == completionRegionBlocked {
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "empty", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{})
	}
	items := server.buildCompletionItems(snapshot, uri, state.Text, offset, region)
	// A version change while the candidates were built invalidates the whole
	// result: return the agreed empty list instead of stale candidates.
	if !server.hasFreshAnalysis(uri) {
		server.observeRequestOutcome(ctx, "completion", uri, state.Version, "stale", started)
		return server.successResponse(message.ID, []protocol.CompletionItem{})
	}
	server.observeRequestOutcome(ctx, "completion", uri, state.Version, completionOutcome(items), started)
	return server.successResponse(message.ID, items)
}

func completionOutcome(items []protocol.CompletionItem) string {
	if len(items) == 0 {
		return "empty"
	}
	return "success"
}

func (server *Server) analyzingCompletionItem(text string, offset int) protocol.CompletionItem {
	zeroRange := protocol.Range{}
	if positionValue, err := position.New(text).PositionAt(offset); err == nil {
		zeroRange = protocol.Range{Start: protocolPosition(positionValue), End: protocolPosition(positionValue)}
	}
	return protocol.CompletionItem{
		Label:      "Analyzing…",
		Kind:       protocol.CompletionItemKindText,
		Detail:     "Analysis in progress; completion is not available yet",
		TextEdit:   &protocol.TextEdit{Range: zeroRange, NewText: ""},
		FilterText: identifierPrefix(text, offset),
		SortText:   "000",
	}
}

func (server *Server) buildCompletionItems(snapshot index.IndexSnapshot, uri, text string, offset int, region completionRegion) []protocol.CompletionItem {
	mapper := position.New(text)
	prefix := identifierPrefix(text, offset)
	prefixStart := offset - len(prefix)
	replaceRange, err := mapper.Range(prefixStart, offset)
	if err != nil {
		return []protocol.CompletionItem{}
	}
	provenance := server.analysisProvenanceFor(uri, snapshot)
	provenanceData := func(source string) map[string]string {
		return map[string]string{
			"source":        source,
			"profileId":     provenance.profileID,
			"passKind":      provenance.passKind,
			"contextStatus": provenance.contextStatus,
		}
	}
	candidates := make([]completionCandidate, 0)

	addCandidate := func(item protocol.CompletionItem, kindOrder int, address string) {
		item.TextEdit = &protocol.TextEdit{Range: protocolRange(replaceRange), NewText: item.Label}
		item.FilterText = item.Label
		group := 1
		if strings.HasPrefix(item.Label, prefix) && prefix != "" {
			group = 0
		}
		candidates = append(candidates, completionCandidate{item: item, group: group, kindOrder: kindOrder, address: address})
	}

	for _, symbol := range snapshot.VisibleSymbolsAt(uri, offset) {
		if !completionSymbolAllowed(region, symbol.Kind) {
			continue
		}
		detail := completionDetail(symbol)
		if isTopsSymbol(symbol) {
			detail = "Tops extension: " + detail
		}
		item := protocol.CompletionItem{
			Label:  symbol.Name,
			Kind:   completionItemKind(symbol),
			Detail: detail,
			Data:   provenanceData("index"),
		}
		address := symbol.SourceVersion.URI + ":" + fmt.Sprint(symbol.NameRange.Start) + ":" + fmt.Sprint(symbol.NameRange.End)
		addCandidate(item, completionKindOrder(symbol.Kind), address)
	}
	if region.allowsMacros() {
		macroNames := make([]string, 0, len(snapshot.PredefinedMacros))
		for name, value := range snapshot.PredefinedMacros {
			if value.Kind == parser.MacroUndefined {
				continue
			}
			macroNames = append(macroNames, name)
		}
		sort.Strings(macroNames)
		for _, name := range macroNames {
			item := protocol.CompletionItem{
				Label:  name,
				Kind:   protocol.CompletionItemKindConstant,
				Detail: "predefined macro",
				Data:   provenanceData("predefined-macro"),
			}
			addCandidate(item, 4, "predefined-macro:"+name)
		}
	}
	for _, spelling := range parser.CppKeywords() {
		if !completionKeywordAllowed(region, spelling, false) {
			continue
		}
		addCandidate(protocol.CompletionItem{Label: spelling, Kind: protocol.CompletionItemKindKeyword, Detail: "keyword", Data: provenanceData("keyword")}, 5, "keyword:"+spelling)
	}
	for _, spelling := range parser.TopsKeywords() {
		if !completionKeywordAllowed(region, spelling, true) {
			continue
		}
		addCandidate(protocol.CompletionItem{Label: spelling, Kind: protocol.CompletionItemKindKeyword, Detail: "Tops keyword", Data: provenanceData("keyword")}, 5, "keyword:"+spelling)
	}
	return finalizeCompletionCandidates(candidates)
}

func (region completionRegion) allowsMacros() bool {
	switch region {
	case completionRegionNormal, completionRegionExpression:
		return true
	default:
		return false
	}
}

func completionKindOrder(kind index.SymbolKind) int {
	switch kind {
	case index.SymbolKindVariable, index.SymbolKindBuiltin, index.SymbolKindField:
		return 0
	case index.SymbolKindFunction:
		return 1
	case index.SymbolKindType:
		return 2
	case index.SymbolKindNamespace:
		return 3
	case index.SymbolKindMacro:
		return 4
	case index.SymbolKindAttribute:
		return 5
	default:
		return 6
	}
}

func completionDetail(symbol index.SymbolRecord) string {
	if symbol.Detail != "" {
		return symbol.Detail
	}
	if symbol.Signature != "" {
		return symbol.Signature
	}
	if symbol.TypeName != "" {
		return symbol.TypeName
	}
	return ""
}

func finalizeCompletionCandidates(candidates []completionCandidate) []protocol.CompletionItem {
	seen := make(map[string]struct{}, len(candidates))
	unique := make([]completionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.item.Label + "\x00" + fmt.Sprint(candidate.item.Kind) + "\x00" + candidate.address
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	sort.SliceStable(unique, func(left, right int) bool {
		if unique[left].group != unique[right].group {
			return unique[left].group < unique[right].group
		}
		if unique[left].kindOrder != unique[right].kindOrder {
			return unique[left].kindOrder < unique[right].kindOrder
		}
		if unique[left].item.Label != unique[right].item.Label {
			return unique[left].item.Label < unique[right].item.Label
		}
		return unique[left].address < unique[right].address
	})
	items := make([]protocol.CompletionItem, len(unique))
	for index, candidate := range unique {
		candidate.item.SortText = fmt.Sprintf("%d%02d", candidate.group, index+1)
		items[index] = candidate.item
	}
	return items
}

// observeRequestOutcome records the request result category together with the
// document identifier and version. Source text and credentials are never logged.
func (server *Server) observeRequestOutcome(ctx context.Context, operation, uri string, version int, result string, started time.Time) {
	server.log(ctx, requestOutcomeLevel(result), "request_result",
		"operation", operation,
		"document_id", logging.DocumentID(uri),
		"document_version", version,
		"result", result,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func requestOutcomeLevel(result string) slog.Level {
	switch result {
	case "invalid-params", "cancelled", "navigation-timeout", "stale":
		return slog.LevelWarn
	default:
		return slog.LevelDebug
	}
}
