package server

import (
	"context"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/semantic"
)

type serverSourceProvider struct {
	documents      *document.Store
	delegate       index.SourceProvider
	cacheMu        sync.RWMutex
	sourceVersions map[string]string
	headerStore    *index.HeaderSnapshotStore
}

func (provider *serverSourceProvider) GetHeaderSnapshot(uri, sourceVersion string, contextVersion int) (index.IndexSnapshot, bool) {
	return provider.headerStore.Get(uri, sourceVersion, contextVersion)
}

func (provider *serverSourceProvider) PutHeaderSnapshot(snapshot index.IndexSnapshot) {
	provider.headerStore.Put(snapshot)
}

func (provider *serverSourceProvider) clearOpenedDocument(uri string) {
	if clearer, ok := provider.delegate.(index.OpenDocumentOverlay); ok {
		clearer.ClearOpenDocument(uri)
	}
	provider.cacheMu.Lock()
	delete(provider.sourceVersions, uri)
	provider.headerStore.Invalidate(uri)
	provider.cacheMu.Unlock()
}

func (provider *serverSourceProvider) OpenDocument(uri string) (index.SourceDocument, bool) {
	if state, ok := provider.documents.Get(uri); ok {
		source := index.SourceDocument{URI: state.URI, Text: state.Text, Version: state.Version, SourceVersion: index.SourceDigest(state.Text)}
		provider.syncOpenedDocument(source)
		return source, true
	}
	source, ok := provider.delegate.OpenDocument(uri)
	if !ok {
		return index.SourceDocument{}, false
	}
	if source.URI == "" {
		source.URI = uri
	}
	if source.Version == 0 {
		source.Version = 1
	}
	if source.SourceVersion == "" {
		source.SourceVersion = index.SourceDigest(source.Text)
	}
	provider.syncOpenedDocument(source)
	return source, true
}

func (provider *serverSourceProvider) syncOpenedDocument(source index.SourceDocument) {
	if isHeaderURI(source.URI) {
		provider.cacheMu.Lock()
		previous, hadPrevious := provider.sourceVersions[source.URI]
		provider.sourceVersions[source.URI] = source.SourceVersion
		if !hadPrevious || previous != source.SourceVersion {
			provider.headerStore.Invalidate(source.URI)
		}
		provider.cacheMu.Unlock()
	}
	local, ok := provider.delegate.(interface{ SetOpenDocument(index.SourceDocument) })
	if !ok {
		return
	}
	local.SetOpenDocument(source)
}

func (provider *serverSourceProvider) currentSourceVersion(uri string) (index.SourceVersion, bool) {
	if state, ok := provider.documents.Get(uri); ok {
		return index.SourceVersion{URI: state.URI, DocumentVersion: state.Version, SourceVersion: index.SourceDigest(state.Text)}, true
	}
	provider.cacheMu.RLock()
	sourceVersion, ok := provider.sourceVersions[uri]
	provider.cacheMu.RUnlock()
	if ok && sourceVersion != "" {
		return index.SourceVersion{URI: uri, SourceVersion: sourceVersion}, true
	}
	source, ok := provider.delegate.OpenDocument(uri)
	if !ok {
		return index.SourceVersion{}, false
	}
	if source.URI == "" {
		source.URI = uri
	}
	if source.SourceVersion == "" {
		source.SourceVersion = index.SourceDigest(source.Text)
	}
	return index.SourceVersion{URI: source.URI, DocumentVersion: source.Version, SourceVersion: source.SourceVersion}, true
}

func (server *Server) syncSourceDocument(state document.DocumentState) {
	if state.LanguageID != "tops-header" && !isHeaderURI(state.URI) {
		return
	}
	provider, ok := server.sourceProvider.(*serverSourceProvider)
	if !ok {
		return
	}
	provider.syncOpenedDocument(index.SourceDocument{URI: state.URI, Text: state.Text, Version: state.Version, SourceVersion: index.SourceDigest(state.Text)})
	if server.semanticSourceCache != nil {
		server.semanticSourceCache.InvalidateSource(semantic.SourceKindHeader, state.URI)
	}
}

func (provider *serverSourceProvider) IncludeRoots() ([]string, bool) {
	return provider.delegate.IncludeRoots()
}

func (provider *serverSourceProvider) ResolveAttribute(requesterURI, spelling string) (index.SymbolLocation, bool) {
	return provider.ResolveAttributeSource(index.SourceResolutionContext{RequesterURI: requesterURI}, spelling)
}

func (provider *serverSourceProvider) ResolveBuiltin(requesterURI, spelling string) (index.SymbolLocation, bool) {
	return provider.ResolveBuiltinSource(index.SourceResolutionContext{RequesterURI: requesterURI}, spelling)
}

func (provider *serverSourceProvider) ResolveAttributeSource(context index.SourceResolutionContext, spelling string) (index.SymbolLocation, bool) {
	if resolver, ok := provider.delegate.(index.VerifiedAttributeSourceResolver); ok {
		source, found := resolver.ResolveAttributeSource(context, spelling)
		return provider.completeSourceLocation(source, context), found
	}
	resolver, ok := provider.delegate.(index.AttributeSourceResolver)
	if !ok {
		return index.SymbolLocation{}, false
	}
	source, found := resolver.ResolveAttribute(context.RequesterURI, spelling)
	if !found || !index.HasCompleteSourceProvenance(context, source) {
		return index.SymbolLocation{}, false
	}
	return provider.completeSourceLocation(source, context), found
}

func (provider *serverSourceProvider) ResolveBuiltinSource(context index.SourceResolutionContext, spelling string) (index.SymbolLocation, bool) {
	if resolver, ok := provider.delegate.(index.VerifiedBuiltinSourceResolver); ok {
		source, found := resolver.ResolveBuiltinSource(context, spelling)
		return provider.completeSourceLocation(source, context), found
	}
	resolver, ok := provider.delegate.(index.BuiltinSourceResolver)
	if !ok {
		return index.SymbolLocation{}, false
	}
	source, found := resolver.ResolveBuiltin(context.RequesterURI, spelling)
	if !found || !index.HasCompleteSourceProvenance(context, source) {
		return index.SymbolLocation{}, false
	}
	return provider.completeSourceLocation(source, context), found
}

func (provider *serverSourceProvider) completeSourceLocation(source index.SymbolLocation, context index.SourceResolutionContext) index.SymbolLocation {
	if source.URI == "" {
		return index.SymbolLocation{}
	}
	document, ok := provider.OpenDocument(source.URI)
	if !ok {
		return index.SymbolLocation{}
	}
	source.SourceVersion.URI = document.URI
	source.SourceVersion.DocumentVersion = document.Version
	source.SourceVersion.ContextVersion = context.ContextVersion
	source.SourceVersion.SourceVersion = document.SourceVersion
	return source
}

func (provider *serverSourceProvider) ResolveInclude(requesterURI, includeSpelling string) (index.SourceDocument, index.HeaderDependencyStatus) {
	return provider.ResolveIncludeForm(requesterURI, includeSpelling, index.IncludeFormQuoted)
}

func (provider *serverSourceProvider) ResolveIncludeForm(requesterURI, includeSpelling string, form index.IncludeForm) (index.SourceDocument, index.HeaderDependencyStatus) {
	includeSpelling = strings.Trim(strings.TrimSpace(includeSpelling), "<>\"")
	if state, ok, ambiguous := provider.openedInclude(requesterURI, includeSpelling, form); ok {
		source := index.SourceDocument{URI: state.URI, Text: state.Text, Version: state.Version, SourceVersion: index.SourceDigest(state.Text)}
		provider.syncOpenedDocument(source)
		return source, index.HeaderDependencyResolved
	} else if ambiguous {
		return index.SourceDocument{}, index.HeaderDependencyUnresolved
	}
	var source index.SourceDocument
	var status index.HeaderDependencyStatus
	if resolver, ok := provider.delegate.(index.IncludeFormResolver); ok {
		source, status = resolver.ResolveIncludeForm(requesterURI, includeSpelling, form)
	} else {
		source, status = provider.delegate.ResolveInclude(requesterURI, includeSpelling)
	}
	if source.URI != "" {
		if state, ok := provider.documents.Get(source.URI); ok {
			resolved := index.SourceDocument{URI: state.URI, Text: state.Text, Version: state.Version, SourceVersion: index.SourceDigest(state.Text)}
			provider.syncOpenedDocument(resolved)
			return resolved, index.HeaderDependencyResolved
		}
	}
	return source, status
}

func (provider *serverSourceProvider) openedInclude(requesterURI, includeSpelling string, form index.IncludeForm) (document.DocumentState, bool, bool) {
	states := provider.documents.All()
	roots, rootsAvailable := provider.IncludeRoots()
	for _, candidateURI := range includeCandidateURIs(requesterURI, includeSpelling, form, roots) {
		for _, state := range states {
			if state.URI == candidateURI {
				return state, true, false
			}
		}
	}
	matches := make([]document.DocumentState, 0, 1)
	for _, state := range states {
		if filepath.Base(uriFilePath(state.URI)) == includeSpelling {
			matches = append(matches, state)
		}
	}
	if len(matches) == 1 && (!rootsAvailable || len(roots) == 0) {
		return matches[0], true, false
	}
	return document.DocumentState{}, false, len(matches) > 1
}

func includeCandidateURIs(requesterURI, includeSpelling string, form index.IncludeForm, roots []string) []string {
	if includeSpelling == "" {
		return nil
	}
	candidates := make([]string, 0, len(roots)+1)
	if filepath.IsAbs(includeSpelling) {
		return []string{fileURI(filepath.Clean(includeSpelling))}
	}
	if form != index.IncludeFormAngle {
		requesterPath := uriFilePath(requesterURI)
		if requesterPath != "" {
			candidates = append(candidates, fileURI(filepath.Clean(filepath.Join(filepath.Dir(requesterPath), includeSpelling))))
		}
	}
	for _, root := range roots {
		candidates = append(candidates, fileURI(filepath.Clean(filepath.Join(root, includeSpelling))))
	}
	result := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func uriFilePath(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Scheme == "file" {
		path, err := url.PathUnescape(parsed.Path)
		if err == nil {
			return filepath.FromSlash(path)
		}
	}
	return filepath.FromSlash(strings.TrimPrefix(uri, "file://"))
}

func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func (server *Server) commitIndexSnapshot(snapshot index.IndexSnapshot) bool {
	if server.symbolIndex == nil || snapshot.RootURI == "" {
		return false
	}
	server.freshnessMu.Lock()
	defer server.freshnessMu.Unlock()
	server.mu.RLock()
	state, documentOpen := server.documents.Get(snapshot.RootURI)
	contextVersion := int(server.contextVersion)
	epoch := server.analysisEpoch
	generation := server.analysis[snapshot.RootURI]
	server.mu.RUnlock()
	if !documentOpen || state.Version != snapshot.SourceVersion.DocumentVersion || snapshot.ContextVersion != contextVersion {
		return false
	}
	if snapshot.Epoch != 0 && snapshot.Epoch != epoch {
		return false
	}
	if snapshot.Generation != 0 && snapshot.Generation != generation {
		return false
	}
	if !server.symbolIndex.Replace(snapshot) {
		return false
	}
	return true
}

func (server *Server) currentIndexSnapshot(uri string) (index.IndexSnapshot, bool) {
	if server.symbolIndex == nil {
		return index.IndexSnapshot{}, false
	}
	return server.symbolIndex.Get(uri)
}

func (server *Server) currentDocumentVersion(uri string) int {
	if state, ok := server.documents.Get(uri); ok {
		return state.Version
	}
	return 0
}

func (server *Server) invalidateHeaderDependents(ctx context.Context, publisher diagnosticPublisher, headerURI string, asynchronous bool) {
	if server.symbolIndex == nil {
		return
	}
	server.freshnessMu.Lock()
	stale := server.markHeaderDependentsStaleLocked(headerURI)
	server.freshnessMu.Unlock()
	for _, uri := range stale {
		if state, ok := server.documents.Get(uri); ok {
			if err := server.clearPublishedDiagnostics(ctx, publisher, uri, state.Version); err != nil {
				server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "header-invalidation", "error_kind", "publisher_write_failed")
				server.reportFatal(err)
			}
			server.scheduleAnalysis(ctx, publisher, state, asynchronous)
		}
	}
}

func (server *Server) requeueOpenDocuments(ctx context.Context, publisher diagnosticPublisher, asynchronous bool) {
	for _, state := range server.documents.All() {
		server.scheduleAnalysis(ctx, publisher, state, asynchronous)
	}
}

func (server *Server) markHeaderDependentsStale(headerURI string) {
	server.freshnessMu.Lock()
	defer server.freshnessMu.Unlock()
	server.markHeaderDependentsStaleLocked(headerURI)
}

func (server *Server) markHeaderDependentsStaleLocked(headerURI string) []string {
	if server.symbolIndex == nil {
		return nil
	}
	if provider, ok := server.sourceProvider.(*serverSourceProvider); ok {
		provider.invalidateHeaderSnapshots(headerURI)
	}
	if server.semanticSourceCache != nil {
		server.semanticSourceCache.InvalidateSource(semantic.SourceKindHeader, headerURI)
	}
	stale := server.symbolIndex.MarkHeaderStale(headerURI)
	for _, uri := range stale {
		server.semanticStore.Remove(uri)
	}
	return stale
}

func (provider *serverSourceProvider) invalidateHeaderSnapshots(headerURI string) {
	provider.cacheMu.Lock()
	defer provider.cacheMu.Unlock()
	provider.headerStore.Invalidate(headerURI)
}

func isHeaderURI(uri string) bool {
	lower := strings.ToLower(uri)
	return strings.HasSuffix(lower, ".h") || strings.HasSuffix(lower, ".hh") || strings.HasSuffix(lower, ".hpp") || strings.HasSuffix(lower, ".hxx")
}
