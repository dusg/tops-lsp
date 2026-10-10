package parser

type TokenKind string

type ConditionalState string

const (
	ConditionalActive   ConditionalState = "active"
	ConditionalInactive ConditionalState = "inactive"
	ConditionalUnknown  ConditionalState = "unknown"
)

const (
	TokenEOF         TokenKind = "eof"
	TokenIdentifier  TokenKind = "identifier"
	TokenKeyword     TokenKind = "keyword"
	TokenLiteral     TokenKind = "literal"
	TokenOperator    TokenKind = "operator"
	TokenPunctuation TokenKind = "punctuation"
	TokenComment     TokenKind = "comment"
	TokenDirective   TokenKind = "directive"
	TokenNewline     TokenKind = "newline"
	TokenUnknown     TokenKind = "unknown"
)

type SourceRange struct {
	Start int
	End   int
}

type Token struct {
	Kind             TokenKind
	Spelling         string
	Range            SourceRange
	ConditionalState ConditionalState
	Unterminated     bool
	LeadingTrivia    []Token
	LineStart        bool
}
