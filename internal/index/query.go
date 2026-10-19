package index

import (
	"sort"
	"strconv"

	"tops-lsp/internal/parser"
)

func (snapshot IndexSnapshot) DefinitionAt(offset int) []SymbolLocation {
	return snapshot.DefinitionAtURI(snapshot.RootURI, offset)
}

func (snapshot IndexSnapshot) DefinitionAtURI(uri string, offset int) []SymbolLocation {
	if snapshot.IndexStatus == IndexStatusStale {
		return nil
	}
	ids := snapshot.symbolIDsAt(uri, offset)
	return snapshot.locationsForSymbols(ids)
}

func (snapshot IndexSnapshot) ReferencesAt(offset int, includeDeclaration bool) []SymbolLocation {
	return snapshot.ReferencesAtURI(snapshot.RootURI, offset, includeDeclaration)
}

func (snapshot IndexSnapshot) ReferencesAtURI(uri string, offset int, includeDeclaration bool) []SymbolLocation {
	if snapshot.IndexStatus == IndexStatusStale {
		return nil
	}
	ids := snapshot.symbolIDsAt(uri, offset)
	locations := make([]SymbolLocation, 0)
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok {
			continue
		}
		if includeDeclaration {
			switch {
			case symbol.Kind == SymbolKindBuiltin && symbol.Builtin.Availability == "available":
				locations = append(locations, symbol.AttributeSources...)
				if len(symbol.AttributeSources) == 0 {
					locations = append(locations, symbol.Declarations...)
					locations = append(locations, symbol.Definitions...)
				}
			case symbol.Kind != SymbolKindBuiltin:
				locations = append(locations, symbol.Declarations...)
				locations = append(locations, symbol.Definitions...)
				locations = append(locations, symbol.AttributeSources...)
			}
		}
		for _, referenceID := range symbol.References {
			if reference, ok := snapshot.References[referenceID]; ok {
				if reference.Status == ReferenceStatusInactive || reference.Condition.State == parser.ConditionalInactive {
					continue
				}
				locations = append(locations, SymbolLocation{
					URI:            reference.URI,
					Range:          reference.Range,
					SelectionRange: reference.Range,
					Role:           SymbolLocationReference,
					SourceVersion:  reference.SourceVersion,
				})
			}
		}
	}
	return uniqueLocations(locations)
}

func (snapshot IndexSnapshot) DocumentSymbols() []DocumentSymbol {
	if snapshot.IndexStatus == IndexStatusStale {
		return nil
	}
	return snapshot.documentSymbolsForIDs(snapshot.RootSymbols)
}

// SymbolsAtURI returns the symbols whose name or declaration covers offset.
func (snapshot IndexSnapshot) SymbolsAtURI(uri string, offset int) []SymbolRecord {
	if snapshot.IndexStatus == IndexStatusStale {
		return nil
	}
	result := make([]SymbolRecord, 0)
	for _, id := range snapshot.symbolIDsAt(uri, offset) {
		symbol, ok := snapshot.Symbols[id]
		if !ok || !symbolQueryable(symbol) {
			continue
		}
		result = append(result, symbol)
	}
	sortSymbolsByName(result)
	return result
}

// VisibleSymbolsAt returns the symbols visible from the innermost scope that
// contains offset, including enclosing scopes and translation-unit symbols.
func (snapshot IndexSnapshot) VisibleSymbolsAt(uri string, offset int) []SymbolRecord {
	if snapshot.IndexStatus == IndexStatusStale {
		return nil
	}
	innermost := scopeAtSnapshot(&snapshot, uri, offset)
	included := make(map[string]struct{})
	for current := innermost; current != ""; {
		scope, ok := snapshot.Scopes[current]
		if !ok {
			break
		}
		for _, id := range scope.Symbols {
			included[id] = struct{}{}
		}
		current = scope.ParentID
	}
	for _, scope := range snapshot.Scopes {
		if scope.Kind != ScopeTranslationUnit && scope.Kind != ScopeHeader {
			continue
		}
		for _, id := range scope.Symbols {
			included[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(included))
	for id := range included {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]SymbolRecord, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok || !symbolQueryable(symbol) {
			continue
		}
		if symbol.Kind == SymbolKindBuiltin && symbol.Builtin.Availability != "" && symbol.Builtin.Availability != "available" {
			continue
		}
		if symbol.ScopeID == innermost && symbol.SourceVersion.URI == uri && symbol.NameRange.Start > offset {
			continue
		}
		key := symbol.Name + "\x00" + string(symbol.Kind) + "\x00" + strconv.Itoa(symbol.NameRange.Start) + "\x00" + strconv.Itoa(symbol.NameRange.End)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, symbol)
	}
	sortSymbolsByName(result)
	return result
}

// symbolQueryable reports whether a symbol may be returned by a position query.
func symbolQueryable(symbol SymbolRecord) bool {
	return symbol.Condition.State != parser.ConditionalInactive && symbol.Status != SymbolStatusStale
}

// sortSymbolsByName orders symbols by name and then by stable symbol id.
func sortSymbolsByName(symbols []SymbolRecord) {
	sort.SliceStable(symbols, func(left, right int) bool {
		if symbols[left].Name != symbols[right].Name {
			return symbols[left].Name < symbols[right].Name
		}
		return symbols[left].SymbolID < symbols[right].SymbolID
	})
}

func (snapshot IndexSnapshot) symbolIDsAt(uri string, offset int) []string {
	ids := make([]string, 0)
	for _, reference := range snapshot.References {
		if reference.URI != uri {
			continue
		}
		if reference.Status == ReferenceStatusInactive || reference.Condition.State == parser.ConditionalInactive {
			continue
		}
		if containsOffset(reference.Range, offset) {
			for _, candidate := range reference.Candidates {
				ids = appendUniqueString(ids, candidate)
			}
		}
	}
	for id, symbol := range snapshot.Symbols {
		if symbolMatchesOffset(symbol, uri, offset) {
			ids = appendUniqueString(ids, id)
		}
	}
	return filterCandidates(snapshot, ids)
}

// symbolMatchesOffset reports whether the symbol's name, declaration,
// definition, attribute source or attribute usage covers offset in uri.
func symbolMatchesOffset(symbol SymbolRecord, uri string, offset int) bool {
	if symbol.Condition.State != parser.ConditionalInactive && symbol.SourceVersion.URI == uri && containsOffset(symbol.NameRange, offset) {
		return true
	}
	for _, location := range symbol.Declarations {
		if location.URI == uri && location.Role == SymbolLocationDeclaration && containsOffset(location.SelectionRange, offset) {
			return true
		}
	}
	for _, location := range symbol.Definitions {
		if location.URI == uri && location.Role == SymbolLocationDefinition && containsOffset(location.SelectionRange, offset) {
			return true
		}
	}
	for _, location := range symbol.AttributeSources {
		if location.URI == uri && containsOffset(location.SelectionRange, offset) {
			return true
		}
	}
	if symbol.Kind != SymbolKindAttribute {
		return false
	}
	for _, usage := range symbol.AttributeUsages {
		selectionRange := usage.SelectionRange
		if selectionRange.Start == selectionRange.End {
			selectionRange = usage.Range
		}
		if usage.Condition.State != parser.ConditionalInactive && containsOffset(selectionRange, offset) {
			return true
		}
	}
	return false
}

func (snapshot IndexSnapshot) locationsForSymbols(ids []string) []SymbolLocation {
	active := make([]SymbolLocation, 0)
	unknown := make([]SymbolLocation, 0)
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok || symbol.Condition.State == parser.ConditionalInactive {
			continue
		}
		if symbol.Kind == SymbolKindBuiltin && symbol.Builtin.Availability != "available" {
			continue
		}
		locations := symbol.Definitions
		if (symbol.Kind == SymbolKindAttribute || symbol.Kind == SymbolKindBuiltin) && len(symbol.AttributeSources) > 0 {
			locations = symbol.AttributeSources
		}
		if len(locations) == 0 {
			locations = symbol.Declarations
		}
		if symbol.Condition.State == parser.ConditionalUnknown {
			unknown = append(unknown, locations...)
		} else {
			active = append(active, locations...)
		}
	}
	if len(active) > 0 {
		return uniqueLocations(active)
	}
	return uniqueLocations(unknown)
}

func filterCandidates(snapshot IndexSnapshot, ids []string) []string {
	filtered := make([]string, 0, len(ids))
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok || symbol.Condition.State == parser.ConditionalInactive {
			continue
		}
		filtered = appendUniqueString(filtered, id)
	}
	return filtered
}

func (snapshot IndexSnapshot) documentSymbolsForIDs(ids []string) []DocumentSymbol {
	result := make([]DocumentSymbol, 0, len(ids))
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok || symbol.Condition.State == parser.ConditionalInactive {
			continue
		}
		kind := symbol.Kind
		if symbol.TypeKind == "alias" {
			kind = SymbolKindObject
		}
		result = append(result, DocumentSymbol{
			Name:           symbol.Name,
			Kind:           kind,
			TypeKind:       symbol.TypeKind,
			Range:          firstLocationRange(symbol),
			SelectionRange: symbol.NameRange,
			Detail:         symbol.Detail,
			Children:       snapshot.documentSymbolsForIDs(symbol.Children),
		})
	}
	return result
}

func firstLocationRange(symbol SymbolRecord) parser.SourceRange {
	if len(symbol.Definitions) > 0 {
		return symbol.Definitions[0].Range
	}
	if len(symbol.Declarations) > 0 {
		return symbol.Declarations[0].Range
	}
	if len(symbol.AttributeUsages) > 0 {
		return symbol.AttributeUsages[0].Range
	}
	return parser.SourceRange{}
}

func containsOffset(sourceRange parser.SourceRange, offset int) bool {
	return sourceRange.Start <= offset && offset < sourceRange.End
}

func uniqueLocations(locations []SymbolLocation) []SymbolLocation {
	result := make([]SymbolLocation, 0, len(locations))
	seen := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		key := location.URI + ":" + strconv.Itoa(location.Range.Start) + ":" + strconv.Itoa(location.Range.End)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, location)
	}
	return result
}
