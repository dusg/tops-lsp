package semantic

import (
	"testing"

	"tops-lsp/internal/index"
)

func TestBuildCallableFactExecutionSpacesAndDeclarationState(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	tests := []struct {
		name        string
		qualifiers  []string
		wantKind    string
		wantDefault bool
		wantPasses  []PassKind
	}{
		{name: "default host", wantKind: "host", wantDefault: true, wantPasses: []PassKind{PassKindHost}},
		{name: "device", qualifiers: []string{"__device__"}, wantKind: "device", wantPasses: []PassKind{PassKindDevice}},
		{name: "host device", qualifiers: []string{"__host__", "__device__"}, wantKind: "host-device", wantPasses: []PassKind{PassKindHost, PassKindDevice}},
		{name: "kernel entry", qualifiers: []string{"__global__"}, wantKind: "kernel-entry", wantPasses: []PassKind{PassKindDevice}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			symbol := index.SymbolRecord{SymbolID: test.name, Name: test.name, Kind: index.SymbolKindFunction, ParameterSummary: "int left, float right", ReturnTypeSummary: "long", ExecutionQualifiers: test.qualifiers, Status: index.SymbolStatusActive}
			fact := BuildCallableFact(symbol, context)
			if fact.ExecutionKind != test.wantKind || fact.DefaultExecutionSpace != test.wantDefault || len(fact.Parameters) != 2 || len(fact.PassKinds) != len(test.wantPasses) {
				t.Fatalf("callable = %+v", fact)
			}
			for index, pass := range test.wantPasses {
				if fact.PassKinds[index] != pass {
					t.Fatalf("pass kinds = %v", fact.PassKinds)
				}
			}
		})
	}
	declaration := BuildCallableFact(index.SymbolRecord{SymbolID: "declaration", Name: "declaration", Kind: index.SymbolKindFunction, Status: index.SymbolStatusDeclarationOnly}, context)
	if !declaration.DeclarationOnly {
		t.Fatalf("declaration-only callable = %+v", declaration)
	}
}
