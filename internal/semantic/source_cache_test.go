package semantic

import "testing"

func TestWorkspaceSourceCacheUsesSourceVersionAndContextVersionKeys(t *testing.T) {
	cache := NewWorkspaceSourceCache()
	key := SourceCacheKey{Kind: SourceKindHeader, URIOrDescriptor: "file:///workspace/include/api.h", SourceVersion: "header-v1", ContextVersion: 2}
	cache.Put(key, SourceCacheValue{URI: key.URIOrDescriptor, Version: "header-v1", Status: "resolved", Values: map[string]string{"api": "int"}})
	if value, ok := cache.Get(key); !ok || value.Values["api"] != "int" || cache.ReadCount(key) != 1 {
		t.Fatalf("cached value = %+v, ok = %v, reads = %d", value, ok, cache.ReadCount(key))
	}
	if _, ok := cache.Get(SourceCacheKey{Kind: SourceKindHeader, URIOrDescriptor: key.URIOrDescriptor, SourceVersion: "header-v2", ContextVersion: 2}); ok {
		t.Fatal("different source version reused cached header")
	}
	cache.InvalidateSource(SourceKindHeader, key.URIOrDescriptor)
	if _, ok := cache.Get(key); ok {
		t.Fatal("invalidated header remained cached")
	}
	macroKey := SourceCacheKey{Kind: SourceKindMacro, URIOrDescriptor: "@workspace-macros", SourceVersion: "macro-v1", ContextVersion: 2}
	cache.Put(macroKey, SourceCacheValue{Status: "resolved"})
	cache.InvalidateContext(2)
	if _, ok := cache.Get(macroKey); ok {
		t.Fatal("context-invalidated macro remained cached")
	}
}
