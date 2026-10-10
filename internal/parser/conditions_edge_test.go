package parser

import "testing"

func TestParseAppliesLocalDefineAndUndefAcrossConditionalRegions(t *testing.T) {
	source := "#define FLAG 1\n#ifdef FLAG\nint first;\n#endif\n#undef FLAG\n#ifndef FLAG\nint second;\n#endif\n"
	result := Parse(source, ParseContext{})
	if len(result.ConditionalRegions) != 2 {
		t.Fatalf("regions = %+v", result.ConditionalRegions)
	}
	for index, region := range result.ConditionalRegions {
		if region.State != ConditionalActive {
			t.Fatalf("region[%d] = %+v", index, region)
		}
	}
	if hasDiagnostic(result.Diagnostics, DiagnosticUnknownCondition) {
		t.Fatalf("unexpected unknown condition: %+v", result.Diagnostics)
	}
}

func TestParseKeepsMacroContinuationOnOneDirectiveLine(t *testing.T) {
	result := Parse("#define ADD_ONE(value) value \\\n++ 1\n#if ADD_ONE(1) == 2\nint value;\n#endif\n", ParseContext{})
	if len(result.ConditionalRegions) != 1 || result.ConditionalRegions[0].State != ConditionalUnknown {
		t.Fatalf("regions = %+v", result.ConditionalRegions)
	}
	if hasDiagnostic(result.Diagnostics, DiagnosticUnexpectedToken) {
		t.Fatalf("continuation produced token diagnostic: %+v", result.Diagnostics)
	}
}

func TestParseFiltersLexicalDiagnosticsFromInactiveBranch(t *testing.T) {
	result := Parse("#if 0\nint broken = @;\n#else\nint valid;\n#endif\n", ParseContext{})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnexpectedToken && diagnostic.Message == "unrecognized character" {
			t.Fatalf("inactive lexical diagnostic leaked: %+v", result.Diagnostics)
		}
	}
}

func TestParseRetainsUnknownBranchLexicalDiagnostics(t *testing.T) {
	result := Parse("#if MAYBE\nint uncertain = @;\n#endif\n", ParseContext{})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnexpectedToken && diagnostic.Message == "unrecognized character" {
			if diagnostic.ConditionalState != ConditionalUnknown {
				t.Fatalf("diagnostic state = %q, want unknown", diagnostic.ConditionalState)
			}
			return
		}
	}
	t.Fatalf("unknown-branch lexical diagnostic missing: %+v", result.Diagnostics)
}

func TestParseRecordsNestedConditionalParent(t *testing.T) {
	result := Parse("#if OUTER\n#if INNER\nint value;\n#endif\n#endif\n", ParseContext{})
	if len(result.ConditionalRegions) != 2 {
		t.Fatalf("regions = %+v", result.ConditionalRegions)
	}
	if result.ConditionalRegions[1].ParentID != 0 {
		t.Fatalf("nested parent = %d, want 0", result.ConditionalRegions[1].ParentID)
	}
}

func TestParseReportsUnbalancedConditionalExpression(t *testing.T) {
	result := Parse("#if (FLAG\nint value;\n#endif\n", ParseContext{})
	if !hasDiagnostic(result.Diagnostics, DiagnosticInvalidDirective) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestParseReportsUnconsumedPreprocessorTokens(t *testing.T) {
	for _, source := range []string{"##\nint value;\n", "# ##\nint value;\n", "#\nint value;\n"} {
		result := Parse(source, ParseContext{})
		if !hasDiagnostic(result.Diagnostics, DiagnosticInvalidDirective) {
			t.Fatalf("source %q diagnostics = %+v", source, result.Diagnostics)
		}
	}
}

func TestParseMarksUnknownBranchSyntaxDiagnostics(t *testing.T) {
	result := Parse("#if MAYBE\nint value = ;\n#endif\n", ParseContext{})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnexpectedToken {
			if diagnostic.ConditionalState != ConditionalUnknown {
				t.Fatalf("diagnostic state = %q, want unknown", diagnostic.ConditionalState)
			}
			return
		}
	}
	t.Fatalf("unknown-branch syntax diagnostic missing: %+v", result.Diagnostics)
}

func TestParseReportsConditionalBranchOrderAndArguments(t *testing.T) {
	source := "#if FLAG\nint first;\n#else extra\nint second;\n#elif OTHER\nint third;\n#else\nint fourth;\n#endif value\n"
	result := Parse(source, ParseContext{})
	invalidDirectives := 0
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticInvalidDirective {
			invalidDirectives++
		}
	}
	if invalidDirectives != 4 {
		t.Fatalf("invalid directive diagnostics = %+v", result.Diagnostics)
	}
}

func TestParseDoesNotApplyUnknownBranchMacroDefinitions(t *testing.T) {
	source := "#if MAYBE\n#define FROM_UNKNOWN 1\n#endif\n#if FROM_UNKNOWN\nint selected;\n#endif\n"
	result := Parse(source, ParseContext{})
	if len(result.ConditionalRegions) != 2 || result.ConditionalRegions[1].State != ConditionalUnknown {
		t.Fatalf("conditional regions = %+v", result.ConditionalRegions)
	}
	unknownDiagnostics := 0
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnknownCondition {
			unknownDiagnostics++
		}
	}
	if unknownDiagnostics != 2 {
		t.Fatalf("unknown condition diagnostics = %+v", result.Diagnostics)
	}
}
