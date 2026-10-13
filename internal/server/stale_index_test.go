package server

import (
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/protocol"
)

func TestNavigationDoesNotUseOlderDocumentIndex(t *testing.T) {
	server := New(nil)
	uri := "file:///workspace/versioned.tops"
	if err := server.documents.Open(uri, "tops", 1, "int old_name();"); err != nil {
		t.Fatal(err)
	}
	if !server.symbolIndex.Replace(index.IndexSnapshot{RootURI: uri, SourceVersion: index.SourceVersion{URI: uri, DocumentVersion: 1, ContextVersion: 1}}) {
		t.Fatal("initial snapshot was not stored")
	}
	if err := server.documents.Change(uri, 2, []document.Change{{Range: &document.Range{Start: document.Position{Line: 0, Character: 4}, End: document.Position{Line: 0, Character: 12}}, Text: "new_name"}}); err != nil {
		t.Fatal(err)
	}
	_, snapshot, _, ok := server.navigationInput(uri, protocol.Position{Line: 0, Character: 4})
	if !ok || snapshot.RootURI != "" {
		t.Fatalf("stale navigation snapshot = %+v, ok = %v", snapshot, ok)
	}
}

func TestCloseRemovesIndexSnapshot(t *testing.T) {
	server := New(nil)
	uri := "file:///workspace/closed.tops"
	if err := server.documents.Open(uri, "tops", 1, "int value;"); err != nil {
		t.Fatal(err)
	}
	if !server.symbolIndex.Replace(index.IndexSnapshot{RootURI: uri, SourceVersion: index.SourceVersion{URI: uri, DocumentVersion: 1, ContextVersion: 1}}) {
		t.Fatal("snapshot was not stored")
	}
	if !server.closeDocument(uri) {
		t.Fatal("document did not close")
	}
	if _, ok := server.symbolIndex.Get(uri); ok {
		t.Fatal("closed document index remained")
	}
}
