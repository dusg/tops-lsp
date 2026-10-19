package semantic

import "strings"

type VectorConstraintRules struct {
	ElementBits            map[string]int
	AllowedLanes           []int
	AlignmentMaxWidthBytes bool
	Provenance             string
}

func (rules VectorConstraintRules) Available() bool {
	return rules.Provenance != "" && len(rules.ElementBits) > 0 && len(rules.AllowedLanes) > 0
}

func gcu400VectorConstraintRules() VectorConstraintRules {
	return VectorConstraintRules{
		ElementBits:            map[string]int{"char": 8, "short": 16, "int": 32, "long": 64, "float": 32, "double": 64},
		AllowedLanes:           []int{1, 2, 4, 8, 16},
		AlignmentMaxWidthBytes: true,
		Provenance:             "profile:gcu400",
	}
}

func vectorRulesForTarget(arch string) (VectorConstraintRules, bool) {
	if strings.EqualFold(strings.TrimSpace(arch), "gcu400") {
		return gcu400VectorConstraintRules(), true
	}
	return VectorConstraintRules{}, false
}

type SemanticProfile struct {
	ProfileID         string
	TargetArch        string
	Fallback          bool
	TargetProvenance  string
	TargetRules       bool
	AddressSpaceRules bool
	VectorConstraints bool
	VectorRules       VectorConstraintRules
}

func DefaultSemanticProfile(targeted bool) SemanticProfile {
	if targeted {
		rules := gcu400VectorConstraintRules()
		return SemanticProfile{ProfileID: "gcu400-default", TargetArch: "gcu400", Fallback: true, TargetProvenance: "fallback_default", TargetRules: true, AddressSpaceRules: true, VectorConstraints: true, VectorRules: rules}
	}
	return SemanticProfile{ProfileID: "cpp-default", TargetProvenance: "unresolved"}
}
