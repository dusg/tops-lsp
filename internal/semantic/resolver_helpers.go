package semantic

func ensureCommandFieldMetadata(command *CompilationCommand, defaultSource string) {
	if command == nil {
		return
	}
	if command.FieldSource == nil {
		command.FieldSource = make(map[string]string, len(commandFieldKeys))
	}
	if command.FieldResolution == nil {
		command.FieldResolution = make(map[string]string, len(commandFieldKeys))
	}
	if command.FieldConflict == nil {
		command.FieldConflict = make(map[string]*FieldConflict, len(commandFieldKeys))
	}
	for _, key := range commandFieldKeys {
		if command.FieldSource[key] == "" {
			command.FieldSource[key] = defaultSource
		}
		if command.FieldResolution[key] == "" {
			if (defaultSource == "compile_commands" || defaultSource == "workspace") && !commandFieldIsEmpty(command, key) {
				command.FieldResolution[key] = "resolved"
			} else {
				command.FieldResolution[key] = commandFieldResolution(command, key)
			}
		}
		if _, ok := command.FieldConflict[key]; !ok {
			command.FieldConflict[key] = nil
		}
	}
}

func commandFieldResolution(command *CompilationCommand, key string) string {
	if commandFieldIsEmpty(command, key) {
		return "unresolved"
	}
	return "resolved"
}

func commandFieldIsEmpty(command *CompilationCommand, key string) bool {
	switch key {
	case "source_uri":
		return command.SourceURI == ""
	case "working_directory":
		return command.WorkingDirectory == ""
	case "raw_arguments":
		return command.RawArguments == nil
	case "normalized_arguments":
		return command.NormalizedArguments == nil
	case "arch":
		return command.Arch == ""
	case "driver":
		return command.Driver == ""
	case "language":
		return command.Language == ""
	case "language_standard":
		return command.LanguageStandard == ""
	case "pass_kind":
		return command.PassKind == ""
	case "toolchain_include_roots":
		return len(command.ToolchainIncludeRoots) == 0
	case "workspace_include_roots":
		return len(command.WorkspaceIncludeRoots) == 0
	case "toolchain_macros":
		return len(command.ToolchainMacros) == 0
	case "workspace_macros":
		return len(command.WorkspaceMacros) == 0
	default:
		return true
	}
}

func unresolvedCommandFields(command CompilationCommand) []string {
	ensureCommandFieldMetadata(&command, command.Source)
	missing := make([]string, 0)
	for _, key := range commandFieldKeys {
		if command.FieldResolution[key] == "unresolved" || command.FieldSource[key] == "unresolved" {
			missing = append(missing, key)
		}
	}
	return missing
}

func applyPassInputs(context *SemanticContext, input ResolverInput) {
	if context == nil || (len(input.PassAvailability) == 0 && len(input.PassProvenance) == 0) {
		return
	}
	if context.PassViews == nil {
		context.PassViews = NewPassViews(context.Command.PassKind, context.Status)
	}
	for pass, availability := range input.PassAvailability {
		view, ok := context.PassViews[pass]
		if !ok {
			continue
		}
		view.PassKind = pass
		view.Availability = availability
		view.Provenance = mergePassProvenance(view.Provenance, input.PassProvenance[pass])
		context.PassViews[pass] = view
		if availability == PassAvailabilityMissing && context.Status == ContextStatusResolved {
			context.Status = ContextStatusPartial
		}
	}
	for pass, provenance := range input.PassProvenance {
		view, ok := context.PassViews[pass]
		if !ok {
			continue
		}
		view.Provenance = mergePassProvenance(view.Provenance, provenance)
		context.PassViews[pass] = view
	}
	configurePassViews(context)
}

func mergePassProvenance(base, extra map[string]string) map[string]string {
	result := CloneStringMap(base)
	if result == nil {
		result = make(map[string]string)
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}
