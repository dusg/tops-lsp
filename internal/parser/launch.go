package parser

import "strings"

func (parser *syntaxParser) launchNode(tokens []Token, start int) *SyntaxNode {
	close := -1
	for index := start + 1; index < len(tokens); index++ {
		if tokens[index].Spelling == ">>>" {
			close = index
			break
		}
	}
	callee := ""
	launchStart := tokens[start].Range.Start
	if start > 0 {
		calleeStart := launchCalleeStart(tokens, start)
		var builder strings.Builder
		for index := calleeStart; index < start; index++ {
			builder.WriteString(tokens[index].Spelling)
		}
		callee = builder.String()
		launchStart = tokens[calleeStart].Range.Start
	}
	configEnd := len(tokens)
	argumentTokens := []Token(nil)
	missingLaunchClose := close < 0
	incomplete := missingLaunchClose
	children := []*SyntaxNode(nil)
	if close >= 0 {
		configEnd = close
		if close+1 >= len(tokens) || tokens[close+1].Spelling != "(" {
			missingRange := SourceRange{Start: tokens[close].Range.End, End: tokens[close].Range.End}
			parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected '(' after kernel launch configuration", missingRange, true, tokens[start].Range, "opening '<<<'")
			children = append(children, missingTokenNode("(", missingRange, tokens[start].ConditionalState))
			incomplete = true
		} else {
			argumentStart := close + 2
			argumentEnd := len(tokens)
			depth := 1
			argumentClosed := false
			for index := close + 2; index < len(tokens); index++ {
				switch tokens[index].Spelling {
				case "(":
					depth++
				case ")":
					depth--
					if depth == 0 {
						argumentEnd = index
						argumentClosed = true
					}
				}
				if argumentClosed {
					break
				}
			}
			argumentTokens = append([]Token(nil), tokens[argumentStart:argumentEnd]...)
			if !argumentClosed {
				insertion := tokens[len(tokens)-1].Range.End
				if tokens[len(tokens)-1].Spelling == ";" {
					insertion = tokens[len(tokens)-1].Range.Start
				}
				missingRange := SourceRange{Start: insertion, End: insertion}
				parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected ')' after kernel launch arguments", missingRange, true, tokens[close+1].Range, "opening '('")
				children = append(children, missingTokenNode(")", missingRange, tokens[start].ConditionalState))
				incomplete = true
			}
		}
	}
	launchEnd := tokens[len(tokens)-1].Range.End
	if close >= 0 {
		launchEnd = tokens[close].Range.End
	}
	if missingLaunchClose {
		missingRange := SourceRange{Start: launchEnd, End: launchEnd}
		parser.addDiagnosticWithRelated(DiagnosticMissingToken, "expected '>>>'", missingRange, true, tokens[start].Range, "opening '<<<'")
	}
	if close >= 0 && close == start+1 {
		parser.addDiagnostic(DiagnosticInvalidTopsAttribute, "kernel launch requires a configuration", tokens[start].Range, false)
	}
	if missingLaunchClose {
		children = append(children, missingTokenNode(">>>", SourceRange{Start: launchEnd, End: launchEnd}, tokens[start].ConditionalState))
	}
	return &SyntaxNode{Kind: NodeTopsLaunch, Name: callee, Callee: callee, Range: SourceRange{Start: launchStart, End: launchEnd}, Children: children, ConfigTokens: append([]Token(nil), tokens[start+1:configEnd]...), ArgumentTokens: argumentTokens, ConditionalState: tokens[start].ConditionalState, Incomplete: incomplete}
}

func launchCalleeStart(tokens []Token, launchStart int) int {
	calleeStart := launchStart - 1
	if tokens[calleeStart].Spelling == ">" {
		depth := 0
		for index := launchStart - 1; index >= 0; index-- {
			switch tokens[index].Spelling {
			case ">":
				depth++
			case "<":
				depth--
				if depth == 0 {
					calleeStart = index - 1
				}
			}
			if depth == 0 {
				break
			}
		}
	}
	for calleeStart >= 2 {
		separator := tokens[calleeStart-1].Spelling
		if separator != "::" && separator != "." && separator != "->" {
			break
		}
		calleeStart -= 2
	}
	if calleeStart < 0 {
		return 0
	}
	return calleeStart
}

func (parser *syntaxParser) hasIncompleteLaunch(start, end int) bool {
	for index := start; index < end && index < len(parser.tokens); index++ {
		if parser.tokens[index].Spelling != "<<<" {
			continue
		}
		for closeIndex := index + 1; closeIndex < end && closeIndex < len(parser.tokens); closeIndex++ {
			if parser.tokens[closeIndex].Spelling == ">>>" {
				break
			}
			if closeIndex+1 == end {
				return true
			}
		}
	}
	return false
}

func (parser *syntaxParser) hasIncompleteLaunchArguments(start, end int) bool {
	for index := start; index < end && index < len(parser.tokens); index++ {
		if parser.tokens[index].Spelling != "<<<" {
			continue
		}
		close := -1
		for closeIndex := index + 1; closeIndex < end && closeIndex < len(parser.tokens); closeIndex++ {
			if parser.tokens[closeIndex].Spelling == ">>>" {
				close = closeIndex
				break
			}
		}
		if close < 0 || close+1 >= end || parser.tokens[close+1].Spelling != "(" {
			continue
		}
		depth := 1
		for argumentIndex := close + 2; argumentIndex < end && argumentIndex < len(parser.tokens); argumentIndex++ {
			switch parser.tokens[argumentIndex].Spelling {
			case "(":
				depth++
			case ")":
				depth--
				if depth == 0 {
					return false
				}
			}
		}
		return true
	}
	return false
}
