package parser

import "testing"

func TestParseBuildsNamespaceAndFunctionTree(t *testing.T) {
	result := Parse(`namespace demo { using Value = int; int add(int left, int right) { return left + right; } }`, ParseContext{LanguageStandard: "c++17"})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if result.Root == nil || len(result.Root.Children) != 1 {
		t.Fatalf("root = %+v", result.Root)
	}
	namespace := result.Root.Children[0]
	if namespace.Kind != NodeNamespace || namespace.Name != "demo" {
		t.Fatalf("namespace = %+v", namespace)
	}
	function := findNode(namespace, NodeFunction)
	if function == nil || function.Name != "add" {
		t.Fatalf("function = %+v", function)
	}
	if findNode(function, NodeStatement) == nil {
		t.Fatalf("function body = %+v", function.Children)
	}
}

func TestParsePreservesQualifiedAndAnonymousNamespaceNames(t *testing.T) {
	result := Parse("namespace outer::inner { int value; } namespace { int hidden; }", ParseContext{LanguageStandard: "c++17"})
	if len(result.Root.Children) != 2 {
		t.Fatalf("root children = %+v", result.Root.Children)
	}
	if result.Root.Children[0].Name != "outer::inner" {
		t.Fatalf("qualified namespace name = %q", result.Root.Children[0].Name)
	}
	if result.Root.Children[1].Name != "" {
		t.Fatalf("anonymous namespace name = %q", result.Root.Children[1].Name)
	}
}

func findNode(node *SyntaxNode, kind NodeKind) *SyntaxNode {
	if node == nil {
		return nil
	}
	if node.Kind == kind {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, kind); found != nil {
			return found
		}
	}
	return nil
}
