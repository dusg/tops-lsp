package parser

import "testing"

func TestParseReportsEmptyTopsAttributeAndMalformedLaunch(t *testing.T) {
	result := Parse("__maxnreg__() __global__ void bad(int* output) { output<<<>>>(); }", ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticInvalidTopsAttribute) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if findNode(result.Root, NodeTopsLaunch) == nil {
		t.Fatalf("launch recovery missing: %+v", result.Root)
	}
}

func hasDiagnostic(diagnostics []ParserDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
