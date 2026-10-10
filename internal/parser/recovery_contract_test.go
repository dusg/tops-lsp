package parser

import "testing"

func TestRecoveryContractAcrossTruncatedContexts(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantNode       NodeKind
		wantValue      string
		wantDiagnostic string
		wantRelated    bool
	}{
		{name: "template", source: "template <typename T", wantNode: NodeMissingToken, wantValue: ">", wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "attribute", source: "[[deprecated(", wantNode: NodeAttribute, wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "parameter", source: "int call(int value", wantNode: NodeMissingToken, wantValue: ")", wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "initializer", source: "int value = (1", wantNode: NodeMissingToken, wantValue: ")", wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "statement", source: "int main() { if (true) { return 1;", wantNode: NodeMissingToken, wantValue: "}", wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "launch", source: "kernel<<<grid,", wantNode: NodeTopsLaunch, wantDiagnostic: DiagnosticMissingToken, wantRelated: true},
		{name: "directive", source: "#if FLAG\nint value;\n", wantNode: NodeMissingToken, wantValue: "#endif", wantDiagnostic: DiagnosticInvalidDirective, wantRelated: true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := Parse(testCase.source, ParseContext{LanguageStandard: "c++17", DocumentVersion: 5, ContextVersion: 8})
			if result.Root == nil || result.Status == ParseStatusComplete {
				t.Fatalf("result = %+v", result)
			}
			node := findNode(result.Root, testCase.wantNode)
			if node == nil {
				t.Fatalf("missing recovery node %q in %+v", testCase.wantNode, result.Root)
			}
			if !node.Incomplete {
				t.Fatalf("recovery node is not incomplete: %+v", node)
			}
			if testCase.wantValue != "" && node.Value != testCase.wantValue {
				t.Fatalf("recovery value = %q, want %q", node.Value, testCase.wantValue)
			}
			diagnostic := firstDiagnostic(result.Diagnostics, testCase.wantDiagnostic)
			if diagnostic == nil {
				t.Fatalf("missing diagnostic %q in %+v", testCase.wantDiagnostic, result.Diagnostics)
			}
			if diagnostic.Range.Start != len(testCase.source) || diagnostic.Range.End != len(testCase.source) {
				t.Fatalf("diagnostic range = %+v", diagnostic.Range)
			}
			if testCase.wantRelated && len(diagnostic.Related) == 0 {
				t.Fatalf("missing related opening range in %+v", diagnostic)
			}
			if diagnostic.DocumentVersion != 5 || diagnostic.ContextVersion != 8 {
				t.Fatalf("diagnostic versions = %+v", diagnostic)
			}
		})
	}
}

func TestRecoveryContractPreservesOpaqueAndErrorNodes(t *testing.T) {
	opaque := Parse("[[vendor::property(1)]] int value;", ParseContext{LanguageStandard: "c++17"})
	if findNode(opaque.Root, NodeOpaque) == nil {
		t.Fatalf("opaque attribute node missing: %+v", opaque.Root)
	}
	errorResult := Parse("int value; )", ParseContext{LanguageStandard: "c++17"})
	errorNode := findNode(errorResult.Root, NodeError)
	if errorNode == nil || errorNode.Value != ")" || errorNode.Incomplete {
		t.Fatalf("error node = %+v", errorNode)
	}
}

func TestRecoveryContractSuppressesCascadeAndKeepsFollowingDeclaration(t *testing.T) {
	result := Parse("int broken( { return 1; } int valid() { return 2; }", ParseContext{LanguageStandard: "c++17"})
	if !containsFunction(result.Root, "valid") {
		t.Fatalf("following declaration missing: %+v", result.Root)
	}
	seen := make(map[string]struct{})
	for _, diagnostic := range result.Diagnostics {
		key := diagnostic.Code + ":" + string(rune(diagnostic.Range.Start)) + ":" + string(rune(diagnostic.Range.End))
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate recovery diagnostic = %+v", diagnostic)
		}
		seen[key] = struct{}{}
	}
}

func firstDiagnostic(diagnostics []ParserDiagnostic, code string) *ParserDiagnostic {
	for index := range diagnostics {
		if diagnostics[index].Code == code {
			return &diagnostics[index]
		}
	}
	return nil
}
