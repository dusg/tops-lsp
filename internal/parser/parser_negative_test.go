package parser

import "testing"

func TestParseClassifiesInvalidTopLevelToken(t *testing.T) {
	result := Parse("int value = ; int valid = 1;", ParseContext{})
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected a syntax diagnostic")
	}
	if !containsFunction(result.Root, "never") && result.Root == nil {
		t.Fatal("parser returned no root")
	}
}

func TestParseMarksCxx20ConstructOutsideCxx17Baseline(t *testing.T) {
	result := Parse("concept Number = true;", ParseContext{LanguageStandard: "c++17"})
	if !hasDiagnostic(result.Diagnostics, DiagnosticUnsupported) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticUnsupported && diagnostic.Severity != SeverityInformation {
			t.Fatalf("unsupported severity = %v", diagnostic.Severity)
		}
	}
}

func TestParseMarksIfConstexprOutsideCxx17Baseline(t *testing.T) {
	for _, standard := range []string{"c++11", "c++14"} {
		result := Parse("if constexpr (true) { return 1; }", ParseContext{LanguageStandard: standard})
		if !hasDiagnostic(result.Diagnostics, DiagnosticUnsupported) {
			t.Fatalf("standard=%s diagnostics = %+v", standard, result.Diagnostics)
		}
	}
}

func TestParseMarksStructuredBindingOutsideCxx17Baseline(t *testing.T) {
	result := Parse("auto [left, right] = pair;", ParseContext{LanguageStandard: "c++14"})
	if !hasDiagnostic(result.Diagnostics, DiagnosticUnsupported) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}

func TestParseMarksGenericLambdaOutsideCxx14Baseline(t *testing.T) {
	result := Parse("auto apply = [](auto value) { return value; };", ParseContext{LanguageStandard: "c++11"})
	if !hasDiagnostic(result.Diagnostics, DiagnosticUnsupported) {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
}
