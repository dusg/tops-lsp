package index

import (
	"testing"

	"tops-lsp/internal/parser"
)

func TestSymbolModelStoresNavigationMetadata(t *testing.T) {
	location := SymbolLocation{
		URI:            "file:///workspace/main.tops",
		Range:          parser.SourceRange{Start: 10, End: 20},
		SelectionRange: parser.SourceRange{Start: 14, End: 17},
		Role:           SymbolLocationDefinition,
	}
	symbol := SymbolRecord{
		SymbolID:            "symbol-1",
		Name:                "add",
		QualifiedName:       "demo::add",
		Kind:                SymbolKindFunction,
		Signature:           "int add(int, int)",
		ParameterSummary:    "int, int",
		ReturnTypeSummary:   "int",
		ExecutionQualifiers: []string{"__device__"},
		Definitions:         []SymbolLocation{location},
		Status:              SymbolStatusActive,
	}
	if symbol.Kind != SymbolKindFunction || len(symbol.Definitions) != 1 {
		t.Fatalf("symbol = %+v", symbol)
	}
	if symbol.Definitions[0].SelectionRange != location.SelectionRange {
		t.Fatalf("selection range = %+v", symbol.Definitions[0].SelectionRange)
	}
}

func TestStableSymbolIDIgnoresBodyAndWhitespaceChanges(t *testing.T) {
	identity := SymbolIdentity{
		OriginURI:       "file:///workspace/main.tops",
		Kind:            SymbolKindFunction,
		QualifiedName:   "demo::add",
		DeclarationRole: "definition",
		Anchor:          "int add(int, int)",
	}
	first := StableSymbolID(identity)
	identity.Anchor = "int   add(int, int)"
	second := StableSymbolID(identity)
	if first != second {
		t.Fatalf("IDs changed for equivalent declaration: %q != %q", first, second)
	}
}

func TestIndexSnapshotStoresConditionAndDependencyStatus(t *testing.T) {
	snapshot := IndexSnapshot{
		RootURI:     "file:///workspace/main.tops",
		IndexStatus: IndexStatusDegraded,
		Dependencies: []HeaderDependency{{
			RequesterURI:    "file:///workspace/main.tops",
			IncludeSpelling: "missing.h",
			Status:          HeaderDependencyMissing,
		}},
	}
	if snapshot.IndexStatus != IndexStatusDegraded || snapshot.Dependencies[0].Status != HeaderDependencyMissing {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestSymbolClassOfSplitsTypesAndKeepsUnknownLast(t *testing.T) {
	cases := []struct {
		kind     SymbolKind
		typeKind string
		want     SymbolClass
	}{
		{SymbolKindFunction, "", SymbolClassFunction},
		{SymbolKindType, "enum", SymbolClassEnum},
		{SymbolKindType, "struct", SymbolClassStruct},
		{SymbolKindType, "union", SymbolClassUnion},
		{SymbolKindType, "alias", SymbolClassAlias},
		{SymbolKindType, "class", SymbolClassClass},
		{SymbolKindType, "", SymbolClassClass},
		{SymbolKindNamespace, "", SymbolClassNamespace},
		{SymbolKindObject, "", SymbolClassObject},
		{SymbolKindField, "", SymbolClassField},
		{SymbolKindAttribute, "", SymbolClassAttribute},
		{SymbolKindVariable, "", SymbolClassVariable},
		{SymbolKindBuiltin, "", SymbolClassBuiltin},
		{SymbolKindMacro, "", SymbolClassMacro},
		{SymbolKindHeader, "", SymbolClassUnknown},
	}
	for _, testCase := range cases {
		if got := SymbolClassOf(testCase.kind, testCase.typeKind); got != testCase.want {
			t.Fatalf("SymbolClassOf(%q, %q) = %q, want %q", testCase.kind, testCase.typeKind, got, testCase.want)
		}
	}
}
