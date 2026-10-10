package parser

import (
	"sort"
	"strings"
)

type syntaxParser struct {
	tokens      []Token
	context     ParseContext
	diagnostics []ParserDiagnostic
}

func Parse(text string, context ParseContext) ParseResult {
	tokenized := Tokenize(text, context)
	regions, conditionalDiagnostics := analyzeConditionals(tokenized.Tokens, context)
	lexicalDiagnostics := filterConditionalDiagnostics(tokenized.Diagnostics, tokenized.Tokens)
	parser := &syntaxParser{tokens: tokenized.Tokens, context: context, diagnostics: append(append([]ParserDiagnostic(nil), lexicalDiagnostics...), conditionalDiagnostics...)}
	root := &SyntaxNode{Kind: NodeTranslationUnit, Range: SourceRange{Start: 0, End: len(text)}, ConditionalState: ConditionalActive}
	root.Children, _ = parser.parseSequence(0, len(parser.tokens)-1, true)
	attachConditionalBranchNodes(regions, parser.tokens, context)
	for _, diagnostic := range conditionalDiagnostics {
		if diagnostic.Code == DiagnosticInvalidDirective && diagnostic.Message == "missing #endif" {
			root.Children = append(root.Children, missingTokenNode("#endif", diagnostic.Range, ConditionalActive))
		}
	}
	result := ParseResult{
		Tokens:             parser.tokens,
		Root:               root,
		ConditionalRegions: regions,
		Diagnostics:        parser.sortedDiagnostics(),
		Status:             ParseStatusComplete,
		DocumentVersion:    context.DocumentVersion,
		ContextVersion:     context.ContextVersion,
		ConsumedBytes:      len(text),
	}
	if len(result.Diagnostics) > 0 {
		result.Status = ParseStatusRecovered
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code == DiagnosticUnexpectedToken || diagnostic.Code == DiagnosticInvalidLiteral || diagnostic.Code == DiagnosticInvalidDirective {
				result.Status = ParseStatusInvalid
				break
			}
		}
	}
	if result.Status == ParseStatusComplete && partialContext(context) {
		result.Status = ParseStatusPartial
	}
	return result
}

func partialContext(context ParseContext) bool {
	return context.LanguageStandard == "" || context.LanguageStandard == "unknown" || context.DriverKind == "" || context.DriverKind == "unknown" || context.CompilerContextStatus == "" || context.CompilerContextStatus != "resolved" || context.ArgumentProvenance == "" || context.ArgumentProvenance == "unknown" || context.TargetProfile == "" || context.TargetProfile == "unknown" || !context.IncludeRootsAvailable
}

func (parser *syntaxParser) parseSequence(start, end int, reportEOF bool) ([]*SyntaxNode, int) {
	nodes := make([]*SyntaxNode, 0)
	index := start
	for index < end {
		index = skipTrivia(parser.tokens, index, end)
		if index >= end {
			break
		}
		if parser.tokens[index].Spelling == "}" {
			parser.addDiagnostic(DiagnosticUnexpectedToken, "unexpected '}'", parser.tokens[index].Range, false)
			nodes = append(nodes, errorNode(parser.tokens[index]))
			index++
			continue
		}
		if parser.tokens[index].Spelling == "#" {
			node, next := parser.parseDirective(index, end)
			if node != nil {
				nodes = append(nodes, node)
			}
			index = next
			continue
		}
		if parser.tokens[index].ConditionalState == ConditionalInactive {
			index++
			continue
		}
		node, next := parser.parseConstruct(index, end, reportEOF)
		if node != nil {
			nodes = append(nodes, node)
		}
		if next <= index {
			index++
		} else {
			index = next
		}
	}
	return nodes, index
}

func (parser *syntaxParser) parseConstruct(start, end int, reportEOF bool) (*SyntaxNode, int) {
	parenDepth := 0
	bracketDepth := 0
	angleDepth := 0
	templateHeader := false
	for index := start; index < end; index++ {
		token := parser.tokens[index]
		if index > start && parenDepth == 0 && bracketDepth == 0 && angleDepth == 0 && parser.isTopLevelFunctionStart(start, index, end) {
			missingRange := token.Range
			parser.addDiagnostic(DiagnosticMissingToken, "expected ';'", missingRange, true)
			children := []*SyntaxNode{missingTokenNode(";", missingRange, token.ConditionalState)}
			return parser.nodeFromRange(start, index, children), index
		}
		if token.Spelling == "template" && index+1 < end && parser.tokens[index+1].Spelling == "<" {
			templateHeader = true
		}
		if templateHeader {
			switch token.Spelling {
			case "<":
				angleDepth++
			case ">":
				if angleDepth > 0 {
					angleDepth--
				}
				if angleDepth == 0 {
					templateHeader = false
				}
			}
		}
		switch token.Spelling {
		case "(":
			parenDepth++
		case ")":
			if parenDepth == 0 {
				parser.addDiagnostic(DiagnosticUnexpectedToken, "unexpected ')'", token.Range, false)
				return parser.nodeFromRange(start, index+1, []*SyntaxNode{errorNode(token)}), index + 1
			}
			parenDepth--
		case "[":
			bracketDepth++
		case "]":
			if bracketDepth == 0 {
				parser.addDiagnostic(DiagnosticUnexpectedToken, "unexpected ']'", token.Range, false)
				return parser.nodeFromRange(start, index+1, []*SyntaxNode{errorNode(token)}), index + 1
			}
			bracketDepth--
		case "{":
			if parenDepth > 0 && bracketDepth == 0 && angleDepth == 0 {
				missingRange := SourceRange{Start: token.Range.Start, End: token.Range.Start}
				if opening, ok := parser.unclosedDelimiter(start, index, "(", ")"); ok {
					parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected ')'", missingRange, true, opening, "opening '('")
				} else {
					parser.addDiagnostic(DiagnosticMissingToken, "expected ')'", missingRange, true)
				}
				close := parser.matchBrace(index+1, end)
				if close < 0 {
					close = end
				}
				children, _ := parser.parseSequence(index+1, close, true)
				children = append(children, missingTokenNode(")", missingRange, token.ConditionalState))
				next := minInt(close+1, end)
				if next < end && parser.tokens[next].Spelling == ";" {
					next++
				}
				return parser.nodeFromRange(start, next, children), next
			}
			if parenDepth != 0 || bracketDepth != 0 || angleDepth != 0 {
				continue
			}
			close := parser.matchBrace(index+1, end)
			if close < 0 {
				endRange := parser.endRange(start, end)
				parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected '}'", SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}, true, token.Range, "opening '{'")
				children, _ := parser.parseSequence(index+1, end, true)
				children = append(children, missingTokenNode("}", SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}, token.ConditionalState))
				return parser.nodeFromRange(start, end, children), end
			}
			children, _ := parser.parseSequence(index+1, close, true)
			next := close + 1
			if next < end && parser.tokens[next].Spelling == ";" {
				next++
			}
			return parser.nodeFromRange(start, next, children), next
		case ";":
			if parenDepth == 0 && bracketDepth == 0 {
				parser.validatePrefix(start, index)
				return parser.nodeFromRange(start, index+1, nil), index + 1
			}
		}
	}

	if reportEOF && start < end {
		endRange := parser.endRange(start, end)
		var recoveryChildren []*SyntaxNode
		if opening, ok := parser.unclosedAttribute(start, end); ok {
			missingRange := SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}
			parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected ']]'", missingRange, true, opening, "opening '[['")
			recoveryChildren = append(recoveryChildren, missingTokenNode("]]", missingRange, parser.tokens[start].ConditionalState))
		} else if parenDepth > 0 && !parser.hasIncompleteLaunchArguments(start, end) {
			missingRange := SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}
			if opening, ok := parser.unclosedDelimiter(start, end, "(", ")"); ok {
				parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected ')'", missingRange, true, opening, "opening '('")
			} else {
				parser.addDiagnostic(DiagnosticMissingToken, "expected ')'", missingRange, true)
			}
			recoveryChildren = append(recoveryChildren, missingTokenNode(")", missingRange, parser.tokens[start].ConditionalState))
		} else if bracketDepth > 0 {
			missingRange := SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}
			if opening, ok := parser.unclosedDelimiter(start, end, "[", "]"); ok {
				parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected ']'", missingRange, true, opening, "opening '['")
			} else {
				parser.addDiagnostic(DiagnosticMissingToken, "expected ']'", missingRange, true)
			}
			recoveryChildren = append(recoveryChildren, missingTokenNode("]", missingRange, parser.tokens[start].ConditionalState))
		} else if angleDepth > 0 {
			missingRange := SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}
			if opening, ok := parser.unclosedDelimiter(start, end, "<", ">"); ok {
				parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected '>'", missingRange, true, opening, "opening '<'")
			} else {
				parser.addDiagnostic(DiagnosticMissingToken, "expected '>'", missingRange, true)
			}
			recoveryChildren = append(recoveryChildren, missingTokenNode(">", missingRange, parser.tokens[start].ConditionalState))
		} else if !parser.hasUnterminatedToken(start, end) && parser.tokens[end-1].Spelling != "}" && !parser.hasIncompleteLaunch(start, end) {
			missingRange := SourceRange{Start: lenRangeEnd(endRange), End: lenRangeEnd(endRange)}
			parser.addDiagnostic(DiagnosticMissingToken, "expected ';'", missingRange, true)
			recoveryChildren = append(recoveryChildren, missingTokenNode(";", missingRange, parser.tokens[start].ConditionalState))
		}
		return parser.nodeFromRange(start, end, recoveryChildren), end
	}
	return parser.nodeFromRange(start, end, nil), end
}

func (parser *syntaxParser) parseDirective(start, end int) (*SyntaxNode, int) {
	index := start
	for index < end && parser.tokens[index].Kind != TokenNewline {
		index++
	}
	return parser.nodeFromRange(start, index, nil), minInt(index+1, end)
}

func (parser *syntaxParser) matchBrace(start, end int) int {
	depth := 1
	for index := start; index < end; index++ {
		switch parser.tokens[index].Spelling {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func (parser *syntaxParser) nodeFromRange(start, end int, children []*SyntaxNode) *SyntaxNode {
	if start >= end || start >= len(parser.tokens) {
		return nil
	}
	end = minInt(end, len(parser.tokens))
	first := parser.tokens[start]
	last := parser.tokens[end-1]
	node := &SyntaxNode{Kind: parser.kindForRange(start, end), Range: SourceRange{Start: first.Range.Start, End: last.Range.End}, Children: children, ConditionalState: first.ConditionalState}
	prefix := parser.significantTokens(start, end)
	node.Name = nameForTokens(prefix)
	parser.decorateStructure(node, prefix)
	parser.validateUnsupported(prefix)
	if token, message, ok := parser.unsupportedConstruct(prefix); ok {
		node.Children = append(node.Children, &SyntaxNode{Kind: NodeOpaque, Name: token.Spelling, Value: message, Range: SourceRange{Start: prefix[0].Range.Start, End: prefix[len(prefix)-1].Range.End}, Tokens: append([]Token(nil), prefix...), ConditionalState: token.ConditionalState})
	}
	for index, token := range prefix {
		if topsKeywordSpellings[token.Spelling] {
			attributeEnd := attributeTokenEnd(prefix, index)
			attribute := &SyntaxNode{Kind: NodeTopsQualifier, Name: token.Spelling, Range: SourceRange{Start: token.Range.Start, End: prefix[attributeEnd-1].Range.End}, Tokens: append([]Token(nil), prefix[index:attributeEnd]...), ConditionalState: token.ConditionalState}
			node.Children = append(node.Children, attribute)
			index = attributeEnd - 1
		} else if token.Spelling == "[[" || token.Spelling == "__attribute__" {
			attributeEnd := attributeTokenEnd(prefix, index)
			attributeNode := &SyntaxNode{Kind: NodeAttribute, Name: token.Spelling, Range: SourceRange{Start: token.Range.Start, End: prefix[attributeEnd-1].Range.End}, Tokens: append([]Token(nil), prefix[index:attributeEnd]...), ConditionalState: token.ConditionalState, Incomplete: attributeIsIncomplete(prefix, index, attributeEnd)}
			if token.Spelling == "[[" {
				attributeName := standardAttributeName(prefix[index:attributeEnd])
				if attributeName != "" && !knownStandardAttribute(attributeName) {
					attributeNode.Children = append(attributeNode.Children, &SyntaxNode{Kind: NodeOpaque, Name: attributeName, Range: attributeNode.Range, Tokens: append([]Token(nil), prefix[index:attributeEnd]...), ConditionalState: token.ConditionalState})
				}
			}
			node.Children = append(node.Children, attributeNode)
			index = attributeEnd - 1
		}
	}
	parser.validateTopsAttributes(prefix)
	for index, token := range prefix {
		if token.Spelling == "<<<" {
			node.Children = append(node.Children, parser.launchNode(prefix, index))
		}
	}
	if node.Kind == NodeTemplate {
		if function := nodeFromTokens(prefix); function != nil {
			node.Children = append([]*SyntaxNode{function}, node.Children...)
		}
	}
	return node
}

func (parser *syntaxParser) hasUnterminatedToken(start, end int) bool {
	for index := end - 1; index >= start; index-- {
		token := parser.tokens[index]
		if token.Kind == TokenComment || token.Kind == TokenNewline {
			continue
		}
		return token.Unterminated
	}
	return false
}

func (parser *syntaxParser) kindForRange(start, end int) NodeKind {
	prefix := parser.significantTokens(start, end)
	if len(prefix) == 0 {
		return NodeOpaque
	}
	if prefix[0].Spelling == "#" {
		return NodePreprocessorDirective
	}
	for _, token := range prefix {
		switch token.Spelling {
		case "namespace":
			return NodeNamespace
		case "using":
			return NodeUsing
		case "class":
			return NodeClass
		case "struct":
			return NodeStruct
		case "enum":
			return NodeEnum
		case "union":
			return NodeUnion
		case "template":
			return NodeTemplate
		}
	}
	first := prefix[0].Spelling
	switch first {
	case "return", "if", "else", "for", "while", "do", "switch", "case", "break", "continue", "throw", "try":
		return NodeStatement
	}
	for index, token := range prefix {
		if token.Spelling == "(" && isFunctionDeclarator(prefix, index) {
			return NodeFunction
		}
	}
	if hasDeclarationSpecifier(prefix) {
		return NodeDeclaration
	}
	if hasExpressionOperator(prefix) {
		return NodeExpression
	}
	return NodeDeclaration
}

func hasDeclarationSpecifier(tokens []Token) bool {
	for _, token := range tokens {
		switch token.Spelling {
		case "auto", "bool", "char", "double", "float", "int", "long", "short", "signed", "unsigned", "void", "wchar_t", "const", "constexpr", "static", "extern", "volatile":
			return true
		}
	}
	return false
}

func nodeFromTokens(tokens []Token) *SyntaxNode {
	for index, token := range tokens {
		if token.Spelling == "(" && index > 0 && !isAttributeName(tokens[index-1].Spelling) {
			name := tokens[index-1]
			if name.Kind == TokenIdentifier || name.Kind == TokenKeyword {
				return &SyntaxNode{Kind: NodeFunction, Name: name.Spelling, Range: SourceRange{Start: tokens[0].Range.Start, End: tokens[len(tokens)-1].Range.End}, ConditionalState: tokens[0].ConditionalState}
			}
		}
	}
	return nil
}

func (parser *syntaxParser) significantTokens(start, end int) []Token {
	result := make([]Token, 0, end-start)
	for index := start; index < end && index < len(parser.tokens); index++ {
		token := parser.tokens[index]
		if token.Kind != TokenComment && token.Kind != TokenNewline {
			result = append(result, token)
		}
	}
	return result
}

func nameForTokens(tokens []Token) string {
	for index, token := range tokens {
		switch token.Spelling {
		case "namespace":
			nameStart := index + 1
			if nameStart < len(tokens) && tokens[nameStart].Spelling == "inline" {
				nameStart++
			}
			if nameStart >= len(tokens) || tokens[nameStart].Spelling == "{" {
				return ""
			}
			var builder strings.Builder
			for candidateIndex := nameStart; candidateIndex < len(tokens); candidateIndex++ {
				candidate := tokens[candidateIndex]
				if candidate.Spelling == "{" || candidate.Spelling == ";" {
					break
				}
				if candidate.Kind == TokenIdentifier || candidate.Spelling == "::" {
					builder.WriteString(candidate.Spelling)
					continue
				}
				break
			}
			return builder.String()
		case "class", "struct", "enum", "union":
			for _, candidate := range tokens[index+1:] {
				if candidate.Kind == TokenIdentifier {
					return candidate.Spelling
				}
			}
		}
	}
	if equalsIndex := topLevelToken(tokens, "="); equalsIndex > 0 {
		for index := equalsIndex - 1; index >= 0; index-- {
			if tokens[index].Kind == TokenIdentifier {
				return tokens[index].Spelling
			}
		}
	}
	for index, token := range tokens {
		if token.Spelling == "(" && index > 0 && !isAttributeName(tokens[index-1].Spelling) {
			return tokens[index-1].Spelling
		}
	}
	for index := len(tokens) - 1; index >= 0; index-- {
		if tokens[index].Kind == TokenIdentifier {
			return tokens[index].Spelling
		}
	}
	return ""
}

func isAttributeName(spelling string) bool {
	return topsKeywordSpellings[spelling] || spelling == "__attribute__"
}

func standardAttributeName(tokens []Token) string {
	for _, token := range tokens[1:] {
		if token.Kind == TokenIdentifier || token.Kind == TokenKeyword {
			return token.Spelling
		}
	}
	return ""
}

func knownStandardAttribute(name string) bool {
	switch name {
	case "deprecated", "fallthrough", "maybe_unused", "nodiscard", "noreturn", "carries_dependency", "no_unique_address":
		return true
	default:
		return false
	}
}

func (parser *syntaxParser) validateTopsAttributes(tokens []Token) {
	for index := 0; index+2 < len(tokens); index++ {
		if !topsKeywordSpellings[tokens[index].Spelling] || tokens[index+1].Spelling != "(" || tokens[index+2].Spelling != ")" {
			continue
		}
		if tokens[index].Spelling == "__thread_dims__" || tokens[index].Spelling == "__cluster_dims__" {
			continue
		}
		parser.addDiagnostic(DiagnosticInvalidTopsAttribute, "Tops attribute requires an argument", SourceRange{Start: tokens[index].Range.Start, End: tokens[index+2].Range.End}, false)
	}
}

func attributeTokenEnd(tokens []Token, start int) int {
	if tokens[start].Spelling == "[[" {
		for index := start + 1; index < len(tokens); index++ {
			if tokens[index].Spelling == "]]" {
				return index + 1
			}
		}
		return len(tokens)
	}
	if start+1 >= len(tokens) || tokens[start+1].Spelling != "(" {
		return start + 1
	}
	depth := 0
	for index := start + 1; index < len(tokens); index++ {
		switch tokens[index].Spelling {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return index + 1
			}
		}
	}
	return len(tokens)
}

func attributeIsIncomplete(tokens []Token, start, end int) bool {
	if start >= end || end > len(tokens) {
		return false
	}
	last := tokens[end-1].Spelling
	if tokens[start].Spelling == "[[" {
		return last != "]]"
	}
	if tokens[start].Spelling == "__attribute__" {
		return last != ")"
	}
	return false
}

func (parser *syntaxParser) unclosedAttribute(start, end int) (SourceRange, bool) {
	for index := start; index < end && index < len(parser.tokens); index++ {
		if parser.tokens[index].Spelling != "[[" {
			continue
		}
		for close := index + 1; close < end && close < len(parser.tokens); close++ {
			if parser.tokens[close].Spelling == "]]" {
				return SourceRange{}, false
			}
		}
		return parser.tokens[index].Range, true
	}
	return SourceRange{}, false
}

func (parser *syntaxParser) validateUnsupported(tokens []Token) {
	if parser.context.LanguageStandard != "c++11" && parser.context.LanguageStandard != "c++14" && parser.context.LanguageStandard != "c++17" {
		return
	}
	if parser.context.LanguageStandard == "c++11" {
		if token, ok := firstGenericLambdaAuto(tokens); ok {
			parser.addDiagnostic(DiagnosticUnsupported, "generic lambdas require C++14 or newer", token.Range, false)
			return
		}
		if token, ok := firstReturnDeductionAuto(tokens); ok {
			parser.addDiagnostic(DiagnosticUnsupported, "return type deduction requires C++14 or newer", token.Range, false)
			return
		}
	}
	for index, token := range tokens {
		switch token.Spelling {
		case "module", "import", "concept", "requires", "co_await", "co_return", "co_yield", "consteval", "constinit":
			parser.addDiagnostic(DiagnosticUnsupported, "syntax is outside the configured C++ baseline", token.Range, false)
			return
		}
		if parser.context.LanguageStandard != "c++17" && token.Spelling == "constexpr" && index > 0 && tokens[index-1].Spelling == "if" {
			parser.addDiagnostic(DiagnosticUnsupported, "syntax requires C++17 or newer", token.Range, false)
			return
		}
		if parser.context.LanguageStandard != "c++17" && token.Spelling == "auto" && isStructuredBinding(tokens, index) {
			parser.addDiagnostic(DiagnosticUnsupported, "syntax requires C++17 or newer", token.Range, false)
			return
		}
	}
}

func firstGenericLambdaAuto(tokens []Token) (Token, bool) {
	for index, token := range tokens {
		if token.Spelling != "[" {
			continue
		}
		close := index + 1
		for close < len(tokens) && tokens[close].Spelling != "]" {
			close++
		}
		if close >= len(tokens) || close+1 >= len(tokens) || tokens[close+1].Spelling != "(" {
			continue
		}
		for parameter := close + 2; parameter < len(tokens) && tokens[parameter].Spelling != ")"; parameter++ {
			if tokens[parameter].Spelling == "auto" {
				return tokens[parameter], true
			}
		}
	}
	return Token{}, false
}

func firstReturnDeductionAuto(tokens []Token) (Token, bool) {
	for index := 0; index+2 < len(tokens); index++ {
		if tokens[index].Spelling != "auto" || tokens[index+1].Kind != TokenIdentifier || tokens[index+2].Spelling != "(" {
			continue
		}
		return tokens[index], true
	}
	return Token{}, false
}

func isStructuredBinding(tokens []Token, autoIndex int) bool {
	if autoIndex+2 >= len(tokens) || tokens[autoIndex+1].Spelling != "[" {
		return false
	}
	for index := autoIndex + 2; index < len(tokens); index++ {
		if tokens[index].Spelling == "]" {
			return index+1 < len(tokens) && tokens[index+1].Spelling == "="
		}
		if tokens[index].Spelling == ";" || tokens[index].Spelling == "{" {
			return false
		}
	}
	return false
}

func hasExpressionOperator(tokens []Token) bool {
	for _, token := range tokens {
		switch token.Spelling {
		case "=", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "&&", "||", "?":
			return true
		}
	}
	return false
}

func (parser *syntaxParser) validatePrefix(start, end int) {
	prefix := parser.significantTokens(start, end)
	if len(prefix) >= 2 && prefix[len(prefix)-1].Spelling == "=" {
		parser.addDiagnostic(DiagnosticUnexpectedToken, "expected expression after '='", prefix[len(prefix)-1].Range, false)
	}
}

func (parser *syntaxParser) endRange(start, end int) SourceRange {
	if start >= end || start >= len(parser.tokens) {
		return SourceRange{}
	}
	end = minInt(end, len(parser.tokens))
	return SourceRange{Start: parser.tokens[start].Range.Start, End: parser.tokens[end-1].Range.End}
}

func lenRangeEnd(sourceRange SourceRange) int {
	return sourceRange.End
}

func (parser *syntaxParser) addDiagnostic(code, message string, sourceRange SourceRange, incomplete bool) {
	parser.diagnostics = append(parser.diagnostics, ParserDiagnostic{Code: code, Severity: severityFor(code), Message: message, Range: sourceRange, Recoverable: true, Incomplete: incomplete, ConditionalState: parser.diagnosticStateAt(sourceRange.Start), DocumentVersion: parser.context.DocumentVersion, ContextVersion: parser.context.ContextVersion})
}

func (parser *syntaxParser) addDiagnosticWithRelated(code, message string, sourceRange SourceRange, incomplete bool, relatedRange SourceRange, relatedMessage string) {
	diagnostic := ParserDiagnostic{Code: code, Severity: severityFor(code), Message: message, Range: sourceRange, Recoverable: true, Incomplete: incomplete, ConditionalState: parser.diagnosticStateAt(sourceRange.Start), DocumentVersion: parser.context.DocumentVersion, ContextVersion: parser.context.ContextVersion}
	diagnostic.Related = []RelatedLocation{{URI: parser.context.DocumentURI, Range: relatedRange, Message: relatedMessage}}
	parser.diagnostics = append(parser.diagnostics, diagnostic)
}

func (parser *syntaxParser) diagnosticStateAt(offset int) ConditionalState {
	return conditionalStateAt(parser.tokens, offset)
}

func missingTokenNode(expected string, sourceRange SourceRange, state ConditionalState) *SyntaxNode {
	return &SyntaxNode{Kind: NodeMissingToken, Range: sourceRange, Value: expected, ConditionalState: state, Incomplete: true}
}

func errorNode(token Token) *SyntaxNode {
	return &SyntaxNode{Kind: NodeError, Range: token.Range, Value: token.Spelling, ConditionalState: token.ConditionalState}
}

func (parser *syntaxParser) unclosedDelimiter(start, end int, opening, closing string) (SourceRange, bool) {
	openings := make([]SourceRange, 0)
	for index := start; index < end && index < len(parser.tokens); index++ {
		switch parser.tokens[index].Spelling {
		case opening:
			openings = append(openings, parser.tokens[index].Range)
		case closing:
			if len(openings) > 0 {
				openings = openings[:len(openings)-1]
			}
		}
	}
	if len(openings) == 0 {
		return SourceRange{}, false
	}
	return openings[len(openings)-1], true
}

func (parser *syntaxParser) sortedDiagnostics() []ParserDiagnostic {
	result := append([]ParserDiagnostic(nil), parser.diagnostics...)
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Range.Start != result[right].Range.Start {
			return result[left].Range.Start < result[right].Range.Start
		}
		return result[left].Code < result[right].Code
	})
	unique := result[:0]
	for _, diagnostic := range result {
		if len(unique) > 0 {
			previous := unique[len(unique)-1]
			if previous.Code == diagnostic.Code && previous.Range == diagnostic.Range {
				continue
			}
		}
		unique = append(unique, diagnostic)
	}
	return unique
}

func skipTrivia(tokens []Token, start, end int) int {
	for start < end && (tokens[start].Kind == TokenComment || tokens[start].Kind == TokenNewline) {
		start++
	}
	return start
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
