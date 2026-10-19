package server

import (
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/semantic"
)

func (server *Server) readSemanticHeader(key semantic.SourceCacheKey) (semantic.SourceCacheValue, error) {
	uri := index.NormalizeSourceURI(key.URIOrDescriptor)
	if uri == "" {
		return semantic.SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "unresolved"}, nil
	}
	server.mu.RLock()
	provider := server.sourceProvider
	server.mu.RUnlock()
	if provider == nil {
		return semantic.SourceCacheValue{URI: uri, Version: key.SourceVersion, Status: "unresolved"}, nil
	}
	document, ok := provider.OpenDocument(uri)
	if !ok {
		return semantic.SourceCacheValue{URI: uri, Version: key.SourceVersion, Status: "missing"}, nil
	}
	version := document.SourceVersion
	if version == "" {
		version = index.SourceDigest(document.Text)
	}
	status := "resolved"
	if key.SourceVersion != "" && version != key.SourceVersion {
		status = "stale"
	}
	return semantic.SourceCacheValue{URI: uri, Version: version, Status: status}, nil
}

func (server *Server) readSemanticMacros(key semantic.SourceCacheKey) (semantic.SourceCacheValue, error) {
	cacheKey := semanticMacroSourceKey(key.Kind, key.URIOrDescriptor, key.SourceVersion)
	server.mu.RLock()
	value, ok := server.semanticMacroSources[cacheKey]
	server.mu.RUnlock()
	if !ok {
		return semantic.SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "unresolved"}, nil
	}
	return value, nil
}

func (server *Server) registerSemanticMacroSource(kind semantic.SourceKind, descriptor, version string, values map[string]string) {
	if server == nil || kind != semantic.SourceKindMacro || descriptor == "" {
		return
	}
	status := "resolved"
	if len(values) == 0 {
		status = "unresolved"
	}
	value := semantic.SourceCacheValue{URI: descriptor, Version: version, Status: status, Values: semantic.CloneStringMap(values)}
	key := semanticMacroSourceKey(kind, descriptor, version)
	server.mu.Lock()
	server.semanticMacroSources[key] = value
	server.mu.Unlock()
}

func (server *Server) loadSemanticMacroSource(kind semantic.SourceKind, descriptor, version string) {
	if server == nil || server.semanticSourceCache == nil || descriptor == "" {
		return
	}
	_, _ = server.semanticSourceAdapter.Load(semantic.SourceCacheKey{Kind: kind, URIOrDescriptor: descriptor, SourceVersion: version, ContextVersion: server.currentContextVersion()})
}

func (server *Server) loadSemanticProvenance(context *semantic.SemanticContext) {
	if server == nil || context == nil || server.semanticSourceCache == nil {
		return
	}
	seen := make(map[string]struct{}, len(context.HeaderVersions))
	for entryIndex := range context.HeaderVersions {
		entry := &context.HeaderVersions[entryIndex]
		uri := index.NormalizeSourceURI(entry.URI)
		if uri == "" {
			continue
		}
		entry.URI = uri
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		version := entry.Version
		if version == "" {
			version = "status:" + string(entry.Status)
		}
		value, err := server.semanticSourceAdapter.Load(semantic.SourceCacheKey{Kind: semantic.SourceKindHeader, URIOrDescriptor: uri, SourceVersion: version, ContextVersion: context.SemanticContextVersion})
		if err != nil || value.Status == "resolved" {
			continue
		}
		if entry.Status == "" || entry.Status == "resolved" {
			entry.Status = semantic.SourceVersionStatus(value.Status)
			entry.Version = ""
		}
	}
	for entryIndex := range context.SourceVersions {
		entry := &context.SourceVersions[entryIndex]
		if entry.Status == "" || entry.URI == context.DocumentURI {
			continue
		}
		for _, header := range context.HeaderVersions {
			if index.NormalizeSourceURI(header.URI) == index.NormalizeSourceURI(entry.URI) {
				*entry = header
				break
			}
		}
	}
	if fingerprint, err := semantic.AggregateSourceVersion(context.DocumentURI, context.SourceVersions); err == nil {
		context.SourceVersionFingerprint = fingerprint
	}
}

func semanticToolchainMacroDescriptor(provenance semantic.ToolchainProvenance) string {
	return strings.Join([]string{"@toolchain-macros", provenance.Identity, provenance.Version, provenance.MacroVersion}, "\x00")
}

func semanticMacroSourceKey(kind semantic.SourceKind, descriptor, version string) string {
	return string(kind) + "\x00" + descriptor + "\x00" + version
}

func mergeSemanticStringMaps(left, right map[string]string) map[string]string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	result := semantic.CloneStringMap(left)
	if result == nil {
		result = make(map[string]string, len(right))
	}
	for key, value := range right {
		if _, exists := result[key]; !exists {
			result[key] = value
		}
	}
	return result
}

func mergeSemanticStrings(left, right []string) []string {
	result := append([]string(nil), left...)
	seen := make(map[string]struct{}, len(result)+len(right))
	for _, value := range result {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
