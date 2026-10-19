package semantic

import (
	"sync"

	"tops-lsp/internal/index"
)

type SourceKind string

const (
	SourceKindHeader SourceKind = "header"
	SourceKindMacro  SourceKind = "macro"
)

type SourceCacheKey struct {
	Kind            SourceKind
	URIOrDescriptor string
	SourceVersion   string
	ContextVersion  int
}

type SourceCacheValue struct {
	URI     string
	Version string
	Status  string
	Values  map[string]string
}

type WorkspaceSourceCache struct {
	mu      sync.RWMutex
	entries map[SourceCacheKey]SourceCacheValue
	reads   map[SourceCacheKey]int
}

func NewWorkspaceSourceCache() *WorkspaceSourceCache {
	return &WorkspaceSourceCache{entries: make(map[SourceCacheKey]SourceCacheValue), reads: make(map[SourceCacheKey]int)}
}

func (cache *WorkspaceSourceCache) Get(key SourceCacheKey) (SourceCacheValue, bool) {
	if cache == nil {
		return SourceCacheValue{}, false
	}
	key = normalizeSourceCacheKey(key)
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	value, ok := cache.entries[key]
	value.Values = CloneStringMap(value.Values)
	return value, ok
}

func (cache *WorkspaceSourceCache) Put(key SourceCacheKey, value SourceCacheValue) {
	if cache == nil {
		return
	}
	key = normalizeSourceCacheKey(key)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.entries[key] = SourceCacheValue{URI: value.URI, Version: value.Version, Status: value.Status, Values: CloneStringMap(value.Values)}
	cache.reads[key]++
}

func (cache *WorkspaceSourceCache) ReadCount(key SourceCacheKey) int {
	if cache == nil {
		return 0
	}
	key = normalizeSourceCacheKey(key)
	cache.mu.RLock()
	defer cache.mu.RUnlock()
	return cache.reads[key]
}

func (cache *WorkspaceSourceCache) InvalidateSource(kind SourceKind, uriOrDescriptor string) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key := range cache.entries {
		if key.Kind == kind && key.URIOrDescriptor == normalizeCacheURI(uriOrDescriptor) {
			delete(cache.entries, key)
		}
	}
}

func (cache *WorkspaceSourceCache) InvalidateContext(contextVersion int) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key := range cache.entries {
		if key.ContextVersion == contextVersion {
			delete(cache.entries, key)
		}
	}
}

func (cache *WorkspaceSourceCache) InvalidateThroughContext(contextVersion int) {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key := range cache.entries {
		if key.ContextVersion <= contextVersion {
			delete(cache.entries, key)
		}
	}
}

func normalizeSourceCacheKey(key SourceCacheKey) SourceCacheKey {
	key.URIOrDescriptor = normalizeCacheURI(key.URIOrDescriptor)
	return key
}

func normalizeCacheURI(value string) string {
	if value == "" || value[0] == '@' {
		return value
	}
	return index.NormalizeSourceURI(value)
}
