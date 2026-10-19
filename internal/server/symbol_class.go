package server

import (
	"tops-lsp/internal/index"
	"tops-lsp/internal/protocol"
)

// symbolClassPresentation renders one symbol class for the three protocol
// surfaces that expose it. Keeping them in one table makes the class-to-kind
// rules single-sourced instead of three parallel switches.
type symbolClassPresentation struct {
	protocolKind   protocol.SymbolKind
	completionKind int
	hoverLabel     string
}

var symbolClassPresentations = map[index.SymbolClass]symbolClassPresentation{
	index.SymbolClassFunction:  {protocolKind: protocol.SymbolKindFunction, completionKind: protocol.CompletionItemKindFunction, hoverLabel: "function"},
	index.SymbolClassEnum:      {protocolKind: protocol.SymbolKindEnum, completionKind: protocol.CompletionItemKindEnum, hoverLabel: "enum"},
	index.SymbolClassStruct:    {protocolKind: protocol.SymbolKindStruct, completionKind: protocol.CompletionItemKindStruct, hoverLabel: "struct"},
	index.SymbolClassUnion:     {protocolKind: protocol.SymbolKindStruct, completionKind: protocol.CompletionItemKindStruct, hoverLabel: "union"},
	index.SymbolClassAlias:     {protocolKind: protocol.SymbolKindClass, completionKind: protocol.CompletionItemKindClass, hoverLabel: "type alias"},
	index.SymbolClassClass:     {protocolKind: protocol.SymbolKindClass, completionKind: protocol.CompletionItemKindClass, hoverLabel: "class"},
	index.SymbolClassNamespace: {protocolKind: protocol.SymbolKindNamespace, completionKind: protocol.CompletionItemKindModule, hoverLabel: "namespace"},
	index.SymbolClassObject:    {protocolKind: protocol.SymbolKindObject, completionKind: protocol.CompletionItemKindVariable, hoverLabel: "symbol"},
	index.SymbolClassField:     {protocolKind: protocol.SymbolKindField, completionKind: protocol.CompletionItemKindField, hoverLabel: "field"},
	index.SymbolClassAttribute: {protocolKind: protocol.SymbolKindProperty, completionKind: protocol.CompletionItemKindProperty, hoverLabel: "attribute"},
	index.SymbolClassVariable:  {protocolKind: protocol.SymbolKindVariable, completionKind: protocol.CompletionItemKindVariable, hoverLabel: "variable"},
	index.SymbolClassBuiltin:   {protocolKind: protocol.SymbolKindVariable, completionKind: protocol.CompletionItemKindVariable, hoverLabel: "Tops builtin"},
	index.SymbolClassMacro:     {protocolKind: protocol.SymbolKindConstant, completionKind: protocol.CompletionItemKindConstant, hoverLabel: "macro"},
	index.SymbolClassUnknown:   {protocolKind: protocol.SymbolKindObject, completionKind: protocol.CompletionItemKindVariable, hoverLabel: "symbol"},
}

func symbolClassPresentationOf(kind index.SymbolKind, typeKind string) symbolClassPresentation {
	if presentation, ok := symbolClassPresentations[index.SymbolClassOf(kind, typeKind)]; ok {
		return presentation
	}
	return symbolClassPresentations[index.SymbolClassUnknown]
}

func protocolSymbolKind(kind index.SymbolKind, typeKind string) protocol.SymbolKind {
	return symbolClassPresentationOf(kind, typeKind).protocolKind
}

func completionItemKind(symbol index.SymbolRecord) int {
	return symbolClassPresentationOf(symbol.Kind, symbol.TypeKind).completionKind
}

func hoverKindLabel(symbol index.SymbolRecord) string {
	return symbolClassPresentationOf(symbol.Kind, symbol.TypeKind).hoverLabel
}
