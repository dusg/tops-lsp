package semantic

import "strings"

type TypeKind string

const (
	TypeKindScalar      TypeKind = "scalar"
	TypeKindPointer     TypeKind = "pointer"
	TypeKindReference   TypeKind = "reference"
	TypeKindNamed       TypeKind = "named"
	TypeKindUserDefined TypeKind = "user-defined"
	TypeKindUnresolved  TypeKind = "unresolved"
)

type ConversionStatus string

const (
	ConversionStatusSupported   ConversionStatus = "supported"
	ConversionStatusUnsupported ConversionStatus = "unsupported"
	ConversionStatusUnresolved  ConversionStatus = "unresolved"
)

type ConversionKind string

const (
	ConversionKindIdentity     ConversionKind = "identity"
	ConversionKindStandard     ConversionKind = "standard"
	ConversionKindPointer      ConversionKind = "pointer"
	ConversionKindReference    ConversionKind = "reference"
	ConversionKindUserDefined  ConversionKind = "user-defined"
	ConversionKindIncompatible ConversionKind = "incompatible"
)

type SemanticType struct {
	Spelling      string
	Kind          TypeKind
	ScalarKind    string
	CVQualifiers  []string
	PointerDepth  int
	Pointee       *SemanticType
	ReferenceKind string
	AddressSpace  string
	UserDefined   bool
	Unresolved    bool
	VectorElement string
	VectorLanes   int
	VectorWidth   int
	Alignment     int
}

var scalarRanks = map[string]int{
	"bool":   0,
	"char":   1,
	"short":  2,
	"int":    3,
	"long":   4,
	"float":  5,
	"double": 6,
}

func ParseSemanticType(spelling string) SemanticType {
	original := strings.TrimSpace(spelling)
	if original == "" {
		return SemanticType{Kind: TypeKindUnresolved, Unresolved: true}
	}
	tokens := strings.Fields(original)
	result := SemanticType{Spelling: original}
	baseTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		switch token {
		case "const", "volatile":
			result.CVQualifiers = append(result.CVQualifiers, token)
		case "__global", "__local", "__shared", "__constant":
			result.AddressSpace = strings.TrimPrefix(token, "__")
		default:
			baseTokens = append(baseTokens, token)
		}
	}
	for index, token := range baseTokens {
		switch token {
		case "__global", "__global__":
			result.AddressSpace = "global"
			baseTokens[index] = ""
			continue
		case "__local", "__local__":
			result.AddressSpace = "local"
			baseTokens[index] = ""
			continue
		case "__shared", "__shared__":
			result.AddressSpace = "shared"
			baseTokens[index] = ""
			continue
		case "__constant", "__constant__":
			result.AddressSpace = "constant"
			baseTokens[index] = ""
			continue
		case "__host__", "__device__", "__kernel__":
			baseTokens[index] = ""
			continue
		}
		if strings.Contains(token, "*") {
			result.PointerDepth += strings.Count(token, "*")
			baseTokens[index] = strings.ReplaceAll(token, "*", "")
		}
		if strings.Contains(token, "&") {
			if strings.Contains(token, "&&") {
				result.ReferenceKind = "rvalue"
			} else {
				result.ReferenceKind = "lvalue"
			}
			baseTokens[index] = strings.ReplaceAll(strings.ReplaceAll(token, "&&", ""), "&", "")
		}
	}
	baseTokens = compactStrings(baseTokens)
	base := strings.Join(baseTokens, " ")
	if result.ReferenceKind != "" {
		result.Kind = TypeKindReference
		pointee := ParseSemanticType(strings.Join(append(append([]string{}, qualifierTokens(result.CVQualifiers)...), base), " "))
		result.Pointee = &pointee
		return result
	}
	if result.PointerDepth > 0 {
		result.Kind = TypeKindPointer
		pointee := ParseSemanticType(base)
		result.Pointee = &pointee
		return result
	}
	if _, ok := scalarRanks[base]; ok {
		result.Kind = TypeKindScalar
		result.ScalarKind = base
		return result
	}
	if strings.HasPrefix(base, "struct ") || strings.HasPrefix(base, "class ") || strings.HasPrefix(base, "enum ") {
		base = strings.TrimSpace(strings.SplitN(base, " ", 2)[1])
	}
	result.Spelling = original
	result.Kind = TypeKindNamed
	result.ScalarKind = base
	result.Unresolved = true
	return result
}

func CheckConversion(source, target SemanticType) CallConversion {
	if source.UserDefined || target.UserDefined || source.Kind == TypeKindUserDefined || target.Kind == TypeKindUserDefined {
		return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusUnsupported, Kind: ConversionKindUserDefined}
	}
	if source.Unresolved || target.Unresolved || source.Kind == TypeKindUnresolved || target.Kind == TypeKindUnresolved {
		return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusUnresolved, Kind: ConversionKindIncompatible}
	}
	if semanticTypeEqual(source, target) {
		return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusSupported, Kind: ConversionKindIdentity}
	}
	if source.Kind == TypeKindScalar && target.Kind == TypeKindScalar {
		if _, sourceKnown := scalarRanks[source.ScalarKind]; sourceKnown {
			if _, targetKnown := scalarRanks[target.ScalarKind]; targetKnown {
				return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusSupported, Kind: ConversionKindStandard}
			}
		}
		return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusUnresolved, Kind: ConversionKindIncompatible}
	}
	if source.Kind == TypeKindPointer && target.Kind == TypeKindPointer && source.PointerDepth == target.PointerDepth && source.Pointee != nil && target.Pointee != nil {
		if source.Pointee.ScalarKind == target.Pointee.ScalarKind && (!containsQualifier(source.Pointee.CVQualifiers, "const") || containsQualifier(target.Pointee.CVQualifiers, "const")) {
			return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusSupported, Kind: ConversionKindPointer}
		}
	}
	if target.Kind == TypeKindReference && target.Pointee != nil {
		conversion := CheckConversion(source, *target.Pointee)
		if conversion.Status == ConversionStatusSupported || conversion.Status == ConversionStatusUnsupported {
			conversion.Kind = ConversionKindReference
			return conversion
		}
	}
	if source.Kind != target.Kind && source.Kind != TypeKindUnresolved && target.Kind != TypeKindUnresolved {
		return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusSupported, Kind: ConversionKindIncompatible}
	}
	return CallConversion{SourceType: source, TargetType: target, Status: ConversionStatusUnresolved, Kind: ConversionKindIncompatible}
}

func semanticTypeEqual(left, right SemanticType) bool {
	if left.Kind != right.Kind || left.ScalarKind != right.ScalarKind || left.PointerDepth != right.PointerDepth || left.ReferenceKind != right.ReferenceKind || left.AddressSpace != right.AddressSpace || left.VectorElement != right.VectorElement || left.VectorLanes != right.VectorLanes || left.VectorWidth != right.VectorWidth || left.Alignment != right.Alignment || !sameStrings(left.CVQualifiers, right.CVQualifiers) {
		return false
	}
	if left.Pointee == nil || right.Pointee == nil {
		return left.Pointee == nil && right.Pointee == nil
	}
	return semanticTypeEqual(*left.Pointee, *right.Pointee)
}

func qualifierTokens(qualifiers []string) []string {
	return append([]string(nil), qualifiers...)
}

func compactStrings(values []string) []string {
	result := values[:0]
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func containsQualifier(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
