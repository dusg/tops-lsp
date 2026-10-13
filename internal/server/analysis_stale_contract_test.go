package server

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
)

func TestParserVersionMismatchPublishesStaleResult(t *testing.T) {
	var output bytes.Buffer
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{
		CompilationContext: &CompilationContext{ContextVersion: 4},
		Parse: func(text string, parseContext parser.ParseContext) parser.ParseResult {
			result := parser.Parse(text, parseContext)
			result.DocumentVersion--
			result.ContextVersion--
			return result
		},
	})
	uri := "file:///workspace/parser-version-mismatch.tops"
	state, err := server.openDocument(uri, "tops", 2, "int value;\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)

	logged := output.String()
	if !strings.Contains(logged, `"reason":"parser-version-mismatch"`) {
		t.Fatalf("parser stale log = %s", logged)
	}
	assertStaleResultEventFields(t, logged, uri, 1, 2, 3, 4)
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() != 1 {
		t.Fatalf("parser stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
	if _, ok := server.CurrentIndexSnapshot(uri); ok {
		t.Fatal("parser version mismatch committed an index snapshot")
	}
}

func TestAnalysisQueueDropStaleResultCarriesHeaderProvenance(t *testing.T) {
	var output bytes.Buffer
	server := NewWithOptions(logging.New(&output, slog.LevelDebug), ServerOptions{SourceProvider: index.NewLocalSourceProvider(nil, nil)})
	headerURI := "file:///workspace/api.h"
	rootURI := "file:///workspace/main.tops"
	oldHeader := "int old_value;\n"
	newHeader := "int new_value;\n"
	if _, err := server.openDocument(headerURI, "tops-header", 1, oldHeader); err != nil {
		t.Fatal(err)
	}
	rootState, err := server.openDocument(rootURI, "tops", 1, "#include \"api.h\"\nint use() { return old_value; }\n")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, rootState)
	if snapshot, ok := server.CurrentIndexSnapshot(rootURI); !ok || len(snapshot.Dependencies) != 1 {
		t.Fatalf("initial root snapshot = %+v, ok = %v", snapshot, ok)
	}

	parseStarted := make(chan struct{})
	releaseParse := make(chan struct{})
	var parseCalls atomic.Int32
	server.parse = func(text string, parseContext parser.ParseContext) parser.ParseResult {
		if parseCalls.Add(1) == 1 {
			close(parseStarted)
			<-releaseParse
		}
		return parser.Parse(text, parseContext)
	}
	server.enqueueAnalysis(context.Background(), nil, rootState)
	<-parseStarted
	server.enqueueAnalysis(context.Background(), nil, rootState)
	if _, err := server.changeDocument(headerURI, 2, []document.Change{{
		Range: &document.Range{Start: document.Position{}, End: document.Position{Line: 1, Character: 0}},
		Text:  newHeader,
	}}); err != nil {
		t.Fatal(err)
	}
	server.markHeaderDependentsStale(headerURI)
	currentRoot, ok := server.Documents().Get(rootURI)
	if !ok {
		t.Fatal("root document was closed")
	}
	server.enqueueAnalysis(context.Background(), nil, currentRoot)
	close(releaseParse)
	server.WaitForAnalysis()

	oldHeaderVersion := index.SourceDigest(oldHeader)
	newHeaderVersion := index.SourceDigest(newHeader)
	var queueDropLog string
	for _, line := range strings.Split(output.String(), "\n") {
		if strings.Contains(line, `"reason":"analysis-queue-drop"`) {
			queueDropLog = line
			break
		}
	}
	if queueDropLog == "" {
		t.Fatalf("queue-drop stale log = %s", output.String())
	}
	for _, field := range []string{"old_generation", "new_generation", "old_epoch", "new_epoch"} {
		if !strings.Contains(queueDropLog, `"`+field+`":`) {
			t.Fatalf("queue-drop stale log missing %q: %s", field, queueDropLog)
		}
	}
	if !strings.Contains(queueDropLog, `"old_header_versions":"`+logging.DocumentID(headerURI)+`:`+oldHeaderVersion+`"`) {
		t.Fatalf("queue-drop stale log lacks old header version: %s", queueDropLog)
	}
	if !strings.Contains(queueDropLog, `"new_header_versions":"`+logging.DocumentID(headerURI)+`:`+newHeaderVersion+`"`) {
		t.Fatalf("queue-drop stale log lacks new header version: %s", queueDropLog)
	}
	if server.OldVersionOverwriteCount() != 0 || server.StaleResultSuppressionCount() < 1 {
		t.Fatalf("queue-drop stale metrics = overwrite:%d suppression:%d", server.OldVersionOverwriteCount(), server.StaleResultSuppressionCount())
	}
}
