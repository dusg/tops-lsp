package parser

import "testing"

func TestParseResultRetainsVersionsAndRecoveryNodes(t *testing.T) {
	result := ParseResult{
		Root: &SyntaxNode{
			Kind: NodeTranslationUnit,
			Children: []*SyntaxNode{{
				Kind:       NodeMissingToken,
				Range:      SourceRange{Start: 12, End: 12},
				Incomplete: true,
			}},
		},
		Status:          ParseStatusRecovered,
		DocumentVersion: 7,
		ContextVersion:  3,
	}

	if result.Root == nil || len(result.Root.Children) != 1 {
		t.Fatalf("result root = %+v", result.Root)
	}
	if result.Root.Children[0].Range.Start != result.Root.Children[0].Range.End {
		t.Fatalf("missing token range = %+v", result.Root.Children[0].Range)
	}
	if !result.Root.Children[0].Incomplete || result.DocumentVersion != 7 || result.ContextVersion != 3 {
		t.Fatalf("result metadata = %+v", result)
	}
}

func TestDiagnosticCodesAndConditionalStatesAreStable(t *testing.T) {
	if DiagnosticMissingToken != "tops-syntax-missing-token" || DiagnosticUnknownCondition != "tops-syntax-unknown-condition" {
		t.Fatalf("diagnostic codes changed: %q %q", DiagnosticMissingToken, DiagnosticUnknownCondition)
	}
	if ConditionalActive == ConditionalInactive || ConditionalUnknown == ConditionalActive {
		t.Fatal("conditional states are not distinct")
	}
}
