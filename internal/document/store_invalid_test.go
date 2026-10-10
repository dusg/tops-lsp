package document_test

import (
	"errors"
	"testing"

	"tops-lsp/internal/document"
)

func TestStoreRejectsInvalidChangesWithoutPartialCommit(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 3, "hello"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	changes := []document.Change{
		{
			Range: &document.Range{
				Start: document.Position{Line: 0, Character: 0},
				End:   document.Position{Line: 0, Character: 0},
			},
			Text: "x",
		},
		{
			Range: &document.Range{
				Start: document.Position{Line: 9, Character: 0},
				End:   document.Position{Line: 9, Character: 0},
			},
			Text: "y",
		},
	}
	if err := store.Change("file:///sample.cpp", 4, changes); err == nil {
		t.Fatal("Change() error = nil")
	}
	state, _ := store.Get("file:///sample.cpp")
	if state.Text != "hello" || state.Version != 3 {
		t.Fatalf("invalid change committed state = %+v", state)
	}
}

func TestStoreRejectsMissingDocumentAndOldVersion(t *testing.T) {
	store := document.NewStore()
	if err := store.Change("file:///missing.cpp", 1, []document.Change{{Text: "x"}}); !errors.Is(err, document.ErrDocumentNotOpen) {
		t.Fatalf("missing document error = %v", err)
	}
	if err := store.Open("file:///sample.cpp", "cpp", 2, "text"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 2, []document.Change{{Text: "x"}}); !errors.Is(err, document.ErrVersionNotIncreasing) {
		t.Fatalf("old version error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 3, nil); !errors.Is(err, document.ErrNoChanges) {
		t.Fatalf("empty changes error = %v", err)
	}
}

func TestStoreRejectsNegativeVersions(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///negative.cpp", "cpp", -1, "text"); err == nil {
		t.Fatal("Open() error = nil for negative version")
	}
	if err := store.Open("file:///sample.cpp", "cpp", 1, "text"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", -1, []document.Change{{
		Range: &document.Range{
			Start: document.Position{Line: 0, Character: 0},
			End:   document.Position{Line: 0, Character: 0},
		},
		Text: "x",
	}}); err == nil {
		t.Fatal("Change() error = nil for negative version")
	}
}

func TestStoreRejectsFullChanges(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "text"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 2, []document.Change{{Text: "new"}}); !errors.Is(err, document.ErrFullChange) {
		t.Fatalf("full change error = %v", err)
	}
}

func TestStoreRejectsCRLFTerminatorPositionAndRangeLengthMismatch(t *testing.T) {
	store := document.NewStore()
	if err := store.Open("file:///sample.cpp", "cpp", 1, "one\r\ntwo"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.Change("file:///sample.cpp", 2, []document.Change{{
		Range: &document.Range{
			Start: document.Position{Line: 0, Character: 4},
			End:   document.Position{Line: 0, Character: 4},
		},
		Text: "x",
	}}); !errors.Is(err, document.ErrInvalidRange) {
		t.Fatalf("CRLF terminator error = %v", err)
	}

	if err := store.Open("file:///utf16.cpp", "cpp", 1, "A😀"); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	wrongLength := 1
	if err := store.Change("file:///utf16.cpp", 2, []document.Change{{
		Range: &document.Range{
			Start: document.Position{Line: 0, Character: 1},
			End:   document.Position{Line: 0, Character: 3},
		},
		RangeLength: &wrongLength,
		Text:        "X",
	}}); !errors.Is(err, document.ErrRangeLengthMismatch) {
		t.Fatalf("rangeLength mismatch error = %v", err)
	}
	state, _ := store.Get("file:///utf16.cpp")
	if state.Text != "A😀" || state.Version != 1 {
		t.Fatalf("mismatched rangeLength modified state = %+v", state)
	}
}
