package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/protocol"
)

func TestObserveIndexDoesNotLogSourcePayload(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	snapshot := index.IndexSnapshot{
		RootURI:       "file:///workspace/secret.tops",
		IndexStatus:   index.IndexStatusDegraded,
		SourceVersion: index.SourceVersion{SourceVersion: "root-digest"},
		Symbols: map[string]index.SymbolRecord{
			"one": {Kind: index.SymbolKindFunction, Name: "compute"},
		},
		Dependencies: []index.HeaderDependency{{TargetURI: "file:///workspace/secret.h", SourceVersion: index.SourceVersion{SourceVersion: "header-digest"}}},
		Diagnostics:  []index.IndexDiagnostic{{Code: "analysis-degraded", Message: "missing header"}},
	}
	server.observeIndex(context.Background(), snapshot)
	logged := output.String()
	if !strings.Contains(logged, "analysis_degraded") || !strings.Contains(logged, "analysis-degraded") || !strings.Contains(logged, "function") {
		t.Fatalf("observability log = %s", logged)
	}
	if !strings.Contains(logged, "root-digest") || !strings.Contains(logged, "header-digest") {
		t.Fatalf("version fields missing from observability log = %s", logged)
	}
	if strings.Contains(logged, "secret.tops") || strings.Contains(logged, "compute") {
		t.Fatalf("observability log contains source payload: %s", logged)
	}
}

func TestObserveStaleIndexLogsVersionTransitionWithoutSource(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	server.observeStaleIndex(context.Background(), "file:///workspace/secret.tops", 1, 2, "document-change")
	logged := output.String()
	if !strings.Contains(logged, "stale_result") || !strings.Contains(logged, "old_document_version") || !strings.Contains(logged, "new_document_version") {
		t.Fatalf("stale log = %s", logged)
	}
	if strings.Contains(logged, "secret.tops") {
		t.Fatalf("stale log contains raw URI: %s", logged)
	}
}

func TestObserveNavigationLogsStableResultFields(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	snapshot := index.IndexSnapshot{
		RootURI:       "file:///workspace/secret.tops",
		IndexStatus:   index.IndexStatusDegraded,
		SourceVersion: index.SourceVersion{DocumentVersion: 2, ContextVersion: 3, SourceVersion: "root-digest"},
		Symbols: map[string]index.SymbolRecord{
			"one": {Kind: index.SymbolKindFunction, Name: "compute", Condition: index.ConditionalProvenance{State: "unknown"}},
		},
		Diagnostics: []index.IndexDiagnostic{{Code: "analysis-degraded", Reason: "ambiguous-candidate"}},
	}
	server.observeNavigation(context.Background(), "definition", snapshot, 0, time.Now())
	server.observeNavigationError(context.Background(), "definition", snapshot.RootURI, &snapshot, 0, "analysis-degraded", "invalid-params", time.Now())
	logged := output.String()
	for _, field := range []string{"navigation_query", "symbol_kind", "diagnostic_code", "error_kind", "analysis-degraded", "ambiguous-candidate"} {
		if !strings.Contains(logged, field) {
			t.Fatalf("navigation log missing %q: %s", field, logged)
		}
	}
	if strings.Contains(logged, "secret.tops") || strings.Contains(logged, "compute") {
		t.Fatalf("navigation log contains source payload: %s", logged)
	}
}

func TestObserveNavigationNoFreshSnapshotLogsStaleResult(t *testing.T) {
	var output bytes.Buffer
	server := New(logging.New(&output, slog.LevelDebug))
	server.observeNavigationError(context.Background(), "definition", "file:///workspace/stale.tops", nil, 0, "stale_result", "no-fresh-snapshot", time.Now())
	logged := output.String()
	if !strings.Contains(logged, "stale_result") || strings.Contains(logged, "navigation_query") {
		t.Fatalf("stale navigation log = %s", logged)
	}
}

func TestObserveStaleResultSeparatesSuppressionFromOverwriteMetrics(t *testing.T) {
	var output bytes.Buffer
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{
		CompilationContext: &CompilationContext{ContextVersion: 2},
	})
	uri := "file:///workspace/stale-metrics.tops"
	if err := server.documents.Open(uri, "tops", 2, "int new_value;\n"); err != nil {
		t.Fatal(err)
	}
	snapshot := index.IndexSnapshot{
		RootURI:        uri,
		ContextVersion: 1,
		SourceVersion:  index.SourceVersion{URI: uri, DocumentVersion: 1, ContextVersion: 1, SourceVersion: "old-document"},
		Dependencies: []index.HeaderDependency{{
			TargetURI:     "file:///workspace/api.h",
			SourceVersion: index.SourceVersion{SourceVersion: "old-header"},
		}},
	}
	server.observeStaleResult(context.Background(), "definition", uri, &snapshot, "query-freshness-check", time.Now())

	if server.OldVersionOverwriteCount() != 0 {
		t.Fatalf("old version overwrite count = %d, want 0", server.OldVersionOverwriteCount())
	}
	if server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("stale result suppression count = %d, want 1", server.StaleResultSuppressionCount())
	}
	assertStaleResultEventFields(t, output.String(), uri, 1, 2, 1, 2)
}

func assertStaleResultEventFields(t *testing.T, logged, uri string, oldDocument, newDocument, oldContext, newContext int) {
	t.Helper()
	for _, field := range []string{
		"stale_result",
		fmt.Sprintf(`"document_id":"%s"`, logging.DocumentID(uri)),
		fmt.Sprintf(`"old_document_version":%d`, oldDocument),
		fmt.Sprintf(`"new_document_version":%d`, newDocument),
		fmt.Sprintf(`"old_context_version":%d`, oldContext),
		fmt.Sprintf(`"new_context_version":%d`, newContext),
		"old_header_versions",
		"new_header_versions",
	} {
		if !strings.Contains(logged, field) {
			t.Fatalf("stale_result log missing %q: %s", field, logged)
		}
	}
	if strings.Contains(logged, uri) {
		t.Fatalf("stale_result log contains raw URI: %s", logged)
	}
}

func TestPublishDiagnosticSetsKeepsOtherDocumentDiagnostics(t *testing.T) {
	server := New(nil)
	publisher := &recordingDiagnosticPublisher{}
	if err := server.publishDiagnosticSets(context.Background(), publisher, map[string][]protocol.Diagnostic{
		"file:///workspace/one.tops": {{Code: "analysis-degraded"}},
	}, map[string]int{"file:///workspace/one.tops": 1}, 1, "file:///workspace/one.tops"); err != nil {
		t.Fatal(err)
	}
	if err := server.publishDiagnosticSets(context.Background(), publisher, map[string][]protocol.Diagnostic{
		"file:///workspace/two.tops": {{Code: "analysis-degraded"}},
	}, map[string]int{"file:///workspace/two.tops": 1}, 1, "file:///workspace/two.tops"); err != nil {
		t.Fatal(err)
	}
	if len(publisher.params) != 2 || len(publisher.params[1].Diagnostics) != 1 || publisher.params[1].URI != "file:///workspace/two.tops" {
		t.Fatalf("diagnostic publications = %+v", publisher.params)
	}
}
