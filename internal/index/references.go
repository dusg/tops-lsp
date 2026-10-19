package index

import (
	"fmt"
	"sort"
	"tops-lsp/internal/parser"
)

func rebuildReferences(snapshot *IndexSnapshot) {
	preserved := make(map[string]ReferenceRecord)
	for id, reference := range snapshot.References {
		if reference.URI != snapshot.RootURI {
			preserved[id] = reference
		}
	}
	snapshot.References = preserved
	for id, symbol := range snapshot.Symbols {
		symbol.References = nil
		snapshot.Symbols[id] = symbol
	}
	for _, reference := range preserved {
		reference = normalizeBuiltinReference(snapshot, reference)
		snapshot.References[reference.ReferenceID] = reference
		attachReference(snapshot, reference)
	}
	declarationRanges := make([]parser.SourceRange, 0)
	for _, symbol := range snapshot.Symbols {
		if symbol.SourceVersion.URI == snapshot.RootURI {
			declarationRanges = append(declarationRanges, symbol.NameRange)
		}
	}
	collectSnapshotReferences(snapshot, snapshot.RootURI, snapshot.Tokens, snapshot.ConditionalRegions, declarationRanges, snapshot.SourceVersion)
	collectAttributeReferencesForSnapshot(snapshot)
}

func collectSnapshotReferences(snapshot *IndexSnapshot, uri string, tokens []parser.Token, regions []parser.ConditionalRegion, declarationRanges []parser.SourceRange, sourceVersion SourceVersion) {
	for tokenIndex, token := range tokens {
		if token.Kind != parser.TokenIdentifier || containsRange(declarationRanges, token.Range) {
			continue
		}
		scopeID := scopeAtSnapshot(snapshot, uri, token.Range.Start)
		qualifiedName := qualifiedReferenceName(tokens, tokenIndex)
		candidates := []string(nil)
		if token.ConditionalState != parser.ConditionalInactive {
			candidates = candidateIDsForReference(tokens, tokenIndex, snapshot, token.Spelling, qualifiedName, scopeID, token.ConditionalState)
		}
		if hasAttributeCandidate(snapshot, candidates) {
			continue
		}
		condition := provenanceAtRegions(regions, token.Range.Start, uri)
		if condition.State == "" {
			condition = ConditionalProvenance{SourceURI: uri, State: token.ConditionalState}
		}
		reference := newReferenceRecord(uri, token, qualifiedName, scopeID, candidates, condition, referenceKind(tokens, tokenIndex, snapshot, candidates), sourceVersion)
		reference.Builtin = builtinProvenanceForSnapshot(snapshot, candidates, token)
		reference = normalizeBuiltinReference(snapshot, reference)
		snapshot.References[reference.ReferenceID] = reference
		attachReference(snapshot, reference)
	}
}

func symbolIDsByName(snapshot *IndexSnapshot, name string) []string {
	ids := make([]string, 0)
	for id, symbol := range snapshot.Symbols {
		if symbol.Name == name {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func scopeAtSnapshot(snapshot *IndexSnapshot, uri string, offset int) string {
	best := ""
	bestSize := int(^uint(0) >> 1)
	for id, scope := range snapshot.Scopes {
		if scope.SourceURI != "" && scope.SourceURI != uri {
			continue
		}
		if scope.Range.Start <= offset && offset < scope.Range.End {
			size := scope.Range.End - scope.Range.Start
			if size < bestSize {
				best = id
				bestSize = size
			}
		}
	}
	return best
}

func newReferenceRecord(uri string, token parser.Token, qualifiedName, scopeID string, candidates []string, condition ConditionalProvenance, kind string, sourceVersion SourceVersion) ReferenceRecord {
	return ReferenceRecord{
		ReferenceID:   fmt.Sprintf("ref:%s:%d:%d", uri, token.Range.Start, token.Range.End),
		URI:           uri,
		Range:         token.Range,
		Name:          token.Spelling,
		QualifiedName: qualifiedName,
		ScopeID:       scopeID,
		Candidates:    append([]string(nil), candidates...),
		Condition:     condition,
		Kind:          kind,
		Status:        referenceStatus(token.ConditionalState, candidates),
		SourceVersion: sourceVersion,
	}
}

func cloneMacroContext(macros map[string]parser.MacroValue) map[string]parser.MacroValue {
	if macros == nil {
		return nil
	}
	if len(macros) == 0 {
		return map[string]parser.MacroValue{}
	}
	result := make(map[string]parser.MacroValue, len(macros))
	for name, value := range macros {
		result[name] = value
	}
	return result
}

func builtinProvenanceForContext(context parser.ParseContext, sourceURI, availability string) BuiltinProvenance {
	return BuiltinProvenance{
		TargetProfile: context.TargetProfile,
		PassKind:      context.PassKind,
		MacroContext:  cloneMacroContext(context.PredefinedMacros),
		SourceURI:     sourceURI,
		Availability:  availability,
	}
}

func builtinProvenanceForSnapshot(snapshot *IndexSnapshot, candidates []string, token parser.Token) BuiltinProvenance {
	if !isBuiltinSpelling(token.Spelling) {
		return BuiltinProvenance{}
	}
	for _, candidate := range candidates {
		if symbol, ok := snapshot.Symbols[candidate]; ok && symbol.Kind == SymbolKindBuiltin {
			provenance := symbol.Builtin
			if provenance.Availability == "" {
				provenance.Availability = "available"
			}
			return provenance
		}
	}
	return BuiltinProvenance{
		TargetProfile: snapshot.TargetProfile,
		PassKind:      snapshot.PassKind,
		MacroContext:  cloneMacroContext(snapshot.PredefinedMacros),
		Availability:  "unresolved",
	}
}

func normalizeBuiltinReference(snapshot *IndexSnapshot, reference ReferenceRecord) ReferenceRecord {
	if reference.Kind != "builtin-use" {
		return reference
	}
	if reference.Builtin.Availability != "available" {
		reference.Candidates = nil
		reference.Status = ReferenceStatusUnresolved
	}
	return reference
}

func referenceStatus(state parser.ConditionalState, candidates []string) ReferenceStatus {
	if state == parser.ConditionalInactive {
		return ReferenceStatusInactive
	}
	if len(candidates) == 0 {
		if state == parser.ConditionalUnknown {
			return ReferenceStatusUnknown
		}
		return ReferenceStatusUnresolved
	}
	if state == parser.ConditionalUnknown {
		return ReferenceStatusUnknown
	}
	if len(candidates) > 1 {
		return ReferenceStatusAmbiguous
	}
	return ReferenceStatusResolved
}

func referenceKind(tokens []parser.Token, index int, snapshot *IndexSnapshot, candidates []string) string {
	if index+1 < len(tokens) && tokens[index+1].Spelling == "(" {
		return "call"
	}
	if index > 0 && (tokens[index-1].Spelling == "." || tokens[index-1].Spelling == "->") {
		return "member-use"
	}
	for _, candidate := range candidates {
		symbol, ok := snapshot.Symbols[candidate]
		if !ok {
			continue
		}
		switch symbol.Kind {
		case SymbolKindType:
			return "type-use"
		case SymbolKindAttribute:
			return "attribute-use"
		case SymbolKindBuiltin:
			return "builtin-use"
		case SymbolKindMacro:
			return "macro-use"
		}
	}
	if isBuiltinSpelling(tokens[index].Spelling) {
		return "builtin-use"
	}
	return "other"
}

func attachReference(snapshot *IndexSnapshot, reference ReferenceRecord) {
	for _, candidate := range reference.Candidates {
		symbol, ok := snapshot.Symbols[candidate]
		if !ok {
			continue
		}
		symbol.References = appendUniqueString(symbol.References, reference.ReferenceID)
		if reference.Status == ReferenceStatusAmbiguous {
			symbol.Status = SymbolStatusAmbiguous
		}
		snapshot.Symbols[candidate] = symbol
	}
}
