package server

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/semantic"
)

// semanticTestProviderOptions describes the command candidate provider used by
// semantic server tests. Zero values fall back to the shared test defaults.
type semanticTestProviderOptions struct {
	IncludeRoots     []string
	Macros           map[string]string
	Version          string
	ToolchainRoots   []string
	ToolchainMacros  map[string]string
	ToolchainVersion string
	MacroVersion     string
	PassAvailability map[semantic.PassKind]semantic.PassAvailability
	PassProvenance   map[semantic.PassKind]map[string]string
}

// newSemanticTestProvider builds a CommandCandidateProvider that returns the
// options above, so tests do not repeat the same literal provider body.
func newSemanticTestProvider(options semanticTestProviderOptions) semantic.CommandCandidateProvider {
	if len(options.IncludeRoots) == 0 {
		options.IncludeRoots = []string{"/workspace/include"}
	}
	if options.Macros == nil {
		options.Macros = map[string]string{"PROJECT": "1"}
	}
	if options.Version == "" {
		options.Version = "workspace-v1"
	}
	if len(options.ToolchainRoots) == 0 {
		options.ToolchainRoots = []string{"/tops/include"}
	}
	if options.ToolchainMacros == nil {
		options.ToolchainMacros = map[string]string{"__GCU_ARCH__": "400"}
	}
	if options.ToolchainVersion == "" {
		options.ToolchainVersion = "toolchain-v1"
	}
	if options.MacroVersion == "" {
		options.MacroVersion = "toolchain-m1"
	}
	return semantic.CommandCandidateProviderFunc(func(input semantic.CommandCandidateProviderInput) (semantic.CommandCandidateProviderOutput, error) {
		return semantic.CommandCandidateProviderOutput{
			DocumentURI:   input.DocumentURI,
			WorkspaceRoot: input.WorkspaceRoot,
			WorkspaceFallback: semantic.WorkspaceFallbackFields{
				IncludeRoots: append([]string(nil), options.IncludeRoots...),
				Macros:       options.Macros,
				Version:      options.Version,
			},
			Toolchain: semantic.ToolchainProvenance{
				Identity:     "topscc",
				Version:      options.ToolchainVersion,
				IncludeRoots: append([]string(nil), options.ToolchainRoots...),
				Macros:       options.ToolchainMacros,
				MacroVersion: options.MacroVersion,
			},
			PassAvailability: options.PassAvailability,
			PassProvenance:   options.PassProvenance,
		}, nil
	})
}

// openSemanticTestDocument opens uri with text and runs one analysis pass, which
// is the shared setup step of the semantic server tests.
func openSemanticTestDocument(t *testing.T, server *Server, uri, languageID, text string) document.DocumentState {
	t.Helper()
	state, err := server.openDocument(uri, languageID, 1, text)
	if err != nil {
		t.Fatalf("openDocument(%q) failed: %v", uri, err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	return state
}

// newSemanticTestServer builds a server with the shared semantic test provider
// and returns it together with the provider options actually used.
func newSemanticTestServer(t *testing.T, options semanticTestProviderOptions, mutate func(*ServerOptions)) *Server {
	t.Helper()
	serverOptions := ServerOptions{CommandCandidateProvider: newSemanticTestProvider(options)}
	if mutate != nil {
		mutate(&serverOptions)
	}
	return NewWithOptions(nil, serverOptions)
}

// semanticTestLogger returns a debug logger writing into buffer for tests that
// assert on emitted log records.
func semanticTestLogger(buffer *bytes.Buffer) *slog.Logger {
	return logging.New(buffer, slog.LevelDebug)
}

// semanticTestSourceProvider is a minimal SourceProvider used by header tests.
type semanticTestSourceProvider struct {
	roots   []string
	headers map[string]index.SourceDocument
	reads   map[string]int
}

func (provider *semanticTestSourceProvider) OpenDocument(uri string) (index.SourceDocument, bool) {
	document, ok := provider.headers[uri]
	return document, ok
}

func (provider *semanticTestSourceProvider) IncludeRoots() ([]string, bool) {
	if len(provider.roots) == 0 {
		return nil, false
	}
	return append([]string(nil), provider.roots...), true
}

func (provider *semanticTestSourceProvider) ResolveInclude(string, string) (index.SourceDocument, index.HeaderDependencyStatus) {
	return index.SourceDocument{}, index.HeaderDependencyUnresolved
}

// parseContextRecorder captures the parse context a server used, so tests can
// assert the resolved semantic context reached the parser.
type parseContextRecorder struct {
	context parser.ParseContext
	calls   int
}

func (recorder *parseContextRecorder) parse(text string, context parser.ParseContext) parser.ParseResult {
	recorder.context = context
	recorder.calls++
	return parser.Parse(text, context)
}
