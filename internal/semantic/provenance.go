package semantic

import "fmt"

type CommandCandidateProviderInput struct {
	DocumentURI   string
	WorkspaceRoot string
}

type CommandCandidateProviderOutput struct {
	DocumentURI       string
	WorkspaceRoot     string
	Candidates        []CommandCandidate
	WorkspaceFallback WorkspaceFallbackFields
	Toolchain         ToolchainProvenance
	PassAvailability  map[PassKind]PassAvailability
	PassProvenance    map[PassKind]map[string]string
}

type CommandCandidateProvider interface {
	ProvideCommandCandidates(CommandCandidateProviderInput) (CommandCandidateProviderOutput, error)
}

type CommandCandidateProviderFunc func(CommandCandidateProviderInput) (CommandCandidateProviderOutput, error)

func (provider CommandCandidateProviderFunc) ProvideCommandCandidates(input CommandCandidateProviderInput) (CommandCandidateProviderOutput, error) {
	if provider == nil {
		return CommandCandidateProviderOutput{}, fmt.Errorf("command candidate provider is nil")
	}
	return provider(input)
}

type SourceReader func(SourceCacheKey) (SourceCacheValue, error)

type SourceProviderAdapter struct {
	Cache      *WorkspaceSourceCache
	ReadHeader SourceReader
	ReadMacros SourceReader
}

func (adapter SourceProviderAdapter) Load(key SourceCacheKey) (SourceCacheValue, error) {
	if adapter.Cache == nil {
		return SourceCacheValue{}, fmt.Errorf("workspace source cache is nil")
	}
	if value, ok := adapter.Cache.Get(key); ok {
		return value, nil
	}
	reader := adapter.ReadHeader
	if key.Kind == SourceKindMacro {
		reader = adapter.ReadMacros
	}
	if reader == nil {
		return SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "unresolved"}, nil
	}
	value, err := reader(key)
	if err != nil {
		return SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "unreadable"}, err
	}
	adapter.Cache.Put(key, value)
	return value, nil
}
