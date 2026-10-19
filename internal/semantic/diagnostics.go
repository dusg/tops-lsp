package semantic

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type SemanticDiagnosticSpec struct {
	Code          string
	Severity      SemanticSeverity
	PassSensitive bool
}

var semanticDiagnosticSpecs = map[string]SemanticDiagnosticSpec{
	"tops-semantic-unknown-name":         {Code: "tops-semantic-unknown-name", Severity: SeverityError},
	"tops-semantic-ambiguous-name":       {Code: "tops-semantic-ambiguous-name", Severity: SeverityWarning},
	"tops-semantic-type-mismatch":        {Code: "tops-semantic-type-mismatch", Severity: SeverityError},
	"tops-semantic-invalid-call":         {Code: "tops-semantic-invalid-call", Severity: SeverityError, PassSensitive: true},
	"tops-semantic-host-device-boundary": {Code: "tops-semantic-host-device-boundary", Severity: SeverityError, PassSensitive: true},
	"tops-semantic-address-space":        {Code: "tops-semantic-address-space", Severity: SeverityError, PassSensitive: true},
	"tops-semantic-vector-constraint":    {Code: "tops-semantic-vector-constraint", Severity: SeverityError, PassSensitive: true},
	"tops-semantic-context-missing":      {Code: "tops-semantic-context-missing", Severity: SeverityWarning},
	"tops-semantic-context-degraded":     {Code: "tops-semantic-context-degraded", Severity: SeverityWarning},
	"tops-semantic-incomplete-input":     {Code: "tops-semantic-incomplete-input", Severity: SeverityHint},
}

func SemanticDiagnosticSpecForCode(code string) (SemanticDiagnosticSpec, bool) {
	spec, ok := semanticDiagnosticSpecs[code]
	return spec, ok
}

func NormalizeSemanticDiagnostics(diagnostics []SemanticDiagnostic) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		spec, ok := SemanticDiagnosticSpecForCode(diagnostic.Code)
		if !ok {
			continue
		}
		diagnostic.SemanticKind = strings.TrimPrefix(spec.Code, "tops-semantic-")
		diagnostic.Severity = spec.Severity
		if spec.PassSensitive {
			if diagnostic.PassKind != string(PassKindHost) && diagnostic.PassKind != string(PassKindDevice) {
				continue
			}
			if diagnostic.Severity == SeverityError && diagnostic.Range.Start == diagnostic.Range.End {
				continue
			}
		} else if diagnostic.PassKind != "not-applicable" {
			continue
		}
		result = append(result, diagnostic)
	}
	return result
}

func DeduplicateSemanticDiagnostics(diagnostics []SemanticDiagnostic) []SemanticDiagnostic {
	result := make([]SemanticDiagnostic, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		candidates := append([]string(nil), diagnostic.Candidates...)
		sort.Strings(candidates)
		condition := diagnostic.ConditionalProvenance
		conditionKey := fmt.Sprintf("%s\x00%d\x00%d\x00%d\x00%d\x00%d\x00%s", condition.SourceURI, condition.RegionID, condition.BranchIndex, condition.ParentRegionID, condition.ExpressionRange.Start, condition.ExpressionRange.End, condition.State)
		key := fmt.Sprintf("%s\x00%d\x00%d\x00%s\x00%s\x00%s\x00%s\x00%s", diagnostic.Code, diagnostic.Range.Start, diagnostic.Range.End, diagnostic.Severity, diagnostic.PassKind, diagnostic.SemanticKind, strings.Join(candidates, ","), conditionKey)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, diagnostic)
	}
	sort.SliceStable(result, func(left, right int) bool {
		leftKey := result[left].Code + strconv.Itoa(result[left].Range.Start) + strconv.Itoa(result[left].Range.End) + result[left].PassKind + result[left].SemanticKind
		rightKey := result[right].Code + strconv.Itoa(result[right].Range.Start) + strconv.Itoa(result[right].Range.End) + result[right].PassKind + result[right].SemanticKind
		return leftKey < rightKey
	})
	return result
}
