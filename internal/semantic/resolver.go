package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"tops-lsp/internal/index"
)

type WorkspaceFallbackFields struct {
	IncludeRoots    []string
	Macros          map[string]string
	Version         string
	FieldSource     map[string]string
	FieldResolution map[string]string
}

type ToolchainProvenance struct {
	Identity        string
	Version         string
	IncludeRoots    []string
	Macros          map[string]string
	MacroVersion    string
	NormalizedRoots []string
	FieldSource     map[string]string
	FieldResolution map[string]string
}

type ResolverInput struct {
	DocumentURI       string
	WorkspaceRoot     string
	Mode              string
	LanguageID        string
	Candidates        []CommandCandidate
	ExistingCommand   *CompilationCommand
	ExistingProfile   *SemanticProfile
	WorkspaceFallback WorkspaceFallbackFields
	Toolchain         ToolchainProvenance
	DocumentVersion   int
	ContextVersion    int
	Generation        uint64
	Epoch             uint64
	PassAvailability  map[PassKind]PassAvailability
	PassProvenance    map[PassKind]map[string]string
}

type normalizedCandidate struct {
	command  CommandCandidate
	uri      string
	workdir  string
	args     []string
	group    string
	instance ConflictSource
}

func ResolveSemanticContext(input ResolverInput) SemanticContext {
	documentURI := index.NormalizeSourceURI(input.DocumentURI)
	if documentURI == "" {
		documentURI = strings.TrimSpace(input.DocumentURI)
	}
	normalized := normalizeCandidates(input, documentURI)
	language, languageKnown, languageConflict := resolveLanguage(input, normalized)
	if input.ExistingCommand != nil && (len(normalized) == 0 || existingCommandWins(normalized)) {
		if commandLanguage, ok := normalizeLanguage(input.ExistingCommand.Language); ok {
			language = commandLanguage
			languageKnown = true
			languageConflict = false
		}
		context := contextFromExisting(input, documentURI, language, languageKnown, languageConflict)
		applyPassInputs(&context, input)
		return context
	}

	groups := make(map[string][]normalizedCandidate)
	for _, candidate := range normalized {
		groups[candidate.group] = append(groups[candidate.group], candidate)
	}
	if len(groups) == 0 {
		if language == "TOPS" {
			context := defaultTopsContext(input, documentURI, languageConflict)
			applyPassInputs(&context, input)
			return context
		}
		context := defaultCppContext(input, documentURI, languageKnown, languageConflict)
		applyPassInputs(&context, input)
		return context
	}

	allCandidates := make([]normalizedCandidate, 0, len(groups))
	for _, group := range groups {
		sort.Slice(group, func(left, right int) bool {
			if commandSourcePrecedence(group[left].instance.Kind) != commandSourcePrecedence(group[right].instance.Kind) {
				return commandSourcePrecedence(group[left].instance.Kind) < commandSourcePrecedence(group[right].instance.Kind)
			}
			return group[left].instance.Instance < group[right].instance.Instance
		})
		allCandidates = append(allCandidates, group[0])
	}
	sort.Slice(allCandidates, func(left, right int) bool {
		if allCandidates[left].instance.Kind != allCandidates[right].instance.Kind {
			return commandSourcePrecedence(allCandidates[left].instance.Kind) < commandSourcePrecedence(allCandidates[right].instance.Kind)
		}
		return allCandidates[left].instance.Instance < allCandidates[right].instance.Instance
	})
	selected := allCandidates[0]
	command := commandFromCandidate(selected)
	mergeMissingCommandSources(&command, input)
	context := SemanticContext{
		DocumentURI:            documentURI,
		Command:                command,
		DocumentVersion:        input.DocumentVersion,
		SemanticContextVersion: input.ContextVersion,
		AnalysisGeneration:     input.Generation,
		Epoch:                  input.Epoch,
		Provenance:             map[string]string{"command": selected.instance.Kind, "command_instance": selected.instance.Instance},
	}
	context.Provenance["candidate_instances"] = candidateInstanceProvenance(normalized)
	if language == "TOPS" {
		context.Profile = profileForTarget(command.Arch, false)
	} else {
		context.Profile = DefaultSemanticProfile(false)
	}
	context.Status = statusForCommand(command, language, languageKnown)
	fieldConflicts := commandConflicts(allCandidates, &context.Command)
	if languageConflict || len(groups) > 1 || len(fieldConflicts) > 0 {
		context.Status = ContextStatusInvalid
		context.Conflicts = fieldConflicts
		if languageConflict && len(context.Conflicts) == 0 {
			context.Conflicts = []string{"language"}
		}
		if language == "TOPS" {
			context.Profile = profileForTarget("gcu400", true)
			context.Profile.TargetRules = false
			context.Provenance["profile"] = "gcu400-default"
			context.Provenance["target_conflict"] = "true"
		}
	}
	context.PassViews = NewPassViews(command.PassKind, context.Status)
	applyPassInputs(&context, input)
	return context
}

func normalizeCandidates(input ResolverInput, documentURI string) []normalizedCandidate {
	result := make([]normalizedCandidate, 0, len(input.Candidates))
	for _, candidate := range input.Candidates {
		if candidate.ParseError != "" || (candidate.SourceKind != "compile_commands" && candidate.SourceKind != "workspace") {
			continue
		}
		uri := normalizeCandidateURI(candidate.SourceURI, input.WorkspaceRoot)
		if uri == "" || uri != documentURI {
			continue
		}
		workdir, err := normalizeWorkingDirectory(candidate.WorkingDirectory, input.WorkspaceRoot)
		if err != nil {
			continue
		}
		arguments := append([]string(nil), candidate.NormalizedArguments...)
		if candidate.RawArguments != nil {
			arguments, err = ExpandResponseFiles(candidate.RawArguments, workdir)
			if err != nil {
				continue
			}
		}
		candidate.SourceURI = uri
		candidate.WorkingDirectory = workdir
		candidate.NormalizedArguments = arguments
		inferCandidateFields(&candidate)
		instance, err := commandCandidateInstance(candidate)
		if err != nil {
			continue
		}
		canonicalArguments, err := CanonicalJSON(arguments)
		if err != nil {
			continue
		}
		result = append(result, normalizedCandidate{
			command:  candidate,
			uri:      uri,
			workdir:  workdir,
			args:     arguments,
			group:    workdir + "\x00" + string(canonicalArguments),
			instance: instance,
		})
	}
	return result
}

func resolveLanguage(input ResolverInput, candidates []normalizedCandidate) (string, bool, bool) {
	candidateLanguages := make(map[string]struct{})
	for _, candidate := range candidates {
		if language, ok := normalizeLanguage(candidate.command.Language); ok {
			candidateLanguages[language] = struct{}{}
		}
	}
	if len(candidateLanguages) > 1 {
		return "TOPS", true, true
	}
	if len(candidateLanguages) == 1 {
		for language := range candidateLanguages {
			return language, true, false
		}
	}
	for _, signal := range []string{input.Mode, input.LanguageID, documentExtension(input.DocumentURI)} {
		if language, ok := normalizeLanguage(signal); ok {
			return language, true, false
		}
	}
	return "C++", false, false
}

func normalizeLanguage(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tops", "topscc":
		return "TOPS", true
	case "cpp", "c++", "cxx", "cc", "clang", "clang++", "g++", "gcc":
		return "C++", true
	default:
		return "", false
	}
}

func existingCommandWins(candidates []normalizedCandidate) bool {
	if len(candidates) == 0 {
		return true
	}
	for _, candidate := range candidates {
		if candidate.instance.Kind != "workspace" {
			return false
		}
	}
	return true
}

func documentExtension(uri string) string {
	parsed, err := url.Parse(uri)
	if err == nil && parsed.Path != "" {
		return strings.TrimPrefix(filepath.Ext(parsed.Path), ".")
	}
	return strings.TrimPrefix(filepath.Ext(uri), ".")
}

func contextFromExisting(input ResolverInput, documentURI, language string, languageKnown, languageConflict bool) SemanticContext {
	command := cloneCompilationCommand(*input.ExistingCommand)
	ensureCommandFieldMetadata(&command, command.Source)
	if command.SourceURI == "" {
		command.SourceURI = documentURI
	}
	command.SourceURI = documentURI
	if command.Language == "" && languageKnown && !languageConflict {
		command.Language = language
		command.FieldResolution["language"] = "resolved"
	}
	mergeMissingCommandSources(&command, input)
	if language == "TOPS" {
		profile := existingCommandProfile(input, command)
		context := newContextFromCommand(documentURI, command, profile, statusForCommand(command, language, languageKnown), input)
		if languageConflict {
			context.Status = ContextStatusInvalid
			context.Profile = profileForTarget("gcu400", true)
			context.Profile.TargetRules = false
		}
		context.MissingFields = unresolvedCommandFields(command)
		return context
	}
	profile := DefaultSemanticProfile(false)
	if input.ExistingProfile != nil && input.ExistingProfile.ProfileID != "" {
		profile = *input.ExistingProfile
	}
	status := statusForCommand(command, language, languageKnown)
	if languageConflict {
		status = ContextStatusInvalid
	}
	context := newContextFromCommand(documentURI, command, profile, status, input)
	context.MissingFields = unresolvedCommandFields(command)
	return context
}

func defaultTopsContext(input ResolverInput, documentURI string, languageConflict bool) SemanticContext {
	command := CompilationCommand{
		Source:              "default",
		SourceURI:           documentURI,
		WorkingDirectory:    normalizedWorkspaceRoot(input.WorkspaceRoot),
		RawArguments:        []string{"topscc", "-arch", "gcu400", "-std=c++17"},
		NormalizedArguments: []string{"topscc", "-arch", "gcu400", "-std=c++17"},
		Arch:                "gcu400",
		Driver:              "topscc",
		Language:            "TOPS",
		LanguageStandard:    "c++17",
		PassKind:            PassKindHostDevice,
		Fallback:            true,
	}
	initializeCommandFieldMaps(&command)
	for _, key := range []string{"arch", "driver", "language", "language_standard", "pass_kind"} {
		command.FieldSource[key] = "fallback_default"
		command.FieldResolution[key] = "resolved"
	}
	command.ToolchainIncludeRoots = append([]string(nil), input.Toolchain.IncludeRoots...)
	command.ToolchainMacros = CloneStringMap(input.Toolchain.Macros)
	command.WorkspaceIncludeRoots = append([]string(nil), input.WorkspaceFallback.IncludeRoots...)
	command.WorkspaceMacros = CloneStringMap(input.WorkspaceFallback.Macros)
	setDefaultSourceMetadata(&command, input)
	status := ContextStatusResolved
	missing := []string(nil)
	if len(command.ToolchainIncludeRoots) == 0 || input.Toolchain.FieldResolution["toolchain_include_roots"] == "unresolved" {
		missing = append(missing, "toolchain_include_roots")
	}
	if len(command.ToolchainMacros) == 0 || input.Toolchain.FieldResolution["toolchain_macros"] == "unresolved" {
		missing = append(missing, "toolchain_macros")
	}
	for _, key := range []string{"workspace_include_roots", "workspace_macros"} {
		if input.WorkspaceFallback.FieldResolution[key] == "unresolved" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		status = ContextStatusPartial
	}
	if languageConflict {
		status = ContextStatusInvalid
	}
	profile := profileForTarget("gcu400", true)
	if status == ContextStatusInvalid {
		profile.TargetRules = false
	}
	context := newContextFromCommand(documentURI, command, profile, status, input)
	context.MissingFields = missing
	context.Provenance["command"] = "fallback_default"
	context.Provenance["profile"] = "gcu400-default"
	return context
}

func defaultCppContext(input ResolverInput, documentURI string, languageKnown, languageConflict bool) SemanticContext {
	command := CompilationCommand{Source: "default", SourceURI: documentURI, Language: "C++", LanguageStandard: "c++17", PassKind: PassKindHost}
	initializeCommandFieldMaps(&command)
	for _, key := range []string{"language", "language_standard", "pass_kind"} {
		command.FieldSource[key] = "fallback_default"
		command.FieldResolution[key] = "resolved"
	}
	status := ContextStatusResolved
	if !languageKnown {
		status = ContextStatusMissing
	}
	if languageConflict {
		status = ContextStatusInvalid
	}
	context := newContextFromCommand(documentURI, command, DefaultSemanticProfile(false), status, input)
	context.Provenance["command"] = "fallback_default"
	return context
}

func newContextFromCommand(documentURI string, command CompilationCommand, profile SemanticProfile, status ContextStatus, input ResolverInput) SemanticContext {
	if status == ContextStatusResolved {
		for _, availability := range input.PassAvailability {
			if availability == PassAvailabilityMissing {
				status = ContextStatusPartial
				break
			}
		}
	}
	context := SemanticContext{
		DocumentURI:            documentURI,
		Command:                command,
		Profile:                profile,
		Status:                 status,
		DocumentVersion:        input.DocumentVersion,
		SemanticContextVersion: input.ContextVersion,
		AnalysisGeneration:     input.Generation,
		Epoch:                  input.Epoch,
		PassViews:              NewPassViews(command.PassKind, status),
		Provenance:             map[string]string{"command": command.Source},
	}
	for passKind, availability := range input.PassAvailability {
		if view, ok := context.PassViews[passKind]; ok {
			view.Availability = availability
			context.PassViews[passKind] = view
		}
	}
	configurePassViews(&context)
	return context
}

func configurePassViews(context *SemanticContext) {
	if context == nil {
		return
	}
	for passKind, view := range context.PassViews {
		view.Status = context.Status
		provenance := map[string]string{
			"profile":   context.Profile.ProfileID,
			"command":   context.Command.Source,
			"pass_kind": string(passKind),
		}
		for key, value := range view.Provenance {
			provenance[key] = value
		}
		view.Provenance = provenance
		if view.Provenance["source"] == "" && context.Command.Fallback {
			view.Provenance["source"] = "fallback_default"
		}
		view.RulesAvailable = nil
		if view.Availability == PassAvailabilityAvailable && context.Status == ContextStatusResolved && context.Profile.TargetRules {
			view.RulesAvailable = []string{"host-device-boundary"}
			if passKind == PassKindDevice {
				if context.Profile.AddressSpaceRules {
					view.RulesAvailable = append(view.RulesAvailable, "address-space")
				}
				if context.Profile.VectorConstraints && context.Profile.VectorRules.Available() {
					view.RulesAvailable = append(view.RulesAvailable, "vector-constraint")
				}
			}
		}
		context.PassViews[passKind] = view
	}
}

func profileForTarget(arch string, fallback bool) SemanticProfile {
	if strings.TrimSpace(arch) == "" || strings.EqualFold(strings.TrimSpace(arch), "unknown") || fallback {
		return DefaultSemanticProfile(true)
	}
	rules, vectorAvailable := vectorRulesForTarget(arch)
	return SemanticProfile{ProfileID: arch, TargetArch: arch, TargetProvenance: "compile_commands", TargetRules: true, AddressSpaceRules: true, VectorConstraints: vectorAvailable, VectorRules: rules}
}

func existingCommandProfile(input ResolverInput, command CompilationCommand) SemanticProfile {
	if input.ExistingProfile != nil && input.ExistingProfile.ProfileID != "" {
		return *input.ExistingProfile
	}
	if strings.TrimSpace(command.Arch) == "" || strings.EqualFold(strings.TrimSpace(command.Arch), "unknown") {
		return profileForTarget("gcu400", true)
	}
	return profileForTarget(command.Arch, false)
}

func statusForCommand(command CompilationCommand, language string, languageKnown bool) ContextStatus {
	if !languageKnown {
		return ContextStatusMissing
	}
	for key, resolution := range command.FieldResolution {
		if resolution == "invalid" {
			return ContextStatusInvalid
		}
		if resolution == "unresolved" && (key == "arch" || key == "driver" || key == "language" || key == "language_standard" || key == "pass_kind" || key == "toolchain_include_roots" || key == "toolchain_macros" || key == "workspace_include_roots" || key == "workspace_macros") {
			return ContextStatusPartial
		}
	}
	if language == "TOPS" && (command.Arch == "" || command.Driver == "" || command.LanguageStandard == "" || command.PassKind == "") {
		return ContextStatusPartial
	}
	return ContextStatusResolved
}

func inferCandidateFields(candidate *CommandCandidate) {
	if candidate.Language == "" {
		for index, argument := range candidate.NormalizedArguments {
			if argument == "-x" && index+1 < len(candidate.NormalizedArguments) {
				candidate.Language = candidate.NormalizedArguments[index+1]
			}
			if strings.HasPrefix(argument, "-x") && len(argument) > 2 {
				candidate.Language = argument[2:]
			}
			if strings.HasPrefix(argument, "--language=") {
				candidate.Language = strings.TrimPrefix(argument, "--language=")
			}
		}
	}
	if candidate.Arch == "" {
		candidate.Arch = argumentValue(candidate.NormalizedArguments, "-arch", "--arch=")
	}
	if candidate.Driver == "" && len(candidate.NormalizedArguments) > 0 {
		if candidate.NormalizedArguments[0] == "topscc" || strings.Contains(candidate.NormalizedArguments[0], "clang") {
			candidate.Driver = candidate.NormalizedArguments[0]
		}
	}
	if candidate.LanguageStandard == "" {
		candidate.LanguageStandard = argumentValue(candidate.NormalizedArguments, "-std", "-std=")
		if candidate.LanguageStandard == "" {
			candidate.LanguageStandard = argumentValue(candidate.NormalizedArguments, "--std", "--std=")
		}
	}
	if candidate.PassKind == "" {
		pass := argumentValue(candidate.NormalizedArguments, "--pass", "--pass=")
		if pass == "" {
			pass = argumentValue(candidate.NormalizedArguments, "-pass", "-pass=")
		}
		candidate.PassKind = PassKind(pass)
	}
	if candidate.Language == "" {
		candidate.Language = candidate.Driver
	}
	if language, ok := normalizeLanguage(candidate.Language); ok {
		candidate.Language = language
	}
	if candidate.PassKind == "" {
		if language, ok := normalizeLanguage(candidate.Language); ok && language == "TOPS" {
			candidate.PassKind = PassKindHostDevice
		} else {
			candidate.PassKind = PassKindHost
		}
	}
}

func argumentValue(arguments []string, exact, prefix string) string {
	for index, argument := range arguments {
		if argument == exact && index+1 < len(arguments) {
			return arguments[index+1]
		}
		if strings.HasPrefix(argument, prefix) {
			return strings.TrimPrefix(argument, prefix)
		}
	}
	return ""
}

func commandFromCandidate(candidate normalizedCandidate) CompilationCommand {
	value := candidate.command
	command := CompilationCommand{
		Source:                value.SourceKind,
		SourceURI:             value.SourceURI,
		WorkingDirectory:      value.WorkingDirectory,
		RawArguments:          append([]string(nil), value.RawArguments...),
		NormalizedArguments:   append([]string(nil), value.NormalizedArguments...),
		Arch:                  value.Arch,
		Driver:                value.Driver,
		Language:              value.Language,
		LanguageStandard:      value.LanguageStandard,
		PassKind:              value.PassKind,
		ToolchainIncludeRoots: append([]string(nil), value.ToolchainRoots...),
		WorkspaceIncludeRoots: append([]string(nil), value.WorkspaceRoots...),
		ToolchainMacros:       CloneStringMap(value.ToolchainMacros),
		WorkspaceMacros:       CloneStringMap(value.WorkspaceMacros),
		Fallback:              false,
	}
	initializeCommandFieldMaps(&command)
	for key, source := range value.FieldSource {
		command.FieldSource[key] = source
	}
	for key, resolution := range value.FieldResolution {
		command.FieldResolution[key] = resolution
	}
	for key, conflict := range value.FieldConflict {
		command.FieldConflict[key] = cloneFieldConflict(conflict)
	}
	for _, key := range CommandFieldKeys() {
		if command.FieldSource[key] == "" {
			command.FieldSource[key] = value.SourceKind
		}
		if command.FieldResolution[key] == "" {
			command.FieldResolution[key] = commandFieldResolution(&command, key)
		}
	}
	return command
}

func candidateInstanceProvenance(candidates []normalizedCandidate) string {
	instances := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		value := candidate.instance.Kind + ":" + candidate.instance.Instance
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		instances = append(instances, value)
	}
	sort.Strings(instances)
	return strings.Join(instances, ",")
}

func mergeMissingCommandSources(command *CompilationCommand, input ResolverInput) {
	if command == nil {
		return
	}
	ensureCommandFieldMetadata(command, command.Source)
	mergeSource := func(key, source, resolution string, available bool, assign func()) {
		if !commandFieldIsEmpty(command, key) {
			return
		}
		if available {
			assign()
			command.FieldSource[key] = source
		}
		if resolution != "" {
			command.FieldResolution[key] = resolution
		} else if available {
			command.FieldResolution[key] = "resolved"
		}
	}
	mergeSource("toolchain_include_roots", "toolchain_default", input.Toolchain.FieldResolution["toolchain_include_roots"], len(input.Toolchain.IncludeRoots) > 0, func() {
		command.ToolchainIncludeRoots = append([]string(nil), input.Toolchain.IncludeRoots...)
	})
	mergeSource("toolchain_macros", "toolchain_default", input.Toolchain.FieldResolution["toolchain_macros"], len(input.Toolchain.Macros) > 0, func() {
		command.ToolchainMacros = CloneStringMap(input.Toolchain.Macros)
	})
	mergeSource("workspace_include_roots", "workspace", input.WorkspaceFallback.FieldResolution["workspace_include_roots"], len(input.WorkspaceFallback.IncludeRoots) > 0, func() {
		command.WorkspaceIncludeRoots = append([]string(nil), input.WorkspaceFallback.IncludeRoots...)
	})
	mergeSource("workspace_macros", "workspace", input.WorkspaceFallback.FieldResolution["workspace_macros"], len(input.WorkspaceFallback.Macros) > 0, func() {
		command.WorkspaceMacros = CloneStringMap(input.WorkspaceFallback.Macros)
	})
}

func commandConflicts(candidates []normalizedCandidate, command *CompilationCommand) []string {
	conflicts := make([]string, 0)
	for _, key := range CommandFieldKeys() {
		pairs := make([]struct {
			source ConflictSource
			value  FieldValue
		}, 0, len(candidates))
		seen := make(map[string]struct{})
		for _, candidate := range candidates {
			value := commandFieldValue(candidate.command, key)
			canonical, err := CanonicalJSON(value)
			if err != nil {
				continue
			}
			if _, exists := seen[string(canonical)]; exists {
				continue
			}
			seen[string(canonical)] = struct{}{}
			pairs = append(pairs, struct {
				source ConflictSource
				value  FieldValue
			}{source: candidate.instance, value: value})
		}
		if len(pairs) < 2 {
			continue
		}
		sort.Slice(pairs, func(left, right int) bool {
			if commandSourcePrecedence(pairs[left].source.Kind) != commandSourcePrecedence(pairs[right].source.Kind) {
				return commandSourcePrecedence(pairs[left].source.Kind) < commandSourcePrecedence(pairs[right].source.Kind)
			}
			return pairs[left].source.Instance < pairs[right].source.Instance
		})
		sources := make([]ConflictSource, 0, len(pairs))
		values := make([]FieldValue, 0, len(pairs))
		for _, pair := range pairs {
			sources = append(sources, pair.source)
			values = append(values, pair.value)
		}
		conflict := &FieldConflict{Sources: sources, Values: values}
		command.FieldConflict[key] = conflict
		command.FieldSource[key] = "invalid"
		command.FieldResolution[key] = "invalid"
		instances := make([]string, 0, len(sources))
		for _, source := range sources {
			instances = append(instances, source.Kind+":"+source.Instance)
		}
		conflicts = append(conflicts, key+":"+strings.Join(instances, ","))
	}
	sort.Strings(conflicts)
	return conflicts
}

func commandFieldValue(command CommandCandidate, key string) FieldValue {
	switch key {
	case "source_uri":
		return command.SourceURI
	case "working_directory":
		return command.WorkingDirectory
	case "raw_arguments":
		return command.RawArguments
	case "normalized_arguments":
		return command.NormalizedArguments
	case "arch":
		return command.Arch
	case "driver":
		return command.Driver
	case "language":
		return command.Language
	case "language_standard":
		return command.LanguageStandard
	case "pass_kind":
		return command.PassKind
	case "toolchain_include_roots":
		return command.ToolchainRoots
	case "workspace_include_roots":
		return command.WorkspaceRoots
	case "toolchain_macros":
		return command.ToolchainMacros
	case "workspace_macros":
		return command.WorkspaceMacros
	default:
		return nil
	}
}

func commandCandidateInstance(candidate CommandCandidate) (ConflictSource, error) {
	object := map[string]any{
		"kind":                 candidate.SourceKind,
		"source_uri":           candidate.SourceURI,
		"working_directory":    candidate.WorkingDirectory,
		"normalized_arguments": candidate.NormalizedArguments,
	}
	if candidate.SourceKind == "workspace" {
		object["version"] = candidate.Version
	}
	canonical, err := CanonicalJSON(object)
	if err != nil {
		return ConflictSource{}, err
	}
	digest := sha256.Sum256(canonical)
	return ConflictSource{Kind: candidate.SourceKind, Instance: hex.EncodeToString(digest[:])}, nil
}

func ToolchainConflictSource(provenance ToolchainProvenance) (ConflictSource, error) {
	object := map[string]any{
		"kind":               "toolchain_default",
		"toolchain_identity": provenance.Identity,
		"toolchain_version":  provenance.Version,
		"normalized_roots":   provenance.NormalizedRoots,
		"macro_version":      provenance.MacroVersion,
	}
	canonical, err := CanonicalJSON(object)
	if err != nil {
		return ConflictSource{}, err
	}
	digest := sha256.Sum256(canonical)
	return ConflictSource{Kind: "toolchain_default", Instance: hex.EncodeToString(digest[:])}, nil
}

func FallbackConflictSource() ConflictSource {
	return ConflictSource{Kind: "fallback_default", Instance: "gcu400-default"}
}

func initializeCommandFieldMaps(command *CompilationCommand) {
	command.FieldSource = make(map[string]string, len(commandFieldKeys))
	command.FieldResolution = make(map[string]string, len(commandFieldKeys))
	command.FieldConflict = make(map[string]*FieldConflict, len(commandFieldKeys))
	for _, key := range commandFieldKeys {
		command.FieldConflict[key] = nil
	}
}

func setDefaultSourceMetadata(command *CompilationCommand, input ResolverInput) {
	for _, key := range commandFieldKeys {
		if command.FieldSource[key] == "" {
			command.FieldSource[key] = "fallback_default"
		}
		if command.FieldResolution[key] == "" {
			command.FieldResolution[key] = "resolved"
		}
	}
	if len(command.ToolchainIncludeRoots) == 0 {
		command.FieldSource["toolchain_include_roots"] = "toolchain_default"
		command.FieldResolution["toolchain_include_roots"] = "unresolved"
	} else {
		command.FieldSource["toolchain_include_roots"] = "toolchain_default"
		if resolution := input.Toolchain.FieldResolution["toolchain_include_roots"]; resolution != "" {
			command.FieldResolution["toolchain_include_roots"] = resolution
		}
	}
	if len(command.ToolchainMacros) == 0 {
		command.FieldSource["toolchain_macros"] = "toolchain_default"
		command.FieldResolution["toolchain_macros"] = "unresolved"
	} else {
		command.FieldSource["toolchain_macros"] = "toolchain_default"
		if resolution := input.Toolchain.FieldResolution["toolchain_macros"]; resolution != "" {
			command.FieldResolution["toolchain_macros"] = resolution
		}
	}
	command.FieldSource["workspace_include_roots"] = "workspace"
	if len(command.WorkspaceIncludeRoots) == 0 {
		command.FieldSource["workspace_include_roots"] = "workspace"
		command.FieldResolution["workspace_include_roots"] = "unresolved"
	} else if resolution := input.WorkspaceFallback.FieldResolution["workspace_include_roots"]; resolution != "" {
		command.FieldResolution["workspace_include_roots"] = resolution
	}
	command.FieldSource["workspace_macros"] = "workspace"
	if len(command.WorkspaceMacros) == 0 {
		command.FieldSource["workspace_macros"] = "workspace"
		command.FieldResolution["workspace_macros"] = "unresolved"
	} else if resolution := input.WorkspaceFallback.FieldResolution["workspace_macros"]; resolution != "" {
		command.FieldResolution["workspace_macros"] = resolution
	}
}

func cloneCompilationCommand(command CompilationCommand) CompilationCommand {
	command.RawArguments = append([]string(nil), command.RawArguments...)
	command.NormalizedArguments = append([]string(nil), command.NormalizedArguments...)
	command.ToolchainIncludeRoots = append([]string(nil), command.ToolchainIncludeRoots...)
	command.WorkspaceIncludeRoots = append([]string(nil), command.WorkspaceIncludeRoots...)
	command.ToolchainMacros = CloneStringMap(command.ToolchainMacros)
	command.WorkspaceMacros = CloneStringMap(command.WorkspaceMacros)
	command.FieldSource = CloneStringMap(command.FieldSource)
	command.FieldResolution = CloneStringMap(command.FieldResolution)
	command.FieldConflict = make(map[string]*FieldConflict, len(command.FieldConflict))
	for key, conflict := range command.FieldConflict {
		command.FieldConflict[key] = cloneFieldConflict(conflict)
	}
	return command
}

func cloneFieldConflict(conflict *FieldConflict) *FieldConflict {
	if conflict == nil {
		return nil
	}
	return &FieldConflict{Sources: append([]ConflictSource(nil), conflict.Sources...), Values: append([]FieldValue(nil), conflict.Values...)}
}

// CloneStringMap returns an independent copy of values so callers can mutate it
// without affecting the resolver input.
func CloneStringMap(values map[string]string) map[string]string {
	return maps.Clone(values)
}

func ClonePassAvailability(values map[PassKind]PassAvailability) map[PassKind]PassAvailability {
	return maps.Clone(values)
}

func ClonePassProvenance(values map[PassKind]map[string]string) map[PassKind]map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[PassKind]map[string]string, len(values))
	for pass, provenance := range values {
		result[pass] = maps.Clone(provenance)
	}
	return result
}

func normalizeCandidateURI(value, workspaceRoot string) string {
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Scheme == "" && !filepath.IsAbs(value) && workspaceRoot != "" {
		value = filepath.Join(workspaceRoot, value)
	}
	return index.NormalizeSourceURI(value)
}

func normalizeWorkingDirectory(value, workspaceRoot string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("working directory is empty")
	}
	if !filepath.IsAbs(value) {
		if workspaceRoot == "" {
			return "", fmt.Errorf("relative working directory has no workspace root")
		}
		value = filepath.Join(workspaceRoot, value)
	}
	abs, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(abs), nil
}

func normalizedWorkspaceRoot(root string) string {
	if root == "" {
		return ""
	}
	value, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return ""
	}
	return filepath.ToSlash(value)
}

func commandSourcePrecedence(kind string) int {
	switch kind {
	case "compile_commands":
		return 0
	case "workspace":
		return 1
	case "toolchain_default":
		return 2
	case "fallback_default":
		return 3
	default:
		return 4
	}
}
