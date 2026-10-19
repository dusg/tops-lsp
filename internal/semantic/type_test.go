package semantic

import "testing"

func TestParseSemanticTypeSupportsScalarPointerAndReference(t *testing.T) {
	integer := ParseSemanticType("const int")
	if integer.Kind != TypeKindScalar || integer.ScalarKind != "int" || len(integer.CVQualifiers) != 1 {
		t.Fatalf("integer = %+v", integer)
	}
	pointer := ParseSemanticType("const int *")
	if pointer.Kind != TypeKindPointer || pointer.PointerDepth != 1 || pointer.Pointee == nil || pointer.Pointee.ScalarKind != "int" {
		t.Fatalf("pointer = %+v", pointer)
	}
	reference := ParseSemanticType("int &")
	if reference.Kind != TypeKindReference || reference.ReferenceKind != "lvalue" || reference.Pointee == nil {
		t.Fatalf("reference = %+v", reference)
	}
}

func TestCheckConversionSupportsStandardAndUnresolvedKinds(t *testing.T) {
	standard := CheckConversion(ParseSemanticType("int"), ParseSemanticType("long"))
	if standard.Status != ConversionStatusSupported || standard.Kind != ConversionKindStandard {
		t.Fatalf("standard conversion = %+v", standard)
	}
	incompatible := CheckConversion(ParseSemanticType("int"), ParseSemanticType("Widget"))
	if incompatible.Status != ConversionStatusUnresolved {
		t.Fatalf("unresolved conversion = %+v", incompatible)
	}
	unsupported := CheckConversion(SemanticType{Spelling: "Meters", Kind: TypeKindUserDefined, UserDefined: true}, SemanticType{Spelling: "Seconds", Kind: TypeKindUserDefined, UserDefined: true})
	if unsupported.Status != ConversionStatusUnsupported {
		t.Fatalf("user-defined conversion = %+v", unsupported)
	}
}

func TestCheckConversionSupportsPointerAndReferenceCompatibility(t *testing.T) {
	source := ParseSemanticType("int *")
	target := ParseSemanticType("const int *")
	conversion := CheckConversion(source, target)
	if conversion.Status != ConversionStatusSupported || conversion.Kind != ConversionKindPointer {
		t.Fatalf("pointer conversion = %+v", conversion)
	}
	reference := CheckConversion(ParseSemanticType("int"), ParseSemanticType("const int &"))
	if reference.Status != ConversionStatusSupported || reference.Kind != ConversionKindReference {
		t.Fatalf("reference conversion = %+v", reference)
	}
}
