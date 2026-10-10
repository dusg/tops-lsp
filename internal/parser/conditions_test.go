package parser

import "testing"

func TestParseTracksActiveAndInactiveConditionalBranches(t *testing.T) {
	result := Parse("#if defined(__GCU_ARCH__)\nint active_value = 1;\n#else\nint inactive_value = ;\n#endif\n", ParseContext{
		PredefinedMacros: map[string]MacroValue{"__GCU_ARCH__": {Kind: MacroDefined}},
	})
	if len(result.ConditionalRegions) != 1 {
		t.Fatalf("conditional regions = %+v", result.ConditionalRegions)
	}
	region := result.ConditionalRegions[0]
	if region.State != ConditionalActive || len(region.Branches) != 2 {
		t.Fatalf("region = %+v", region)
	}
	if region.Branches[0].State != ConditionalActive || region.Branches[1].State != ConditionalInactive {
		t.Fatalf("branches = %+v", region.Branches)
	}
	if hasDiagnostic(result.Diagnostics, DiagnosticUnexpectedToken) {
		t.Fatalf("inactive branch diagnostic leaked: %+v", result.Diagnostics)
	}
}

func TestParseReportsUnknownConditionalWithoutChoosingTarget(t *testing.T) {
	result := Parse("#if __GCU_ARCH__ == 400\nint gcu400_value;\n#else\nint other_value;\n#endif\n", ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticUnknownCondition) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	if len(result.ConditionalRegions) != 1 || result.ConditionalRegions[0].State != ConditionalUnknown {
		t.Fatalf("regions = %+v", result.ConditionalRegions)
	}
	if result.ConditionalRegions[0].Branches[1].State != ConditionalUnknown {
		t.Fatalf("else branch = %+v", result.ConditionalRegions[0].Branches[1])
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnknownCondition && (diagnostic.Severity != SeverityInformation || diagnostic.ConditionalState != ConditionalUnknown) {
			t.Fatalf("unknown diagnostic = %+v", diagnostic)
		}
	}
}

func TestParseReportsMissingEndif(t *testing.T) {
	result := Parse("#if defined(FLAG)\nint value;\n", ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticInvalidDirective) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}
