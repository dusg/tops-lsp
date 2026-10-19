package semantic

import (
	"sort"
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func LookupName(snapshot index.IndexSnapshot, scopeID, name string) []index.SymbolRecord {
	if name == "" {
		return nil
	}
	if strings.Contains(name, "::") {
		active, unknown := lookupQualifiedNames(snapshot, name)
		if len(active) > 0 {
			return active
		}
		if len(unknown) > 0 {
			return unknown
		}
	}
	unknown := make([]index.SymbolRecord, 0)
	seen := make(map[string]struct{})
	for current := scopeID; current != ""; {
		scope, ok := snapshot.Scopes[current]
		if !ok {
			break
		}
		active := make([]index.SymbolRecord, 0)
		scopeUnknown := make([]index.SymbolRecord, 0)
		for _, symbolID := range scope.Symbols {
			if _, exists := seen[symbolID]; exists {
				continue
			}
			symbol, ok := snapshot.Symbols[symbolID]
			if !ok || symbol.Name != name || symbol.Condition.State == parser.ConditionalInactive {
				continue
			}
			seen[symbolID] = struct{}{}
			if symbol.Condition.State == parser.ConditionalUnknown {
				scopeUnknown = append(scopeUnknown, symbol)
			} else {
				active = append(active, symbol)
			}
		}
		if len(active) > 0 {
			sort.Slice(active, func(left, right int) bool { return active[left].SymbolID < active[right].SymbolID })
			return active
		}
		if len(unknown) == 0 && len(scopeUnknown) > 0 {
			unknown = append(unknown, scopeUnknown...)
		}
		current = scope.ParentID
	}
	sort.Slice(unknown, func(left, right int) bool { return unknown[left].SymbolID < unknown[right].SymbolID })
	return unknown
}

func lookupQualifiedNames(snapshot index.IndexSnapshot, name string) (active, unknown []index.SymbolRecord) {
	for _, symbol := range snapshot.Symbols {
		if symbol.QualifiedName != name || symbol.Condition.State == parser.ConditionalInactive {
			continue
		}
		if symbol.Condition.State == parser.ConditionalUnknown {
			unknown = append(unknown, symbol)
		} else {
			active = append(active, symbol)
		}
	}
	sort.Slice(active, func(left, right int) bool { return active[left].SymbolID < active[right].SymbolID })
	sort.Slice(unknown, func(left, right int) bool { return unknown[left].SymbolID < unknown[right].SymbolID })
	return active, unknown
}
