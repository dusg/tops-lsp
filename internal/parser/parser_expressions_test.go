package parser

import "testing"

func TestParseRecognizesTemplateAndExpressionNodes(t *testing.T) {
	result := Parse(`template <typename T> T transform(T value) { if (value > 0) return value + 1; return value; }`, ParseContext{LanguageStandard: "c++17"})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if findNode(result.Root, NodeTemplate) == nil {
		t.Fatalf("template node missing: %+v", result.Root)
	}
	if findNode(result.Root, NodeExpression) == nil && findNode(result.Root, NodeStatement) == nil {
		t.Fatalf("expression/statement nodes missing: %+v", result.Root)
	}
}

func TestParseReportsMissingSemicolonAndKeepsFollowingFunction(t *testing.T) {
	result := Parse("int broken() { return 1 } int valid_after_error() { return 2; }", ParseContext{})
	if len(result.Diagnostics) == 0 || result.Diagnostics[0].Code != DiagnosticMissingToken {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if findNode(result.Root, NodeFunction) == nil {
		t.Fatalf("function nodes missing: %+v", result.Root)
	}
	if !containsFunction(result.Root, "valid_after_error") {
		t.Fatalf("following function missing: %+v", result.Root)
	}
}

func TestParseSplitsFollowingFunctionAfterMissingDeclarationSemicolon(t *testing.T) {
	result := Parse("int value = 1\nint main() { return value;", ParseContext{LanguageStandard: "c++17"})
	if !containsFunction(result.Root, "main") {
		t.Fatalf("following function missing: %+v", result.Root)
	}
	if !hasDiagnostic(result.Diagnostics, DiagnosticMissingToken) {
		t.Fatalf("missing declaration recovery diagnostic = %+v", result.Diagnostics)
	}
}

func TestParseBuildsStructuredExpressionAndDeclarationParts(t *testing.T) {
	result := Parse("int value = left + right * scale; int next = make(value);", ParseContext{LanguageStandard: "c++17"})
	declarations := namedNodes(result.Root, NodeDeclaration)
	if len(declarations) != 2 {
		t.Fatalf("declarations = %+v", declarations)
	}
	first := declarations[0]
	if first.Expression == nil || first.Expression.Operator != "+" || len(first.Expression.Children) != 2 {
		t.Fatalf("expression = %+v", first.Expression)
	}
	if first.Expression.Children[1].Operator != "*" {
		t.Fatalf("operator precedence tree = %+v", first.Expression)
	}
	if len(first.TypeTokens) != 1 || first.TypeTokens[0].Spelling != "int" || len(first.DeclaratorTokens) == 0 || first.DeclaratorTokens[0].Spelling != "value" {
		t.Fatalf("declaration parts type=%v declarator=%v", first.TypeTokens, first.DeclaratorTokens)
	}
	if declarations[1].Name != "next" || declarations[1].Expression == nil || declarations[1].Expression.Operator != "call" {
		t.Fatalf("following declaration = %+v", declarations[1])
	}
}

func namedNodes(root *SyntaxNode, kind NodeKind) []*SyntaxNode {
	result := make([]*SyntaxNode, 0)
	var visit func(*SyntaxNode)
	visit = func(node *SyntaxNode) {
		if node == nil {
			return
		}
		if node.Kind == kind {
			result = append(result, node)
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(root)
	return result
}

func containsFunction(node *SyntaxNode, name string) bool {
	if node == nil {
		return false
	}
	if node.Kind == NodeFunction && node.Name == name {
		return true
	}
	for _, child := range node.Children {
		if containsFunction(child, name) {
			return true
		}
	}
	return false
}
