package parser

func parsePrattExpression(tokens []Token) *SyntaxNode {
	parser := &expressionParser{tokens: tokens}
	return parser.parse(0)
}

type expressionParser struct {
	tokens []Token
	index  int
}

func (parser *expressionParser) parse(minPrecedence int) *SyntaxNode {
	left := parser.parsePrefix()
	if left == nil {
		return nil
	}
	for parser.index < len(parser.tokens) {
		token := parser.tokens[parser.index]
		if token.Spelling == "(" {
			left = parser.parseCall(left)
			continue
		}
		if token.Spelling == "." || token.Spelling == "->" {
			parser.index++
			member := parser.parsePrefix()
			if member == nil {
				return left
			}
			left = &SyntaxNode{Kind: NodeExpression, Operator: token.Spelling, Range: SourceRange{Start: left.Range.Start, End: member.Range.End}, Children: []*SyntaxNode{left, member}, ConditionalState: left.ConditionalState}
			continue
		}
		precedence, ok := expressionPrecedence(token.Spelling)
		if !ok || precedence < minPrecedence {
			break
		}
		parser.index++
		right := parser.parse(precedence + 1)
		if right == nil {
			break
		}
		left = &SyntaxNode{Kind: NodeExpression, Operator: token.Spelling, Range: SourceRange{Start: left.Range.Start, End: right.Range.End}, Children: []*SyntaxNode{left, right}, ConditionalState: left.ConditionalState}
	}
	return left
}

func (parser *expressionParser) parsePrefix() *SyntaxNode {
	if parser.index >= len(parser.tokens) {
		return nil
	}
	token := parser.tokens[parser.index]
	parser.index++
	if token.Spelling == "(" {
		inner := parser.parse(0)
		if parser.index < len(parser.tokens) && parser.tokens[parser.index].Spelling == ")" {
			close := parser.tokens[parser.index]
			parser.index++
			if inner != nil {
				return &SyntaxNode{Kind: NodeExpression, Operator: "group", Range: SourceRange{Start: token.Range.Start, End: close.Range.End}, Children: []*SyntaxNode{inner}, ConditionalState: token.ConditionalState}
			}
		}
		return inner
	}
	if isUnaryOperator(token.Spelling) {
		operand := parser.parsePrefix()
		children := []*SyntaxNode(nil)
		end := token.Range.End
		if operand != nil {
			children = []*SyntaxNode{operand}
			end = operand.Range.End
		}
		return &SyntaxNode{Kind: NodeExpression, Operator: token.Spelling, Range: SourceRange{Start: token.Range.Start, End: end}, Children: children, ConditionalState: token.ConditionalState}
	}
	return &SyntaxNode{Kind: NodeExpression, Name: token.Spelling, Range: token.Range, Tokens: []Token{token}, ConditionalState: token.ConditionalState}
}

func (parser *expressionParser) parseCall(callee *SyntaxNode) *SyntaxNode {
	open := parser.tokens[parser.index]
	parser.index++
	argument := parser.parse(0)
	children := []*SyntaxNode{callee}
	if argument != nil {
		children = append(children, argument)
	}
	end := open.Range.End
	if parser.index < len(parser.tokens) && parser.tokens[parser.index].Spelling == ")" {
		end = parser.tokens[parser.index].Range.End
		parser.index++
	}
	return &SyntaxNode{Kind: NodeExpression, Operator: "call", Range: SourceRange{Start: callee.Range.Start, End: end}, Children: children, ConditionalState: callee.ConditionalState}
}

func expressionPrecedence(spelling string) (int, bool) {
	switch spelling {
	case ",":
		return 1, true
	case "=", "+=", "-=", "*=", "/=", "%=":
		return 2, true
	case "?":
		return 3, true
	case "||":
		return 4, true
	case "&&":
		return 5, true
	case "|":
		return 6, true
	case "^":
		return 7, true
	case "&":
		return 8, true
	case "==", "!=":
		return 9, true
	case "<", ">", "<=", ">=":
		return 10, true
	case "<<", ">>":
		return 11, true
	case "+", "-":
		return 12, true
	case "*", "/", "%":
		return 13, true
	default:
		return 0, false
	}
}

func isUnaryOperator(spelling string) bool {
	switch spelling {
	case "+", "-", "!", "~", "*", "&", "++", "--":
		return true
	default:
		return false
	}
}
