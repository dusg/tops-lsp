package server

import (
	"context"
	"errors"

	"tops-lsp/internal/document"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
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
	mapper := position.New(state.Text)
	diagnostics := make([]protocol.Diagnostic, 0, len(result.Diagnostics))
	for _, sourceDiagnostic := range result.Diagnostics {
		rangeValue, err := mapper.Range(sourceDiagnostic.Range.Start, sourceDiagnostic.Range.End)
		if err != nil {
			return err
		}
		diagnostic := protocol.Diagnostic{
			Range:    protocol.Range{Start: protocol.Position{Line: rangeValue.Start.Line, Character: rangeValue.Start.Character}, End: protocol.Position{Line: rangeValue.End.Line, Character: rangeValue.End.Character}},
			Severity: protocolSeverity(sourceDiagnostic.Severity),
			Code:     sourceDiagnostic.Code,
			Source:   "tops-lsp",
			Message:  sourceDiagnostic.Message,
			Data: &protocol.DiagnosticData{
				ConditionalState: string(sourceDiagnostic.ConditionalState),
				Recoverable:      sourceDiagnostic.Recoverable,
				Incomplete:       sourceDiagnostic.Incomplete,
				ContextVersion:   sourceDiagnostic.ContextVersion,
			},
		}
		for _, related := range sourceDiagnostic.Related {
			relatedRange, relatedErr := mapper.Range(related.Range.Start, related.Range.End)
			if relatedErr != nil {
				return relatedErr
			}
			uri := related.URI
			if uri == "" {
				uri = state.URI
			}
			diagnostic.RelatedInformation = append(diagnostic.RelatedInformation, protocol.DiagnosticRelatedInformation{
				Location: protocol.Location{URI: uri, Range: protocol.Range{Start: protocol.Position{Line: relatedRange.Start.Line, Character: relatedRange.Start.Character}, End: protocol.Position{Line: relatedRange.End.Line, Character: relatedRange.End.Character}}},
				Message:  related.Message,
			})
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	version := state.Version
	contextVersion := result.ContextVersion
	return publisher.Publish(ctx, protocol.PublishDiagnosticsParams{URI: state.URI, Version: &version, ContextVersion: &contextVersion, Diagnostics: diagnostics})
}

func (server *Server) publishParserDiagnosticsIfCurrent(ctx context.Context, publisher diagnosticPublisher, snapshot analysisSnapshot, result parser.ParseResult) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	if !server.analysisCurrentSnapshotLocked(snapshot) {
		return nil
	}
	return server.publishParserDiagnostics(ctx, publisher, snapshot.state, result)
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
