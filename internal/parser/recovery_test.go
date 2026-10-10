package parser

import "testing"

func TestParseRecoversMissingSemicolonAtEOF(t *testing.T) {
	source := "int value = 1"
	result := Parse(source, ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticMissingToken) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticMissingToken && diagnostic.Range.Start != len(source) {
			t.Fatalf("missing token range = %+v", diagnostic.Range)
		}
	}
	if result.Root == nil || result.Status != ParseStatusRecovered {
		t.Fatalf("result = %+v", result)
	}
	missing := findNode(result.Root, NodeMissingToken)
	if missing == nil || !missing.Incomplete || missing.Range != (SourceRange{Start: len(source), End: len(source)}) {
		t.Fatalf("missing token node = %+v", missing)
	}
}

func TestParseRecoversMissingBraceAndKeepsFollowingFunction(t *testing.T) {
	result := Parse("int broken() { return 1; int valid_after_error() { return 2; }", ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticMissingToken) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if !containsFunction(result.Root, "valid_after_error") {
		t.Fatalf("following function missing: %+v", result.Root)
	}
}

func TestParseKeepsIncompleteKernelLaunchNode(t *testing.T) {
	result := Parse("kernel<<<grid,", ParseContext{})
	if findNode(result.Root, NodeTopsLaunch) == nil {
		t.Fatalf("launch node missing: %+v", result.Root)
	}
	if !hasDiagnostic(result.Diagnostics, DiagnosticMissingToken) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestParseSuppressesCascadeAfterUnterminatedLiteral(t *testing.T) {
	result := Parse(`int value = "unfinished`, ParseContext{})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != DiagnosticUnterminated {
		t.Fatalf("unterminated literal diagnostics = %+v", result.Diagnostics)
	}
}

func TestParseRecoveryReportsOpeningDelimiterLocation(t *testing.T) {
	result := Parse("int main( { return 1; }", ParseContext{})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != DiagnosticMissingToken || diagnostic.Message != "expected ')'" {
			continue
		}
		if len(diagnostic.Related) != 1 || diagnostic.Related[0].Range != (SourceRange{Start: 8, End: 9}) {
			t.Fatalf("related location = %+v", diagnostic.Related)
		}
		return
	}
	t.Fatalf("missing ')' diagnostic = %+v", result.Diagnostics)
}

func TestParseRecoversMissingTemplateClose(t *testing.T) {
	source := "template <typename T"
	result := Parse(source, ParseContext{LanguageStandard: "c++17"})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticMissingToken && diagnostic.Message == "expected '>'" {
			if diagnostic.Range != (SourceRange{Start: len(source), End: len(source)}) {
				t.Fatalf("template diagnostic range = %+v", diagnostic.Range)
			}
			missing := findNode(result.Root, NodeMissingToken)
			if missing == nil || missing.Value != ">" {
				t.Fatalf("template missing node = %+v", missing)
			}
			return
		}
	}
	t.Fatalf("template diagnostics = %+v", result.Diagnostics)
}

func TestParseCreatesErrorNodeForUnexpectedDelimiter(t *testing.T) {
	source := "int value; )"
	result := Parse(source, ParseContext{})
	errorNode := findNode(result.Root, NodeError)
	if errorNode == nil || errorNode.Range != (SourceRange{Start: len("int value; "), End: len(source)}) {
		t.Fatalf("error node = %+v diagnostics=%+v", errorNode, result.Diagnostics)
	}
}

func TestParseCreatesMissingDirectiveNodeAtEOF(t *testing.T) {
	source := "#if FLAG\nint value;\n"
	result := Parse(source, ParseContext{})
	missing := findNode(result.Root, NodeMissingToken)
	if missing == nil || missing.Value != "#endif" || missing.Range != (SourceRange{Start: len(source), End: len(source)}) || !missing.Incomplete {
		t.Fatalf("missing directive node = %+v diagnostics=%+v", missing, result.Diagnostics)
	}
}

func TestRecoveryContractCoversNestedConstructs(t *testing.T) {
	cases := []struct {
		name          string
		source        string
		expectedValue string
		expectedKind  NodeKind
	}{
		{name: "attribute", source: "[[deprecated(", expectedValue: "]]", expectedKind: NodeAttribute},
		{name: "parameter", source: "int function(", expectedValue: ")", expectedKind: NodeFunction},
		{name: "initializer", source: "int value = {1,", expectedValue: "}", expectedKind: NodeDeclaration},
		{name: "statement", source: "if (value", expectedValue: ")", expectedKind: NodeStatement},
		{name: "launch", source: "kernel<<<grid,", expectedValue: ">>>", expectedKind: NodeTopsLaunch},
		{name: "directive", source: "#if FLAG\nint value;\n", expectedValue: "#endif", expectedKind: NodePreprocessorDirective},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := Parse(testCase.source, ParseContext{})
			if result.Root == nil || len(result.Diagnostics) == 0 {
				t.Fatalf("result = %+v", result)
			}
			if findNode(result.Root, testCase.expectedKind) == nil {
				t.Fatalf("expected node %q missing: %+v", testCase.expectedKind, result.Root)
			}
			missing := findNodeWithValue(result.Root, NodeMissingToken, testCase.expectedValue)
			if missing == nil || !missing.Incomplete || missing.Range.Start != missing.Range.End {
				t.Fatalf("expected recovery node %q missing: %+v", testCase.expectedValue, result.Root)
			}
		})
	}
}

func findNodeWithValue(node *SyntaxNode, kind NodeKind, value string) *SyntaxNode {
	if node == nil {
		return nil
	}
	if node.Kind == kind && node.Value == value {
		return node
	}
	for _, child := range node.Children {
		if found := findNodeWithValue(child, kind, value); found != nil {
			return found
		}
	}
	return nil
}
