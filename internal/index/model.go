package index

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"tops-lsp/internal/parser"
)

type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindType      SymbolKind = "type"
	SymbolKindAttribute SymbolKind = "attribute"
	SymbolKindBuiltin   SymbolKind = "builtin"
	SymbolKindNamespace SymbolKind = "namespace"
	SymbolKindField     SymbolKind = "field"
	SymbolKindVariable  SymbolKind = "variable"
	SymbolKindMacro     SymbolKind = "macro"
	SymbolKindHeader    SymbolKind = "header"
	SymbolKindObject    SymbolKind = "object"
)

// SymbolClass is the coarse category of an indexed symbol. Protocol document
// symbols, completion items and hover labels all derive from it so the kind and
// type-kind rules exist in exactly one place.
type SymbolClass string

const (
	SymbolClassFunction  SymbolClass = "function"
	SymbolClassEnum      SymbolClass = "enum"
	SymbolClassStruct    SymbolClass = "struct"
	SymbolClassUnion     SymbolClass = "union"
	SymbolClassAlias     SymbolClass = "alias"
	SymbolClassClass     SymbolClass = "class"
	SymbolClassNamespace SymbolClass = "namespace"
	SymbolClassObject    SymbolClass = "object"
	SymbolClassField     SymbolClass = "field"
	SymbolClassAttribute SymbolClass = "attribute"
	SymbolClassVariable  SymbolClass = "variable"
	SymbolClassBuiltin   SymbolClass = "builtin"
	SymbolClassMacro     SymbolClass = "macro"
	SymbolClassUnknown   SymbolClass = "unknown"
)

// SymbolClassOf classifies a symbol kind. Type symbols are split by their type
// kind; unrecognised kinds fall back to SymbolClassUnknown.
func SymbolClassOf(kind SymbolKind, typeKind string) SymbolClass {
	switch kind {
	case SymbolKindFunction:
		return SymbolClassFunction
	case SymbolKindType:
		switch typeKind {
		case "enum":
			return SymbolClassEnum
		case "struct":
			return SymbolClassStruct
		case "union":
			return SymbolClassUnion
		case "alias":
			return SymbolClassAlias
		}
		return SymbolClassClass
	case SymbolKindNamespace:
		return SymbolClassNamespace
	case SymbolKindObject:
		return SymbolClassObject
	case SymbolKindField:
		return SymbolClassField
	case SymbolKindAttribute:
		return SymbolClassAttribute
	case SymbolKindVariable:
		return SymbolClassVariable
	case SymbolKindBuiltin:
		return SymbolClassBuiltin
	case SymbolKindMacro:
		return SymbolClassMacro
	default:
		return SymbolClassUnknown
	}
}

type SymbolStatus string

const (
	SymbolStatusActive          SymbolStatus = "active"
	SymbolStatusConditional     SymbolStatus = "conditional"
	SymbolStatusDeclarationOnly SymbolStatus = "declaration-only"
	SymbolStatusIncomplete      SymbolStatus = "incomplete"
	SymbolStatusUnresolved      SymbolStatus = "unresolved"
	SymbolStatusAmbiguous       SymbolStatus = "ambiguous"
	SymbolStatusStale           SymbolStatus = "stale"
)

type IndexStatus string

const (
	IndexStatusComplete IndexStatus = "complete"
	IndexStatusPartial  IndexStatus = "partial"
	IndexStatusDegraded IndexStatus = "degraded"
	IndexStatusStale    IndexStatus = "stale"
)

type SymbolLocationRole string

const (
	SymbolLocationDeclaration SymbolLocationRole = "declaration"
	SymbolLocationDefinition  SymbolLocationRole = "definition"
	SymbolLocationReference   SymbolLocationRole = "reference"
	SymbolLocationBuiltin     SymbolLocationRole = "builtin-source"
	SymbolLocationAttribute   SymbolLocationRole = "attribute-source"
)

type HeaderDependencyStatus string

const (
	HeaderDependencyResolved   HeaderDependencyStatus = "resolved"
	HeaderDependencyUnresolved HeaderDependencyStatus = "unresolved"
	HeaderDependencyMissing    HeaderDependencyStatus = "missing"
	HeaderDependencyUnreadable HeaderDependencyStatus = "unreadable"
	HeaderDependencyCycle      HeaderDependencyStatus = "cycle"
	HeaderDependencyStale      HeaderDependencyStatus = "stale"
)

type IncludeForm string

const (
	IncludeFormQuoted IncludeForm = "quoted"
	IncludeFormAngle  IncludeForm = "angle"
)

type SourceVersion struct {
	URI             string
	DocumentVersion int
	ContextVersion  int
	SourceVersion   string
	Generation      uint64
	Epoch           uint64
}

type SymbolLocation struct {
	URI            string
	Range          parser.SourceRange
	SelectionRange parser.SourceRange
	Role           SymbolLocationRole
	Condition      ConditionalProvenance
	SourceVersion  SourceVersion
}

type BuiltinProvenance struct {
	TargetProfile string
	PassKind      string
	MacroContext  map[string]parser.MacroValue
	SourceURI     string
	Availability  string
}

type SymbolIdentity struct {
	OriginURI       string
	Kind            SymbolKind
	QualifiedName   string
	DeclarationRole string
	Anchor          string
}

type SymbolRecord struct {
	SymbolID            string
	Name                string
	QualifiedName       string
	Kind                SymbolKind
	TypeKind            string
	TypeName            string
	ScopeID             string
	ScopeName           string
	NameRange           parser.SourceRange
	Declarations        []SymbolLocation
	Definitions         []SymbolLocation
	References          []string
	Condition           ConditionalProvenance
	Origin              string
	OriginKind          string
	Detail              string
	Signature           string
	ParameterSummary    string
	ReturnTypeSummary   string
	ExecutionQualifiers []string
	AttributeUsages     []SymbolLocation
	AttributeSources    []SymbolLocation
	AttributeProvenance BuiltinProvenance
	Builtin             BuiltinProvenance
	Status              SymbolStatus
	SourceVersion       SourceVersion
	Children            []string
}

type ReferenceStatus string

const (
	ReferenceStatusResolved   ReferenceStatus = "resolved"
	ReferenceStatusAmbiguous  ReferenceStatus = "ambiguous"
	ReferenceStatusUnresolved ReferenceStatus = "unresolved"
	ReferenceStatusInactive   ReferenceStatus = "inactive"
	ReferenceStatusUnknown    ReferenceStatus = "unknown"
)

type ReferenceRecord struct {
	ReferenceID   string
	URI           string
	Range         parser.SourceRange
	Name          string
	QualifiedName string
	ScopeID       string
	Candidates    []string
	Condition     ConditionalProvenance
	Kind          string
	Status        ReferenceStatus
	SourceVersion SourceVersion
	Builtin       BuiltinProvenance
}

type ScopeKind string

const (
	ScopeTranslationUnit ScopeKind = "translation-unit"
	ScopeNamespace       ScopeKind = "namespace"
	ScopeClass           ScopeKind = "class"
	ScopeStruct          ScopeKind = "struct"
	ScopeUnion           ScopeKind = "union"
	ScopeFunction        ScopeKind = "function"
	ScopeBlock           ScopeKind = "block"
	ScopeHeader          ScopeKind = "header"
)

type ScopeRecord struct {
	ScopeID       string
	SourceURI     string
	Kind          ScopeKind
	Name          string
	QualifiedName string
	ParentID      string
	Range         parser.SourceRange
	Symbols       []string
	Children      []string
}

type ConditionalProvenance struct {
	SourceURI        string
	RegionID         int
	BranchIndex      int
	ExpressionRange  parser.SourceRange
	State            parser.ConditionalState
	ParentRegionID   int
	ConditionTokens  []parser.Token
	ParentConditions []ConditionalProvenance
}

type HeaderDependency struct {
	RequesterURI    string
	IncludeSpelling string
	IncludeForm     IncludeForm
	IncludeRange    parser.SourceRange
	TargetURI       string
	SearchRoots     []string
	Condition       ConditionalProvenance
	SourceVersion   SourceVersion
	Status          HeaderDependencyStatus
	Dependents      []string
}

type IndexDiagnostic struct {
	SourceURI        string
	SourceVersion    SourceVersion
	Code             string
	Message          string
	Range            parser.SourceRange
	Reason           string
	Condition        ConditionalProvenance
	ConditionalState parser.ConditionalState
}

type DocumentSymbol struct {
	Name           string
	Kind           SymbolKind
	TypeKind       string
	Range          parser.SourceRange
	SelectionRange parser.SourceRange
	Detail         string
	Children       []DocumentSymbol
}

type IndexSnapshot struct {
	RootURI                 string
	SourceVersion           SourceVersion
	ContextVersion          int
	Generation              uint64
	Epoch                   uint64
	ParseStatus             parser.ParseStatus
	IndexStatus             IndexStatus
	ASTRoot                 *parser.SyntaxNode
	Tokens                  []parser.Token
	ConditionalRegions      []parser.ConditionalRegion
	ConditionalRegionsByURI map[string][]parser.ConditionalRegion
	Sources                 map[string]string
	TargetProfile           string
	PassKind                string
	PredefinedMacros        map[string]parser.MacroValue
	Scopes                  map[string]ScopeRecord
	Symbols                 map[string]SymbolRecord
	References              map[string]ReferenceRecord
	RootSymbols             []string
	Dependencies            []HeaderDependency
	Diagnostics             []IndexDiagnostic
}

func StableSymbolID(identity SymbolIdentity) string {
	anchor := strings.Join(strings.Fields(identity.Anchor), " ")
	input := strings.Join([]string{identity.OriginURI, string(identity.Kind), identity.QualifiedName, anchor}, "\x00")
	digest := sha256.Sum256([]byte(input))
	return identity.OriginURI + "#" + string(identity.Kind) + ":" + hex.EncodeToString(digest[:12])
}
