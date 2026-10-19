package server

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/semantic"
)

func (server *Server) resolveSemanticContext(state document.DocumentState, contextVersion int) semantic.SemanticContext {
	server.mu.RLock()
	compilationContext := cloneCompilationContext(server.compilationContext)
	provider := server.sourceProvider
	commandCandidateProvider := server.commandCandidateProvider
	server.mu.RUnlock()

	input := semantic.ResolverInput{
		DocumentURI:     state.URI,
		WorkspaceRoot:   documentWorkspaceRoot(state.URI),
		Mode:            compilationContext.Mode,
		LanguageID:      state.LanguageID,
		DocumentVersion: state.Version,
		ContextVersion:  contextVersion,
	}
	if provider != nil {
		if roots, available := provider.IncludeRoots(); available {
			input.WorkspaceFallback.IncludeRoots = append([]string(nil), roots...)
		}
	}
	if len(compilationContext.PredefinedMacros) > 0 {
		input.WorkspaceFallback.Macros = mergeSemanticStringMaps(input.WorkspaceFallback.Macros, semanticParserMacros(compilationContext.PredefinedMacros))
		input.WorkspaceFallback.FieldResolution = semantic.CloneStringMap(input.WorkspaceFallback.FieldResolution)
		if input.WorkspaceFallback.FieldResolution == nil {
			input.WorkspaceFallback.FieldResolution = make(map[string]string)
		}
		input.WorkspaceFallback.FieldResolution["workspace_macros"] = "resolved"
	}
	if commandCandidateProvider != nil {
		output, err := commandCandidateProvider.ProvideCommandCandidates(semantic.CommandCandidateProviderInput{
			DocumentURI:   state.URI,
			WorkspaceRoot: documentWorkspaceRoot(state.URI),
		})
		if err == nil {
			input.Candidates = append([]semantic.CommandCandidate(nil), output.Candidates...)
			input.WorkspaceFallback.IncludeRoots = mergeSemanticStrings(input.WorkspaceFallback.IncludeRoots, output.WorkspaceFallback.IncludeRoots)
			input.WorkspaceFallback.Macros = mergeSemanticStringMaps(input.WorkspaceFallback.Macros, output.WorkspaceFallback.Macros)
			input.WorkspaceFallback.Version = output.WorkspaceFallback.Version
			input.WorkspaceFallback.FieldSource = semantic.CloneStringMap(output.WorkspaceFallback.FieldSource)
			input.WorkspaceFallback.FieldResolution = semantic.CloneStringMap(output.WorkspaceFallback.FieldResolution)
			input.Toolchain = output.Toolchain
			input.PassAvailability = semantic.ClonePassAvailability(output.PassAvailability)
			input.PassProvenance = semantic.ClonePassProvenance(output.PassProvenance)
			server.registerSemanticMacroSource(semantic.SourceKindMacro, "@workspace-macros", output.WorkspaceFallback.Version, output.WorkspaceFallback.Macros)
			toolchainDescriptor := semanticToolchainMacroDescriptor(output.Toolchain)
			server.registerSemanticMacroSource(semantic.SourceKindMacro, toolchainDescriptor, output.Toolchain.MacroVersion, output.Toolchain.Macros)
			server.loadSemanticMacroSource(semantic.SourceKindMacro, "@workspace-macros", output.WorkspaceFallback.Version)
			server.loadSemanticMacroSource(semantic.SourceKindMacro, toolchainDescriptor, output.Toolchain.MacroVersion)
		}
	}
	if hasCompilationCommand(compilationContext) {
		command := semantic.CompilationCommand{
			Source:           "compile_commands",
			SourceURI:        state.URI,
			Language:         compilationContext.Language,
			LanguageStandard: compilationContext.LanguageStandard,
			Driver:           compilationContext.DriverKind,
			Arch:             compilationContext.TargetProfile,
			PassKind:         semantic.PassKind(compilationContext.PassKind),
		}
		if command.LanguageStandard == "unknown" {
			command.LanguageStandard = ""
		}
		if command.Driver == "unknown" {
			command.Driver = ""
		}
		if command.Arch == "unknown" {
			command.Arch = ""
		}
		if command.PassKind == semantic.PassKind("unknown") {
			command.PassKind = ""
		}
		input.ExistingCommand = &command
		if strings.EqualFold(compilationContext.CompilerContextStatus, string(semantic.ContextStatusResolved)) && len(input.Candidates) == 0 {
			if input.Toolchain.FieldResolution == nil {
				input.Toolchain.FieldResolution = make(map[string]string)
			}
			input.Toolchain.FieldResolution["toolchain_include_roots"] = "resolved"
			input.Toolchain.FieldResolution["toolchain_macros"] = "resolved"
			if input.WorkspaceFallback.FieldResolution == nil {
				input.WorkspaceFallback.FieldResolution = make(map[string]string)
			}
			input.WorkspaceFallback.FieldResolution["workspace_include_roots"] = "resolved"
			input.WorkspaceFallback.FieldResolution["workspace_macros"] = "resolved"
			input.PassAvailability = make(map[semantic.PassKind]semantic.PassAvailability)
			input.PassProvenance = make(map[semantic.PassKind]map[string]string)
			passes := []semantic.PassKind{command.PassKind}
			if command.PassKind == semantic.PassKindHostDevice {
				passes = []semantic.PassKind{semantic.PassKindHost, semantic.PassKindDevice}
			}
			for _, pass := range passes {
				if pass != semantic.PassKindHost && pass != semantic.PassKindDevice {
					continue
				}
				input.PassAvailability[pass] = semantic.PassAvailabilityAvailable
				input.PassProvenance[pass] = map[string]string{"source": "compilation_context", "pass_kind": string(pass)}
			}
		}
	}
	context := semantic.ResolveSemanticContext(input)
	context.SourceVersions = []semantic.SourceVersionEntry{{URI: state.URI, Version: strconv.Itoa(state.Version)}}
	if fingerprint, err := semantic.AggregateSourceVersion(state.URI, context.SourceVersions); err == nil {
		context.SourceVersionFingerprint = fingerprint
	}
	return context
}

func semanticParserMacros(values map[string]parser.MacroValue) map[string]string {
	result := make(map[string]string, len(values))
	for name, value := range values {
		switch value.Kind {
		case parser.MacroInteger:
			result[name] = strconv.FormatInt(value.Value, 10)
		case parser.MacroDefined, parser.MacroFunction:
			result[name] = "1"
		case parser.MacroUndefined:
			result[name] = "0"
		}
	}
	return result
}

func (server *Server) commitSemanticContext(snapshot analysisSnapshot, context semantic.SemanticContext) bool {
	status := semantic.SnapshotStatusDegraded
	if context.Status == semantic.ContextStatusResolved {
		status = semantic.SnapshotStatusComplete
	}
	return server.commitSemanticSnapshot(snapshot, semantic.SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: context.SourceVersionFingerprint,
		DocumentVersion:          snapshot.state.Version,
		SemanticContextVersion:   snapshot.context.version,
		SourceVersions:           context.SourceVersions,
		HeaderVersions:           context.HeaderVersions,
		Generation:               snapshot.generation,
		Epoch:                    snapshot.epoch,
		Status:                   status,
	})
}

func (server *Server) commitSemanticSnapshot(snapshot analysisSnapshot, result semantic.SemanticSnapshot) bool {
	if !server.analysisCurrentSnapshot(snapshot) {
		return false
	}
	if !server.semanticSnapshotSourcesCurrent(result) {
		return false
	}
	result.Generation = snapshot.generation
	result.Epoch = snapshot.epoch
	result.DocumentVersion = snapshot.state.Version
	result.SemanticContextVersion = snapshot.context.version
	return server.semanticStore.Replace(result)
}

func (server *Server) semanticSnapshotSourcesCurrent(snapshot semantic.SemanticSnapshot) bool {
	entries := append([]semantic.SourceVersionEntry(nil), snapshot.SourceVersions...)
	entries = append(entries, snapshot.HeaderVersions...)
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, ok := seen[entry.URI]; ok {
			continue
		}
		seen[entry.URI] = struct{}{}
		current, ok := server.semanticSourceVersion(entry.URI)
		if entry.Version != "" {
			if !ok || current.SourceVersion != entry.Version {
				return false
			}
			continue
		}
		if entry.Status != "" && ok && current.SourceVersion != "" {
			return false
		}
	}
	return true
}

func (server *Server) semanticSourceVersion(uri string) (index.SourceVersion, bool) {
	if state, ok := server.documents.Get(uri); ok {
		return index.SourceVersion{URI: state.URI, DocumentVersion: state.Version, SourceVersion: index.SourceDigest(state.Text)}, true
	}
	if provider, ok := server.sourceProvider.(*serverSourceProvider); ok {
		return provider.currentSourceVersion(uri)
	}
	return index.SourceVersion{}, false
}

func enrichSemanticSourceVersions(context semantic.SemanticContext, snapshot index.IndexSnapshot, rootText string) semantic.SemanticContext {
	root := semantic.SourceVersionEntry{URI: snapshot.RootURI, Version: index.SourceDigest(rootText)}
	context.SourceVersions = []semantic.SourceVersionEntry{root}
	context.HeaderVersions = nil
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		entry := semantic.SourceVersionEntry{URI: dependency.TargetURI}
		if dependency.SourceVersion.SourceVersion != "" {
			entry.Version = dependency.SourceVersion.SourceVersion
		} else {
			switch dependency.Status {
			case index.HeaderDependencyMissing:
				entry.Status = semantic.SourceVersionStatusMissing
			case index.HeaderDependencyUnreadable:
				entry.Status = semantic.SourceVersionStatusUnreadable
			case index.HeaderDependencyStale:
				entry.Status = semantic.SourceVersionStatusStale
			default:
				entry.Status = semantic.SourceVersionStatusUnresolved
			}
		}
		context.HeaderVersions = append(context.HeaderVersions, entry)
		context.SourceVersions = append(context.SourceVersions, entry)
	}
	context = degradeSemanticContextForSources(context, snapshot)
	if fingerprint, err := semantic.AggregateSourceVersion(snapshot.RootURI, context.SourceVersions); err == nil {
		context.SourceVersionFingerprint = fingerprint
	}
	return context
}

func degradeSemanticContextForSources(context semantic.SemanticContext, snapshot index.IndexSnapshot) semantic.SemanticContext {
	for _, dependency := range snapshot.Dependencies {
		if dependency.Status == index.HeaderDependencyResolved {
			continue
		}
		if context.Status == semantic.ContextStatusResolved {
			context.Status = semantic.ContextStatusPartial
		}
		if dependency.TargetURI == "" {
			candidates := includeCandidateURIs(dependency.RequesterURI, dependency.IncludeSpelling, dependency.IncludeForm, dependency.SearchRoots)
			if len(candidates) > 0 {
				entry := semantic.SourceVersionEntry{URI: index.NormalizeSourceURI(candidates[0]), Status: semantic.SourceVersionStatus(dependencySourceStatus(dependency.Status))}
				context.HeaderVersions = append(context.HeaderVersions, entry)
				context.SourceVersions = append(context.SourceVersions, entry)
			}
		}
	}
	if context.Status != semantic.ContextStatusResolved {
		for passKind, view := range context.PassViews {
			view.Status = context.Status
			view.RulesAvailable = nil
			context.PassViews[passKind] = view
		}
	}
	if fingerprint, err := semantic.AggregateSourceVersion(context.DocumentURI, context.SourceVersions); err == nil {
		context.SourceVersionFingerprint = fingerprint
	}
	return context
}

func dependencySourceStatus(status index.HeaderDependencyStatus) string {
	switch status {
	case index.HeaderDependencyMissing:
		return string(semantic.SourceVersionStatusMissing)
	case index.HeaderDependencyUnreadable:
		return string(semantic.SourceVersionStatusUnreadable)
	case index.HeaderDependencyStale:
		return string(semantic.SourceVersionStatusStale)
	default:
		return string(semantic.SourceVersionStatusUnresolved)
	}
}

func hasCompilationCommand(context CompilationContext) bool {
	return (context.Language != "" && !strings.EqualFold(context.Language, "unknown")) ||
		(context.Mode != "" && !strings.EqualFold(context.Mode, "unknown")) ||
		(context.LanguageStandard != "" && context.LanguageStandard != "unknown") ||
		(context.DriverKind != "" && context.DriverKind != "unknown") ||
		(context.TargetProfile != "" && context.TargetProfile != "unknown") ||
		(context.PassKind != "" && context.PassKind != "unknown")
}

func documentPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Path != "" {
		return parsed.Path
	}
	return uri
}

func documentWorkspaceRoot(uri string) string {
	path := documentPath(uri)
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}
