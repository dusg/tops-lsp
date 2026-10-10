package parser

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenizeResult struct {
	Tokens        []Token
	Diagnostics   []ParserDiagnostic
	ConsumedBytes int
}

var keywordSpellings = map[string]bool{
	"alignas": true, "alignof": true, "asm": true, "auto": true, "bool": true,
	"break": true, "case": true, "catch": true, "char": true, "char16_t": true, "char32_t": true, "class": true,
	"const": true, "constexpr": true, "const_cast": true, "consteval": true, "constinit": true, "continue": true,
	"decltype": true, "default": true, "delete": true, "do": true, "double": true,
	"dynamic_cast": true, "else": true, "enum": true, "explicit": true, "export": true,
	"extern": true, "false": true, "final": true, "float": true, "for": true,
	"friend": true, "goto": true, "if": true, "inline": true, "int": true, "thread_local": true,
	"long": true, "mutable": true, "namespace": true, "new": true, "noexcept": true,
	"nullptr": true, "operator": true, "private": true, "protected": true,
	"public": true, "register": true, "reinterpret_cast": true, "return": true,
	"short": true, "signed": true, "sizeof": true, "static": true, "static_assert": true,
	"static_cast": true, "struct": true, "switch": true, "template": true,
	"this": true, "throw": true, "true": true, "try": true, "typedef": true,
	"typename": true, "union": true, "unsigned": true, "using": true, "virtual": true,
	"void": true, "volatile": true, "wchar_t": true, "while": true,
	"concept": true, "requires": true, "co_await": true, "co_return": true, "co_yield": true, "char8_t": true,
	"and": true, "or": true, "not": true, "bitand": true, "bitor": true, "compl": true, "xor": true, "and_eq": true, "or_eq": true, "not_eq": true,
	"module": true, "import": true,
}

var topsKeywordSpellings = map[string]bool{
	"__global__": true, "__device__": true, "__host__": true, "__cooperative__": true,
	"__sp__": true, "__scalar_only__": true, "__noinline__": true, "__forceinline__": true,
	"__alwaysinline__": true, "__force_noinline__": true, "__inline_hint__": true,
	"__constant__": true, "__shared__": true, "__local__": true, "__private__": true,
	"__cluster_shared__": true, "__local_stack__": true, "__mmu_pointer__": true,
	"__restrict__": true, "__thread_dims__": true, "__cluster_dims__": true,
	"__launch_bounds__": true, "__maxnreg__": true, "__block_tile__": true,
	"__valigned__": true, "__vector": true, "__vector2": true, "__vector4": true,
	"__vector8": true, "__fp16": true, "__bf16": true,
}

var operatorSpellings = []string{
	"<<<=", ">>>=", "<<<", ">>>", "...", "->*", "<<=", ">>=", "##", "[[", "]]",
	"::", "->", ".*", "++", "--", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<=>",
	"(", ")", "{", "}", "[", "]", ";", ",", ".", ":", "?", "~", "!", "=", "+",
	"-", "*", "/", "%", "&", "|", "^", "<", ">", "#",
}

var (
	integerLiteralPattern = regexp.MustCompile(`^(?:0|[1-9][0-9']*|0[0-7][0-7']*|0[bB][01][01']*|0[xX][0-9a-fA-F][0-9a-fA-F']*)(?:[uU](?:[lL]{1,2})?|[lL]{1,2}[uU]?)?$`)
	floatLiteralPattern   = regexp.MustCompile(`^(?:(?:[0-9][0-9']*\.[0-9']*|\.[0-9][0-9']*|[0-9][0-9']*\.[0-9']*[eE][+-]?[0-9][0-9']*|\.[0-9][0-9']*[eE][+-]?[0-9][0-9']*|[0-9][0-9']*[eE][+-]?[0-9][0-9']*)[fFlL]?|0[xX](?:[0-9a-fA-F][0-9a-fA-F']*\.[0-9a-fA-F']*|\.[0-9a-fA-F][0-9a-fA-F']*|[0-9a-fA-F][0-9a-fA-F']*)[pP][+-]?[0-9][0-9']*[fFlL]?)$`)
)

func Tokenize(text string, context ParseContext) TokenizeResult {
	result := TokenizeResult{}
	lineStart := true
	for offset := 0; offset < len(text); {
		start := offset
		current := text[offset]
		if current == ' ' || current == '\t' || current == '\f' || current == '\v' {
			offset++
			continue
		}
		if current == '\r' && offset+1 < len(text) && text[offset+1] == '\n' {
			offset++
			continue
		}
		if current == '\\' {
			if offset+1 < len(text) && text[offset+1] == '\n' {
				offset += 2
				lineStart = false
				continue
			}
			if offset+2 < len(text) && text[offset+1] == '\r' && text[offset+2] == '\n' {
				offset += 3
				lineStart = false
				continue
			}
		}
		if current == '\n' || current == '\r' {
			offset++
			result.Tokens = append(result.Tokens, Token{Kind: TokenNewline, Spelling: text[start:offset], Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, LineStart: lineStart})
			lineStart = true
			continue
		}
		if strings.HasPrefix(text[offset:], "//") {
			offset += 2
			for offset < len(text) && text[offset] != '\n' && text[offset] != '\r' {
				offset++
			}
			result.Tokens = append(result.Tokens, Token{Kind: TokenComment, Spelling: text[start:offset], Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, LineStart: lineStart})
			lineStart = false
			continue
		}
		if strings.HasPrefix(text[offset:], "/*") {
			offset += 2
			closed := false
			for offset+1 < len(text) {
				if text[offset] == '*' && text[offset+1] == '/' {
					offset += 2
					closed = true
					break
				}
				offset++
			}
			if !closed {
				offset = len(text)
			}
			result.Tokens = append(result.Tokens, Token{Kind: TokenComment, Spelling: text[start:offset], Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, Unterminated: !closed, LineStart: lineStart})
			if !closed {
				result.Diagnostics = append(result.Diagnostics, diagnosticFor(context, DiagnosticUnterminated, "unterminated block comment", SourceRange{Start: start, End: offset}, true))
			}
			lineStart = false
			continue
		}
		if current == '"' || current == '\'' {
			var closed, invalid bool
			offset, closed, invalid = scanQuoted(text, offset, current)
			appendLiteral(&result, text, context, start, lineStart, literalScanResult{End: offset, Closed: closed, Invalid: invalid, UnterminatedMessage: "unterminated literal", InvalidMessage: "invalid literal"})
			lineStart = false
			continue
		}
		if rawOffset, ok := rawStringPrefixOffset(text, offset); ok {
			var closed, invalid bool
			offset, closed, invalid = scanRawString(text, rawOffset)
			appendLiteral(&result, text, context, start, lineStart, literalScanResult{End: offset, Closed: closed, Invalid: invalid, UnterminatedMessage: "unterminated raw literal", InvalidMessage: "invalid raw literal delimiter"})
			lineStart = false
			continue
		}
		if quoteOffset, ok := prefixedQuoteOffset(text, offset); ok {
			var closed, invalid bool
			offset, closed, invalid = scanQuoted(text, quoteOffset, text[quoteOffset])
			appendLiteral(&result, text, context, start, lineStart, literalScanResult{End: offset, Closed: closed, Invalid: invalid, UnterminatedMessage: "unterminated literal", InvalidMessage: "invalid literal"})
			lineStart = false
			continue
		}
		if isIdentifierStartAt(text, offset) {
			offset = scanIdentifier(text, offset)
			spelling := text[start:offset]
			kind := TokenIdentifier
			if keywordSpellings[spelling] || topsKeywordSpellings[spelling] {
				kind = TokenKeyword
			}
			result.Tokens = append(result.Tokens, Token{Kind: kind, Spelling: spelling, Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, LineStart: lineStart})
			lineStart = false
			continue
		}
		if isDigitAt(text, offset) {
			offset = scanNumber(text, offset)
			appendLiteral(&result, text, context, start, lineStart, literalScanResult{End: offset, Closed: true, Invalid: !validNumericLiteral(text[start:offset]), InvalidMessage: "invalid numeric literal"})
			lineStart = false
			continue
		}
		matched := ""
		for _, spelling := range operatorSpellings {
			if strings.HasPrefix(text[offset:], spelling) && len(spelling) > len(matched) {
				matched = spelling
			}
		}
		if matched != "" {
			offset += len(matched)
			result.Tokens = append(result.Tokens, Token{Kind: TokenPunctuation, Spelling: matched, Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, LineStart: lineStart})
			lineStart = false
			continue
		}
		_, size := utf8.DecodeRuneInString(text[offset:])
		if size == 0 {
			size = 1
		}
		offset += size
		result.Tokens = append(result.Tokens, Token{Kind: TokenUnknown, Spelling: text[start:offset], Range: SourceRange{Start: start, End: offset}, ConditionalState: ConditionalActive, LineStart: lineStart})
		result.Diagnostics = append(result.Diagnostics, diagnosticFor(context, DiagnosticUnexpectedToken, "unrecognized character", SourceRange{Start: start, End: offset}, false))
		lineStart = false
	}
	result.Tokens = append(result.Tokens, Token{Kind: TokenEOF, Range: SourceRange{Start: len(text), End: len(text)}, ConditionalState: ConditionalActive, LineStart: lineStart})
	result.ConsumedBytes = len(text)
	return result
}

func diagnosticFor(context ParseContext, code, message string, sourceRange SourceRange, incomplete bool) ParserDiagnostic {
	return ParserDiagnostic{Code: code, Severity: severityFor(code), Message: message, Range: sourceRange, Recoverable: true, Incomplete: incomplete, ConditionalState: ConditionalActive, DocumentVersion: context.DocumentVersion, ContextVersion: context.ContextVersion}
}

type literalScanResult struct {
	End                 int
	Closed              bool
	Invalid             bool
	UnterminatedMessage string
	InvalidMessage      string
}

func appendLiteral(result *TokenizeResult, text string, context ParseContext, start int, lineStart bool, scan literalScanResult) {
	result.Tokens = append(result.Tokens, Token{Kind: TokenLiteral, Spelling: text[start:scan.End], Range: SourceRange{Start: start, End: scan.End}, ConditionalState: ConditionalActive, Unterminated: !scan.Closed, LineStart: lineStart})
	if scan.Closed && !scan.Invalid {
		return
	}
	code := DiagnosticUnterminated
	message := scan.UnterminatedMessage
	incomplete := true
	if scan.Invalid {
		code = DiagnosticInvalidLiteral
		message = scan.InvalidMessage
		incomplete = false
	}
	result.Diagnostics = append(result.Diagnostics, diagnosticFor(context, code, message, SourceRange{Start: start, End: scan.End}, incomplete))
}

func severityFor(code string) DiagnosticSeverity {
	if code == DiagnosticUnterminated {
		return SeverityHint
	}
	if code == DiagnosticUnknownCondition || code == DiagnosticUnsupported {
		return SeverityInformation
	}
	return SeverityError
}

func scanQuoted(text string, offset int, quote byte) (int, bool, bool) {
	invalid := false
	for index := offset + 1; index < len(text); index++ {
		switch text[index] {
		case '\\':
			if index+1 >= len(text) {
				return len(text), false, invalid
			}
			next := text[index+1]
			if next == '\r' {
				if index+2 < len(text) && text[index+2] == '\n' {
					index += 2
				} else {
					index++
				}
				continue
			}
			if next == '\n' {
				index++
				continue
			}
			if !validEscape(text, index) {
				invalid = true
			}
			index++
		case '\r', '\n':
			return index, false, true
		case quote:
			if quote == '\'' && index == offset+1 {
				invalid = true
			}
			return index + 1, true, invalid
		}
	}
	return len(text), false, invalid
}

func validEscape(text string, slashIndex int) bool {
	next := text[slashIndex+1]
	switch next {
	case '\\', '\'', '"', '?', 'a', 'b', 'f', 'n', 'r', 't', 'v':
		return true
	case 'x':
		return slashIndex+2 < len(text) && isHexDigit(text[slashIndex+2])
	case 'u':
		return hasHexDigits(text, slashIndex+2, 4)
	case 'U':
		return hasHexDigits(text, slashIndex+2, 8)
	default:
		return next >= '0' && next <= '7'
	}
}

func hasHexDigits(text string, start, count int) bool {
	if start+count > len(text) {
		return false
	}
	for index := start; index < start+count; index++ {
		if !isHexDigit(text[index]) {
			return false
		}
	}
	return true
}

func isHexDigit(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f') || (value >= 'A' && value <= 'F')
}

func scanRawString(text string, offset int) (int, bool, bool) {
	open := strings.IndexByte(text[offset+2:], '(')
	if open < 0 {
		return len(text), false, true
	}
	open += offset + 2
	delimiter := text[offset+2 : open]
	closing := ")" + delimiter + "\""
	end := strings.Index(text[open+1:], closing)
	if end < 0 {
		return len(text), false, !validRawDelimiter(delimiter)
	}
	return open + 1 + end + len(closing), true, !validRawDelimiter(delimiter)
}

func validRawDelimiter(delimiter string) bool {
	if len(delimiter) > 16 {
		return false
	}
	for _, runeValue := range delimiter {
		if unicode.IsSpace(runeValue) || runeValue == '(' || runeValue == ')' || runeValue == '\\' || runeValue < 0x20 || runeValue == 0x7f {
			return false
		}
	}
	return true
}

func rawStringPrefixOffset(text string, offset int) (int, bool) {
	for _, prefix := range []string{"u8R", "uR", "UR", "LR", "R"} {
		if strings.HasPrefix(text[offset:], prefix+`"`) {
			return offset + len(prefix) - 1, true
		}
	}
	return 0, false
}

func prefixedQuoteOffset(text string, offset int) (int, bool) {
	if offset+1 < len(text) && (text[offset] == 'u' || text[offset] == 'U' || text[offset] == 'L') && (text[offset+1] == '"' || text[offset+1] == '\'') {
		return offset + 1, true
	}
	if offset+2 < len(text) && text[offset] == 'u' && text[offset+1] == '8' && (text[offset+2] == '"' || text[offset+2] == '\'') {
		return offset + 2, true
	}
	return 0, false
}

func isIdentifierStartAt(text string, offset int) bool {
	runeValue, size := utf8.DecodeRuneInString(text[offset:])
	return size > 0 && (runeValue == '_' || unicode.IsLetter(runeValue))
}

func scanIdentifier(text string, offset int) int {
	for offset < len(text) {
		runeValue, size := utf8.DecodeRuneInString(text[offset:])
		if size == 0 || !(runeValue == '_' || unicode.IsLetter(runeValue) || unicode.IsDigit(runeValue)) {
			break
		}
		offset += size
	}
	return offset
}

func isDigitAt(text string, offset int) bool {
	return text[offset] >= '0' && text[offset] <= '9'
}

func scanNumber(text string, offset int) int {
	start := offset
	for offset < len(text) {
		current := text[offset]
		if (current >= 'a' && current <= 'z') || (current >= 'A' && current <= 'Z') || (current >= '0' && current <= '9') || strings.ContainsRune("._'", rune(current)) {
			offset++
			continue
		}
		if (current == '+' || current == '-') && offset > start {
			previous := text[offset-1]
			if previous == 'e' || previous == 'E' || previous == 'p' || previous == 'P' {
				offset++
				continue
			}
		}
		break
	}
	return offset
}

func validNumericLiteral(spelling string) bool {
	if strings.Contains(spelling, "''") || strings.HasPrefix(spelling, "'") || strings.HasSuffix(spelling, "'") {
		return false
	}
	for index := 0; index < len(spelling); index++ {
		if spelling[index] != '\'' {
			continue
		}
		if index == 0 || index+1 >= len(spelling) || !isLiteralDigit(spelling[index-1]) || !isLiteralDigit(spelling[index+1]) {
			return false
		}
	}
	return integerLiteralPattern.MatchString(spelling) || floatLiteralPattern.MatchString(spelling)
}

func isLiteralDigit(value byte) bool {
	return (value >= '0' && value <= '9') || (value >= 'a' && value <= 'f') || (value >= 'A' && value <= 'F')
}
