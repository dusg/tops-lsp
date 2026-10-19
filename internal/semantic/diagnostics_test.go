package semantic

import (
	"testing"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestSemanticDiagnosticRegistryAndPassAwareDeduplication(t *testing.T) {
	for _, code := range []string{
		"tops-semantic-unknown-name",
		"tops-semantic-ambiguous-name",
		"tops-semantic-type-mismatch",
		"tops-semantic-invalid-call",
		"tops-semantic-host-device-boundary",
		"tops-semantic-address-space",
		"tops-semantic-vector-constraint",
		"tops-semantic-context-missing",
		"tops-semantic-context-degraded",
		"tops-semantic-incomplete-input",
	} {
		if _, ok := SemanticDiagnosticSpecForCode(code); !ok {
			t.Fatalf("missing diagnostic code %q", code)
		}
	}
	rangeValue := parser.SourceRange{Start: 2, End: 4}
	duplicates := []SemanticDiagnostic{
		{Code: "tops-semantic-host-device-boundary", Range: rangeValue, PassKind: "host"},
		{Code: "tops-semantic-host-device-boundary", Range: rangeValue, PassKind: "host"},
		{Code: "tops-semantic-host-device-boundary", Range: rangeValue, PassKind: "device"},
	}
	unique := DeduplicateSemanticDiagnostics(duplicates)
	if len(unique) != 2 {
		t.Fatalf("deduplicated diagnostics = %+v", unique)
	}
	conditioned := DeduplicateSemanticDiagnostics([]SemanticDiagnostic{
		{Code: "tops-semantic-context-degraded", Range: rangeValue, PassKind: "not-applicable", ConditionalProvenance: index.ConditionalProvenance{State: parser.ConditionalActive}},
		{Code: "tops-semantic-context-degraded", Range: rangeValue, PassKind: "not-applicable", ConditionalProvenance: index.ConditionalProvenance{State: parser.ConditionalUnknown}},
	})
	if len(conditioned) != 2 {
		t.Fatalf("conditional diagnostics = %+v", conditioned)
	}
}

func TestNormalizeSemanticDiagnosticsAppliesStableRegistryAndRangeGate(t *testing.T) {
	diagnostics := NormalizeSemanticDiagnostics([]SemanticDiagnostic{
		{Code: "unknown-code", Range: parser.SourceRange{Start: 1, End: 2}},
		{Code: "tops-semantic-address-space", Severity: SeverityError, PassKind: "device"},
		{Code: "tops-semantic-host-device-boundary", Severity: SeverityError, PassKind: "host", Range: parser.SourceRange{Start: 1, End: 2}},
	})
	if len(diagnostics) != 1 || diagnostics[0].Code != "tops-semantic-host-device-boundary" || diagnostics[0].SemanticKind != "host-device-boundary" || diagnostics[0].Severity != SeverityError {
		t.Fatalf("normalized diagnostics = %+v", diagnostics)
	}
}
