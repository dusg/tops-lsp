package index

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"tops-lsp/internal/parser"
)

type SourceDocument struct {
	URI           string
	Text          string
	Version       int
	SourceVersion string
}

type SourceProvider interface {
	OpenDocument(uri string) (SourceDocument, bool)
	IncludeRoots() ([]string, bool)
	ResolveInclude(requesterURI, includeSpelling string) (SourceDocument, HeaderDependencyStatus)
}

type OpenDocumentOverlay interface {
	ClearOpenDocument(uri string)
}

type SourceResolutionContext struct {
	RequesterURI      string
	ContextVersion    int
	TargetProfile     string
	PassKind          string
	MacroContext      map[string]parser.MacroValue
	AllowedSourceURIs []string
}

type IncludeFormResolver interface {
	ResolveIncludeForm(requesterURI, includeSpelling string, form IncludeForm) (SourceDocument, HeaderDependencyStatus)
}

type HeaderSnapshotCache interface {
	GetHeaderSnapshot(uri, sourceVersion string, contextVersion int) (IndexSnapshot, bool)
	PutHeaderSnapshot(snapshot IndexSnapshot)
}

func resolveInclude(provider SourceProvider, requesterURI, includeSpelling string, form IncludeForm) (SourceDocument, HeaderDependencyStatus) {
	if resolver, ok := provider.(IncludeFormResolver); ok {
		return resolver.ResolveIncludeForm(requesterURI, includeSpelling, form)
	}
	return provider.ResolveInclude(requesterURI, includeSpelling)
}

type AttributeSourceResolver interface {
	ResolveAttribute(requesterURI, spelling string) (SymbolLocation, bool)
}

type VerifiedAttributeSourceResolver interface {
	ResolveAttributeSource(context SourceResolutionContext, spelling string) (SymbolLocation, bool)
}

type BuiltinSourceResolver interface {
	ResolveBuiltin(requesterURI, spelling string) (SymbolLocation, bool)
}

type VerifiedBuiltinSourceResolver interface {
	ResolveBuiltinSource(context SourceResolutionContext, spelling string) (SymbolLocation, bool)
}

func HasCompleteSourceProvenance(context SourceResolutionContext, source SymbolLocation) bool {
	if context.RequesterURI == "" || context.ContextVersion <= 0 || context.TargetProfile == "" || context.TargetProfile == "unknown" || context.PassKind == "" || context.PassKind == "unknown" || context.MacroContext == nil || len(context.AllowedSourceURIs) == 0 {
		return false
	}
	if source.URI == "" || !IsHeaderURI(source.URI) {
		return false
	}
	normalizedSourceURI := normalizeSourceURI(source.URI)
	for _, allowedURI := range context.AllowedSourceURIs {
		if normalizedSourceURI != "" && normalizedSourceURI == normalizeSourceURI(allowedURI) {
			return true
		}
	}
	return false
}

type EmptySourceProvider struct{}

func (EmptySourceProvider) OpenDocument(string) (SourceDocument, bool) {
	return SourceDocument{}, false
}

func (EmptySourceProvider) IncludeRoots() ([]string, bool) {
	return nil, false
}

func (EmptySourceProvider) ResolveInclude(string, string) (SourceDocument, HeaderDependencyStatus) {
	return SourceDocument{}, HeaderDependencyUnresolved
}

type LocalSourceProvider struct {
	opened            map[string]SourceDocument
	initial           map[string]SourceDocument
	overlays          map[string]SourceDocument
	roots             []string
	trustedHeaderURIs map[string]struct{}
	cacheMu           sync.RWMutex
	headerStore       *HeaderSnapshotStore
}

func NewLocalSourceProvider(opened map[string]SourceDocument, roots []string) *LocalSourceProvider {
	return NewLocalSourceProviderWithOptions(opened, roots, LocalSourceProviderOptions{})
}

type LocalSourceProviderOptions struct {
	TrustedHeaderURIs []string
}

func NewLocalSourceProviderWithOptions(opened map[string]SourceDocument, roots []string, options LocalSourceProviderOptions) *LocalSourceProvider {
	openedCopy := make(map[string]SourceDocument, len(opened))
	for uri, document := range opened {
		if document.SourceVersion == "" {
			document.SourceVersion = SourceDigest(document.Text)
		}
		openedCopy[uri] = document
	}
	rootCopy := make([]string, 0, len(roots))
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			if absolute, err := filepath.Abs(root); err == nil {
				root = absolute
			}
		}
		rootCopy = append(rootCopy, root)
	}
	trustedHeaderURIs := make(map[string]struct{}, len(options.TrustedHeaderURIs))
	for _, uri := range options.TrustedHeaderURIs {
		if normalized := normalizeSourceURI(uri); normalized != "" {
			trustedHeaderURIs[normalized] = struct{}{}
		}
	}
	initialCopy := make(map[string]SourceDocument, len(openedCopy))
	for uri, document := range openedCopy {
		initialCopy[uri] = document
	}
	return &LocalSourceProvider{opened: openedCopy, initial: initialCopy, overlays: make(map[string]SourceDocument), roots: rootCopy, trustedHeaderURIs: trustedHeaderURIs, headerStore: NewHeaderSnapshotStore()}
}

func (provider *LocalSourceProvider) SetOpenDocument(document SourceDocument) {
	if provider == nil || document.URI == "" {
		return
	}
	document = normalizeSourceDocument(document, document.URI)
	provider.cacheMu.Lock()
	defer provider.cacheMu.Unlock()
	previous, hadPrevious := provider.overlays[document.URI]
	if !hadPrevious {
		previous, hadPrevious = provider.opened[document.URI]
	}
	provider.overlays[document.URI] = document
	if !hadPrevious || previous.Version != document.Version || previous.SourceVersion != document.SourceVersion || previous.Text != document.Text {
		provider.headerStore.Invalidate(document.URI)
	}
}

func (provider *LocalSourceProvider) ClearOpenDocument(uri string) {
	if provider == nil {
		return
	}
	provider.cacheMu.Lock()
	defer provider.cacheMu.Unlock()
	delete(provider.overlays, uri)
	if document, ok := provider.initial[uri]; ok {
		provider.opened[uri] = document
	} else {
		delete(provider.opened, uri)
	}
	provider.headerStore.Invalidate(uri)
}

func (provider *LocalSourceProvider) GetHeaderSnapshot(uri, sourceVersion string, contextVersion int) (IndexSnapshot, bool) {
	if provider == nil {
		return IndexSnapshot{}, false
	}
	return provider.headerStore.Get(uri, sourceVersion, contextVersion)
}

func (provider *LocalSourceProvider) PutHeaderSnapshot(snapshot IndexSnapshot) {
	if provider == nil {
		return
	}
	provider.headerStore.Put(snapshot)
}

// InvalidateHeaderSnapshots drops the cached snapshots for changedURI and every
// root URI that transitively depends on it.
func (provider *LocalSourceProvider) InvalidateHeaderSnapshots(changedURI string) {
	if provider == nil {
		return
	}
	provider.headerStore.Invalidate(changedURI)
}

func (provider *LocalSourceProvider) OpenDocument(uri string) (SourceDocument, bool) {
	if provider == nil {
		return SourceDocument{}, false
	}
	provider.cacheMu.RLock()
	defer provider.cacheMu.RUnlock()
	if document, ok := provider.overlays[uri]; ok {
		return document, true
	}
	document, ok := provider.opened[uri]
	return document, ok
}

func (provider *LocalSourceProvider) IncludeRoots() ([]string, bool) {
	if provider == nil || len(provider.roots) == 0 {
		return nil, false
	}
	return append([]string(nil), provider.roots...), true
}

func (provider *LocalSourceProvider) ResolveInclude(requesterURI, includeSpelling string) (SourceDocument, HeaderDependencyStatus) {
	return provider.ResolveIncludeForm(requesterURI, includeSpelling, IncludeFormQuoted)
}

func (provider *LocalSourceProvider) ResolveIncludeForm(requesterURI, includeSpelling string, form IncludeForm) (SourceDocument, HeaderDependencyStatus) {
	if provider == nil {
		return SourceDocument{}, HeaderDependencyUnresolved
	}
	includeSpelling = strings.Trim(strings.TrimSpace(includeSpelling), "<>\"")
	if includeSpelling == "" {
		return SourceDocument{}, HeaderDependencyUnresolved
	}
	paths := includeCandidatePaths(requesterURI, includeSpelling, provider.roots, form)
	for _, path := range paths {
		uri := fileURI(path)
		if document, ok := provider.OpenDocument(uri); ok {
			return normalizeSourceDocument(document, uri), HeaderDependencyResolved
		}
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return SourceDocument{URI: uri}, HeaderDependencyUnreadable
		}
		if info.IsDir() {
			return SourceDocument{URI: uri}, HeaderDependencyUnreadable
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return SourceDocument{URI: uri}, HeaderDependencyUnreadable
		}
		text := string(body)
		document := normalizeSourceDocument(SourceDocument{URI: uri, Text: text, Version: 1, SourceVersion: SourceDigest(text)}, uri)
		provider.cacheMu.Lock()
		provider.opened[uri] = document
		provider.cacheMu.Unlock()
		return document, HeaderDependencyResolved
	}
	openedMatches := make([]SourceDocument, 0, 1)
	provider.cacheMu.RLock()
	opened := make(map[string]SourceDocument, len(provider.opened))
	for uri, document := range provider.opened {
		opened[uri] = document
	}
	for uri, document := range provider.overlays {
		opened[uri] = document
	}
	provider.cacheMu.RUnlock()
	for uri, document := range opened {
		if filepath.Base(uriPath(uri)) == includeSpelling {
			openedMatches = append(openedMatches, normalizeSourceDocument(document, uri))
		}
	}
	if len(openedMatches) == 1 {
		return openedMatches[0], HeaderDependencyResolved
	}
	if len(openedMatches) > 1 {
		return SourceDocument{}, HeaderDependencyUnresolved
	}
	if len(provider.roots) == 0 {
		return SourceDocument{}, HeaderDependencyUnresolved
	}
	return SourceDocument{}, HeaderDependencyMissing
}

func (provider *LocalSourceProvider) ResolveAttributeSource(context SourceResolutionContext, spelling string) (SymbolLocation, bool) {
	return provider.resolveHeaderSymbol(context, spelling, SymbolKindAttribute)
}

func (provider *LocalSourceProvider) ResolveBuiltinSource(context SourceResolutionContext, spelling string) (SymbolLocation, bool) {
	return provider.resolveHeaderSymbol(context, spelling, SymbolKindBuiltin)
}

func (provider *LocalSourceProvider) ResolveBuiltin(requesterURI, spelling string) (SymbolLocation, bool) {
	return provider.ResolveBuiltinSource(SourceResolutionContext{RequesterURI: requesterURI}, spelling)
}

func (provider *LocalSourceProvider) resolveHeaderSymbol(context SourceResolutionContext, spelling string, kind SymbolKind) (SymbolLocation, bool) {
	if provider == nil || spelling == "" || context.TargetProfile == "" || context.TargetProfile == "unknown" || context.PassKind == "" || context.PassKind == "unknown" {
		return SymbolLocation{}, false
	}
	provider.cacheMu.RLock()
	opened := make(map[string]SourceDocument, len(provider.opened))
	for uri, document := range provider.opened {
		opened[uri] = document
	}
	for uri, document := range provider.overlays {
		opened[uri] = document
	}
	provider.cacheMu.RUnlock()
	uris := make([]string, 0, len(opened))
	for uri := range opened {
		if IsHeaderURI(uri) {
			uris = append(uris, uri)
		}
	}
	sort.Strings(uris)
	for _, uri := range uris {
		if _, trusted := provider.trustedHeaderURIs[uri]; !trusted {
			continue
		}
		allowed := false
		for _, allowedURI := range context.AllowedSourceURIs {
			if allowedURI == uri {
				allowed = true
				break
			}
		}
		if !allowed {
			continue
		}
		document := normalizeSourceDocument(opened[uri], uri)
		parseContext := parser.ParseContext{
			DocumentURI:           uri,
			LanguageID:            "tops-header",
			CompilerContextStatus: "resolved",
			TargetProfile:         context.TargetProfile,
			PassKind:              context.PassKind,
			PredefinedMacros:      cloneMacroContext(context.MacroContext),
			DocumentVersion:       document.Version,
			ContextVersion:        context.ContextVersion,
		}
		snapshot := Build(document.Text, parser.Parse(document.Text, parseContext), parseContext)
		for _, symbol := range snapshot.Symbols {
			matches := symbol.Name == spelling
			if !matches && kind == SymbolKindAttribute {
				matches = strings.Contains(symbol.Detail, spelling)
			}
			if !matches || symbol.Kind != kind {
				continue
			}
			locations := symbol.Definitions
			if len(locations) == 0 {
				locations = symbol.Declarations
			}
			if len(locations) == 0 {
				continue
			}
			location := locations[0]
			location.SourceVersion = SourceVersion{URI: uri, DocumentVersion: document.Version, ContextVersion: context.ContextVersion, SourceVersion: sourceVersionForDocument(document)}
			return location, true
		}
	}
	return SymbolLocation{}, false
}

// IsHeaderURI reports whether uri names a C/C++ header file.
func IsHeaderURI(uri string) bool {
	path := strings.ToLower(uriPath(uri))
	return strings.HasSuffix(path, ".h") || strings.HasSuffix(path, ".hh") || strings.HasSuffix(path, ".hpp") || strings.HasSuffix(path, ".hxx")
}

func includeCandidatePaths(requesterURI, includeSpelling string, roots []string, form IncludeForm) []string {
	paths := make([]string, 0, len(roots)+1)
	if filepath.IsAbs(includeSpelling) {
		return []string{filepath.Clean(includeSpelling)}
	}
	if form != IncludeFormAngle {
		if directory := filepath.Dir(uriPath(requesterURI)); directory != "." && directory != "" {
			paths = append(paths, filepath.Clean(filepath.Join(directory, includeSpelling)))
		}
	}
	for _, root := range roots {
		paths = append(paths, filepath.Clean(filepath.Join(root, includeSpelling)))
	}
	return uniquePaths(paths)
}

func uniquePaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

// URIPath converts a file:// URI back into a filesystem path.
func URIPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Scheme == "file" {
		if path, err := url.PathUnescape(parsed.Path); err == nil {
			return filepath.FromSlash(path)
		}
	}
	return filepath.FromSlash(strings.TrimPrefix(uri, "file://"))
}

func uriPath(uri string) string {
	return URIPath(uri)
}

// NormalizeSourceDocument fills in the URI, version and source digest defaults
// for a source document.
func NormalizeSourceDocument(document SourceDocument, uri string) SourceDocument {
	if document.URI == "" {
		document.URI = uri
	}
	if document.Version == 0 {
		document.Version = 1
	}
	if document.SourceVersion == "" {
		document.SourceVersion = SourceDigest(document.Text)
	}
	return document
}

func normalizeSourceDocument(document SourceDocument, uri string) SourceDocument {
	return NormalizeSourceDocument(document, uri)
}

// FileURI converts an absolute filesystem path into a file:// URI.
func FileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func fileURI(path string) string {
	return FileURI(path)
}

func normalizeSourceURI(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" {
		if parsed.Scheme == "file" {
			return fileURI(filepath.Clean(uriPath(value)))
		}
		return value
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return ""
	}
	return fileURI(filepath.Clean(absolute))
}

// NormalizeSourceURI returns the canonical URI used for source identity.
func NormalizeSourceURI(value string) string {
	return normalizeSourceURI(value)
}

func SourceDigest(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:12])
}
