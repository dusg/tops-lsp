package server

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
)

func (server *Server) observeIndex(ctx context.Context, snapshot index.IndexSnapshot, started ...time.Time) {
	if snapshot.IndexStatus != index.IndexStatusDegraded && snapshot.IndexStatus != index.IndexStatusPartial {
		return
	}
	server.log(ctx, slog.LevelWarn, "analysis_degraded",
		"document_id", logging.DocumentID(snapshot.RootURI),
		"operation", "index",
		"symbol_kind", indexSymbolKinds(snapshot),
		"document_version", snapshot.SourceVersion.DocumentVersion,
		"context_version", snapshot.ContextVersion,
		"source_version", snapshot.SourceVersion.SourceVersion,
		"header_versions", indexHeaderVersions(snapshot),
		"condition", indexConditionStates(snapshot),
		"index_status", snapshot.IndexStatus,
		"candidate_count", len(snapshot.Symbols),
		"diagnostic_code", indexDiagnosticCodes(snapshot),
		"error_kind", indexErrorKind(snapshot),
		"duration_ms", elapsedMilliseconds(started),
	)
}

func (server *Server) observeStaleIndex(ctx context.Context, uri string, oldVersion, newVersion int, reason string, started ...time.Time) {
	oldSnapshot := server.staleIndexSnapshot(uri, oldVersion)
	start := time.Now()
	if len(started) > 0 {
		start = started[0]
	}
	server.observeStaleResult(ctx, "index", uri, &oldSnapshot, reason, start)
}

func (server *Server) observeStaleResult(ctx context.Context, operation, uri string, snapshot *index.IndexSnapshot, reason string, started time.Time) {
	server.staleResultSuppressionCount.Add(1)
	oldDocumentVersion := 0
	oldContextVersion := 0
	oldHeaderVersions := ""
	oldSourceVersion := ""
	oldGeneration := uint64(0)
	oldEpoch := uint64(0)
	if snapshot != nil {
		oldDocumentVersion = snapshot.SourceVersion.DocumentVersion
		oldContextVersion = snapshot.ContextVersion
		oldHeaderVersions = indexHeaderVersions(*snapshot)
		oldSourceVersion = snapshot.SourceVersion.SourceVersion
		oldGeneration = snapshot.SourceVersion.Generation
		if oldGeneration == 0 {
			oldGeneration = snapshot.Generation
		}
		oldEpoch = snapshot.SourceVersion.Epoch
		if oldEpoch == 0 {
			oldEpoch = snapshot.Epoch
		}
	} else if current, ok := server.currentIndexSnapshot(uri); ok {
		oldDocumentVersion = current.SourceVersion.DocumentVersion
		oldContextVersion = current.ContextVersion
		oldHeaderVersions = indexHeaderVersions(current)
		oldSourceVersion = current.SourceVersion.SourceVersion
		oldGeneration = current.SourceVersion.Generation
		if oldGeneration == 0 {
			oldGeneration = current.Generation
		}
		oldEpoch = current.SourceVersion.Epoch
		if oldEpoch == 0 {
			oldEpoch = current.Epoch
		}
	}
	newDocumentVersion := server.currentDocumentVersion(uri)
	newContextVersion := server.currentContextVersion()
	newGeneration, newEpoch := server.currentAnalysisVersion(uri)
	newHeaderVersions := server.currentHeaderVersionsForSnapshot(uri, snapshot)
	newSourceVersion := ""
	if current, ok := server.currentIndexSnapshot(uri); ok {
		newSourceVersion = current.SourceVersion.SourceVersion
	}
	semanticProfileID := ""
	semanticContextStatus := ""
	semanticPassKind := ""
	semanticSourceVersion := ""
	semanticHeaderVersionSet := ""
	if server.semanticStore != nil {
		if current, ok := server.semanticStore.Get(uri); ok {
			semanticProfileID = current.ProfileID
			semanticContextStatus = string(current.ContextStatus)
			semanticPassKind = string(current.Context.Command.PassKind)
			semanticSourceVersion = current.SourceVersionFingerprint
			semanticHeaderVersionSet = semanticHeaderVersions(current)
		}
	}
	server.log(ctx, slog.LevelInfo, "stale_result",
		"operation", operation,
		"document_id", logging.DocumentID(uri),
		"old_document_version", oldDocumentVersion,
		"new_document_version", newDocumentVersion,
		"old_context_version", oldContextVersion,
		"new_context_version", newContextVersion,
		"old_generation", oldGeneration,
		"new_generation", newGeneration,
		"old_epoch", oldEpoch,
		"new_epoch", newEpoch,
		"old_header_versions", oldHeaderVersions,
		"new_header_versions", newHeaderVersions,
		"old_source_version", oldSourceVersion,
		"new_source_version", newSourceVersion,
		"semantic_profile_id", semanticProfileID,
		"semantic_context_status", semanticContextStatus,
		"semantic_pass_kind", semanticPassKind,
		"semantic_source_version", semanticSourceVersion,
		"semantic_header_versions", semanticHeaderVersionSet,
		"reason", reason,
		"duration_ms", time.Since(started).Milliseconds(),
	)
}

func semanticHeaderVersions(snapshot semantic.SemanticSnapshot) string {
	values := make([]string, 0, len(snapshot.HeaderVersions))
	for _, entry := range snapshot.HeaderVersions {
		if entry.URI == "" {
			continue
		}
		value := entry.Version
		if value == "" {
			value = "status:" + string(entry.Status)
		}
		values = append(values, logging.DocumentID(entry.URI)+":"+value)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func (server *Server) observeStaleResultOnce(ctx context.Context, operation, uri string, snapshot *index.IndexSnapshot, reason string, started time.Time) {
	if state, ok := ctx.Value(requestStateKey{}).(*requestState); ok && state != nil && !state.MarkStaleObserved() {
		return
	}
	server.observeStaleResult(ctx, operation, uri, snapshot, reason, started)
}

func (server *Server) observeNavigation(ctx context.Context, operation string, snapshot index.IndexSnapshot, resultCount int, started time.Time) {
	result := "success"
	if resultCount == 0 {
		result = "empty"
	}
	fields := server.navigationLogFields(operation, snapshot.RootURI, &snapshot)
	fields = append(fields,
		"result", result,
		"result_count", resultCount,
		"diagnostic_code", indexDiagnosticCodes(snapshot),
		"error_kind", indexErrorKind(snapshot),
		"duration_ms", time.Since(started).Milliseconds(),
	)
	server.log(ctx, slog.LevelDebug, "navigation_query", fields...)
}

// navigationLogFields builds the snapshot fields shared by navigation_query
// success and error logs.
func (server *Server) navigationLogFields(operation, uri string, snapshot *index.IndexSnapshot) []any {
	indexStatus := index.IndexStatusStale
	symbolKind := "unknown"
	profileID := ""
	passKind := ""
	documentVersion := 0
	contextVersion := server.currentContextVersion()
	sourceVersion := ""
	headerVersions := ""
	condition := ""
	candidateCount := 0
	if snapshot != nil {
		indexStatus = snapshot.IndexStatus
		symbolKind = indexSymbolKinds(*snapshot)
		profileID = snapshot.TargetProfile
		passKind = snapshot.PassKind
		documentVersion = snapshot.SourceVersion.DocumentVersion
		contextVersion = snapshot.ContextVersion
		sourceVersion = snapshot.SourceVersion.SourceVersion
		headerVersions = indexHeaderVersions(*snapshot)
		condition = indexConditionStates(*snapshot)
		candidateCount = len(snapshot.Symbols)
	}
	return []any{
		"operation", operation,
		"document_id", logging.DocumentID(uri),
		"symbol_kind", symbolKind,
		"index_status", indexStatus,
		"profile_id", profileID,
		"pass_kind", passKind,
		"document_version", documentVersion,
		"context_version", contextVersion,
		"source_version", sourceVersion,
		"header_versions", headerVersions,
		"condition", condition,
		"candidate_count", candidateCount,
	}
}

func (server *Server) observeNavigationError(ctx context.Context, operation, uri string, snapshot *index.IndexSnapshot, resultCount int, diagnosticCode, errorKind string, started time.Time) {
	if diagnosticCode == "stale_result" {
		server.observeStaleResultOnce(ctx, operation, uri, snapshot, errorKind, started)
		return
	}
	fields := server.navigationLogFields(operation, uri, snapshot)
	fields = append(fields,
		"result_count", resultCount,
		"diagnostic_code", diagnosticCode,
		"error_kind", errorKind,
		"duration_ms", time.Since(started).Milliseconds(),
	)
	server.log(ctx, slog.LevelDebug, "navigation_query", fields...)
}

func (server *Server) publishIndexDiagnostics(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState, snapshot index.IndexSnapshot) error {
	if publisher == nil {
		return nil
	}
	byURI, versions, err := indexDiagnosticSet(state, snapshot)
	if err != nil {
		return err
	}
	if _, ok := byURI[state.URI]; !ok {
		byURI[state.URI] = nil
		versions[state.URI] = state.Version
	}
	return server.publishDiagnosticSets(ctx, publisher, byURI, versions, snapshot.ContextVersion, state.URI)
}

func (server *Server) publishAnalysisDiagnosticsWithSemantic(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState, snapshot index.IndexSnapshot, result parser.ParseResult, semanticSnapshot *semantic.SemanticSnapshot) error {
	if publisher == nil {
		return nil
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if !server.analysisCurrentSnapshotLocked(analysisSnapshot{state: state, context: parserContextSnapshot{version: snapshot.ContextVersion}, generation: snapshot.SourceVersion.Generation, epoch: snapshot.Epoch}) {
		return nil
	}
	byURI, versions, err := indexDiagnosticSet(state, snapshot)
	if err != nil {
		return err
	}
	parserDiagnostics, err := parserDiagnosticsForState(state, result)
	if err != nil {
		return err
	}
	byURI[state.URI] = append(byURI[state.URI], parserDiagnostics...)
	if _, ok := versions[state.URI]; !ok {
		versions[state.URI] = state.Version
	}
	if semanticSnapshot != nil {
		semanticDiagnostics, err := semanticDiagnosticsForState(state, *semanticSnapshot)
		if err != nil {
			return err
		}
		byURI[state.URI] = append(byURI[state.URI], semanticDiagnostics...)
	}
	for uri := range byURI {
		byURI[uri] = dedupeAndSortDiagnostics(byURI[uri])
	}
	return server.publishDiagnosticSets(ctx, publisher, byURI, versions, snapshot.ContextVersion, state.URI)
}

func indexDiagnosticSet(state document.DocumentState, snapshot index.IndexSnapshot) (map[string][]protocol.Diagnostic, map[string]int, error) {
	byURI := make(map[string][]protocol.Diagnostic)
	versions := make(map[string]int)
	for _, sourceDiagnostic := range snapshot.Diagnostics {
		sourceURI := sourceDiagnostic.SourceURI
		if sourceURI == "" {
			sourceURI = state.URI
		}
		text := state.Text
		if sourceURI != state.URI {
			var ok bool
			text, ok = snapshot.Sources[sourceURI]
			if !ok {
				return nil, nil, position.ErrInvalidRange
			}
		}
		mapper := position.New(text)
		rangeValue, err := mapper.Range(sourceDiagnostic.Range.Start, sourceDiagnostic.Range.End)
		if err != nil {
			return nil, nil, err
		}
		condition := sourceDiagnostic.Condition
		if condition.State == "" && sourceDiagnostic.ConditionalState != "" {
			condition = index.ConditionalProvenance{SourceURI: sourceURI, State: sourceDiagnostic.ConditionalState}
		}
		provenance, err := conditionalProvenance(condition, snapshot.RootURI, func(uri string) (string, bool) {
			text, ok := snapshot.Sources[uri]
			return text, ok
		}, true)
		if err != nil {
			return nil, nil, err
		}
		byURI[sourceURI] = append(byURI[sourceURI], protocol.Diagnostic{
			Range:    protocolRange(rangeValue),
			Severity: protocol.DiagnosticSeverityInfo,
			Code:     sourceDiagnostic.Code,
			Source:   "tops-lsp",
			Message:  sourceDiagnostic.Message,
			Data: &protocol.DiagnosticData{
				ConditionalState:      string(condition.State),
				ConditionalProvenance: provenance,
				ContextVersion:        snapshot.ContextVersion,
				SourceVersion:         sourceDiagnostic.SourceVersion.SourceVersion,
				Reason:                sourceDiagnostic.Reason,
				ErrorKind:             sourceDiagnostic.Reason,
			},
		})
		version := sourceDiagnostic.SourceVersion.DocumentVersion
		if version == 0 {
			version = state.Version
		}
		versions[sourceURI] = version
	}
	return byURI, versions, nil
}

func (server *Server) publishDiagnosticSets(ctx context.Context, publisher diagnosticPublisher, byURI map[string][]protocol.Diagnostic, versions map[string]int, contextVersion int, ownerURI string) error {
	server.diagnosticMu.Lock()
	defer server.diagnosticMu.Unlock()
	for uri := range server.publishedDiagnosticURIsByDocument[ownerURI] {
		if _, ok := byURI[uri]; !ok {
			byURI[uri] = nil
		}
	}
	uris := make([]string, 0, len(byURI))
	for uri := range byURI {
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	for _, uri := range uris {
		version, hasVersion := versions[uri]
		var versionPointer *int
		if hasVersion {
			versionPointer = &version
		}
		if err := publisher.Publish(ctx, protocol.PublishDiagnosticsParams{URI: uri, Version: versionPointer, ContextVersion: &contextVersion, Diagnostics: byURI[uri]}); err != nil {
			return err
		}
	}
	current := make(map[string]struct{}, len(byURI))
	semanticURIs := make(map[string]struct{})
	for uri := range byURI {
		current[uri] = struct{}{}
		for _, diagnostic := range byURI[uri] {
			if strings.HasPrefix(diagnostic.Code, "tops-semantic-") {
				semanticURIs[uri] = struct{}{}
				break
			}
		}
	}
	server.publishedDiagnosticURIsByDocument[ownerURI] = current
	server.semanticDiagnosticURIsByDocument[ownerURI] = semanticURIs
	return nil
}

func (server *Server) clearPublishedDiagnostics(ctx context.Context, publisher diagnosticPublisher, ownerURI string, version int) error {
	if ownerURI == "" {
		return nil
	}
	if publisher == nil {
		server.diagnosticMu.Lock()
		delete(server.publishedDiagnosticURIsByDocument, ownerURI)
		delete(server.semanticDiagnosticURIsByDocument, ownerURI)
		server.diagnosticMu.Unlock()
		return nil
	}
	server.diagnosticMu.Lock()
	semanticURIs := server.semanticDiagnosticURIsByDocument[ownerURI]
	server.diagnosticMu.Unlock()
	if len(semanticURIs) == 0 {
		return nil
	}
	return server.clearDocumentDiagnostics(ctx, publisher, ownerURI, version)
}

// clearDocumentDiagnostics always publishes an empty diagnostic array so the
// client drops every previously published problem for the document. This is a
// cleanup notification: tests and clients must not treat it as the analysis
// result for the same version, because both may share uri and version.
func (server *Server) clearDocumentDiagnostics(ctx context.Context, publisher diagnosticPublisher, ownerURI string, version int) error {
	if ownerURI == "" {
		return nil
	}
	if publisher == nil {
		server.diagnosticMu.Lock()
		delete(server.publishedDiagnosticURIsByDocument, ownerURI)
		delete(server.semanticDiagnosticURIsByDocument, ownerURI)
		server.diagnosticMu.Unlock()
		return nil
	}
	server.diagnosticMu.Lock()
	delete(server.semanticDiagnosticURIsByDocument, ownerURI)
	server.diagnosticMu.Unlock()
	byURI := map[string][]protocol.Diagnostic{ownerURI: nil}
	versions := make(map[string]int)
	if version > 0 {
		versions[ownerURI] = version
	}
	return server.publishDiagnosticSets(ctx, publisher, byURI, versions, server.currentContextVersion(), ownerURI)
}

func (server *Server) clearAllPublishedDiagnostics(ctx context.Context, publisher diagnosticPublisher) error {
	server.diagnosticMu.Lock()
	owners := make([]string, 0, len(server.publishedDiagnosticURIsByDocument))
	for ownerURI := range server.publishedDiagnosticURIsByDocument {
		owners = append(owners, ownerURI)
	}
	server.diagnosticMu.Unlock()
	sort.Strings(owners)
	if publisher == nil {
		server.diagnosticMu.Lock()
		server.publishedDiagnosticURIsByDocument = make(map[string]map[string]struct{})
		server.semanticDiagnosticURIsByDocument = make(map[string]map[string]struct{})
		server.diagnosticMu.Unlock()
		return nil
	}
	for _, ownerURI := range owners {
		if err := server.clearDocumentDiagnostics(ctx, publisher, ownerURI, server.currentDocumentVersion(ownerURI)); err != nil {
			return err
		}
	}
	return nil
}

func elapsedMilliseconds(started []time.Time) int64 {
	if len(started) == 0 || started[0].IsZero() {
		return 0
	}
	return time.Since(started[0]).Milliseconds()
}

func indexErrorKind(snapshot index.IndexSnapshot) string {
	for _, diagnostic := range snapshot.Diagnostics {
		if diagnostic.Reason != "" {
			return diagnostic.Reason
		}
	}
	if snapshot.IndexStatus == index.IndexStatusPartial {
		return "partial-index"
	}
	if snapshot.IndexStatus == index.IndexStatusDegraded {
		return "degraded-index"
	}
	return "none"
}

func indexSymbolKinds(snapshot index.IndexSnapshot) string {
	kinds := make(map[index.SymbolKind]struct{})
	for _, symbol := range snapshot.Symbols {
		kinds[symbol.Kind] = struct{}{}
	}
	values := make([]string, 0, len(kinds))
	for kind := range kinds {
		values = append(values, string(kind))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func indexConditionStates(snapshot index.IndexSnapshot) string {
	states := make(map[string]struct{})
	for _, symbol := range snapshot.Symbols {
		states[string(symbol.Condition.State)] = struct{}{}
	}
	for _, reference := range snapshot.References {
		states[string(reference.Condition.State)] = struct{}{}
	}
	values := make([]string, 0, len(states))
	for state := range states {
		values = append(values, state)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func indexHeaderVersions(snapshot index.IndexSnapshot) string {
	versions := make([]string, 0)
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI == "" || dependency.SourceVersion.SourceVersion == "" {
			continue
		}
		versions = append(versions, logging.DocumentID(dependency.TargetURI)+":"+dependency.SourceVersion.SourceVersion)
	}
	sort.Strings(versions)
	return strings.Join(versions, ",")
}

func (server *Server) currentHeaderVersions(uri string) string {
	snapshot, ok := server.currentIndexSnapshot(uri)
	if !ok {
		return ""
	}
	if provider, ok := server.sourceProvider.(*serverSourceProvider); ok {
		if versions := provider.currentHeaderVersions(snapshot.Dependencies); versions != "" {
			return versions
		}
	}
	return indexHeaderVersions(snapshot)
}

func (server *Server) currentHeaderVersionsForSnapshot(uri string, snapshot *index.IndexSnapshot) string {
	if snapshot != nil {
		if provider, ok := server.sourceProvider.(*serverSourceProvider); ok {
			if versions := provider.currentHeaderVersions(snapshot.Dependencies); versions != "" {
				return versions
			}
		}
	}
	return server.currentHeaderVersions(uri)
}

func (server *Server) currentAnalysisVersion(uri string) (uint64, uint64) {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.analysis[uri], server.analysisEpoch
}

func (server *Server) staleIndexSnapshot(uri string, oldVersion int) index.IndexSnapshot {
	if current, ok := server.currentIndexSnapshot(uri); ok {
		current.SourceVersion.DocumentVersion = oldVersion
		return current
	}
	contextVersion := server.currentContextVersion()
	generation, epoch := server.currentAnalysisVersion(uri)
	return index.IndexSnapshot{
		RootURI:        uri,
		ContextVersion: contextVersion,
		Generation:     generation,
		Epoch:          epoch,
		SourceVersion: index.SourceVersion{
			URI:             uri,
			DocumentVersion: oldVersion,
			ContextVersion:  contextVersion,
			Generation:      generation,
			Epoch:           epoch,
		},
	}
}

func (provider *serverSourceProvider) currentHeaderVersions(dependencies []index.HeaderDependency) string {
	versions := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		sourceVersion, ok := provider.currentSourceVersion(dependency.TargetURI)
		if !ok || sourceVersion.SourceVersion == "" {
			continue
		}
		versions = append(versions, logging.DocumentID(dependency.TargetURI)+":"+sourceVersion.SourceVersion)
	}
	sort.Strings(versions)
	return strings.Join(versions, ",")
}

func indexDiagnosticCodes(snapshot index.IndexSnapshot) string {
	codes := make([]string, 0, len(snapshot.Diagnostics))
	for _, diagnostic := range snapshot.Diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	sort.Strings(codes)
	return strings.Join(codes, ",")
}
