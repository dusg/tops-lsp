package parser

import (
	"strconv"
)

type conditionValue struct {
	state       ConditionalState
	number      int64
	numberKnown bool
}

type conditionalFrame struct {
	regionIndex int
	parentState ConditionalState
	branchIndex int
	anyTrue     bool
	anyUnknown  bool
	elseSeen    bool
}

func analyzeConditionals(tokens []Token, context ParseContext) ([]ConditionalRegion, []ParserDiagnostic) {
	macros := cloneMacros(context.PredefinedMacros)
	states := make([]ConditionalState, len(tokens))
	for index := range states {
		states[index] = ConditionalActive
	}
	regions := make([]ConditionalRegion, 0)
	diagnostics := make([]ParserDiagnostic, 0)
	stack := make([]conditionalFrame, 0)
	currentState := ConditionalActive

	for index := 0; index < len(tokens); {
		if tokens[index].Spelling == "##" && tokens[index].LineStart {
			lineEnd := index
			for lineEnd < len(tokens) && tokens[lineEnd].Kind != TokenNewline {
				lineEnd++
			}
			lineRange := SourceRange{Start: tokens[index].Range.Start, End: lineEndRange(tokens, index, lineEnd)}
			diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "unexpected ## at start of directive", lineRange, false))
			index = nextLine(lineEnd, len(tokens))
			continue
		}
		if tokens[index].Spelling != "#" || !tokens[index].LineStart {
			states[index] = currentState
			index++
			continue
		}
		lineEnd := index
		for lineEnd < len(tokens) && tokens[lineEnd].Kind != TokenNewline {
			states[lineEnd] = currentState
			lineEnd++
		}
		directiveIndex := nextSignificant(tokens, index+1, lineEnd)
		lineRange := SourceRange{Start: tokens[index].Range.Start, End: lineEndRange(tokens, index, lineEnd)}
		if directiveIndex >= lineEnd {
			diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "missing preprocessor directive name", lineRange, false))
			index = nextLine(lineEnd, len(tokens))
			continue
		}
		directive := tokens[directiveIndex].Spelling
		argumentStart := directiveIndex + 1
		argumentEnd := lineEnd
		for argumentStart < argumentEnd && (tokens[argumentStart].Kind == TokenComment || tokens[argumentStart].Kind == TokenNewline) {
			argumentStart++
		}
		switch directive {
		case "if", "ifdef", "ifndef":
			condition := ConditionalUnknown
			conditionValid := true
			if directive == "if" {
				condition, conditionValid = evaluateCondition(tokens[argumentStart:argumentEnd], macros)
			} else if argumentStart < argumentEnd {
				condition = conditionForMacro(tokens[argumentStart].Spelling, macros)
				if directive == "ifndef" {
					condition = invertState(condition)
				}
				if nextSignificant(tokens, argumentStart+1, argumentEnd) < argumentEnd {
					conditionValid = false
				}
			} else {
				conditionValid = false
			}
			branchState := combineConditionalState(currentState, condition)
			parentID := -1
			if len(stack) > 0 {
				parentID = stack[len(stack)-1].regionIndex
			}
			region := ConditionalRegion{ConditionRange: conditionRange(tokens, argumentStart, argumentEnd), State: branchState, ParentID: parentID}
			region.Branches = append(region.Branches, ConditionalBranch{DirectiveRange: lineRange, BodyRange: SourceRange{Start: lineEndStart(tokens, lineEnd), End: lineEndStart(tokens, lineEnd)}, Condition: cloneTokens(tokens[argumentStart:argumentEnd]), State: branchState})
			regions = append(regions, region)
			frame := conditionalFrame{regionIndex: len(regions) - 1, parentState: currentState, branchIndex: 0, anyTrue: branchState == ConditionalActive, anyUnknown: branchState == ConditionalUnknown}
			stack = append(stack, frame)
			currentState = branchState
			if !conditionValid {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "invalid conditional expression", conditionRange(tokens, argumentStart, argumentEnd), false))
			} else if condition == ConditionalUnknown {
				diagnostic := diagnosticFor(context, DiagnosticUnknownCondition, "conditional expression cannot be resolved with the current macro context", conditionRange(tokens, argumentStart, argumentEnd), false)
				diagnostic.Severity = SeverityInformation
				diagnostic.ConditionalState = ConditionalUnknown
				diagnostics = append(diagnostics, diagnostic)
			}
		case "elif":
			if len(stack) == 0 {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "unexpected #elif", lineRange, false))
				break
			}
			frame := &stack[len(stack)-1]
			if frame.elseSeen {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "#elif after #else", lineRange, false))
				break
			}
			region := &regions[frame.regionIndex]
			region.Branches[frame.branchIndex].BodyRange.End = tokens[index].Range.Start
			condition, conditionValid := evaluateCondition(tokens[argumentStart:argumentEnd], macros)
			branchState := nextBranchState(frame, condition)
			frame.branchIndex++
			region.Branches = append(region.Branches, ConditionalBranch{DirectiveRange: lineRange, BodyRange: SourceRange{Start: lineEndStart(tokens, lineEnd), End: lineEndStart(tokens, lineEnd)}, Condition: cloneTokens(tokens[argumentStart:argumentEnd]), State: branchState})
			region.State = mergeRegionState(region.State, branchState)
			currentState = branchState
			if !conditionValid {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "invalid conditional expression", conditionRange(tokens, argumentStart, argumentEnd), false))
			} else if condition == ConditionalUnknown {
				diagnostic := diagnosticFor(context, DiagnosticUnknownCondition, "conditional expression cannot be resolved with the current macro context", conditionRange(tokens, argumentStart, argumentEnd), false)
				diagnostic.Severity = SeverityInformation
				diagnostic.ConditionalState = ConditionalUnknown
				diagnostics = append(diagnostics, diagnostic)
			}
		case "else":
			if len(stack) == 0 {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "unexpected #else", lineRange, false))
				break
			}
			frame := &stack[len(stack)-1]
			if frame.elseSeen {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "duplicate #else", lineRange, false))
				break
			}
			frame.elseSeen = true
			region := &regions[frame.regionIndex]
			if nextSignificant(tokens, argumentStart, argumentEnd) < argumentEnd {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "#else cannot have arguments", lineRange, false))
			}
			region.Branches[frame.branchIndex].BodyRange.End = tokens[index].Range.Start
			branchState := nextBranchState(frame, ConditionalActive)
			frame.branchIndex++
			region.Branches = append(region.Branches, ConditionalBranch{DirectiveRange: lineRange, BodyRange: SourceRange{Start: lineEndStart(tokens, lineEnd), End: lineEndStart(tokens, lineEnd)}, State: branchState})
			region.State = mergeRegionState(region.State, branchState)
			currentState = branchState
		case "endif":
			if len(stack) == 0 {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "unexpected #endif", lineRange, false))
				break
			}
			if nextSignificant(tokens, argumentStart, argumentEnd) < argumentEnd {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "#endif cannot have arguments", lineRange, false))
			}
			frame := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			region := &regions[frame.regionIndex]
			region.Branches[frame.branchIndex].BodyRange.End = tokens[index].Range.Start
			region.ConditionRange.End = lineRange.End
			currentState = frame.parentState
		case "define":
			if currentState == ConditionalActive && argumentStart < argumentEnd {
				name := tokens[argumentStart].Spelling
				value := MacroValue{Kind: MacroDefined}
				if argumentStart+1 < argumentEnd && tokens[argumentStart+1].Spelling == "(" {
					value = MacroValue{Kind: MacroFunction}
				} else if argumentStart+1 < argumentEnd {
					if integer, err := strconv.ParseInt(tokens[argumentStart+1].Spelling, 0, 64); err == nil {
						value = MacroValue{Kind: MacroInteger, Value: integer}
					}
				}
				macros[name] = value
			} else if currentState == ConditionalActive {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "#define requires a macro name", lineRange, false))
			}
		case "undef":
			if currentState == ConditionalActive && argumentStart < argumentEnd {
				macros[tokens[argumentStart].Spelling] = MacroValue{Kind: MacroUndefined}
			} else if currentState == ConditionalActive {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "#undef requires a macro name", lineRange, false))
			}
		default:
			if directive == "#" || directive == "##" {
				diagnostics = append(diagnostics, diagnosticFor(context, DiagnosticInvalidDirective, "unexpected preprocessor token", lineRange, false))
			}
		}
		index = nextLine(lineEnd, len(tokens))
	}

	for len(stack) > 0 {
		frame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		region := &regions[frame.regionIndex]
		region.Branches[frame.branchIndex].BodyRange.End = lenSource(tokens)
		rangeAtEOF := SourceRange{Start: lenSource(tokens), End: lenSource(tokens)}
		diagnostic := diagnosticFor(context, DiagnosticInvalidDirective, "missing #endif", rangeAtEOF, true)
		diagnostic.Related = []RelatedLocation{{Range: region.ConditionRange, Message: "opening conditional directive"}}
		diagnostics = append(diagnostics, diagnostic)
	}
	for index := range tokens {
		tokens[index].ConditionalState = states[index]
	}
	attachDirectiveDiagnostics(regions, diagnostics)
	return regions, diagnostics
}

func attachDirectiveDiagnostics(regions []ConditionalRegion, diagnostics []ParserDiagnostic) {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != DiagnosticInvalidDirective && diagnostic.Code != DiagnosticUnknownCondition {
			continue
		}
		for regionIndex := range regions {
			region := &regions[regionIndex]
			attached := false
			for _, branch := range region.Branches {
				if diagnostic.Range.Start >= branch.DirectiveRange.Start && diagnostic.Range.Start <= branch.DirectiveRange.End {
					region.DirectiveDiagnostics = append(region.DirectiveDiagnostics, diagnostic)
					attached = true
					break
				}
			}
			if attached {
				continue
			}
			if diagnostic.Message == "missing #endif" && len(diagnostic.Related) > 0 && diagnostic.Related[0].Range.Start == region.ConditionRange.Start {
				region.DirectiveDiagnostics = append(region.DirectiveDiagnostics, diagnostic)
			}
		}
	}
}

func evaluateCondition(tokens []Token, macros map[string]MacroValue) (ConditionalState, bool) {
	value := &conditionParser{tokens: tokens, macros: macros, valid: true}
	result := value.parseOr()
	if value.index != len(tokens) {
		value.valid = false
	}
	if result.state == "" {
		return ConditionalUnknown, false
	}
	return result.state, value.valid
}

type conditionParser struct {
	tokens []Token
	index  int
	macros map[string]MacroValue
	valid  bool
}

func (parser *conditionParser) parseOr() conditionValue {
	left := parser.parseAnd()
	for parser.match("||") {
		right := parser.parseAnd()
		left = combineValues(left, right, "||")
	}
	return left
}

func (parser *conditionParser) parseAnd() conditionValue {
	left := parser.parseComparison()
	for parser.match("&&") {
		right := parser.parseComparison()
		left = combineValues(left, right, "&&")
	}
	return left
}

func (parser *conditionParser) parseComparison() conditionValue {
	left := parser.parseUnary()
	if parser.index >= len(parser.tokens) || (parser.tokens[parser.index].Spelling != "==" && parser.tokens[parser.index].Spelling != "!=") {
		return left
	}
	operator := parser.tokens[parser.index].Spelling
	parser.index++
	right := parser.parseUnary()
	if !left.numberKnown || !right.numberKnown {
		return conditionValue{state: ConditionalUnknown}
	}
	match := left.number == right.number
	if operator == "!=" {
		match = !match
	}
	return booleanValue(match)
}

func (parser *conditionParser) parseUnary() conditionValue {
	if parser.match("!") {
		return invertValue(parser.parseUnary())
	}
	if parser.match("(") {
		value := parser.parseOr()
		if !parser.match(")") {
			parser.valid = false
		}
		return value
	}
	if parser.index >= len(parser.tokens) {
		parser.valid = false
		return conditionValue{state: ConditionalUnknown}
	}
	spelling := parser.tokens[parser.index].Spelling
	parser.index++
	if spelling == "defined" {
		if parser.match("(") {
			if parser.index >= len(parser.tokens) {
				parser.valid = false
				return conditionValue{state: ConditionalUnknown}
			}
			name := parser.tokens[parser.index].Spelling
			parser.index++
			if !parser.match(")") {
				parser.valid = false
			}
			return macroDefinedValue(parser.macros, name)
		}
		if parser.index < len(parser.tokens) {
			name := parser.tokens[parser.index].Spelling
			parser.index++
			return macroDefinedValue(parser.macros, name)
		}
		parser.valid = false
		return conditionValue{state: ConditionalUnknown}
	}
	if integer, err := strconv.ParseInt(spelling, 0, 64); err == nil {
		return conditionValue{state: boolState(integer != 0), number: integer, numberKnown: true}
	}
	if parser.tokens[parser.index-1].Kind != TokenIdentifier && parser.tokens[parser.index-1].Kind != TokenKeyword {
		parser.valid = false
	}
	return macroValue(parser.macros, spelling)
}

func (parser *conditionParser) match(spelling string) bool {
	if parser.index >= len(parser.tokens) || parser.tokens[parser.index].Spelling != spelling {
		return false
	}
	parser.index++
	return true
}

func macroDefinedValue(macros map[string]MacroValue, name string) conditionValue {
	value := macros[name]
	switch value.Kind {
	case MacroUndefined:
		return booleanValue(false)
	case MacroDefined, MacroFunction, MacroInteger:
		return booleanValue(true)
	default:
		return conditionValue{state: ConditionalUnknown}
	}
}

func macroValue(macros map[string]MacroValue, name string) conditionValue {
	value, ok := macros[name]
	if !ok {
		return conditionValue{state: ConditionalUnknown}
	}
	switch value.Kind {
	case MacroUndefined:
		return conditionValue{state: ConditionalInactive, number: 0, numberKnown: true}
	case MacroDefined:
		return conditionValue{state: ConditionalActive, number: 1, numberKnown: true}
	case MacroFunction:
		return conditionValue{state: ConditionalUnknown}
	case MacroInteger:
		return conditionValue{state: boolState(value.Value != 0), number: value.Value, numberKnown: true}
	default:
		return conditionValue{state: ConditionalUnknown}
	}
}

func conditionForMacro(name string, macros map[string]MacroValue) ConditionalState {
	return macroDefinedValue(macros, name).state
}

func booleanValue(value bool) conditionValue {
	if value {
		return conditionValue{state: ConditionalActive, number: 1, numberKnown: true}
	}
	return conditionValue{state: ConditionalInactive, number: 0, numberKnown: true}
}

func boolState(value bool) ConditionalState {
	if value {
		return ConditionalActive
	}
	return ConditionalInactive
}

func combineValues(left, right conditionValue, operator string) conditionValue {
	if left.state == ConditionalUnknown || right.state == ConditionalUnknown {
		return conditionValue{state: ConditionalUnknown}
	}
	if operator == "&&" {
		return booleanValue(left.state == ConditionalActive && right.state == ConditionalActive)
	}
	return booleanValue(left.state == ConditionalActive || right.state == ConditionalActive)
}

func invertValue(value conditionValue) conditionValue {
	if value.state == ConditionalUnknown {
		return value
	}
	return booleanValue(value.state == ConditionalInactive)
}

func combineConditionalState(parent, child ConditionalState) ConditionalState {
	if parent == ConditionalInactive || child == ConditionalInactive {
		return ConditionalInactive
	}
	if parent == ConditionalUnknown || child == ConditionalUnknown {
		return ConditionalUnknown
	}
	return ConditionalActive
}

func nextBranchState(frame *conditionalFrame, condition ConditionalState) ConditionalState {
	if frame.parentState == ConditionalInactive || frame.anyTrue {
		return ConditionalInactive
	}
	if frame.anyUnknown {
		return ConditionalUnknown
	}
	if condition == ConditionalActive {
		frame.anyTrue = true
		return ConditionalActive
	}
	if condition == ConditionalUnknown {
		frame.anyUnknown = true
		return ConditionalUnknown
	}
	return ConditionalInactive
}

func mergeRegionState(left, right ConditionalState) ConditionalState {
	if left == ConditionalUnknown || right == ConditionalUnknown {
		return ConditionalUnknown
	}
	if left == ConditionalActive || right == ConditionalActive {
		return ConditionalActive
	}
	return ConditionalInactive
}

func invertState(state ConditionalState) ConditionalState {
	if state == ConditionalActive {
		return ConditionalInactive
	}
	if state == ConditionalInactive {
		return ConditionalActive
	}
	return ConditionalUnknown
}

func cloneMacros(source map[string]MacroValue) map[string]MacroValue {
	result := make(map[string]MacroValue, len(source))
	for name, value := range source {
		result[name] = value
	}
	return result
}

func cloneTokens(source []Token) []Token {
	return append([]Token(nil), source...)
}

func nextSignificant(tokens []Token, start, end int) int {
	for start < end && (tokens[start].Kind == TokenComment || tokens[start].Kind == TokenNewline) {
		start++
	}
	return start
}

func nextLine(lineEnd, length int) int {
	if lineEnd < length {
		return lineEnd + 1
	}
	return lineEnd
}

func lineEndRange(tokens []Token, start, end int) int {
	if end > start {
		return tokens[end-1].Range.End
	}
	return tokens[start].Range.End
}

func lineEndStart(tokens []Token, index int) int {
	if index < len(tokens) {
		return tokens[index].Range.End
	}
	return lenSource(tokens)
}

func conditionRange(tokens []Token, start, end int) SourceRange {
	if start >= end {
		return SourceRange{}
	}
	return SourceRange{Start: tokens[start].Range.Start, End: tokens[end-1].Range.End}
}

func lenSource(tokens []Token) int {
	if len(tokens) == 0 {
		return 0
	}
	return tokens[len(tokens)-1].Range.End
}
