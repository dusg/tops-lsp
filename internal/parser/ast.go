package parser

type NodeKind string

const (
	NodeTranslationUnit       NodeKind = "translation-unit"
	NodeNamespace             NodeKind = "namespace"
	NodeUsing                 NodeKind = "using"
	NodeDeclaration           NodeKind = "declaration"
	NodeFunction              NodeKind = "function"
	NodeClass                 NodeKind = "class"
	NodeStruct                NodeKind = "struct"
	NodeEnum                  NodeKind = "enum"
	NodeUnion                 NodeKind = "union"
	NodeTemplate              NodeKind = "template"
	NodeStatement             NodeKind = "statement"
	NodeExpression            NodeKind = "expression"
	NodeAttribute             NodeKind = "attribute"
	NodeTopsQualifier         NodeKind = "tops-qualifier"
	NodeTopsLaunch            NodeKind = "tops-launch"
	NodePreprocessorDirective NodeKind = "preprocessor-directive"
	NodeConditionalRegion     NodeKind = "conditional-region"
	NodeOpaque                NodeKind = "opaque"
	NodeError                 NodeKind = "error"
	NodeMissingToken          NodeKind = "missing-token"
)

type SyntaxNode struct {
	Kind             NodeKind
	Range            SourceRange
	Name             string
	Value            string
	Operator         string
	Callee           string
	TypeTokens       []Token
	DeclaratorTokens []Token
	Expression       *SyntaxNode
	Children         []*SyntaxNode
	Tokens           []Token
	ConfigTokens     []Token
	ArgumentTokens   []Token
	ConditionalState ConditionalState
	Incomplete       bool
}

type ConditionalBranch struct {
	DirectiveRange SourceRange
	BodyRange      SourceRange
	Condition      []Token
	State          ConditionalState
	Nodes          []*SyntaxNode
}

type ConditionalRegion struct {
	ConditionRange       SourceRange
	Branches             []ConditionalBranch
	State                ConditionalState
	ParentID             int
	DirectiveDiagnostics []ParserDiagnostic
}

type MacroValueKind string

const (
	MacroUndefined MacroValueKind = "undefined"
	MacroDefined   MacroValueKind = "defined"
	MacroFunction  MacroValueKind = "function"
	MacroInteger   MacroValueKind = "integer"
	MacroUnknown   MacroValueKind = "unknown"
)

type MacroValue struct {
	Kind  MacroValueKind
	Value int64
}

type ParseContext struct {
	DocumentURI           string
	LanguageID            string
	LanguageStandard      string
	DriverKind            string
	CompilerContextStatus string
	ArgumentProvenance    string
	PredefinedMacros      map[string]MacroValue
	TargetProfile         string
	PassKind              string
	IncludeRoots          []string
	IncludeRootsAvailable bool
	DocumentVersion       int
	ContextVersion        int
}

type ParseStatus string

const (
	ParseStatusComplete  ParseStatus = "complete"
	ParseStatusRecovered ParseStatus = "recovered"
	ParseStatusPartial   ParseStatus = "partial"
	ParseStatusInvalid   ParseStatus = "invalid"
)

type ParseResult struct {
	Tokens             []Token
	Root               *SyntaxNode
	ConditionalRegions []ConditionalRegion
	Diagnostics        []ParserDiagnostic
	Status             ParseStatus
	DocumentVersion    int
	ContextVersion     int
	ConsumedBytes      int
}
