package parser

import "testing"

func TestParseRecognizesTopsQualifiersAndAttributes(t *testing.T) {
	result := Parse("__thread_dims__(8, 1, 1) __global__ void kernel(__shared__ int* output) { __local__ __valigned__ int buffer[8]; }", ParseContext{})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	function := findNode(result.Root, NodeFunction)
	if function == nil || function.Name != "kernel" {
		t.Fatalf("function = %+v", function)
	}
	if !hasTopsQualifier(function, "__global__") || !hasTopsQualifier(function, "__thread_dims__") {
		t.Fatalf("function qualifiers = %+v", function.Children)
	}
	threadDims := findTopsQualifier(function, "__thread_dims__")
	if threadDims == nil || len(threadDims.Tokens) < 5 {
		t.Fatalf("thread dims tokens = %+v", threadDims)
	}
}

func TestParsePreservesUnknownAttributeAsOpaqueNode(t *testing.T) {
	result := Parse("[[vendor::property(1)]] int value;", ParseContext{LanguageStandard: "c++17"})
	attribute := findNode(result.Root, NodeAttribute)
	opaque := findNode(result.Root, NodeOpaque)
	if attribute == nil || opaque == nil || opaque.Name != "vendor" {
		t.Fatalf("attribute tree = %+v diagnostics=%+v", result.Root, result.Diagnostics)
	}
}

func findTopsQualifier(node *SyntaxNode, spelling string) *SyntaxNode {
	if node == nil {
		return nil
	}
	if node.Kind == NodeTopsQualifier && node.Name == spelling {
		return node
	}
	for _, child := range node.Children {
		if found := findTopsQualifier(child, spelling); found != nil {
			return found
		}
	}
	return nil
}

func hasTopsQualifier(node *SyntaxNode, spelling string) bool {
	if node == nil {
		return false
	}
	if node.Kind == NodeTopsQualifier && node.Name == spelling {
		return true
	}
	for _, child := range node.Children {
		if hasTopsQualifier(child, spelling) {
			return true
		}
	}
	return false
}
