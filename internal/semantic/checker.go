package semantic

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func BuildSemanticSnapshot(snapshot index.IndexSnapshot, context SemanticContext) (SemanticSnapshot, error) {
	if err := context.Validate(); err != nil {
		return SemanticSnapshot{}, err
	}
	if index.NormalizeSourceURI(snapshot.RootURI) != index.NormalizeSourceURI(context.DocumentURI) {
		return SemanticSnapshot{}, fmt.Errorf("semantic snapshot root source does not match context document")
	}
	result := SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: context.SourceVersionFingerprint,
		DocumentVersion:          context.DocumentVersion,
		SemanticContextVersion:   context.SemanticContextVersion,
		SourceVersions:           append([]SourceVersionEntry(nil), context.SourceVersions...),
		HeaderVersions:           append([]SourceVersionEntry(nil), context.HeaderVersions...),
		Generation:               context.AnalysisGeneration,
		Epoch:                    context.Epoch,
		Status:                   SnapshotStatusComplete,
		Types:                    make(map[string]SemanticType),
	}
	if context.Status != ContextStatusResolved {
		result.Status = SnapshotStatusDegraded
	}
	if err := result.ValidateSourceVersion(); err != nil {
		return SemanticSnapshot{}, err
	}
	if unsupportedReason := context.Provenance["out_of_scope_kind"]; unsupportedReason != "" {
		result.Status = SnapshotStatusDegraded
		result.UnsupportedReason = unsupportedReason
	}
	if (context.ParseStatus != "" && context.ParseStatus != parser.ParseStatusComplete) || (context.IndexStatus != "" && context.IndexStatus != string(index.IndexStatusComplete)) {
		result.Status = SnapshotStatusDegraded
	}
	result.Scopes = buildScopeFacts(snapshot)
	result.Declarations, result.Types = buildDeclarationFacts(snapshot)
	result.Callables = buildCallableFacts(snapshot, context)
	result.References = buildReferenceFacts(snapshot)
	result.Calls = buildCallFacts(snapshot, context)
	if result.UnsupportedReason == "" {
		result.Diagnostics = buildSemanticDiagnostics(snapshot, result.References, result.Calls, context)
		result.Diagnostics = append(result.Diagnostics, buildDeclarationTargetDiagnostics(snapshot, context)...)
		result.Diagnostics = append(result.Diagnostics, buildDeclarationTypeDiagnostics(snapshot, context)...)
		result.Diagnostics = append(result.Diagnostics, buildPassDiagnostics(snapshot, result.Calls, context)...)
		if incomplete, ok := incompleteInputDiagnostic(snapshot, context); ok {
			result.Diagnostics = append(result.Diagnostics, incomplete)
		}
		if context.Status != ContextStatusResolved && !hasContextDiagnostic(result.Diagnostics) {
			result.Diagnostics = append(result.Diagnostics, contextStatusDiagnostic(snapshot, context))
		}
	}
	result.Diagnostics = attachDiagnosticRelatedInformation(snapshot, NormalizeSemanticDiagnostics(result.Diagnostics))
	result.Diagnostics = DeduplicateSemanticDiagnostics(result.Diagnostics)
	if result.InvalidReason == "" {
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Severity == SeverityError {
				result.Status = SnapshotStatusDegraded
				result.InvalidReason = diagnostic.Code
				break
			}
		}
	}
	if result.SourceVersionFingerprint == "" {
		return SemanticSnapshot{}, fmt.Errorf("semantic snapshot source version fingerprint is empty")
	}
	return result, nil
}

func incompleteInputDiagnostic(snapshot index.IndexSnapshot, context SemanticContext) (SemanticDiagnostic, bool) {
	if context.IncompleteRange.Start == context.IncompleteRange.End {
		if context.IndexStatus == "" && (context.ParseStatus == parser.ParseStatusRecovered || context.ParseStatus == parser.ParseStatusInvalid) {
			if source := snapshot.Sources[snapshot.RootURI]; len(source) > 0 {
				context.IncompleteRange = parser.SourceRange{Start: 0, End: 1}
			}
		}
		if context.IncompleteRange.Start == context.IncompleteRange.End {
			return SemanticDiagnostic{}, false
		}
	}
	rangeValue := context.IncompleteRange
	if rangeValue.Start == rangeValue.End {
		return SemanticDiagnostic{}, false
	}
	return SemanticDiagnostic{
		Code:            "tops-semantic-incomplete-input",
		Severity:        SeverityHint,
		SemanticKind:    "incomplete-input",
		Message:         "input is incomplete; semantic results may be partial",
		Range:           rangeValue,
		PassKind:        "not-applicable",
		ProfileID:       context.Profile.ProfileID,
		ContextStatus:   context.Status,
		DocumentVersion: context.DocumentVersion,
		ContextVersion:  context.SemanticContextVersion,
		SourceVersion:   context.SourceVersionFingerprint,
		Incomplete:      true,
		Recoverable:     true,
	}, true
}

func hasContextDiagnostic(diagnostics []SemanticDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "tops-semantic-context-missing" || diagnostic.Code == "tops-semantic-context-degraded" {
			return true
		}
	}
	return false
}

func contextStatusDiagnostic(snapshot index.IndexSnapshot, context SemanticContext) SemanticDiagnostic {
	rangeValue := parser.SourceRange{Start: 0, End: 1}
	if source := snapshot.Sources[snapshot.RootURI]; len(source) == 0 {
		rangeValue = parser.SourceRange{}
	}
	code, severity := contextStatusCode(context.Status)
	return SemanticDiagnostic{Code: code, Severity: severity, SemanticKind: strings.TrimPrefix(code, "tops-semantic-"), Message: semanticDiagnosticMessage(code), Range: rangeValue, PassKind: "not-applicable", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: context.DocumentVersion, ContextVersion: context.SemanticContextVersion, SourceVersion: context.SourceVersionFingerprint}
}

func buildPassDiagnostics(snapshot index.IndexSnapshot, calls []CallFact, context SemanticContext) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0)
	for _, call := range calls {
		if len(call.Candidates) != 1 {
			continue
		}
		caller, ok := callerCallableFact(snapshot, call, context)
		if !ok {
			continue
		}
		symbol, ok := snapshot.Symbols[call.Candidates[0]]
		if !ok || symbol.Kind != index.SymbolKindFunction {
			continue
		}
		callee := BuildCallableFact(symbol, context)
		passFacts := call.PassFacts
		if len(passFacts) == 0 {
			passFacts = []CallPassFact{{PassKind: context.Command.PassKind, Availability: PassAvailabilityAvailable}}
			if context.Command.PassKind == PassKindHostDevice {
				passFacts = []CallPassFact{{PassKind: PassKindHost, Availability: PassAvailabilityAvailable}, {PassKind: PassKindDevice, Availability: PassAvailabilityAvailable}}
			}
		}
		for _, passFact := range passFacts {
			if passFact.Availability != PassAvailabilityAvailable || (passFact.PassKind != PassKindHost && passFact.PassKind != PassKindDevice) {
				continue
			}
			pass := passFact.PassKind
			caller.Range = call.Range
			if callee.Range.Start == callee.Range.End {
				callee.Range = call.Range
			}
			if diagnostic, ok := CheckCallBoundary(caller, callee, pass, context); ok {
				result = append(result, diagnostic)
			}
		}
	}
	return result
}

func callerCallableFact(snapshot index.IndexSnapshot, call CallFact, context SemanticContext) (CallableFact, bool) {
	bestSize := int(^uint(0) >> 1)
	var best CallableFact
	for _, symbol := range snapshot.Symbols {
		if symbol.Kind != index.SymbolKindFunction || symbol.SourceVersion.URI != call.URI {
			continue
		}
		containsCall := false
		for _, location := range append(append([]index.SymbolLocation(nil), symbol.Declarations...), symbol.Definitions...) {
			if location.Range.Start <= call.Range.Start && call.Range.End <= location.Range.End {
				containsCall = true
				break
			}
		}
		if !containsCall {
			continue
		}
		size := symbol.NameRange.End - symbol.NameRange.Start
		for _, location := range append(append([]index.SymbolLocation(nil), symbol.Declarations...), symbol.Definitions...) {
			if location.Range.Start <= call.Range.Start && call.Range.End <= location.Range.End {
				size = location.Range.End - location.Range.Start
				break
			}
		}
		if size >= bestSize {
			continue
		}
		best = BuildCallableFact(symbol, context)
		bestSize = size
	}
	if best.SymbolID == "" {
		return CallableFact{}, false
	}
	return best, true
}

func buildScopeFacts(snapshot index.IndexSnapshot) []ScopeFact {
	ids := make([]string, 0, len(snapshot.Scopes))
	for id := range snapshot.Scopes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]ScopeFact, 0, len(ids))
	for _, id := range ids {
		scope := snapshot.Scopes[id]
		result = append(result, ScopeFact{ScopeID: scope.ScopeID, SourceURI: scope.SourceURI, Kind: scope.Kind, Name: scope.Name, QualifiedName: scope.QualifiedName, ParentID: scope.ParentID, Range: scope.Range, Symbols: append([]string(nil), scope.Symbols...), Children: append([]string(nil), scope.Children...)})
	}
	return result
}

func buildDeclarationFacts(snapshot index.IndexSnapshot) ([]DeclarationFact, map[string]SemanticType) {
	ids := make([]string, 0, len(snapshot.Symbols))
	for id := range snapshot.Symbols {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	facts := make([]DeclarationFact, 0, len(ids))
	types := make(map[string]SemanticType)
	for _, id := range ids {
		symbol := snapshot.Symbols[id]
		typeValue := semanticTypeForSymbol(snapshot, symbol)
		types[id] = typeValue
		facts = append(facts, DeclarationFact{SymbolID: symbol.SymbolID, Name: symbol.Name, QualifiedName: symbol.QualifiedName, Kind: symbol.Kind, Type: typeValue, ScopeID: symbol.ScopeID, Range: symbol.NameRange, Status: symbol.Status, Condition: symbol.Condition, SourceVersion: symbol.SourceVersion})
	}
	return facts, types
}

func semanticTypeForSymbol(snapshot index.IndexSnapshot, symbol index.SymbolRecord) SemanticType {
	if node := findSymbolSyntaxNode(snapshot.ASTRoot, symbol.Name, symbol.NameRange); node != nil {
		return semanticTypeFromNode(node, symbol.Name)
	}
	spelling := symbol.TypeName
	if symbol.Kind == index.SymbolKindFunction {
		spelling = symbol.ReturnTypeSummary
	}
	return ParseSemanticType(spelling)
}

func findSymbolSyntaxNode(node *parser.SyntaxNode, name string, nameRange parser.SourceRange) *parser.SyntaxNode {
	if node == nil {
		return nil
	}
	if node.Name == name && node.NameRange == nameRange {
		return node
	}
	for _, child := range node.Children {
		if found := findSymbolSyntaxNode(child, name, nameRange); found != nil {
			return found
		}
	}
	return nil
}

func semanticTypeFromNode(node *parser.SyntaxNode, name string) SemanticType {
	if node == nil {
		return SemanticType{Kind: TypeKindUnresolved, Unresolved: true}
	}
	parts := tokenSpellings(node.TypeTokens)
	nameIndex := -1
	for index, token := range node.DeclaratorTokens {
		if token.Spelling == name {
			nameIndex = index
			break
		}
	}
	if nameIndex >= 0 {
		parts = append(parts, tokenSpellings(node.DeclaratorTokens[:nameIndex])...)
	}
	result := ParseSemanticType(strings.Join(parts, " "))
	lanes := 0
	if nameIndex >= 0 {
		for index := nameIndex + 1; index+2 < len(node.DeclaratorTokens); index++ {
			if node.DeclaratorTokens[index].Spelling != "[" || node.DeclaratorTokens[index+2].Spelling != "]" {
				continue
			}
			lanes, _ = strconv.Atoi(node.DeclaratorTokens[index+1].Spelling)
			break
		}
	}
	if lanes == 0 {
		for _, token := range append(append([]parser.Token(nil), node.TypeTokens...), node.DeclaratorTokens...) {
			if strings.HasPrefix(token.Spelling, "__vector") {
				lanes, _ = strconv.Atoi(strings.TrimPrefix(token.Spelling, "__vector"))
				if lanes == 0 {
					lanes = 1
				}
				break
			}
		}
	}
	if lanes > 0 {
		result.VectorElement = result.ScalarKind
		result.VectorLanes = lanes
		result.VectorWidth = scalarBitWidth(result.ScalarKind) * lanes
		result.Alignment = scalarBitWidth(result.ScalarKind) / 8
	}
	return result
}

func scalarBitWidth(kind string) int {
	return map[string]int{"char": 8, "short": 16, "int": 32, "long": 64, "float": 32, "double": 64}[kind]
}

func buildDeclarationTargetDiagnostics(snapshot index.IndexSnapshot, context SemanticContext) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0)
	ids := make([]string, 0, len(snapshot.Symbols))
	for id := range snapshot.Symbols {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		symbol := snapshot.Symbols[id]
		if symbol.Condition.State == parser.ConditionalInactive || symbol.Condition.State == parser.ConditionalUnknown {
			continue
		}
		typeValue := semanticTypeForSymbol(snapshot, symbol)
		if typeValue.VectorLanes > 0 {
			if diagnostic, ok := CheckVectorConstraintAt(context, typeValue.VectorElement, typeValue.VectorLanes, typeValue.VectorWidth, typeValue.Alignment, symbol.NameRange); ok {
				result = append(result, diagnostic)
			}
		}
		if symbol.Kind != index.SymbolKindFunction {
			continue
		}
		for _, returnValue := range functionReturnValues(snapshot, symbol) {
			if diagnostic, ok := CheckAddressSpaceAt(context, returnValue.SourceType, typeValue, returnValue.Range); ok {
				result = append(result, diagnostic)
			}
		}
	}
	return result
}

func buildDeclarationTypeDiagnostics(snapshot index.IndexSnapshot, context SemanticContext) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0)
	for _, symbol := range snapshot.Symbols {
		if symbol.Kind == index.SymbolKindFunction || symbol.Condition.State == parser.ConditionalInactive || symbol.Condition.State == parser.ConditionalUnknown {
			continue
		}
		node := findSymbolSyntaxNode(snapshot.ASTRoot, symbol.Name, symbol.NameRange)
		if node == nil {
			continue
		}
		tokens := node.Tokens
		if len(tokens) == 0 {
			tokens = append(append([]parser.Token(nil), node.TypeTokens...), node.DeclaratorTokens...)
		}
		equals := -1
		for index, token := range tokens {
			if token.Spelling == "=" {
				equals = index
				break
			}
		}
		if equals < 0 || equals+1 >= len(tokens) {
			continue
		}
		end := equals + 1
		for end < len(tokens) && tokens[end].Spelling != ";" {
			end++
		}
		if end <= equals+1 {
			continue
		}
		sourceType := inferArgumentType(snapshot, symbol.ScopeID, tokens[equals+1:end])
		targetType := semanticTypeForSymbol(snapshot, symbol)
		conversion := CheckConversion(sourceType, targetType)
		if conversion.Status != ConversionStatusSupported || conversion.Kind != ConversionKindIncompatible {
			continue
		}
		rangeValue := parser.SourceRange{Start: tokens[equals+1].Range.Start, End: tokens[end-1].Range.End}
		result = append(result, SemanticDiagnostic{Code: "tops-semantic-type-mismatch", Severity: SeverityError, SemanticKind: "type-mismatch", Message: semanticDiagnosticMessage("tops-semantic-type-mismatch"), Range: rangeValue, PassKind: "not-applicable", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: context.DocumentVersion, ContextVersion: context.SemanticContextVersion, SourceVersion: context.SourceVersionFingerprint})
	}
	return result
}

type functionReturnValue struct {
	SourceType SemanticType
	Range      parser.SourceRange
}

func functionReturnValues(snapshot index.IndexSnapshot, symbol index.SymbolRecord) []functionReturnValue {
	node := findSymbolSyntaxNode(snapshot.ASTRoot, symbol.Name, symbol.NameRange)
	if node == nil {
		return nil
	}
	result := make([]functionReturnValue, 0)
	tokens := make([]parser.Token, 0)
	for _, token := range snapshot.Tokens {
		if node.Range.Start <= token.Range.Start && token.Range.End <= node.Range.End {
			tokens = append(tokens, token)
		}
	}
	for index, token := range tokens {
		if token.Spelling != "return" || index+1 >= len(tokens) || tokens[index+1].Kind != parser.TokenIdentifier {
			continue
		}
		name := tokens[index+1].Spelling
		for _, candidate := range snapshot.Symbols {
			if candidate.Name != name || candidate.Kind != indexSymbolVariable() || candidate.SourceVersion.URI != symbol.SourceVersion.URI {
				continue
			}
			if candidate.NameRange.Start < node.Range.Start || candidate.NameRange.End > node.Range.End {
				continue
			}
			sourceType := semanticTypeForSymbol(snapshot, candidate)
			if sourceType.AddressSpace == "" {
				if parameterType, ok := parameterTypeFromFunctionNode(node, name); ok {
					sourceType = parameterType
				}
			}
			result = append(result, functionReturnValue{SourceType: sourceType, Range: tokens[index+1].Range})
			break
		}
	}
	return result
}

func indexSymbolVariable() index.SymbolKind {
	return index.SymbolKindVariable
}

func parameterTypeFromFunctionNode(node *parser.SyntaxNode, name string) (SemanticType, bool) {
	open := -1
	for index, token := range node.DeclaratorTokens {
		if token.Spelling == "(" {
			open = index
			break
		}
	}
	if open < 0 {
		return SemanticType{}, false
	}
	depth := 0
	start := open + 1
	for index := start; index < len(node.DeclaratorTokens); index++ {
		token := node.DeclaratorTokens[index]
		switch token.Spelling {
		case "(":
			depth++
		case ")":
			if depth == 0 {
				if value, ok := parameterTypeTokens(node.DeclaratorTokens[start:index], name); ok {
					return value, true
				}
				return SemanticType{}, false
			}
			depth--
		case ",":
			if depth == 0 {
				if value, ok := parameterTypeTokens(node.DeclaratorTokens[start:index], name); ok {
					return value, true
				}
				start = index + 1
			}
		}
	}
	return SemanticType{}, false
}

func parameterTypeTokens(tokens []parser.Token, name string) (SemanticType, bool) {
	for index, token := range tokens {
		if token.Spelling == name {
			return ParseSemanticType(strings.Join(tokenSpellings(tokens[:index]), " ")), true
		}
	}
	return SemanticType{}, false
}

func buildCallableFacts(snapshot index.IndexSnapshot, context SemanticContext) []CallableFact {
	ids := make([]string, 0)
	for id, symbol := range snapshot.Symbols {
		if symbol.Kind == index.SymbolKindFunction {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	result := make([]CallableFact, 0, len(ids))
	for _, id := range ids {
		result = append(result, BuildCallableFact(snapshot.Symbols[id], context))
	}
	return result
}

func buildReferenceFacts(snapshot index.IndexSnapshot) []ReferenceFact {
	ids := make([]string, 0, len(snapshot.References))
	for id := range snapshot.References {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]ReferenceFact, 0, len(ids))
	for _, id := range ids {
		reference := snapshot.References[id]
		result = append(result, ReferenceFact{ReferenceID: reference.ReferenceID, URI: reference.URI, Range: reference.Range, Name: reference.Name, QualifiedName: reference.QualifiedName, ScopeID: reference.ScopeID, Candidates: append([]string(nil), reference.Candidates...), Status: reference.Status, Kind: reference.Kind, Condition: reference.Condition, SourceVersion: reference.SourceVersion})
	}
	return result
}

func buildCallFacts(snapshot index.IndexSnapshot, context SemanticContext) []CallFact {
	result := make([]CallFact, 0)
	for _, reference := range buildReferenceFacts(snapshot) {
		if reference.Kind != "call" {
			continue
		}
		status := CandidateStatusUnresolved
		if reference.Status == index.ReferenceStatusResolved {
			status = CandidateStatusResolved
		} else if reference.Status == index.ReferenceStatusAmbiguous {
			status = CandidateStatusAmbiguous
		}
		fact := CallFact{CallID: reference.ReferenceID, URI: reference.URI, Range: reference.Range, CalleeName: reference.Name, Candidates: append([]string(nil), reference.Candidates...), CandidateStatus: status, CallerKind: "not-applicable", CalleeKind: "not-applicable", Condition: reference.Condition}
		var callee CallableFact
		if len(reference.Candidates) == 1 {
			if callable, ok := snapshot.Symbols[reference.Candidates[0]]; ok {
				callee = BuildCallableFact(callable, context)
				fact.CalleeSymbolID = callee.SymbolID
				fact.CalleeKind = callee.ExecutionKind
				arguments := callArgumentGroups(snapshot, index.ReferenceRecord{URI: reference.URI, Range: reference.Range, ScopeID: reference.ScopeID})
				fact.ArgumentsKnown = arguments != nil
				fact.ArgumentCount = len(arguments)
				fact.Conversions = callConversions(snapshot, index.ReferenceRecord{URI: reference.URI, Range: reference.Range, ScopeID: reference.ScopeID}, callee)
			}
		}
		caller, callerOK := callerCallableFact(snapshot, fact, context)
		if callerOK {
			fact.CallerSymbolID = caller.SymbolID
			fact.CallerKind = caller.ExecutionKind
		}
		for _, pass := range []PassKind{PassKindHost, PassKindDevice} {
			view, ok := context.PassViews[pass]
			if !ok {
				continue
			}
			passFact := CallPassFact{PassKind: pass, Availability: view.Availability}
			if callerOK {
				passFact.CallerSymbolID = caller.SymbolID
				passFact.CallerKind = caller.ExecutionKind
			}
			if callee.SymbolID != "" {
				passFact.CalleeSymbolID = callee.SymbolID
				passFact.CalleeKind = callee.ExecutionKind
			}
			fact.PassFacts = append(fact.PassFacts, passFact)
		}
		for _, passFact := range fact.PassFacts {
			if passFact.Availability == PassAvailabilityAvailable {
				fact.PassKind = passFact.PassKind
				break
			}
		}
		result = append(result, fact)
	}
	return result
}

func buildSemanticDiagnostics(snapshot index.IndexSnapshot, references []ReferenceFact, calls []CallFact, context SemanticContext) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0)
	contextDiagnosticAdded := false
	for _, reference := range references {
		if isTopsQualifier(reference.Name) {
			continue
		}
		if reference.Condition.State == parser.ConditionalInactive || reference.Status == index.ReferenceStatusInactive {
			continue
		}
		if reference.Condition.State == parser.ConditionalUnknown || reference.Status == index.ReferenceStatusUnknown {
			if !contextDiagnosticAdded {
				result = append(result, contextStatusDiagnostic(snapshot, context))
				contextDiagnosticAdded = true
			}
			continue
		}
		code := ""
		severity := SemanticSeverity("")
		switch {
		case reference.Kind == "type-mismatch":
			code, severity = "tops-semantic-type-mismatch", SeverityError
		case reference.Kind == "call" && reference.Status == index.ReferenceStatusResolved && len(reference.Candidates) == 1 && snapshot.Symbols[reference.Candidates[0]].Kind != index.SymbolKindFunction:
			code, severity = "tops-semantic-invalid-call", SeverityError
		case reference.Status == index.ReferenceStatusUnresolved:
			if context.Status == ContextStatusResolved {
				code, severity = "tops-semantic-unknown-name", SeverityError
			} else if !contextDiagnosticAdded {
				result = append(result, contextStatusDiagnostic(snapshot, context))
				contextDiagnosticAdded = true
				continue
			}
		case reference.Status == index.ReferenceStatusAmbiguous:
			code, severity = "tops-semantic-ambiguous-name", SeverityWarning
		}
		if code == "" {
			continue
		}
		result = append(result, semanticDiagnosticForReference(code, severity, reference, context))
	}
	for _, call := range calls {
		if context.ParseStatus != "" && context.ParseStatus != parser.ParseStatusComplete {
			continue
		}
		if call.CandidateStatus != CandidateStatusResolved || len(call.Candidates) != 1 {
			continue
		}
		callable, ok := snapshot.Symbols[call.Candidates[0]]
		if !ok || callable.Kind != index.SymbolKindFunction {
			continue
		}
		parameters := BuildCallableFact(callable, context).Parameters
		if !call.ArgumentsKnown {
			continue
		}
		if call.ArgumentCount != len(parameters) {
			result = append(result, semanticDiagnosticForCall("tops-semantic-invalid-call", SeverityError, call, context))
			continue
		}
		for _, conversion := range call.Conversions {
			if conversion.Status == ConversionStatusUnsupported || conversion.Status == ConversionStatusUnresolved {
				continue
			}
			if conversion.Kind == ConversionKindIncompatible {
				result = append(result, semanticDiagnosticForCall("tops-semantic-type-mismatch", SeverityError, call, context))
				break
			}
			if diagnostic, ok := CheckAddressSpaceAt(context, conversion.SourceType, conversion.TargetType, call.Range); ok {
				result = append(result, diagnostic)
			}
			if element, lanes, width, alignment, ok := vectorTypeValues(conversion.TargetType); ok {
				if diagnostic, vectorOK := CheckVectorConstraintAt(context, element, lanes, width, alignment, call.Range); vectorOK {
					result = append(result, diagnostic)
				}
			}
		}
	}
	return result
}

func callConversions(snapshot index.IndexSnapshot, reference index.ReferenceRecord, callable CallableFact) []CallConversion {
	arguments := callArgumentGroups(snapshot, reference)
	conversions := make([]CallConversion, 0, len(arguments))
	for index, argument := range arguments {
		if index >= len(callable.Parameters) {
			break
		}
		conversions = append(conversions, CheckConversion(inferArgumentType(snapshot, reference.ScopeID, argument), callable.Parameters[index].Type))
	}
	return conversions
}

func callArgumentGroups(snapshot index.IndexSnapshot, reference index.ReferenceRecord) [][]parser.Token {
	tokens := snapshot.Tokens
	if len(tokens) == 0 {
		if source := snapshot.Sources[reference.URI]; source != "" {
			tokens = parser.Tokenize(source, parser.ParseContext{DocumentURI: reference.URI}).Tokens
		}
	}
	open := -1
	for index, token := range tokens {
		if token.Range.Start >= reference.Range.End && token.Spelling == "(" {
			open = index
			break
		}
	}
	if open < 0 {
		return nil
	}
	groups := make([][]parser.Token, 0)
	current := make([]parser.Token, 0)
	depth := 0
	for _, token := range tokens[open+1:] {
		switch token.Spelling {
		case "(", "[", "{":
			depth++
		case ")":
			if depth == 0 {
				if len(current) > 0 {
					groups = append(groups, current)
				}
				return groups
			}
			depth--
		case "]", "}":
			if depth > 0 {
				depth--
			}
		case ",":
			if depth == 0 {
				groups = append(groups, current)
				current = nil
				continue
			}
		}
		current = append(current, token)
	}
	return groups
}

func inferArgumentType(snapshot index.IndexSnapshot, scopeID string, tokens []parser.Token) SemanticType {
	if len(tokens) == 0 {
		return SemanticType{Kind: TypeKindUnresolved, Unresolved: true}
	}
	if len(tokens) == 2 && tokens[0].Spelling == "&" {
		value := inferArgumentType(snapshot, scopeID, tokens[1:])
		value.Kind = TypeKindPointer
		value.PointerDepth++
		value.Pointee = &SemanticType{Spelling: value.Spelling, Kind: value.Kind, ScalarKind: value.ScalarKind}
		return value
	}
	spelling := tokens[0].Spelling
	switch {
	case spelling == "true" || spelling == "false":
		return ParseSemanticType("bool")
	case strings.HasPrefix(spelling, "\"") || strings.HasPrefix(spelling, "u8\""):
		return ParseSemanticType("const char *")
	case strings.HasPrefix(spelling, "'"):
		return ParseSemanticType("char")
	case strings.Contains(spelling, ".") && looksLikeNumericLiteral(spelling):
		return ParseSemanticType("double")
	case strings.ContainsAny(spelling, "eE") && looksLikeNumericLiteral(spelling):
		return ParseSemanticType("double")
	case isIntegerLiteral(spelling):
		return ParseSemanticType("int")
	}
	for _, symbol := range snapshot.Symbols {
		if symbol.Name == spelling && symbol.ScopeID == scopeID {
			if symbol.TypeName != "" {
				typeValue := ParseSemanticType(symbol.TypeName)
				if typeValue.Kind == TypeKindNamed {
					typeValue.Kind = TypeKindUserDefined
					typeValue.UserDefined = true
					typeValue.Unresolved = false
				}
				return typeValue
			}
			if symbol.ReturnTypeSummary != "" {
				return ParseSemanticType(symbol.ReturnTypeSummary)
			}
		}
	}
	return ParseSemanticType(strings.Join(tokenSpellings(tokens), " "))
}

func looksLikeNumericLiteral(value string) bool {
	if value == "" {
		return false
	}
	first := value[0]
	return (first >= '0' && first <= '9') || first == '+' || first == '-'
}

func tokenSpellings(tokens []parser.Token) []string {
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, token.Spelling)
	}
	return result
}

func isIntegerLiteral(value string) bool {
	if value == "" {
		return false
	}
	if _, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimSuffix(value, "u"), "U"), 0, 64); err == nil {
		return true
	}
	return false
}

func semanticDiagnosticForCall(code string, severity SemanticSeverity, call CallFact, context SemanticContext) SemanticDiagnostic {
	passKind := "not-applicable"
	if call.PassKind == PassKindHost || call.PassKind == PassKindDevice {
		passKind = string(call.PassKind)
	} else if code == "tops-semantic-invalid-call" && call.PassKind == PassKindHostDevice {
		passKind = string(PassKindDevice)
	}
	return SemanticDiagnostic{Code: code, Severity: severity, SemanticKind: strings.TrimPrefix(code, "tops-semantic-"), Message: semanticDiagnosticMessage(code), Range: call.Range, PassKind: passKind, ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: context.DocumentVersion, ContextVersion: context.SemanticContextVersion, SourceVersion: context.SourceVersionFingerprint, ConditionalProvenance: call.Condition, Candidates: append([]string(nil), call.Candidates...)}
}

func isTopsQualifier(name string) bool {
	switch name {
	case "__global", "__global__", "__local", "__local__", "__shared", "__shared__", "__constant", "__constant__", "__host__", "__device__", "__kernel__", "__vector", "__vector2", "__vector4", "__vector8":
		return true
	default:
		return false
	}
}

func contextStatusCode(status ContextStatus) (string, SemanticSeverity) {
	if status == ContextStatusMissing {
		return "tops-semantic-context-missing", SeverityWarning
	}
	return "tops-semantic-context-degraded", SeverityWarning
}

func semanticDiagnosticForReference(code string, severity SemanticSeverity, reference ReferenceFact, context SemanticContext) SemanticDiagnostic {
	passKind := "not-applicable"
	if code == "tops-semantic-invalid-call" {
		if context.Command.PassKind == PassKindHost || context.Command.PassKind == PassKindDevice {
			passKind = string(context.Command.PassKind)
		} else if context.Command.PassKind == PassKindHostDevice {
			passKind = string(PassKindDevice)
		}
	}
	return SemanticDiagnostic{Code: code, Severity: severity, SemanticKind: strings.TrimPrefix(code, "tops-semantic-"), Message: semanticDiagnosticMessage(code), Range: reference.Range, PassKind: passKind, ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: context.DocumentVersion, ContextVersion: context.SemanticContextVersion, SourceVersion: context.SourceVersionFingerprint, ConditionalProvenance: reference.Condition, Candidates: append([]string(nil), reference.Candidates...)}
}

func attachDiagnosticRelatedInformation(snapshot index.IndexSnapshot, diagnostics []SemanticDiagnostic) []SemanticDiagnostic {
	for diagnosticIndex := range diagnostics {
		diagnostic := &diagnostics[diagnosticIndex]
		if len(diagnostic.Related) > 0 || len(diagnostic.Candidates) == 0 {
			continue
		}
		candidates := append([]string(nil), diagnostic.Candidates...)
		sort.Strings(candidates)
		for _, candidateID := range candidates {
			symbol, ok := snapshot.Symbols[candidateID]
			if !ok || symbol.NameRange.Start == symbol.NameRange.End {
				continue
			}
			diagnostic.Related = append(diagnostic.Related, SemanticRelatedInformation{URI: symbol.SourceVersion.URI, Range: symbol.NameRange, Message: "candidate " + symbol.QualifiedName})
		}
	}
	return diagnostics
}

func semanticDiagnosticMessage(code string) string {
	switch code {
	case "tops-semantic-unknown-name":
		return "name is not visible in the current scope"
	case "tops-semantic-ambiguous-name":
		return "name has multiple visible candidates"
	case "tops-semantic-context-missing":
		return "semantic context is unavailable"
	case "tops-semantic-type-mismatch":
		return "argument or value type is not compatible"
	case "tops-semantic-invalid-call":
		return "the selected symbol is not callable"
	default:
		return "semantic context is partially available"
	}
}
