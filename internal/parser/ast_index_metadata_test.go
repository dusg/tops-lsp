package parser

import "testing"

func TestParseRecordsDeclarationNavigationMetadata(t *testing.T) {
	source := "namespace demo { using Value = int; struct Point { int x; }; int add(int left, int right) { return left + right; } }"
	result := Parse(source, ParseContext{LanguageStandard: "c++17"})
	if result.Root == nil {
		t.Fatal("parser returned no root")
	}

	namespace := findNode(result.Root, NodeNamespace)
	if namespace == nil {
		t.Fatal("namespace node not found")
	}

	alias := findNode(namespace, NodeUsing)
	assertNodeNameRange(t, source, alias, "Value")
	if alias.DeclarationRole != DeclarationRoleDeclaration {
		t.Fatalf("alias declaration role = %q, want declaration", alias.DeclarationRole)
	}

	point := findNode(namespace, NodeStruct)
	assertNodeNameRange(t, source, point, "Point")
	if point.DeclarationRole != DeclarationRoleDefinition {
		t.Fatalf("struct declaration role = %q, want definition", point.DeclarationRole)
	}
	if point.BodyRange.Start >= point.BodyRange.End {
		t.Fatalf("struct body range = %+v", point.BodyRange)
	}

	function := findNode(namespace, NodeFunction)
	assertNodeNameRange(t, source, function, "add")
	if function.DeclarationRole != DeclarationRoleDefinition {
		t.Fatalf("function declaration role = %q, want definition", function.DeclarationRole)
	}
	if function.BodyRange.Start >= function.BodyRange.End {
		t.Fatalf("function body range = %+v", function.BodyRange)
	}
}

func TestParseRecordsIncompleteDeclarationMetadata(t *testing.T) {
	source := "int add(int left,"
	result := Parse(source, ParseContext{LanguageStandard: "c++17"})
	function := findNode(result.Root, NodeFunction)
	assertNodeNameRange(t, source, function, "add")
	if function.DeclarationRole != DeclarationRoleUnknown {
		t.Fatalf("incomplete declaration role = %q, want unknown", function.DeclarationRole)
	}
	if findNode(function, NodeMissingToken) == nil {
		t.Fatalf("incomplete function recovery = %+v", function.Children)
	}
}

func assertNodeNameRange(t *testing.T, source string, node *SyntaxNode, want string) {
	t.Helper()
	if node == nil {
		t.Fatalf("node for %q not found", want)
	}
	if node.NameRange.Start < 0 || node.NameRange.End > len(source) || node.NameRange.Start >= node.NameRange.End {
		t.Fatalf("name range for %q = %+v", want, node.NameRange)
	}
	if got := source[node.NameRange.Start:node.NameRange.End]; got != want {
		t.Fatalf("name range text = %q, want %q", got, want)
	}
}
