package server

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/semantic"
)

func TestSemanticDefaultCommandAndStaleSnapshotLifecycle(t *testing.T) {
	server := New(nil)
	state, err := server.openDocument("file:///workspace/default.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	initial, ok := server.CurrentSemanticContext(state.URI)
	if !ok || initial.Profile.ProfileID != "gcu400-default" || initial.Command.Arch != "gcu400" || initial.Command.PassKind != semantic.PassKindHostDevice || initial.Status != semantic.ContextStatusPartial {
		t.Fatalf("default semantic context = %+v, ok = %v", initial, ok)
	}
	if _, err := server.changeDocument(state.URI, 2, []document.Change{{Range: &document.Range{Start: document.Position{}, End: document.Position{Character: len(state.Text)}}, Text: "int changed;"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := server.CurrentSemanticContext(state.URI); ok {
		t.Fatal("old semantic context remained after document change")
	}
	updated, ok := server.Documents().Get(state.URI)
	if !ok || updated.Version != 2 {
		t.Fatalf("updated document = %+v, ok = %v", updated, ok)
	}
	server.parseAndPublish(context.Background(), nil, updated)
	final, ok := server.CurrentSemanticContext(state.URI)
	if !ok || final.DocumentVersion != 2 || final.SourceVersionFingerprint == initial.SourceVersionFingerprint {
		t.Fatalf("final semantic context = %+v, ok = %v", final, ok)
	}
}

func TestMissingHeaderDegradesResolvedSemanticContext(t *testing.T) {
	server := newSemanticTestServer(t, semanticTestProviderOptions{
		Macros:           map[string]string{"PROJECT": "1"},
		Version:          "",
		ToolchainVersion: "test",
		MacroVersion:     "test",
	}, func(options *ServerOptions) {
		options.SourceProvider = testSourceProvider{roots: []string{"/workspace/include"}}
	})
	state, err := server.openDocument("file:///workspace/missing-header.tops", "tops", 1, "#include \"missing.h\"\n__device__ int value;\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	resolved, ok := server.CurrentSemanticContext(state.URI)
	if !ok || resolved.Status != semantic.ContextStatusPartial || len(resolved.HeaderVersions) == 0 || resolved.HeaderVersions[0].Status != semantic.SourceVersionStatusMissing {
		t.Fatalf("semantic context = %+v, ok = %v", resolved, ok)
	}
}

func TestSemanticStaleObservationIncludesSemanticVersionsAndProvenance(t *testing.T) {
	var logs bytes.Buffer
	server := New(logging.New(&logs, slog.LevelDebug))
	state, err := server.openDocument("file:///workspace/stale-observation.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	snapshot, ok := server.CurrentIndexSnapshot(state.URI)
	if !ok {
		t.Fatal("index snapshot was not committed")
	}
	server.observeStaleResult(context.Background(), "semantic", state.URI, &snapshot, "test-stale", time.Now())
	logged := logs.String()
	for _, field := range []string{"semantic_profile_id", "semantic_context_status", "semantic_pass_kind", "old_source_version", "new_source_version", "old_header_versions", "new_header_versions", "semantic_source_version"} {
		if !strings.Contains(logged, field) {
			t.Fatalf("stale_result log missing %q: %s", field, logged)
		}
	}
	if strings.Contains(logged, state.Text) {
		t.Fatalf("stale_result log contains source text: %s", logged)
	}
}
