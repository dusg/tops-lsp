package server

import (
	"context"
	"log/slog"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
)

func (server *Server) partialParseContextAt(state document.DocumentState, contextVersion int) parser.ParseContext {
	includeRoots := []string(nil)
	includeRootsAvailable := false
	if server.sourceProvider != nil {
		includeRoots, includeRootsAvailable = server.sourceProvider.IncludeRoots()
	}
	server.mu.RLock()
	compilationContext := cloneCompilationContext(server.compilationContext)
	server.mu.RUnlock()
	return parser.ParseContext{
		DocumentURI:           state.URI,
		LanguageID:            state.LanguageID,
		LanguageStandard:      compilationContext.LanguageStandard,
		DriverKind:            compilationContext.DriverKind,
		CompilerContextStatus: compilationContext.CompilerContextStatus,
		ArgumentProvenance:    compilationContext.ArgumentProvenance,
		PredefinedMacros:      compilationContext.PredefinedMacros,
		TargetProfile:         compilationContext.TargetProfile,
		PassKind:              compilationContext.PassKind,
		IncludeRoots:          append([]string(nil), includeRoots...),
		IncludeRootsAvailable: includeRootsAvailable,
		DocumentVersion:       state.Version,
		ContextVersion:        contextVersion,
	}
}

func (server *Server) parseAndPublish(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState) {
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
	snapshot := analysisSnapshot{state: state, context: parserContextSnapshot{version: contextVersion}, generation: generation, epoch: server.analysisEpoch}
	if indexed, ok := server.currentIndexSnapshot(state.URI); ok {
		snapshot.headerDependencies = cloneHeaderDependencies(indexed.Dependencies)
	}
	return snapshot
}

func (server *Server) parseSnapshot(ctx context.Context, publisher diagnosticPublisher, snapshot analysisSnapshot) {
	started := time.Now()
	if !server.analysisCurrentSnapshot(snapshot) {
		server.observeStaleAnalysisSnapshot(ctx, snapshot, "analysis-snapshot", started)
		return
	}
	parseContext := server.partialParseContextAt(snapshot.state, snapshot.context.version)
	parse := server.parse
	if parse == nil {
		parse = parser.Parse
	}
	result := parse(snapshot.state.Text, parseContext)
	if result.DocumentVersion != snapshot.state.Version || result.ContextVersion != snapshot.context.version {
		server.observeStaleParseResult(ctx, snapshot, result, "parser-version-mismatch", started)
		return
	}
	indexed := index.BuildWithProvider(snapshot.state.Text, result, parseContext, server.sourceProvider)
	indexed.Generation = snapshot.generation
	indexed.Epoch = snapshot.epoch
	indexed.SourceVersion.Generation = snapshot.generation
	indexed.SourceVersion.Epoch = snapshot.epoch
	if !server.commitIndexSnapshot(indexed) {
		server.observeStaleResult(ctx, "index", snapshot.state.URI, &indexed, "snapshot-mismatch", started)
		return
	}
	server.observeIndex(ctx, indexed, started)
	if err := server.publishAnalysisDiagnostics(ctx, publisher, snapshot.state, indexed, result); err != nil {
		server.log(ctx, slog.LevelError, "index_diagnostics_publish_failed", "document_id", logging.DocumentID(snapshot.state.URI), "error_kind", "publisher_write_failed")
		server.reportFatal(err)
	}
	server.log(ctx, slog.LevelInfo, "document_parsed", "document_id", logging.DocumentID(snapshot.state.URI), "document_version", snapshot.state.Version, "context_version", result.ContextVersion, "parse_status", result.Status, "diagnostic_count", len(result.Diagnostics), "diagnostic_codes", diagnosticCodes(result.Diagnostics))
}

func (server *Server) observeStaleAnalysisSnapshot(ctx context.Context, snapshot analysisSnapshot, reason string, started time.Time) {
	indexed := analysisStaleIndexSnapshot(snapshot, snapshot.state.Version, snapshot.context.version)
	server.observeStaleResult(ctx, "index", snapshot.state.URI, &indexed, reason, started)
}

func (server *Server) observeStaleParseResult(ctx context.Context, snapshot analysisSnapshot, result parser.ParseResult, reason string, started time.Time) {
	indexed := analysisStaleIndexSnapshot(snapshot, result.DocumentVersion, result.ContextVersion)
	server.observeStaleResult(ctx, "index", snapshot.state.URI, &indexed, reason, started)
}

func analysisStaleIndexSnapshot(snapshot analysisSnapshot, documentVersion, contextVersion int) index.IndexSnapshot {
	return index.IndexSnapshot{
		RootURI:        snapshot.state.URI,
		ContextVersion: contextVersion,
		Generation:     snapshot.generation,
		Epoch:          snapshot.epoch,
		Dependencies:   cloneHeaderDependencies(snapshot.headerDependencies),
		SourceVersion: index.SourceVersion{
			URI:             snapshot.state.URI,
			DocumentVersion: documentVersion,
			ContextVersion:  contextVersion,
			Generation:      snapshot.generation,
			Epoch:           snapshot.epoch,
		},
	}
}

func cloneHeaderDependencies(dependencies []index.HeaderDependency) []index.HeaderDependency {
	if len(dependencies) == 0 {
		return nil
	}
	cloned := make([]index.HeaderDependency, len(dependencies))
	for i, dependency := range dependencies {
		cloned[i] = dependency
		cloned[i].SearchRoots = append([]string(nil), dependency.SearchRoots...)
		cloned[i].Dependents = append([]string(nil), dependency.Dependents...)
	}
	return cloned
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
	server.InvalidateContextAndSchedule(context.Background(), nil, false)
}

func (server *Server) InvalidateContextAndSchedule(ctx context.Context, publisher diagnosticPublisher, asynchronous bool) {
	server.invalidateContext(ctx, publisher, asynchronous, nil)
}

func (server *Server) UpdateCompilationContext(compilationContext CompilationContext) {
	server.invalidateContext(context.Background(), nil, false, &compilationContext)
}

func (server *Server) invalidateContext(ctx context.Context, publisher diagnosticPublisher, asynchronous bool, compilationContext *CompilationContext) {
	server.freshnessMu.Lock()
	server.mu.Lock()
	if compilationContext != nil {
		server.compilationContext = cloneCompilationContext(*compilationContext)
	}
	server.contextVersion++
	server.analysisEpoch++
	for uri := range server.analysis {
		server.analysis[uri]++
	}
	for _, queue := range server.analysisQueues {
		queue.pending = nil
	}
	server.mu.Unlock()
	server.freshnessMu.Unlock()
	if server.symbolIndex != nil {
		server.symbolIndex.Clear()
	}
	for _, state := range server.documents.All() {
		server.scheduleAnalysis(ctx, publisher, state, asynchronous)
	}
}

func defaultCompilationContext() CompilationContext {
	return CompilationContext{
		LanguageStandard:      "unknown",
		DriverKind:            "unknown",
		CompilerContextStatus: "partial",
		ArgumentProvenance:    "unknown",
		PredefinedMacros:      map[string]parser.MacroValue{},
		TargetProfile:         "unknown",
		PassKind:              "unknown",
	}
}

func cloneCompilationContext(value CompilationContext) CompilationContext {
	if value.PredefinedMacros == nil {
		value.PredefinedMacros = map[string]parser.MacroValue{}
	} else {
		macros := make(map[string]parser.MacroValue, len(value.PredefinedMacros))
		for name, macro := range value.PredefinedMacros {
			macros[name] = macro
		}
		value.PredefinedMacros = macros
	}
	return value
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
