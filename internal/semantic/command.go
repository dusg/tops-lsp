package semantic

type PassKind string

const (
	PassKindHost       PassKind = "host"
	PassKindDevice     PassKind = "device"
	PassKindHostDevice PassKind = "host+device"
)

type FieldValue = any

type ConflictSource struct {
	Kind     string `json:"kind"`
	Instance string `json:"instance"`
}

type FieldConflict struct {
	Sources []ConflictSource `json:"sources"`
	Values  []FieldValue     `json:"values"`
}

type CompilationCommand struct {
	Source                string
	SourceURI             string
	WorkingDirectory      string
	RawArguments          []string
	NormalizedArguments   []string
	Arch                  string
	Driver                string
	Language              string
	LanguageStandard      string
	PassKind              PassKind
	ToolchainIncludeRoots []string
	WorkspaceIncludeRoots []string
	ToolchainMacros       map[string]string
	WorkspaceMacros       map[string]string
	FieldSource           map[string]string
	FieldConflict         map[string]*FieldConflict
	FieldResolution       map[string]string
	Fallback              bool
	Version               int
}

type CommandCandidate struct {
	SourceKind          string
	SourceURI           string
	Version             string
	WorkingDirectory    string
	RawArguments        []string
	NormalizedArguments []string
	Language            string
	Arch                string
	Driver              string
	LanguageStandard    string
	PassKind            PassKind
	ToolchainRoots      []string
	WorkspaceRoots      []string
	ToolchainMacros     map[string]string
	WorkspaceMacros     map[string]string
	FieldSource         map[string]string
	FieldResolution     map[string]string
	FieldConflict       map[string]*FieldConflict
	ParseError          string
}
