package server_test

import (
	"context"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/server"
)

func TestDocumentsPreservesStoreAPI(t *testing.T) {
	var store *document.Store = server.New(nil).Documents()
	if store == nil {
		t.Fatal("Documents() returned nil")
	}
}

func TestDocumentNotificationsUpdateAndCloseStore(t *testing.T) {
	instance := server.New(nil)
	instance.Handle(context.Background(), requestWithRawID(t, "1", "initialize", nil))

	open := protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///sample.cpp",
			LanguageID: "cpp",
			Version:    1,
			Text:       "hello",
		},
	}
	response, exit := instance.Handle(context.Background(), notificationMessage(t, "textDocument/didOpen", open))
	if response != nil || exit {
		t.Fatalf("didOpen response = %s exit = %v", response, exit)
	}

	change := protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{URI: "file:///sample.cpp", Version: 2},
		ContentChanges: []protocol.ContentChange{{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: "world",
		}},
	}
	response, exit = instance.Handle(context.Background(), notificationMessage(t, "textDocument/didChange", change))
	if response != nil || exit {
		t.Fatalf("didChange response = %s exit = %v", response, exit)
	}
	state, ok := instance.Documents().Get("file:///sample.cpp")
	if !ok || state.Text != "world" || state.Version != 2 {
		t.Fatalf("state = %+v exists = %v", state, ok)
	}

	response, exit = instance.Handle(context.Background(), notificationMessage(t, "textDocument/didClose", protocol.DidCloseTextDocumentParams{TextDocument: protocol.TextDocumentIdentifier{URI: "file:///sample.cpp"}}))
	if response != nil || exit {
		t.Fatalf("didClose response = %s exit = %v", response, exit)
	}
	if _, ok := instance.Documents().Get("file:///sample.cpp"); ok {
		t.Fatal("document remains after didClose")
	}
}

func TestInvalidDocumentNotificationKeepsLastValidState(t *testing.T) {
	instance := server.New(nil)
	instance.Handle(context.Background(), requestWithRawID(t, "1", "initialize", nil))
	instance.Handle(context.Background(), notificationMessage(t, "textDocument/didOpen", protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: "file:///sample.cpp", LanguageID: "cpp", Version: 4, Text: "hello"},
	}))

	response, exit := instance.Handle(context.Background(), notificationMessage(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{URI: "file:///sample.cpp", Version: 4},
		ContentChanges: []protocol.ContentChange{{
			Range: &protocol.Range{Start: protocol.Position{Line: 5, Character: 0}, End: protocol.Position{Line: 5, Character: 0}},
			Text:  "bad",
		}},
	}))
	if response != nil || exit {
		t.Fatalf("invalid didChange response = %s exit = %v", response, exit)
	}
	state, _ := instance.Documents().Get("file:///sample.cpp")
	if state.Text != "hello" || state.Version != 4 {
		t.Fatalf("invalid change modified state = %+v", state)
	}

	response, exit = instance.Handle(context.Background(), notificationMessage(t, "textDocument/didChange", protocol.DidChangeTextDocumentParams{
		TextDocument:   protocol.VersionedTextDocumentIdentifier{URI: "file:///sample.cpp", Version: 5},
		ContentChanges: []protocol.ContentChange{{Text: "full"}},
	}))
	if response != nil || exit {
		t.Fatalf("full didChange response = %s exit = %v", response, exit)
	}
	state, _ = instance.Documents().Get("file:///sample.cpp")
	if state.Text != "hello" || state.Version != 4 {
		t.Fatalf("full change modified state = %+v", state)
	}
}
