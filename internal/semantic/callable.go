package semantic

import (
	"strings"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func BuildCallableFact(symbol index.SymbolRecord, context SemanticContext) CallableFact {
	fact := CallableFact{
		SymbolID:        symbol.SymbolID,
		Name:            symbol.Name,
		QualifiedName:   symbol.QualifiedName,
		ScopeID:         symbol.ScopeID,
		ReturnType:      ParseSemanticType(symbol.ReturnTypeSummary),
		DeclarationOnly: symbol.Status == index.SymbolStatusDeclarationOnly,
		Range:           symbol.NameRange,
		Provenance:      map[string]string{"source": symbol.SourceVersion.URI},
	}
	for index, parameter := range splitParameters(symbol.ParameterSummary) {
		fact.Parameters = append(fact.Parameters, CallableParameter{Name: "parameter" + string(rune('0'+index)), Type: ParseSemanticType(parameter)})
	}
	fact.ExecutionKind, fact.DefaultExecutionSpace = callableExecution(symbol, context)
	switch fact.ExecutionKind {
	case "device", "kernel-entry":
		fact.PassKinds = []PassKind{PassKindDevice}
	case "host-device":
		fact.PassKinds = []PassKind{PassKindHost, PassKindDevice}
	default:
		fact.PassKinds = []PassKind{PassKindHost}
	}
	return fact
}

func splitParameters(summary string) []string {
	if strings.TrimSpace(summary) == "" || strings.TrimSpace(summary) == "void" {
		return nil
	}
	parts := strings.Split(summary, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) > 1 && isIdentifier(fields[len(fields)-1]) {
			part = strings.Join(fields[:len(fields)-1], " ")
		}
		result = append(result, part)
	}
	return result
}

func callableExecution(symbol index.SymbolRecord, context SemanticContext) (string, bool) {
	hasHost, hasDevice, hasKernel := false, false, false
	for _, qualifier := range symbol.ExecutionQualifiers {
		switch qualifier {
		case "__host__":
			hasHost = true
		case "__device__":
			hasDevice = true
		case "__global__":
			hasKernel = true
		}
	}
	if hasKernel {
		return "kernel-entry", false
	}
	if hasHost && hasDevice {
		return "host-device", false
	}
	if hasDevice {
		return "device", false
	}
	if hasHost {
		return "host", false
	}
	if context.Status == ContextStatusResolved && context.Command.Language == "TOPS" && (context.ParseStatus == "" || context.ParseStatus == parser.ParseStatusComplete) {
		return "host", true
	}
	return "host", false
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || character == '_' || (index > 0 && character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return true
}
