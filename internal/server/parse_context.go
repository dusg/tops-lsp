package server

import (
	"context"
	"log/slog"

	"tops-lsp/internal/document"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
)

func (server *Server) partialParseContextAt(state document.DocumentState, contextVersion int) parser.ParseContext {
	return parser.ParseContext{
		DocumentURI:           state.URI,
		LanguageID:            state.LanguageID,
		LanguageStandard:      "unknown",
		DriverKind:            "unknown",
		CompilerContextStatus: "partial",
		ArgumentProvenance:    "unknown",
		PredefinedMacros:      map[string]parser.MacroValue{},
		TargetProfile:         "unknown",
		PassKind:              "unknown",
		IncludeRootsAvailable: false,
		DocumentVersion:       state.Version,
		ContextVersion:        contextVersion,
	}
}

func (server *Server) parseAndPublish(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState) {
	if publisher == nil {
		return
	}
	server.parseSnapshot(ctx, publisher, server.newAnalysisSnapshot(state))
}

func (server *Server) newAnalysisSnapshot(state document.DocumentState) analysisSnapshot {
	server.mu.Lock()
	defer server.mu.Unlock()
	if current, ok := server.documents.Get(state.URI); ok {
		state = current
	}
	server.analysis[state.URI]++
	generation := server.analysis[state.URI]
	contextVersion := int(server.contextVersion)
	return analysisSnapshot{state: state, context: parserContextSnapshot{version: contextVersion}, generation: generation, epoch: server.analysisEpoch}
}

func (server *Server) parseSnapshot(ctx context.Context, publisher diagnosticPublisher, snapshot analysisSnapshot) {
	if !server.analysisCurrentSnapshot(snapshot) {
		return
	}
	parseContext := server.partialParseContextAt(snapshot.state, snapshot.context.version)
	parse := server.parse
	if parse == nil {
		parse = parser.Parse
	}
	result := parse(snapshot.state.Text, parseContext)
	if result.DocumentVersion != snapshot.state.Version || result.ContextVersion != snapshot.context.version {
		return
	}
	server.log(ctx, slog.LevelInfo, "document_parsed", "document_id", logging.DocumentID(snapshot.state.URI), "document_version", snapshot.state.Version, "context_version", result.ContextVersion, "parse_status", result.Status, "diagnostic_count", len(result.Diagnostics), "diagnostic_codes", diagnosticCodes(result.Diagnostics))
	if err := server.publishParserDiagnosticsIfCurrent(ctx, publisher, snapshot, result); err != nil {
		server.log(ctx, slog.LevelError, "diagnostics_publish_failed", "document_id", logging.DocumentID(snapshot.state.URI), "document_version", snapshot.state.Version, "error_kind", "publisher_write_failed")
		server.reportFatal(err)
	}
}

func diagnosticCodes(diagnostics []parser.ParserDiagnostic) []string {
	codes := make([]string, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, ok := seen[diagnostic.Code]; ok {
			continue
		}
		seen[diagnostic.Code] = struct{}{}
		codes = append(codes, diagnostic.Code)
	}
	return codes
}

func (server *Server) nextAnalysis(uri string) uint64 {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.analysis[uri]++
	return server.analysis[uri]
}

func (server *Server) analysisCurrent(uri string, contextVersion int, generation uint64) bool {
	server.mu.RLock()
	defer server.mu.RUnlock()
	state, ok := server.documents.Get(uri)
	if !ok {
		return false
	}
	return server.analysisCurrentSnapshotLocked(analysisSnapshot{state: state, context: parserContextSnapshot{version: contextVersion}, generation: generation, epoch: server.analysisEpoch})
}

func (server *Server) analysisCurrentSnapshot(snapshot analysisSnapshot) bool {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.analysisCurrentSnapshotLocked(snapshot)
}

func (server *Server) analysisCurrentSnapshotLocked(snapshot analysisSnapshot) bool {
	if server.analysisStopping || server.analysisEpoch != snapshot.epoch {
		return false
	}
	state, ok := server.documents.Get(snapshot.state.URI)
	return ok && state == snapshot.state && server.analysis[snapshot.state.URI] == snapshot.generation && server.contextVersion == uint64(snapshot.context.version)
}

func (server *Server) currentContextVersion() int {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return int(server.contextVersion)
}

func (server *Server) InvalidateContext() {
	server.mu.Lock()
	server.contextVersion++
	server.analysisEpoch++
	for uri := range server.analysis {
		server.analysis[uri]++
	}
	for _, queue := range server.analysisQueues {
		queue.pending = nil
	}
	server.mu.Unlock()
}

func (server *Server) invalidateAnalysis(uri string) {
	server.mu.Lock()
	server.analysis[uri]++
	server.analysisEpoch++
	if queue := server.analysisQueues[uri]; queue != nil {
		queue.pending = nil
	}
	server.mu.Unlock()
}
