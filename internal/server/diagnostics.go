package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
	"tops-lsp/internal/transport"
)

type diagnosticPublisher interface {
	Publish(context.Context, protocol.PublishDiagnosticsParams) error
}

type frameDiagnosticPublisher struct {
	writer *transport.Writer
}

func (publisher *frameDiagnosticPublisher) Publish(_ context.Context, params protocol.PublishDiagnosticsParams) error {
	if publisher == nil || publisher.writer == nil {
		return errors.New("diagnostic publisher has no writer")
	}
	body, err := protocol.MarshalNotification("textDocument/publishDiagnostics", params)
	if err != nil {
		return err
	}
	return publisher.writer.WriteFrame(body)
}

func (server *Server) publishParserDiagnostics(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState, result parser.ParseResult) error {
	if publisher == nil {
		return nil
	}
	diagnostics, err := parserDiagnosticsForState(state, result)
	if err != nil {
		return err
	}
	version := state.Version
	contextVersion := result.ContextVersion
	return publisher.Publish(ctx, protocol.PublishDiagnosticsParams{URI: state.URI, Version: &version, ContextVersion: &contextVersion, Diagnostics: diagnostics})
}

func parserDiagnosticsForState(state document.DocumentState, result parser.ParseResult) ([]protocol.Diagnostic, error) {
	mapper := position.New(state.Text)
	diagnostics := make([]protocol.Diagnostic, 0, len(result.Diagnostics))
	for _, sourceDiagnostic := range result.Diagnostics {
		rangeValue, err := mapper.Range(sourceDiagnostic.Range.Start, sourceDiagnostic.Range.End)
		if err != nil {
			return nil, err
		}
		diagnostic := protocol.Diagnostic{
			Range:    protocolRange(rangeValue),
			Severity: protocolSeverity(sourceDiagnostic.Severity),
			Code:     sourceDiagnostic.Code,
			Source:   "tops-lsp",
			Message:  sourceDiagnostic.Message,
			Data: &protocol.DiagnosticData{
				ConditionalState: string(sourceDiagnostic.ConditionalState),
				Recoverable:      sourceDiagnostic.Recoverable,
				Incomplete:       sourceDiagnostic.Incomplete,
				DocumentVersion:  sourceDiagnostic.DocumentVersion,
				ContextVersion:   sourceDiagnostic.ContextVersion,
			},
		}
		for _, related := range sourceDiagnostic.Related {
			relatedRange, relatedErr := mapper.Range(related.Range.Start, related.Range.End)
			if relatedErr != nil {
				return nil, relatedErr
			}
			uri := related.URI
			if uri == "" {
				uri = state.URI
			}
			diagnostic.RelatedInformation = append(diagnostic.RelatedInformation, protocol.DiagnosticRelatedInformation{
				Location: protocol.Location{URI: uri, Range: protocolRange(relatedRange)},
				Message:  related.Message,
			})
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}

func protocolSeverity(severity parser.DiagnosticSeverity) protocol.DiagnosticSeverity {
	switch severity {
	case parser.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case parser.SeverityInformation:
		return protocol.DiagnosticSeverityInfo
	case parser.SeverityHint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}

// dedupeAndSortDiagnostics applies the published diagnostic contract: identical
// code, range, severity and message collapse to one entry, and ordering is
// stable by position, severity, code and message.
func dedupeAndSortDiagnostics(diagnostics []protocol.Diagnostic) []protocol.Diagnostic {
	if len(diagnostics) == 0 {
		return diagnostics
	}
	result := make([]protocol.Diagnostic, 0, len(diagnostics))
	seen := make(map[string]struct{}, len(diagnostics))
	for _, diagnostic := range diagnostics {
		key := fmt.Sprintf("%s\x00%d\x00%d\x00%d\x00%d\x00%d\x00%s",
			diagnostic.Code,
			diagnostic.Range.Start.Line, diagnostic.Range.Start.Character,
			diagnostic.Range.End.Line, diagnostic.Range.End.Character,
			diagnostic.Severity, diagnostic.Message)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, diagnostic)
	}
	sort.SliceStable(result, func(left, right int) bool {
		return diagnosticSortLess(result[left], result[right])
	})
	return result
}

func diagnosticSortLess(left, right protocol.Diagnostic) bool {
	if left.Range.Start.Line != right.Range.Start.Line {
		return left.Range.Start.Line < right.Range.Start.Line
	}
	if left.Range.Start.Character != right.Range.Start.Character {
		return left.Range.Start.Character < right.Range.Start.Character
	}
	if left.Range.End.Line != right.Range.End.Line {
		return left.Range.End.Line < right.Range.End.Line
	}
	if left.Range.End.Character != right.Range.End.Character {
		return left.Range.End.Character < right.Range.End.Character
	}
	if left.Severity != right.Severity {
		return left.Severity < right.Severity
	}
	if left.Code != right.Code {
		return left.Code < right.Code
	}
	return left.Message < right.Message
}

func semanticDiagnosticsForState(state document.DocumentState, snapshot semantic.SemanticSnapshot) ([]protocol.Diagnostic, error) {
	if err := snapshot.ValidateSourceVersion(); err != nil {
		return nil, nil
	}
	mapper := position.New(state.Text)
	diagnostics := make([]protocol.Diagnostic, 0, len(snapshot.Diagnostics))
	for _, sourceDiagnostic := range snapshot.Diagnostics {
		if sourceDiagnostic.ProfileID == "" || sourceDiagnostic.SourceVersion == "" {
			continue
		}
		spec, ok := semantic.SemanticDiagnosticSpecForCode(sourceDiagnostic.Code)
		if !ok || sourceDiagnostic.SemanticKind != strings.TrimPrefix(sourceDiagnostic.Code, "tops-semantic-") {
			continue
		}
		if sourceDiagnostic.Severity == semantic.SeverityError {
			if string(sourceDiagnostic.ContextStatus) != string(semantic.ContextStatusResolved) || sourceDiagnostic.Range.Start == sourceDiagnostic.Range.End {
				continue
			}
		}
		if spec.PassSensitive {
			if (sourceDiagnostic.PassKind != "host" && sourceDiagnostic.PassKind != "device") || sourceDiagnostic.Range.Start == sourceDiagnostic.Range.End {
				continue
			}
		} else if sourceDiagnostic.PassKind != "not-applicable" {
			continue
		}
		rangeValue, err := mapper.Range(sourceDiagnostic.Range.Start, sourceDiagnostic.Range.End)
		if err != nil {
			return nil, err
		}
		diagnostic := protocol.Diagnostic{
			Range:    protocolRange(rangeValue),
			Severity: semanticProtocolSeverity(sourceDiagnostic.Severity),
			Code:     sourceDiagnostic.Code,
			Source:   "tops-lsp",
			Message:  sourceDiagnostic.Message,
			Data: &protocol.DiagnosticData{
				DocumentVersion: sourceDiagnostic.DocumentVersion,
				ContextVersion:  sourceDiagnostic.ContextVersion,
				SourceVersion:   sourceDiagnostic.SourceVersion,
				ContextStatus:   string(sourceDiagnostic.ContextStatus),
				ProfileID:       sourceDiagnostic.ProfileID,
				PassKind:        sourceDiagnostic.PassKind,
				SemanticKind:    sourceDiagnostic.SemanticKind,
				Recoverable:     sourceDiagnostic.Recoverable,
				Incomplete:      sourceDiagnostic.Incomplete,
			},
		}
		if sourceDiagnostic.ConditionalProvenance.State != "" {
			conditional, conditionalErr := conditionalProvenance(sourceDiagnostic.ConditionalProvenance, state.URI, func(string) (string, bool) { return state.Text, true }, false)
			if conditionalErr != nil {
				return nil, conditionalErr
			}
			diagnostic.Data.ConditionalProvenance = conditional
		}
		for _, related := range sourceDiagnostic.Related {
			if related.URI != "" && related.URI != state.URI {
				continue
			}
			relatedRange, relatedErr := mapper.Range(related.Range.Start, related.Range.End)
			if relatedErr != nil {
				return nil, relatedErr
			}
			diagnostic.RelatedInformation = append(diagnostic.RelatedInformation, protocol.DiagnosticRelatedInformation{Location: protocol.Location{URI: state.URI, Range: protocolRange(relatedRange)}, Message: related.Message})
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, nil
}

// conditionalProvenance converts an index conditional provenance into its
// protocol form. textForURI supplies the source text used to encode the
// expression range; includeTokens controls whether condition token spellings
// are copied into the result.
func conditionalProvenance(condition index.ConditionalProvenance, defaultSourceURI string, textForURI func(string) (string, bool), includeTokens bool) (*protocol.ConditionalProvenance, error) {
	if condition.State == "" {
		return nil, nil
	}
	sourceURI := condition.SourceURI
	if sourceURI == "" {
		sourceURI = defaultSourceURI
	}
	result := &protocol.ConditionalProvenance{
		SourceURI:      sourceURI,
		RegionID:       condition.RegionID,
		BranchIndex:    condition.BranchIndex,
		State:          string(condition.State),
		ParentRegionID: condition.ParentRegionID,
	}
	if includeTokens && len(condition.ConditionTokens) > 0 {
		result.ConditionTokens = make([]string, 0, len(condition.ConditionTokens))
		for _, token := range condition.ConditionTokens {
			result.ConditionTokens = append(result.ConditionTokens, token.Spelling)
		}
	}
	if condition.ExpressionRange.Start != condition.ExpressionRange.End {
		text, ok := textForURI(sourceURI)
		if !ok {
			return nil, position.ErrInvalidRange
		}
		rangeValue, err := position.New(text).Range(condition.ExpressionRange.Start, condition.ExpressionRange.End)
		if err != nil {
			return nil, err
		}
		expressionRange := protocolRange(rangeValue)
		result.ExpressionRange = &expressionRange
	}
	for _, parent := range condition.ParentConditions {
		parentValue, err := conditionalProvenance(parent, defaultSourceURI, textForURI, includeTokens)
		if err != nil {
			return nil, err
		}
		if parentValue != nil {
			result.ParentConditions = append(result.ParentConditions, *parentValue)
		}
	}
	return result, nil
}

func semanticProtocolSeverity(severity semantic.SemanticSeverity) protocol.DiagnosticSeverity {
	switch severity {
	case semantic.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case semantic.SeverityInformation:
		return protocol.DiagnosticSeverityInfo
	case semantic.SeverityHint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityError
	}
}
