package parser

import "strings"

func attachConditionalBranchNodes(regions []ConditionalRegion, tokens []Token, context ParseContext) {
	for regionIndex := range regions {
		for branchIndex := range regions[regionIndex].Branches {
			branch := &regions[regionIndex].Branches[branchIndex]
			branchTokens := tokensInRange(tokens, branch.BodyRange)
			if len(branchTokens) == 0 {
				continue
			}
			for tokenIndex := range branchTokens {
				branchTokens[tokenIndex].ConditionalState = ConditionalActive
			}
			branchParser := &syntaxParser{tokens: branchTokens, context: context}
			branch.Nodes, _ = branchParser.parseSequence(0, len(branchTokens), true)
		}
	}
}

func tokensInRange(tokens []Token, sourceRange SourceRange) []Token {
	result := make([]Token, 0)
	for _, token := range tokens {
		if token.Kind == TokenEOF {
			continue
		}
		if token.Range.Start >= sourceRange.Start && token.Range.End <= sourceRange.End {
			result = append(result, token)
		}
	}
	return result
}

func (parser *syntaxParser) isTopLevelFunctionStart(start, index, end int) bool {
	if !parser.tokens[index].LineStart {
		return false
	}
	if !hasTopLevelDeclarator(parser.significantTokens(start, index)) {
		return false
	}
	if !potentialDeclarationStart(parser.tokens[index].Spelling) {
		return false
	}
	for candidate := index + 1; candidate+1 < end; candidate++ {
		token := parser.tokens[candidate]
		if token.Spelling == ";" || token.Spelling == "=" || token.Spelling == "{" {
			return false
		}
		if (token.Kind == TokenIdentifier || token.Kind == TokenKeyword) && parser.tokens[candidate+1].Spelling == "(" {
			return true
		}
	}
	return false
}

func hasTopLevelDeclarator(tokens []Token) bool {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	angleDepth := 0
	for _, token := range tokens {
		switch token.Spelling {
		case "(":
			parenDepth++
		case ")":
			parenDepth--
		case "[":
			bracketDepth++
		case "]":
			bracketDepth--
		case "{":
			braceDepth++
		case "}":
			braceDepth--
		case "<":
			angleDepth++
		case ">":
			if angleDepth > 0 {
				angleDepth--
			}
		}
		if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && angleDepth == 0 && token.Kind == TokenIdentifier && !isAttributeName(token.Spelling) && !strings.HasPrefix(token.Spelling, "__") {
			return true
		}
	}
	return false
}

func potentialDeclarationStart(spelling string) bool {
	if topsKeywordSpellings[spelling] {
		return true
	}
	switch spelling {
	case "auto", "bool", "char", "class", "const", "constexpr", "double", "enum", "extern", "float", "inline", "int", "long", "namespace", "short", "signed", "static", "struct", "template", "typedef", "typename", "union", "unsigned", "using", "void", "volatile", "wchar_t":
		return true
	default:
		return false
	}
}

func (parser *syntaxParser) decorateStructure(node *SyntaxNode, tokens []Token) {
	switch node.Kind {
	case NodeDeclaration:
		node.TypeTokens, node.DeclaratorTokens = splitDeclarationTokens(tokens)
	case NodeFunction:
		node.TypeTokens, node.DeclaratorTokens = splitFunctionTokens(tokens)
	}
	if expressionTokens := expressionTokensForNode(node.Kind, tokens); len(expressionTokens) > 0 {
		node.Expression = parsePrattExpression(expressionTokens)
	}
}

func splitDeclarationTokens(tokens []Token) ([]Token, []Token) {
	searchEnd := len(tokens)
	if equalsIndex := topLevelToken(tokens, "="); equalsIndex >= 0 {
		searchEnd = equalsIndex
	}
	nameIndex := -1
	for index, token := range tokens[:searchEnd] {
		if token.Kind == TokenIdentifier {
			nameIndex = index
		}
	}
	if nameIndex <= 0 {
		return nil, append([]Token(nil), tokens...)
	}
	return append([]Token(nil), tokens[:nameIndex]...), append([]Token(nil), tokens[nameIndex:]...)
}

func splitFunctionTokens(tokens []Token) ([]Token, []Token) {
	openParen := -1
	for index, token := range tokens {
		if token.Spelling == "(" {
			openParen = index
			break
		}
	}
	if openParen <= 0 {
		return nil, append([]Token(nil), tokens...)
	}
	nameIndex := openParen - 1
	for nameIndex >= 0 && (tokens[nameIndex].Spelling == ">" || tokens[nameIndex].Spelling == "]") {
		nameIndex--
	}
	if nameIndex < 0 {
		return nil, append([]Token(nil), tokens...)
	}
	return append([]Token(nil), tokens[:nameIndex]...), append([]Token(nil), tokens[nameIndex:]...)
}

func expressionTokensForNode(kind NodeKind, tokens []Token) []Token {
	if len(tokens) == 0 {
		return nil
	}
	start := 0
	end := len(tokens)
	if tokens[end-1].Spelling == ";" {
		end--
	}
	switch kind {
	case NodeDeclaration:
		if equalsIndex := topLevelToken(tokens, "="); equalsIndex >= 0 && equalsIndex+1 < end {
			start = equalsIndex + 1
		}
	case NodeStatement:
		switch tokens[0].Spelling {
		case "return", "throw", "co_return":
			start = 1
		case "if", "while", "switch", "for":
			openParen := topLevelToken(tokens, "(")
			if openParen >= 0 {
				closeParen := matchingToken(tokens, openParen, "(", ")")
				if closeParen > openParen {
					start = openParen + 1
					end = closeParen
				}
			}
		}
	case NodeExpression:
	default:
		return nil
	}
	if start >= end {
		return nil
	}
	return append([]Token(nil), tokens[start:end]...)
}

func topLevelToken(tokens []Token, spelling string) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for index, token := range tokens {
		if token.Spelling == spelling && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			return index
		}
		switch token.Spelling {
		case "(":
			parenDepth++
		case ")":
			parenDepth--
		case "[":
			bracketDepth++
		case "]":
			bracketDepth--
		case "{":
			braceDepth++
		case "}":
			braceDepth--
		}
	}
	return -1
}

func matchingToken(tokens []Token, start int, opening, closing string) int {
	depth := 0
	for index := start; index < len(tokens); index++ {
		switch tokens[index].Spelling {
		case opening:
			depth++
		case closing:
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func isFunctionDeclarator(tokens []Token, openParen int) bool {
	if openParen <= 0 || openParen >= len(tokens) {
		return false
	}
	if topLevelToken(tokens[:openParen], "=") >= 0 {
		return false
	}
	name := tokens[openParen-1]
	if name.Kind != TokenIdentifier && name.Kind != TokenKeyword {
		return false
	}
	if hasDeclarationSpecifier(tokens[:openParen]) || tokens[0].Spelling == "template" || topsKeywordSpellings[tokens[0].Spelling] {
		return true
	}
	return openParen >= 2 && tokens[openParen-2].Kind == TokenIdentifier
}

func (parser *syntaxParser) unsupportedConstruct(tokens []Token) (Token, string, bool) {
	if parser.context.LanguageStandard == "c++11" {
		if token, ok := firstGenericLambdaAuto(tokens); ok {
			return token, "generic lambdas require C++14 or newer", true
		}
		if token, ok := firstReturnDeductionAuto(tokens); ok {
			return token, "return type deduction requires C++14 or newer", true
		}
	}
	for _, token := range tokens {
		switch token.Spelling {
		case "module", "import", "concept", "requires", "co_await", "co_return", "co_yield", "consteval", "constinit":
			return token, "syntax is outside the configured C++ baseline", true
		}
	}
	for index, token := range tokens {
		if parser.context.LanguageStandard != "c++17" && token.Spelling == "constexpr" && index > 0 && tokens[index-1].Spelling == "if" {
			return token, "syntax requires C++17 or newer", true
		}
		if parser.context.LanguageStandard != "c++17" && token.Spelling == "auto" && isStructuredBinding(tokens, index) {
			return token, "syntax requires C++17 or newer", true
		}
	}
	return Token{}, "", false
}
