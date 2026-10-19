package server

import (
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

type completionRegion int

const (
	completionRegionNormal completionRegion = iota
	completionRegionBlocked
	completionRegionDeclarationStart
	completionRegionAttribute
	completionRegionType
	completionRegionScope
	completionRegionExpression
)

var declarationStartTokens = map[string]bool{
	";": true,
	"{": true,
	"}": true,
	":": true,
}

var typeContextTokens = map[string]bool{
	"auto": true, "bool": true, "char": true, "char8_t": true, "char16_t": true, "char32_t": true,
	"double": true, "float": true, "int": true, "long": true, "short": true, "signed": true,
	"unsigned": true, "void": true, "wchar_t": true, "const": true, "constexpr": true,
	"constinit": true, "consteval": true, "decltype": true, "mutable": true, "static": true,
	"volatile": true, "struct": true, "class": true, "enum": true, "union": true, "typename": true,
	"using": true, "typedef": true, "namespace": true, "template": true, "extern": true,
	"__vector": true, "__vector2": true, "__vector4": true, "__vector8": true, "__fp16": true, "__bf16": true,
}

var expressionTokens = map[string]bool{
	"(": true, ",": true, "=": true, "+": true, "-": true, "*": true, "/": true, "%": true,
	"&": true, "|": true, "^": true, "!": true, "~": true, "?": true, "<": true, ">": true,
	"==": true, "!=": true, "<=": true, ">=": true, "&&": true, "||": true, "+=": true,
	"-=": true, "*=": true, "/=": true, "%=": true, "&=": true, "|=": true, "^=": true,
	"<<": true, ">>": true, "<<=": true, ">>=": true, "return": true, "case": true, "throw": true,
	"co_return": true, "co_yield": true, "new": true, "delete": true, "sizeof": true, "alignof": true,
}

var topsTypeKeywords = map[string]bool{
	"__vector": true, "__vector2": true, "__vector4": true, "__vector8": true, "__fp16": true, "__bf16": true,
}

// classifyCompletionRegion maps the cursor position to the syntax rule table in
// contracts/completion-candidates.md section 3. The previous non-whitespace
// token decides the category; the first matching rule wins.
func classifyCompletionRegion(snapshot index.IndexSnapshot, uri string, offset int) completionRegion {
	if offset < 0 || offset > len(snapshot.Sources[uri]) {
		return completionRegionBlocked
	}
	if completionRegionBlockedAt(snapshot, uri, offset) {
		return completionRegionBlocked
	}
	previous := previousSignificantToken(snapshot.Tokens, offset)
	prefix := identifierPrefix(snapshot.Sources[uri], offset)
	if strings.HasPrefix(prefix, "__") {
		return completionRegionAttribute
	}
	if previous == "" {
		return completionRegionDeclarationStart
	}
	switch {
	case previous == "." || previous == "->":
		return completionRegionBlocked
	case previous == "::":
		return completionRegionScope
	case declarationStartTokens[previous]:
		return completionRegionDeclarationStart
	case typeContextTokens[previous]:
		return completionRegionType
	case expressionTokens[previous]:
		return completionRegionExpression
	}
	return completionRegionNormal
}

func completionRegionBlockedAt(snapshot index.IndexSnapshot, uri string, offset int) bool {
	text := snapshot.Sources[uri]
	lineStart, lineEnd := lineBounds(text, offset)
	// One pass covers comments, literals, unterminated literals and directives;
	// each case returns the same verdict the three separate scans produced.
	for _, token := range snapshot.Tokens {
		switch token.Kind {
		case parser.TokenComment, parser.TokenLiteral:
			if token.Range.Start <= offset && offset < token.Range.End {
				return true
			}
			if token.Unterminated && token.Range.Start <= offset {
				return true
			}
		case parser.TokenDirective:
			if token.Range.Start < lineEnd && token.Range.End > lineStart {
				return true
			}
		}
	}
	for _, region := range snapshot.ConditionalRegions {
		for _, branch := range region.Branches {
			if branch.State != parser.ConditionalUnknown {
				continue
			}
			if branch.BodyRange.Start <= offset && offset <= branch.BodyRange.End {
				return true
			}
		}
	}
	return false
}

func previousSignificantToken(tokens []parser.Token, offset int) string {
	significant := ""
	for _, token := range tokens {
		if token.Range.End > offset {
			break
		}
		switch token.Kind {
		case parser.TokenNewline, parser.TokenComment, parser.TokenEOF:
			continue
		}
		significant = token.Spelling
	}
	return significant
}

func lineBounds(text string, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	start := strings.LastIndex(text[:offset], "\n") + 1
	end := len(text)
	if next := strings.Index(text[offset:], "\n"); next >= 0 {
		end = offset + next
	}
	return start, end
}

func identifierPrefix(text string, offset int) string {
	if offset < 0 {
		return ""
	}
	if offset > len(text) {
		offset = len(text)
	}
	start := offset
	for start > 0 && isIdentifierCharacter(text[start-1]) {
		start--
	}
	return text[start:offset]
}

func isIdentifierCharacter(value byte) bool {
	return value == '_' || (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

// completionSymbolAllowed applies the "allowed candidate categories" column of
// the syntax rule table.
func completionSymbolAllowed(region completionRegion, kind index.SymbolKind) bool {
	switch region {
	case completionRegionNormal:
		return true
	case completionRegionBlocked:
		return false
	case completionRegionDeclarationStart:
		return kind == index.SymbolKindType || kind == index.SymbolKindFunction || kind == index.SymbolKindAttribute
	case completionRegionAttribute:
		return kind == index.SymbolKindAttribute
	case completionRegionType:
		return kind == index.SymbolKindType
	case completionRegionScope:
		return kind == index.SymbolKindNamespace || kind == index.SymbolKindType
	case completionRegionExpression:
		switch kind {
		case index.SymbolKindFunction, index.SymbolKindVariable, index.SymbolKindBuiltin, index.SymbolKindField, index.SymbolKindMacro, index.SymbolKindAttribute:
			return true
		}
		return false
	}
	return false
}

func completionKeywordAllowed(region completionRegion, spelling string, tops bool) bool {
	if tops {
		switch region {
		case completionRegionAttribute, completionRegionDeclarationStart, completionRegionNormal:
			return true
		case completionRegionType:
			return topsTypeKeywords[spelling]
		default:
			return false
		}
	}
	switch region {
	case completionRegionBlocked:
		return false
	case completionRegionAttribute, completionRegionScope:
		return false
	default:
		return true
	}
}
