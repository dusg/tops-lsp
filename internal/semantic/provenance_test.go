package semantic

import "testing"

func TestSourceProviderAdapterCachesHeaderAndMacroReads(t *testing.T) {
	cache := NewWorkspaceSourceCache()
	headerReads := 0
	macroReads := 0
	adapter := SourceProviderAdapter{
		Cache: cache,
		ReadHeader: func(key SourceCacheKey) (SourceCacheValue, error) {
			headerReads++
			return SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "resolved", Values: map[string]string{"header": "1"}}, nil
		},
		ReadMacros: func(key SourceCacheKey) (SourceCacheValue, error) {
			macroReads++
			return SourceCacheValue{URI: key.URIOrDescriptor, Version: key.SourceVersion, Status: "resolved", Values: map[string]string{"macro": "1"}}, nil
		},
	}
	headerKey := SourceCacheKey{Kind: SourceKindHeader, URIOrDescriptor: "file:///workspace/include/api.h", SourceVersion: "v1", ContextVersion: 1}
	if _, err := adapter.Load(headerKey); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Load(headerKey); err != nil {
		t.Fatal(err)
	}
	if headerReads != 1 {
		t.Fatalf("header reads = %d", headerReads)
	}
	macroKey := SourceCacheKey{Kind: SourceKindMacro, URIOrDescriptor: "@workspace-macros", SourceVersion: "m1", ContextVersion: 1}
	if _, err := adapter.Load(macroKey); err != nil {
		t.Fatal(err)
	}
	if macroReads != 1 {
		t.Fatalf("macro reads = %d", macroReads)
	}
	if _, err := adapter.Load(SourceCacheKey{Kind: SourceKindHeader, URIOrDescriptor: headerKey.URIOrDescriptor, SourceVersion: "v2", ContextVersion: 1}); err != nil {
		t.Fatal(err)
	}
	if headerReads != 2 {
		t.Fatalf("header reads after version change = %d", headerReads)
	}
}

func TestCommandCandidateProviderDoesNotMakeResolverDecisions(t *testing.T) {
	provider := CommandCandidateProviderFunc(func(input CommandCandidateProviderInput) (CommandCandidateProviderOutput, error) {
		return CommandCandidateProviderOutput{DocumentURI: input.DocumentURI, WorkspaceRoot: input.WorkspaceRoot, WorkspaceFallback: WorkspaceFallbackFields{IncludeRoots: []string{"/workspace/include"}}, Toolchain: ToolchainProvenance{Identity: "topscc"}}, nil
	})
	output, err := provider.ProvideCommandCandidates(CommandCandidateProviderInput{DocumentURI: "file:///workspace/main.tops", WorkspaceRoot: "/workspace"})
	if err != nil || output.DocumentURI == "" || len(output.Candidates) != 0 || len(output.WorkspaceFallback.IncludeRoots) != 1 || output.Toolchain.Identity != "topscc" {
		t.Fatalf("provider output = %+v, err = %v", output, err)
	}
}
