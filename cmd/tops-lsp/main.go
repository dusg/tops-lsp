package main

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/semantic"
	"tops-lsp/internal/server"
	"tops-lsp/internal/transport"
)

func main() {
	logger := logging.New(os.Stderr, slog.LevelInfo)
	instance := server.NewWithOptions(logger, server.ServerOptions{
		SourceProvider:           configuredSourceProvider(),
		CommandCandidateProvider: configuredCommandCandidateProvider(),
		CompilationContext:       configuredCompilationContext(),
		NavigationGate:           configuredNavigationGate(),
	})
	status := instance.Run(context.Background(), transport.NewReader(os.Stdin), transport.NewWriter(os.Stdout))
	os.Exit(status)
}

func configuredSourceProvider() index.SourceProvider {
	roots := configuredPathList("TOPS_LSP_INCLUDE_ROOTS")
	if len(roots) == 0 {
		return index.EmptySourceProvider{}
	}
	return index.NewLocalSourceProviderWithOptions(nil, roots, index.LocalSourceProviderOptions{TrustedHeaderURIs: configuredHeaderURIs(os.Getenv("TOPS_LSP_VERIFIED_HEADER_URIS"))})
}

func configuredCommandCandidateProvider() semantic.CommandCandidateProvider {
	return semantic.NewFileCommandCandidateProvider(semantic.FileCommandCandidateProviderOptions{
		CompileCommandsPath: strings.TrimSpace(os.Getenv("TOPS_LSP_COMPILE_COMMANDS")),
		WorkspaceFallback: semantic.WorkspaceFallbackFields{
			IncludeRoots: configuredPathList("TOPS_LSP_INCLUDE_ROOTS"),
			Macros:       configuredMacros("TOPS_LSP_WORKSPACE_MACROS"),
			Version:      strings.TrimSpace(os.Getenv("TOPS_LSP_WORKSPACE_VERSION")),
		},
		Toolchain: semantic.ToolchainProvenance{
			Identity:     strings.TrimSpace(os.Getenv("TOPS_LSP_TOOLCHAIN_IDENTITY")),
			Version:      strings.TrimSpace(os.Getenv("TOPS_LSP_TOOLCHAIN_VERSION")),
			IncludeRoots: configuredPathList("TOPS_LSP_TOOLCHAIN_INCLUDE_ROOTS"),
			Macros:       configuredMacros("TOPS_LSP_TOOLCHAIN_MACROS"),
			MacroVersion: strings.TrimSpace(os.Getenv("TOPS_LSP_TOOLCHAIN_MACRO_VERSION")),
		},
	})
}

func configuredPathList(name string) []string {
	values := make([]string, 0)
	for _, value := range strings.Split(strings.TrimSpace(os.Getenv(name)), string(os.PathListSeparator)) {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func configuredMacros(name string) map[string]string {
	macros := make(map[string]string)
	for _, value := range strings.Split(strings.TrimSpace(os.Getenv(name)), ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		macroName, macroValue, found := strings.Cut(value, "=")
		macroName = strings.TrimSpace(macroName)
		if macroName == "" {
			continue
		}
		if !found {
			macroValue = "1"
		}
		macros[macroName] = macroValue
	}
	return macros
}

func configuredHeaderURIs(value string) []string {
	result := make([]string, 0)
	for _, item := range strings.Split(value, string(os.PathListSeparator)) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if parsed, err := url.Parse(item); err == nil && parsed.Scheme != "" {
			result = append(result, item)
			continue
		}
		absolute, err := filepath.Abs(item)
		if err != nil {
			continue
		}
		result = append(result, (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String())
	}
	return result
}

func configuredNavigationGate() func(context.Context, string) error {
	if strings.TrimSpace(os.Getenv("TOPS_LSP_NAVIGATION_BLOCK")) != "1" {
		return nil
	}
	return func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}
}

func configuredCompilationContext() *server.CompilationContext {
	target := strings.TrimSpace(os.Getenv("TOPS_LSP_TARGET_PROFILE"))
	pass := strings.TrimSpace(os.Getenv("TOPS_LSP_PASS_KIND"))
	if target == "" && pass == "" && strings.TrimSpace(os.Getenv("TOPS_LSP_PREDEFINED_MACROS")) == "" {
		return nil
	}
	macros := make(map[string]parser.MacroValue)
	for _, name := range strings.Split(os.Getenv("TOPS_LSP_PREDEFINED_MACROS"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			macros[name] = parser.MacroValue{Kind: parser.MacroDefined}
		}
	}
	context := &server.CompilationContext{
		LanguageStandard:      "c++17",
		DriverKind:            "tops-lsp",
		CompilerContextStatus: "resolved",
		ArgumentProvenance:    "environment",
		PredefinedMacros:      macros,
		TargetProfile:         target,
		PassKind:              pass,
	}
	if context.TargetProfile == "" {
		context.TargetProfile = "unknown"
	}
	if context.PassKind == "" {
		context.PassKind = "unknown"
	}
	if target == "" || pass == "" {
		context.CompilerContextStatus = "partial"
	}
	return context
}
