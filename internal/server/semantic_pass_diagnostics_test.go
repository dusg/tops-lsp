package server

import (
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
)

func TestSemanticPassDiagnosticsKeepHostAndDeviceSeparateOnWire(t *testing.T) {
	state := document.DocumentState{URI: "file:///workspace/pass.tops", Version: 4, Text: "int value;\n"}
	context := semanticTestContextForServer()
	snapshot := semantic.SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: context.SourceVersionFingerprint,
		DocumentVersion:          4,
		SemanticContextVersion:   2,
		SourceVersions:           context.SourceVersions,
		Status:                   semantic.SnapshotStatusComplete,
		Diagnostics: []semantic.SemanticDiagnostic{
			{Code: "tops-semantic-host-device-boundary", SemanticKind: "host-device-boundary", Severity: semantic.SeverityError, Message: "host boundary", Range: parser.SourceRange{Start: 0, End: 3}, PassKind: "host", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: 4, ContextVersion: 2, SourceVersion: context.SourceVersionFingerprint},
			{Code: "tops-semantic-host-device-boundary", SemanticKind: "host-device-boundary", Severity: semantic.SeverityError, Message: "device boundary", Range: parser.SourceRange{Start: 0, End: 3}, PassKind: "device", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: 4, ContextVersion: 2, SourceVersion: context.SourceVersionFingerprint},
		},
	}
	diagnostics, err := semanticDiagnosticsForState(state, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 || diagnostics[0].Data == nil || diagnostics[1].Data == nil || diagnostics[0].Data.PassKind == diagnostics[1].Data.PassKind {
		t.Fatalf("wire diagnostics = %+v", diagnostics)
	}
}

func TestSemanticDiagnosticsMapUnicodeCRLFEOFAndZeroWidthRanges(t *testing.T) {
	context := semanticTestContextForServer()
	text := "前\r\nvalue"
	rootVersion := context.SourceVersionFingerprint
	state := document.DocumentState{URI: context.DocumentURI, Version: 7, Text: text}
	snapshot := semantic.SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: rootVersion,
		DocumentVersion:          7,
		SemanticContextVersion:   2,
		SourceVersions:           context.SourceVersions,
		Status:                   semantic.SnapshotStatusComplete,
		Diagnostics: []semantic.SemanticDiagnostic{
			{Code: "tops-semantic-host-device-boundary", SemanticKind: "host-device-boundary", Severity: semantic.SeverityError, Message: "unicode range", Range: parser.SourceRange{Start: 5, End: 10}, PassKind: "host", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: 7, ContextVersion: 2, SourceVersion: rootVersion},
			{Code: "tops-semantic-incomplete-input", SemanticKind: "incomplete-input", Severity: semantic.SeverityHint, Message: "at eof", Range: parser.SourceRange{Start: len(text), End: len(text)}, PassKind: "not-applicable", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: 7, ContextVersion: 2, SourceVersion: rootVersion},
			{Code: "tops-semantic-host-device-boundary", SemanticKind: "host-device-boundary", Severity: semantic.SeverityError, Message: "zero width", Range: parser.SourceRange{Start: len(text), End: len(text)}, PassKind: "device", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, DocumentVersion: 7, ContextVersion: 2, SourceVersion: rootVersion},
		},
	}
	diagnostics, err := semanticDiagnosticsForState(state, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 {
		t.Fatalf("mapped diagnostics = %+v", diagnostics)
	}
	boundary := diagnostics[0]
	if boundary.Range.Start.Line != 1 || boundary.Range.Start.Character != 0 || boundary.Range.End.Line != 1 || boundary.Range.End.Character != 5 {
		t.Fatalf("unicode CRLF range = %+v", boundary.Range)
	}
	if boundary.Data == nil || boundary.Data.DocumentVersion != 7 || boundary.Data.ContextVersion != 2 || boundary.Data.SourceVersion != rootVersion || boundary.Data.ContextStatus != string(context.Status) || boundary.Data.ProfileID == "" || boundary.Data.PassKind != "host" || boundary.Data.SemanticKind != "host-device-boundary" {
		t.Fatalf("wire data = %+v", boundary.Data)
	}
	if diagnostics[1].Range.Start.Line != 1 || diagnostics[1].Range.Start.Character != 5 || diagnostics[1].Range.End != diagnostics[1].Range.Start {
		t.Fatalf("EOF range = %+v", diagnostics[1].Range)
	}
}

func TestSemanticSeverityMapsToLSPSeverity(t *testing.T) {
	for _, test := range []struct {
		severity semantic.SemanticSeverity
		want     protocol.DiagnosticSeverity
	}{
		{semantic.SeverityError, protocol.DiagnosticSeverityError},
		{semantic.SeverityWarning, protocol.DiagnosticSeverityWarning},
		{semantic.SeverityInformation, protocol.DiagnosticSeverityInfo},
		{semantic.SeverityHint, protocol.DiagnosticSeverityHint},
	} {
		got := semanticProtocolSeverity(test.severity)
		if got != test.want {
			t.Fatalf("severity %q = %d, want %d", test.severity, got, test.want)
		}
	}
}
func TestSemanticPassErrorWithoutRangeIsSuppressed(t *testing.T) {
	context := semanticTestContextForServer()
	snapshot := semantic.SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: context.SourceVersionFingerprint,
		SourceVersions:           context.SourceVersions,
		Diagnostics: []semantic.SemanticDiagnostic{{
			Code: "tops-semantic-address-space", Severity: semantic.SeverityError, SemanticKind: "address-space", PassKind: "device", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, SourceVersion: context.SourceVersionFingerprint,
		}},
	}
	diagnostics, err := semanticDiagnosticsForState(document.DocumentState{URI: context.DocumentURI, Text: "int value;"}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("zero-range pass diagnostic = %+v", diagnostics)
	}
}

func TestSemanticDiagnosticsSuppressForgedOrMissingRootSource(t *testing.T) {
	context := semanticTestContextForServer()
	snapshot := semantic.SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: "sha256:forged",
		SourceVersions:           []semantic.SourceVersionEntry{{URI: "file:///workspace/other.tops", Version: "other-v1"}},
		Diagnostics: []semantic.SemanticDiagnostic{{
			Code: "tops-semantic-host-device-boundary", SemanticKind: "host-device-boundary", Severity: semantic.SeverityError, Range: parser.SourceRange{Start: 0, End: 1}, PassKind: "host", ProfileID: context.Profile.ProfileID, ContextStatus: context.Status, SourceVersion: "sha256:forged",
		}},
	}
	diagnostics, err := semanticDiagnosticsForState(document.DocumentState{URI: context.DocumentURI, Text: "int value;"}, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("invalid source fingerprint published diagnostics = %+v", diagnostics)
	}
}

func semanticTestContextForServer() semantic.SemanticContext {
	documentURI := "file:///workspace/pass.tops"
	sourceVersions := []semantic.SourceVersionEntry{{URI: documentURI, Version: "root-v1"}}
	fingerprint, _ := semantic.AggregateSourceVersion(documentURI, sourceVersions)
	return semantic.SemanticContext{
		DocumentURI:              documentURI,
		Command:                  semantic.CompilationCommand{Language: "TOPS", Arch: "gcu400", Driver: "topscc", LanguageStandard: "c++17", PassKind: semantic.PassKindHostDevice},
		Profile:                  semantic.DefaultSemanticProfile(true),
		Status:                   semantic.ContextStatusResolved,
		SemanticContextVersion:   2,
		SourceVersions:           sourceVersions,
		SourceVersionFingerprint: fingerprint,
		PassViews:                semantic.NewPassViews(semantic.PassKindHostDevice, semantic.ContextStatusResolved),
	}
}
