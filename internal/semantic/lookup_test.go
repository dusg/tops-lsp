package semantic

import (
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestLookupNameRespectsScopeQualificationAndConditionState(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.Scopes["inner"] = index.ScopeRecord{ScopeID: "inner", ParentID: "root", SourceURI: snapshot.RootURI, Kind: index.ScopeBlock, Symbols: []string{"inner-value"}}
	snapshot.Symbols["inner-value"] = index.SymbolRecord{SymbolID: "inner-value", Name: "value", QualifiedName: "value", ScopeID: "inner", Condition: index.ConditionalProvenance{State: parser.ConditionalActive}}
	snapshot.Symbols["inactive"] = index.SymbolRecord{SymbolID: "inactive", Name: "value", QualifiedName: "inactive", ScopeID: "root", Condition: index.ConditionalProvenance{State: parser.ConditionalInactive}}
	local := LookupName(snapshot, "inner", "value")
	if len(local) != 1 || local[0].SymbolID != "inner-value" {
		t.Fatalf("local lookup = %+v", local)
	}
	qualified := LookupName(snapshot, "root", "Point")
	if len(qualified) != 1 || qualified[0].SymbolID != "Point" {
		t.Fatalf("qualified lookup = %+v", qualified)
	}
}

func TestLookupNameKeepsAllQualifiedCandidatesInStableOrder(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	snapshot.Symbols["Point-active"] = index.SymbolRecord{SymbolID: "Point-active", Name: "Point", QualifiedName: "demo::Point", Condition: index.ConditionalProvenance{State: parser.ConditionalActive}}
	snapshot.Symbols["Point-unknown"] = index.SymbolRecord{SymbolID: "Point-unknown", Name: "Point", QualifiedName: "demo::Point", Condition: index.ConditionalProvenance{State: parser.ConditionalUnknown}}
	matches := LookupName(snapshot, "root", "demo::Point")
	if len(matches) != 1 || matches[0].SymbolID != "Point-active" {
		t.Fatalf("qualified matches = %+v", matches)
	}
	delete(snapshot.Symbols, "Point-active")
	matches = LookupName(snapshot, "root", "demo::Point")
	if len(matches) != 1 || matches[0].SymbolID != "Point-unknown" {
		t.Fatalf("unknown qualified matches = %+v", matches)
	}
}
