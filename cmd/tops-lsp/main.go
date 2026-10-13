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
	"tops-lsp/internal/server"
	"tops-lsp/internal/transport"
)

func main() {
	logger := logging.New(os.Stderr, slog.LevelInfo)
	instance := server.NewWithOptions(logger, server.ServerOptions{
		SourceProvider:     configuredSourceProvider(),
		CompilationContext: configuredCompilationContext(),
		NavigationGate:     configuredNavigationGate(),
	})
	status := instance.Run(context.Background(), transport.NewReader(os.Stdin), transport.NewWriter(os.Stdout))
	os.Exit(status)
}

func configuredSourceProvider() index.SourceProvider {
	value := strings.TrimSpace(os.Getenv("TOPS_LSP_INCLUDE_ROOTS"))
	if value == "" {
		return index.EmptySourceProvider{}
	}
	roots := make([]string, 0)
	for _, root := range strings.Split(value, string(os.PathListSeparator)) {
		if root = strings.TrimSpace(root); root != "" {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return index.EmptySourceProvider{}
	}
	return index.NewLocalSourceProviderWithOptions(nil, roots, index.LocalSourceProviderOptions{TrustedHeaderURIs: configuredHeaderURIs(os.Getenv("TOPS_LSP_VERIFIED_HEADER_URIS"))})
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
