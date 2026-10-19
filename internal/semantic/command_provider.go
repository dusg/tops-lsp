package semantic

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"tops-lsp/internal/index"
)

type FileCommandCandidateProviderOptions struct {
	CompileCommandsPath string
	WorkspaceCandidates []CommandCandidate
	WorkspaceFallback   WorkspaceFallbackFields
	Toolchain           ToolchainProvenance
	PassAvailability    map[PassKind]PassAvailability
	PassProvenance      map[PassKind]map[string]string
}

type FileCommandCandidateProvider struct {
	mu      sync.Mutex
	options FileCommandCandidateProviderOptions
	cache   map[string]compileCommandsCacheEntry
}

type compileCommandsCacheEntry struct {
	modTime time.Time
	size    int64
	records []compileCommandRecord
}

type compileCommandRecord struct {
	Directory string   `json:"directory"`
	File      string   `json:"file"`
	Arguments []string `json:"arguments"`
	Command   string   `json:"command"`
}

func NewFileCommandCandidateProvider(options FileCommandCandidateProviderOptions) *FileCommandCandidateProvider {
	return &FileCommandCandidateProvider{options: cloneFileCommandCandidateProviderOptions(options), cache: make(map[string]compileCommandsCacheEntry)}
}

func (provider *FileCommandCandidateProvider) Invalidate() {
	if provider == nil {
		return
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.cache = make(map[string]compileCommandsCacheEntry)
}

func (provider *FileCommandCandidateProvider) ProvideCommandCandidates(input CommandCandidateProviderInput) (CommandCandidateProviderOutput, error) {
	if provider == nil {
		return CommandCandidateProviderOutput{}, fmt.Errorf("file command candidate provider is nil")
	}
	output := CommandCandidateProviderOutput{
		DocumentURI:       input.DocumentURI,
		WorkspaceRoot:     input.WorkspaceRoot,
		WorkspaceFallback: normalizedWorkspaceFallback(provider.options.WorkspaceFallback),
		Toolchain:         normalizedToolchainProvenance(provider.options.Toolchain),
		PassAvailability:  ClonePassAvailability(provider.options.PassAvailability),
		PassProvenance:    ClonePassProvenance(provider.options.PassProvenance),
	}
	for _, candidate := range provider.options.WorkspaceCandidates {
		output.Candidates = append(output.Candidates, cloneCommandCandidate(candidate))
	}
	databasePath := provider.databasePath(input.WorkspaceRoot)
	if databasePath == "" {
		return output, nil
	}
	records, err := provider.loadCompileCommands(databasePath)
	if err != nil {
		return output, err
	}
	for _, record := range records {
		candidate := provider.compileCommandCandidate(record, input.WorkspaceRoot)
		if candidate.ParseError != "" || candidate.SourceURI != "" {
			output.Candidates = append(output.Candidates, candidate)
		}
	}
	if len(output.PassAvailability) == 0 {
		output.PassAvailability, output.PassProvenance = passInputsForCandidates(output.Candidates)
	}
	return output, nil
}

func (provider *FileCommandCandidateProvider) databasePath(workspaceRoot string) string {
	configured := strings.TrimSpace(provider.options.CompileCommandsPath)
	if configured != "" {
		if !filepath.IsAbs(configured) && workspaceRoot != "" {
			configured = filepath.Join(workspaceRoot, configured)
		}
		return filepath.Clean(configured)
	}
	for current := filepath.Clean(workspaceRoot); current != "." && current != ""; current = filepath.Dir(current) {
		candidate := filepath.Join(current, "compile_commands.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return ""
}

func (provider *FileCommandCandidateProvider) loadCompileCommands(path string) ([]compileCommandRecord, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	provider.mu.Lock()
	if cached, ok := provider.cache[path]; ok && cached.size == info.Size() && cached.modTime.Equal(info.ModTime()) {
		records := cloneCompileCommandRecords(cached.records)
		provider.mu.Unlock()
		return records, nil
	}
	provider.mu.Unlock()

	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []compileCommandRecord
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("parse compile commands %q: %w", path, err)
	}
	provider.mu.Lock()
	provider.cache[path] = compileCommandsCacheEntry{modTime: info.ModTime(), size: info.Size(), records: cloneCompileCommandRecords(records)}
	provider.mu.Unlock()
	return records, nil
}

func (provider *FileCommandCandidateProvider) compileCommandCandidate(record compileCommandRecord, workspaceRoot string) CommandCandidate {
	workingDirectory := strings.TrimSpace(record.Directory)
	if workingDirectory == "" {
		workingDirectory = workspaceRoot
	}
	if !filepath.IsAbs(workingDirectory) && workspaceRoot != "" {
		workingDirectory = filepath.Join(workspaceRoot, workingDirectory)
	}
	workingDirectory, err := filepath.Abs(filepath.Clean(workingDirectory))
	if err != nil {
		return CommandCandidate{SourceKind: "compile_commands", ParseError: err.Error()}
	}
	filePath := strings.TrimSpace(record.File)
	if filePath == "" {
		return CommandCandidate{SourceKind: "compile_commands", WorkingDirectory: workingDirectory, ParseError: "compile command file is empty"}
	}
	if parsed, parseErr := url.Parse(filePath); parseErr == nil && parsed.Scheme == "file" {
		filePath = parsed.Path
	}
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(workingDirectory, filePath)
	}
	sourceURI := index.NormalizeSourceURI((&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(filePath))}).String())
	arguments := append([]string(nil), record.Arguments...)
	if len(arguments) == 0 && strings.TrimSpace(record.Command) != "" {
		arguments, err = splitCommandLine(record.Command)
		if err != nil {
			return CommandCandidate{SourceKind: "compile_commands", SourceURI: sourceURI, WorkingDirectory: workingDirectory, ParseError: err.Error()}
		}
	}
	if len(arguments) == 0 {
		return CommandCandidate{SourceKind: "compile_commands", SourceURI: sourceURI, WorkingDirectory: workingDirectory, ParseError: "compile command arguments are empty"}
	}
	candidate := CommandCandidate{
		SourceKind:          "compile_commands",
		SourceURI:           sourceURI,
		WorkingDirectory:    workingDirectory,
		RawArguments:        append([]string(nil), arguments...),
		NormalizedArguments: append([]string(nil), arguments...),
	}
	populateCommandCandidateFields(&candidate, workingDirectory)
	return candidate
}

func populateCommandCandidateFields(candidate *CommandCandidate, workingDirectory string) {
	if candidate == nil {
		return
	}
	inferCandidateFields(candidate)
	for index := 0; index < len(candidate.NormalizedArguments); index++ {
		argument := candidate.NormalizedArguments[index]
		value := ""
		switch {
		case argument == "-I" || argument == "-isystem":
			if index+1 < len(candidate.NormalizedArguments) {
				index++
				value = candidate.NormalizedArguments[index]
			}
		case strings.HasPrefix(argument, "-I") && len(argument) > 2:
			value = strings.TrimPrefix(argument, "-I")
		case strings.HasPrefix(argument, "-isystem") && len(argument) > len("-isystem"):
			value = strings.TrimPrefix(argument, "-isystem")
		}
		if value != "" {
			if !filepath.IsAbs(value) {
				value = filepath.Join(workingDirectory, value)
			}
			candidate.WorkspaceRoots = append(candidate.WorkspaceRoots, filepath.Clean(value))
		}
	}
	for index := 0; index < len(candidate.NormalizedArguments); index++ {
		argument := candidate.NormalizedArguments[index]
		value := ""
		switch {
		case argument == "-D":
			if index+1 < len(candidate.NormalizedArguments) {
				index++
				value = candidate.NormalizedArguments[index]
			}
		case strings.HasPrefix(argument, "-D") && len(argument) > 2:
			value = strings.TrimPrefix(argument, "-D")
		}
		if name, macroValue, ok := splitMacro(value); ok {
			if candidate.WorkspaceMacros == nil {
				candidate.WorkspaceMacros = make(map[string]string)
			}
			candidate.WorkspaceMacros[name] = macroValue
		}
	}
	metadata := CompilationCommand{
		SourceURI:             candidate.SourceURI,
		WorkingDirectory:      candidate.WorkingDirectory,
		RawArguments:          candidate.RawArguments,
		NormalizedArguments:   candidate.NormalizedArguments,
		Arch:                  candidate.Arch,
		Driver:                candidate.Driver,
		Language:              candidate.Language,
		LanguageStandard:      candidate.LanguageStandard,
		PassKind:              candidate.PassKind,
		ToolchainIncludeRoots: candidate.ToolchainRoots,
		WorkspaceIncludeRoots: candidate.WorkspaceRoots,
		ToolchainMacros:       candidate.ToolchainMacros,
		WorkspaceMacros:       candidate.WorkspaceMacros,
	}
	candidate.FieldSource = make(map[string]string, len(commandFieldKeys))
	candidate.FieldResolution = make(map[string]string, len(commandFieldKeys))
	candidate.FieldConflict = make(map[string]*FieldConflict, len(commandFieldKeys))
	for _, key := range commandFieldKeys {
		if commandFieldIsEmpty(&metadata, key) {
			candidate.FieldSource[key] = "unresolved"
			candidate.FieldResolution[key] = "unresolved"
		} else {
			candidate.FieldSource[key] = "compile_commands"
			candidate.FieldResolution[key] = "resolved"
		}
		candidate.FieldConflict[key] = nil
	}
}

func normalizedWorkspaceFallback(value WorkspaceFallbackFields) WorkspaceFallbackFields {
	result := WorkspaceFallbackFields{IncludeRoots: append([]string(nil), value.IncludeRoots...), Macros: CloneStringMap(value.Macros), Version: value.Version, FieldSource: CloneStringMap(value.FieldSource), FieldResolution: CloneStringMap(value.FieldResolution)}
	if result.FieldSource == nil {
		result.FieldSource = make(map[string]string)
	}
	if result.FieldResolution == nil {
		result.FieldResolution = make(map[string]string)
	}
	if result.FieldSource["workspace_include_roots"] == "" {
		if len(result.IncludeRoots) > 0 {
			result.FieldSource["workspace_include_roots"] = "workspace"
		} else {
			result.FieldSource["workspace_include_roots"] = "unresolved"
		}
	}
	if result.FieldResolution["workspace_include_roots"] == "" {
		if len(result.IncludeRoots) > 0 {
			result.FieldResolution["workspace_include_roots"] = "resolved"
		} else {
			result.FieldResolution["workspace_include_roots"] = "unresolved"
		}
	}
	if result.FieldSource["workspace_macros"] == "" {
		if len(result.Macros) > 0 {
			result.FieldSource["workspace_macros"] = "workspace"
		} else {
			result.FieldSource["workspace_macros"] = "unresolved"
		}
	}
	if result.FieldResolution["workspace_macros"] == "" {
		if len(result.Macros) > 0 {
			result.FieldResolution["workspace_macros"] = "resolved"
		} else {
			result.FieldResolution["workspace_macros"] = "unresolved"
		}
	}
	return result
}

func normalizedToolchainProvenance(value ToolchainProvenance) ToolchainProvenance {
	result := ToolchainProvenance{Identity: value.Identity, Version: value.Version, IncludeRoots: append([]string(nil), value.IncludeRoots...), Macros: CloneStringMap(value.Macros), MacroVersion: value.MacroVersion, NormalizedRoots: append([]string(nil), value.NormalizedRoots...), FieldSource: CloneStringMap(value.FieldSource), FieldResolution: CloneStringMap(value.FieldResolution)}
	if result.FieldSource == nil {
		result.FieldSource = make(map[string]string)
	}
	if result.FieldResolution == nil {
		result.FieldResolution = make(map[string]string)
	}
	if result.FieldSource["toolchain_include_roots"] == "" {
		if len(result.IncludeRoots) > 0 {
			result.FieldSource["toolchain_include_roots"] = "toolchain_default"
		} else {
			result.FieldSource["toolchain_include_roots"] = "unresolved"
		}
	}
	if result.FieldResolution["toolchain_include_roots"] == "" {
		if len(result.IncludeRoots) > 0 {
			result.FieldResolution["toolchain_include_roots"] = "resolved"
		} else {
			result.FieldResolution["toolchain_include_roots"] = "unresolved"
		}
	}
	if result.FieldSource["toolchain_macros"] == "" {
		if len(result.Macros) > 0 {
			result.FieldSource["toolchain_macros"] = "toolchain_default"
		} else {
			result.FieldSource["toolchain_macros"] = "unresolved"
		}
	}
	if result.FieldResolution["toolchain_macros"] == "" {
		if len(result.Macros) > 0 {
			result.FieldResolution["toolchain_macros"] = "resolved"
		} else {
			result.FieldResolution["toolchain_macros"] = "unresolved"
		}
	}
	return result
}

func splitMacro(value string) (string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}
	name, macroValue, found := strings.Cut(value, "=")
	if !found {
		macroValue = "1"
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false
	}
	return name, macroValue, true
}

func splitCommandLine(value string) ([]string, error) {
	arguments := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			arguments = append(arguments, current.String())
			current.Reset()
		}
	}
	for _, character := range value {
		if escaped {
			current.WriteRune(character)
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			continue
		}
		switch {
		case character == '\'' || character == '"':
			quote = character
		case unicode.IsSpace(character):
			flush()
		default:
			current.WriteRune(character)
		}
	}
	if escaped {
		return nil, fmt.Errorf("command ends with an escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("command has an unterminated quote")
	}
	flush()
	return arguments, nil
}

func cloneFileCommandCandidateProviderOptions(value FileCommandCandidateProviderOptions) FileCommandCandidateProviderOptions {
	result := value
	result.WorkspaceCandidates = make([]CommandCandidate, 0, len(value.WorkspaceCandidates))
	for _, candidate := range value.WorkspaceCandidates {
		result.WorkspaceCandidates = append(result.WorkspaceCandidates, cloneCommandCandidate(candidate))
	}
	result.WorkspaceFallback = normalizedWorkspaceFallback(value.WorkspaceFallback)
	result.Toolchain = normalizedToolchainProvenance(value.Toolchain)
	result.PassAvailability = ClonePassAvailability(value.PassAvailability)
	result.PassProvenance = ClonePassProvenance(value.PassProvenance)
	return result
}

func passInputsForCandidates(candidates []CommandCandidate) (map[PassKind]PassAvailability, map[PassKind]map[string]string) {
	availability := make(map[PassKind]PassAvailability)
	provenance := make(map[PassKind]map[string]string)
	for _, candidate := range candidates {
		passes := []PassKind{candidate.PassKind}
		if candidate.PassKind == PassKindHostDevice {
			passes = []PassKind{PassKindHost, PassKindDevice}
		}
		for _, pass := range passes {
			if pass != PassKindHost && pass != PassKindDevice {
				continue
			}
			availability[pass] = PassAvailabilityAvailable
			provenance[pass] = map[string]string{"source": candidate.SourceKind, "pass_kind": string(pass)}
		}
	}
	return availability, provenance
}

func cloneCommandCandidate(value CommandCandidate) CommandCandidate {
	value.RawArguments = append([]string(nil), value.RawArguments...)
	value.NormalizedArguments = append([]string(nil), value.NormalizedArguments...)
	value.ToolchainRoots = append([]string(nil), value.ToolchainRoots...)
	value.WorkspaceRoots = append([]string(nil), value.WorkspaceRoots...)
	value.ToolchainMacros = CloneStringMap(value.ToolchainMacros)
	value.WorkspaceMacros = CloneStringMap(value.WorkspaceMacros)
	value.FieldSource = CloneStringMap(value.FieldSource)
	value.FieldResolution = CloneStringMap(value.FieldResolution)
	value.FieldConflict = cloneFieldConflictMap(value.FieldConflict)
	return value
}

func cloneFieldConflictMap(values map[string]*FieldConflict) map[string]*FieldConflict {
	if values == nil {
		return nil
	}
	result := make(map[string]*FieldConflict, len(values))
	for key, value := range values {
		result[key] = cloneFieldConflict(value)
	}
	return result
}

func cloneCompileCommandRecords(values []compileCommandRecord) []compileCommandRecord {
	result := make([]compileCommandRecord, 0, len(values))
	for _, value := range values {
		result = append(result, compileCommandRecord{Directory: value.Directory, File: value.File, Arguments: append([]string(nil), value.Arguments...), Command: value.Command})
	}
	return result
}
