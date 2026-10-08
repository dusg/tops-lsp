package document_test

import (
	"testing"

	"tops-lsp/internal/document"
)

func TestStoreAppliesMultipleUTF16ChangesAtomically(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "A😀\n尾"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	digit := 0
	changes := []document.Change{
		{
			Range: &document.Range{
				Start: document.Position{Line: 0, Character: 1},
				End:   document.Position{Line: 0, Character: 3},
			},
			Text: "Z",
		},
		{
			Range: &document.Range{
				Start: document.Position{Line: 1, Character: 1},
				End:   document.Position{Line: 1, Character: 1},
			},
			RangeLength: &digit,
			Text:        "!",
		},
	}
	if err := store.Change("file:///sample.cpp", 2, changes); err != nil {
		t.Fatalf("Change() error = %v", err)
	}

	state, ok := store.Get("file:///sample.cpp")
	if !ok {
		t.Fatal("document was not stored")
	}
	if state.Text != "AZ\n尾!" || state.Version != 2 || state.LastChangeCount != 2 {
		t.Fatalf("state = %+v", state)
	}
}

func TestStoreCloseRemovesDocument(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "text"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !store.Close("file:///sample.cpp") {
		t.Fatal("Close() = false")
	}
	if _, ok := store.Get("file:///sample.cpp"); ok {
		t.Fatal("document remains after Close()")
	}
	if store.Close("file:///sample.cpp") {
		t.Fatal("second Close() = true")
	}
}

func TestStoreAllowsEndOfLineRange(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "hello"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 2, []document.Change{{
		Range: &document.Range{
			Start: document.Position{Line: 0, Character: 0},
			End:   document.Position{Line: 0, Character: 5},
		},
		Text: "world",
	}}); err != nil {
		t.Fatalf("Change() error = %v", err)
	}
	state, _ := store.Get("file:///sample.cpp")
	if state.Text != "world" || state.Version != 2 {
		t.Fatalf("state = %+v", state)
	}
}

func TestStoreHandlesCRLFLineEnds(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "one\r\ntwo"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 2, []document.Change{{
		Range: &document.Range{
			Start: document.Position{Line: 0, Character: 0},
			End:   document.Position{Line: 0, Character: 3},
		},
		Text: "uno",
	}}); err != nil {
		t.Fatalf("Change() error = %v", err)
	}
	state, _ := store.Get("file:///sample.cpp")
	if state.Text != "uno\r\ntwo" {
		t.Fatalf("CRLF state = %q", state.Text)
	}
}
