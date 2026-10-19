package semantic

import (
	"testing"

	"tops-lsp/internal/parser"
)

func TestTargetRulesCheckAddressSpaceAndVectorConstraints(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	source := ParseSemanticType("__global int *")
	target := ParseSemanticType("__local int *")
	if diagnostic, ok := CheckAddressSpaceAt(context, source, target, parser.SourceRange{}); !ok || diagnostic.Code != "tops-semantic-address-space" || diagnostic.PassKind != string(PassKindDevice) {
		t.Fatalf("address-space diagnostic = %+v, ok = %v", diagnostic, ok)
	}
	if _, ok := CheckVectorConstraintAt(context, "int", 3, 96, 4, parser.SourceRange{}); !ok {
		t.Fatal("invalid vector constraint was accepted")
	}
	if _, ok := CheckVectorConstraintAt(context, "int", 4, 128, 4, parser.SourceRange{}); ok {
		t.Fatal("valid vector constraint was rejected")
	}
}

func TestVectorConstraintsUseProfileRulesAndProvenance(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	context.Profile.VectorRules = VectorConstraintRules{
		ElementBits:            map[string]int{"int": 32},
		AllowedLanes:           []int{1, 2, 4},
		AlignmentMaxWidthBytes: true,
		Provenance:             "profile:gcu400",
	}
	if _, ok := CheckVectorConstraintAt(context, "int", 8, 256, 4, parser.SourceRange{}); !ok {
		t.Fatal("profile lane constraints were ignored")
	}
	if _, ok := CheckVectorConstraintAt(context, "int", 4, 128, 4, parser.SourceRange{}); ok {
		t.Fatal("profile vector constraint rejected valid vector")
	}
	context.Profile.VectorRules.Provenance = ""
	if _, ok := CheckVectorConstraintAt(context, "int", 8, 256, 4, parser.SourceRange{}); ok {
		t.Fatal("vector rule without provenance produced a deterministic error")
	}
}

func TestTargetRulesSuppressWithoutPassProvenance(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	context.Command.Source = "compile_commands"
	context.PassViews[PassKindDevice] = PassView{PassKind: PassKindDevice, Availability: PassAvailabilityAvailable, Status: ContextStatusResolved}
	if _, ok := CheckAddressSpaceAt(context, ParseSemanticType("__global int *"), ParseSemanticType("__local int *"), parser.SourceRange{}); ok {
		t.Fatal("address-space rule ran without pass provenance")
	}
	if _, ok := CheckVectorConstraintAt(context, "int", 3, 96, 4, parser.SourceRange{}); ok {
		t.Fatal("vector rule ran without pass provenance")
	}
}

func TestTargetRulesSuppressErrorsOutsideResolvedTargetProfile(t *testing.T) {
	context := semanticTestContext(ContextStatusPartial)
	if _, ok := CheckVectorConstraintAt(context, "int", 3, 96, 4, parser.SourceRange{}); ok {
		t.Fatal("partial context produced a vector error")
	}
	context = semanticTestContext(ContextStatusResolved)
	context.Profile.TargetRules = false
	if _, ok := CheckAddressSpaceAt(context, ParseSemanticType("__global int *"), ParseSemanticType("__local int *"), parser.SourceRange{}); ok {
		t.Fatal("target-neutral profile produced an address-space error")
	}
}
