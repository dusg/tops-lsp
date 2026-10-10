package parser

type DiagnosticSeverity uint8

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

const (
	DiagnosticUnexpectedToken      = "tops-syntax-unexpected-token"
	DiagnosticMissingToken         = "tops-syntax-missing-token"
	DiagnosticUnterminated         = "tops-syntax-unterminated"
	DiagnosticInvalidLiteral       = "tops-syntax-invalid-literal"
	DiagnosticInvalidDirective     = "tops-syntax-invalid-directive"
	DiagnosticUnknownCondition     = "tops-syntax-unknown-condition"
	DiagnosticUnsupported          = "tops-syntax-unsupported"
	DiagnosticInvalidTopsAttribute = "tops-syntax-invalid-tops-attribute"
)

type RelatedLocation struct {
	URI     string
	Range   SourceRange
	Message string
}

type ParserDiagnostic struct {
	Code             string
	Severity         DiagnosticSeverity
	Message          string
	Range            SourceRange
	Related          []RelatedLocation
	Recoverable      bool
	Incomplete       bool
	ConditionalState ConditionalState
	DocumentVersion  int
	ContextVersion   int
}

func filterConditionalDiagnostics(diagnostics []ParserDiagnostic, tokens []Token) []ParserDiagnostic {
	filtered := make([]ParserDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		state := conditionalStateAt(tokens, diagnostic.Range.Start)
		if state == ConditionalInactive {
			continue
		}
		if state != "" {
			diagnostic.ConditionalState = state
		}
		filtered = append(filtered, diagnostic)
	}
	return filtered
}

func conditionalStateAt(tokens []Token, offset int) ConditionalState {
	for _, token := range tokens {
		if token.Range.Start > offset {
			break
		}
		if token.Range.Start <= offset && (offset < token.Range.End || token.Range.Start == token.Range.End) {
			return token.ConditionalState
		}
	}
	return ConditionalActive
}
