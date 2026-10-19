package semantic

import (
	"fmt"
	"strings"

	"tops-lsp/internal/parser"
)

func CheckAddressSpaceAt(context SemanticContext, source, target SemanticType, rangeValue parser.SourceRange) (SemanticDiagnostic, bool) {
	if !targetRuleAvailable(context, PassKindDevice) || !context.Profile.AddressSpaceRules || source.Kind != TypeKindPointer || target.Kind != TypeKindPointer || source.AddressSpace == target.AddressSpace {
		return SemanticDiagnostic{}, false
	}
	return targetDiagnosticAt("tops-semantic-address-space", "address-space", fmt.Sprintf("address space %q cannot convert to %q", source.AddressSpace, target.AddressSpace), context, rangeValue), true
}

func CheckVectorConstraintAt(context SemanticContext, element string, lanes, width, alignment int, rangeValue parser.SourceRange) (SemanticDiagnostic, bool) {
	rules := context.Profile.VectorRules
	if !targetRuleAvailable(context, PassKindDevice) || !context.Profile.VectorConstraints || !rules.Available() {
		return SemanticDiagnostic{}, false
	}
	elementBits := rules.ElementBits[element]
	validLanes := false
	for _, allowed := range rules.AllowedLanes {
		if lanes == allowed {
			validLanes = true
			break
		}
	}
	invalidAlignment := alignment <= 0
	if rules.AlignmentMaxWidthBytes && width > 0 && alignment > width/8 {
		invalidAlignment = true
	}
	if elementBits == 0 || !validLanes || width != lanes*elementBits || invalidAlignment {
		return targetDiagnosticAt("tops-semantic-vector-constraint", "vector-constraint", "vector element, lane, width, or alignment constraint is invalid", context, rangeValue), true
	}
	return SemanticDiagnostic{}, false
}

func targetDiagnosticAt(code, kind, message string, context SemanticContext, rangeValue parser.SourceRange) SemanticDiagnostic {
	return SemanticDiagnostic{Code: code, Severity: SeverityError, SemanticKind: kind, Message: message, Range: rangeValue, PassKind: string(PassKindDevice), ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: context.DocumentVersion, ContextVersion: context.SemanticContextVersion, SourceVersion: context.SourceVersionFingerprint}
}

func targetRuleAvailable(context SemanticContext, pass PassKind) bool {
	view, ok := context.PassViews[pass]
	return ok && view.Availability == PassAvailabilityAvailable && passViewHasProvenance(context, view) && view.Status == ContextStatusResolved && context.Status == ContextStatusResolved && context.Profile.TargetRules
}

func vectorTypeValues(value SemanticType) (string, int, int, int, bool) {
	if value.VectorElement != "" && value.VectorLanes > 0 {
		return value.VectorElement, value.VectorLanes, value.VectorWidth, value.Alignment, true
	}
	if !strings.Contains(value.Spelling, "vector") {
		return "", 0, 0, 0, false
	}
	return value.VectorElement, value.VectorLanes, value.VectorWidth, value.Alignment, value.VectorLanes > 0
}
