package semantic

import (
	"testing"

	"tops-lsp/internal/parser"
)

func TestSemanticConditionStatesPreserveMacroProvenance(t *testing.T) {
	tests := []struct {
		name   string
		macros map[string]parser.MacroValue
		want   parser.ConditionalState
	}{
		{name: "active", macros: map[string]parser.MacroValue{"FLAG": {Kind: parser.MacroInteger, Value: 1}}, want: parser.ConditionalActive},
		{name: "inactive", macros: map[string]parser.MacroValue{"FLAG": {Kind: parser.MacroInteger, Value: 0}}, want: parser.ConditionalInactive},
		{name: "unknown", macros: map[string]parser.MacroValue{}, want: parser.ConditionalUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := "#if FLAG\nint value;\n#endif\n"
			result := parser.Parse(source, parser.ParseContext{DocumentURI: "file:///workspace/condition.tops", LanguageID: "tops", CompilerContextStatus: "resolved", TargetProfile: "gcu400", PassKind: "host+device", PredefinedMacros: test.macros, DocumentVersion: 1, ContextVersion: 1})
			if len(result.ConditionalRegions) != 1 || result.ConditionalRegions[0].State != test.want {
				t.Fatalf("conditional regions = %+v", result.ConditionalRegions)
			}
		})
	}
}
