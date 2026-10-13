package index

import (
	"testing"

	"tops-lsp/internal/parser"
)

func TestBuildKeepsCompleteSymbolsInPartialInput(t *testing.T) {
	source := "int ready() { return 1; } int incomplete(int value"
	context := resolvedParseContext("file:///workspace/partial.tops")
	result := parser.Parse(source, context)
	snapshot := Build(source, result, context)
	if snapshot.IndexStatus != IndexStatusPartial {
		t.Fatalf("index status = %q", snapshot.IndexStatus)
	}
	if symbolByName(snapshot, "ready") == nil {
		t.Fatalf("complete symbol missing: %+v", snapshot.Symbols)
	}
	incomplete := symbolByName(snapshot, "incomplete")
	if incomplete == nil || incomplete.Status != SymbolStatusIncomplete {
		t.Fatalf("incomplete symbol = %+v", incomplete)
	}
}

func TestBuildReplacesRemovedDeclarationsWithoutOldSymbols(t *testing.T) {
	context := resolvedParseContext("file:///workspace/versioned.tops")
	first := "int old_name() { return 1; }"
	firstResult := parser.Parse(first, context)
	firstSnapshot := Build(first, firstResult, context)
	if symbolByName(firstSnapshot, "old_name") == nil {
		t.Fatal("old symbol missing from first snapshot")
	}
	second := "int new_name() { return 2; }"
	context.DocumentVersion = 2
	secondResult := parser.Parse(second, context)
	secondSnapshot := Build(second, secondResult, context)
	if symbolByName(secondSnapshot, "old_name") != nil || symbolByName(secondSnapshot, "new_name") == nil {
		t.Fatalf("second symbols = %+v", secondSnapshot.Symbols)
	}
}

func TestBuildIgnoresCommentsAndStringsWhenCollectingReferences(t *testing.T) {
	source := "int target(); // target\nconst char* text = \"target\"; int use() { return target(); }"
	context := resolvedParseContext("file:///workspace/filter.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	target := symbolByName(snapshot, "target")
	if target == nil || len(target.References) != 1 {
		t.Fatalf("target references = %+v", target)
	}
}

func TestBuildQueryAtRecoveredErrorNodeDoesNotPanic(t *testing.T) {
	source := "int broken( { return 1; }\nint valid();"
	context := resolvedParseContext("file:///workspace/error-node.tops")
	snapshot := Build(source, parser.Parse(source, context), context)
	_ = snapshot.DefinitionAt(sourceOffset(source, "broken", 0))
	if symbolByName(snapshot, "valid") == nil {
		t.Fatalf("valid symbol missing after recovery: %+v", snapshot.Symbols)
	}
}
