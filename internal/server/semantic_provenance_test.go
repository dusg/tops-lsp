package server

import (
	"context"
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/semantic"
)

type semanticProvenanceSourceProvider struct {
	headers map[string]index.SourceDocument
	reads   map[string]int
}

func (provider *semanticProvenanceSourceProvider) OpenDocument(uri string) (index.SourceDocument, bool) {
	document, ok := provider.headers[uri]
	if ok {
		provider.reads[uri]++
	}
	return document, ok
}

func (provider *semanticProvenanceSourceProvider) IncludeRoots() ([]string, bool) {
	return []string{"/workspace/include"}, true
}

func (provider *semanticProvenanceSourceProvider) ResolveInclude(_, includeSpelling string) (index.SourceDocument, index.HeaderDependencyStatus) {
	for uri, document := range provider.headers {
		if includeSpelling == "api.h" && uri == "file:///workspace/include/api.h" {
			return document, index.HeaderDependencyResolved
		}
	}
	return index.SourceDocument{}, index.HeaderDependencyMissing
}

func TestServerSemanticProvenanceCachesReferencedHeaderByVersionAndContext(t *testing.T) {
	headerURI := "file:///workspace/include/api.h"
	provider := &semanticProvenanceSourceProvider{
		headers: map[string]index.SourceDocument{
			headerURI: {URI: headerURI, Version: 1, Text: "int api_value;", SourceVersion: index.SourceDigest("int api_value;")},
		},
		reads: make(map[string]int),
	}
	server := NewWithOptions(nil, ServerOptions{
		SourceProvider: provider,
		CommandCandidateProvider: newSemanticTestProvider(semanticTestProviderOptions{
			Macros:           map[string]string{"PROJECT": "1"},
			ToolchainRoots:   []string{"/opt/tops/include"},
			ToolchainMacros:  map[string]string{"__TOPS__": "1"},
			ToolchainVersion: "toolchain-v1",
			MacroVersion:     "macros-v1",
		}),
	})
	state, err := server.openDocument("file:///workspace/main.tops", "tops", 1, "#include \"api.h\"\nint use() { return api_value; }\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	server.parseAndPublish(context.Background(), nil, state)

	headerKey := semantic.SourceCacheKey{Kind: semantic.SourceKindHeader, URIOrDescriptor: headerURI, SourceVersion: index.SourceDigest("int api_value;"), ContextVersion: 1}
	if got := server.semanticSourceCache.ReadCount(headerKey); got != 1 {
		t.Fatalf("header cache reads after repeated analysis = %d, want 1", got)
	}

	server.InvalidateContext()
	server.WaitForAnalysis()
	headerKey.ContextVersion = 2
	if got := server.semanticSourceCache.ReadCount(headerKey); got != 1 {
		t.Fatalf("header cache reads after context invalidation = %d, want 1", got)
	}
}
