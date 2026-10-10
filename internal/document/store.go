package document

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	sourceposition "tops-lsp/internal/position"
)

var (
	ErrInvalidURI           = errors.New("document URI must not be empty")
	ErrInvalidVersion       = errors.New("document version must not be negative")
	ErrDocumentNotOpen      = errors.New("document is not open")
	ErrVersionNotIncreasing = errors.New("document version must increase")
	ErrNoChanges            = errors.New("document changes must not be empty")
	ErrFullChange           = errors.New("full document changes are not supported")
	ErrInvalidRange         = errors.New("document change range is invalid")
	ErrRangeLengthMismatch  = errors.New("document change rangeLength does not match range")
)

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

type Change struct {
	Range       *Range
	RangeLength *int
	Text        string
}

type DocumentState struct {
	URI             string
	LanguageID      string
	Version         int
	Text            string
	LastChangeCount int
}

type Store struct {
	mu   sync.RWMutex
	docs map[string]DocumentState
}

func NewStore() *Store {
	return &Store{docs: make(map[string]DocumentState)}
}

func (store *Store) Open(uri, languageID string, version int, text string) error {
	key, err := normalizeURI(uri)
	if err != nil {
		return err
	}
	if version < 0 {
		return ErrInvalidVersion
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.docs[key] = DocumentState{
		URI:        key,
		LanguageID: languageID,
		Version:    version,
		Text:       text,
	}
	return nil
}

func (store *Store) Change(uri string, version int, changes []Change) error {
	key, err := normalizeURI(uri)
	if err != nil {
		return err
	}
	if version < 0 {
		return ErrInvalidVersion
	}
	if len(changes) == 0 {
		return ErrNoChanges
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.docs[key]
	if !ok {
		return ErrDocumentNotOpen
	}
	if version <= current.Version {
		return ErrVersionNotIncreasing
	}

	candidate := current.Text
	for index, change := range changes {
		candidate, err = applyChange(candidate, change)
		if err != nil {
			return &ChangeError{Index: index, Err: err}
		}
	}
	current.Text = candidate
	current.Version = version
	current.LastChangeCount = len(changes)
	store.docs[key] = current
	return nil
}

func (store *Store) Close(uri string) bool {
	key, err := normalizeURI(uri)
	if err != nil {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if _, ok := store.docs[key]; !ok {
		return false
	}
	delete(store.docs, key)
	return true
}

func (store *Store) Get(uri string) (DocumentState, bool) {
	key, err := normalizeURI(uri)
	if err != nil {
		return DocumentState{}, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	document, ok := store.docs[key]
	return document, ok
}

type ChangeError struct {
	Index int
	Err   error
}

func (err *ChangeError) Error() string {
	return fmt.Sprintf("change %d: %v", err.Index, err.Err)
}

func (err *ChangeError) Unwrap() error {
	return err.Err
}

func normalizeURI(uri string) (string, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", ErrInvalidURI
	}
	return uri, nil
}

func applyChange(text string, change Change) (string, error) {
	if change.Range == nil {
		return "", ErrFullChange
	}
	start, err := offsetForPosition(text, change.Range.Start)
	if err != nil {
		return "", err
	}
	end, err := offsetForPosition(text, change.Range.End)
	if err != nil {
		return "", err
	}
	if start > end {
		return "", ErrInvalidRange
	}
	if change.RangeLength != nil {
		if *change.RangeLength < 0 {
			return "", ErrRangeLengthMismatch
		}
		length, err := utf16Length(text[start:end])
		if err != nil || length != *change.RangeLength {
			return "", ErrRangeLengthMismatch
		}
	}
	return text[:start] + change.Text + text[end:], nil
}

func offsetForPosition(text string, position Position) (int, error) {
	offset, err := sourceposition.New(text).OffsetAt(sourceposition.Position{
		Line:      position.Line,
		Character: position.Character,
	})
	if err != nil {
		return 0, ErrInvalidRange
	}
	return offset, nil
}

func utf16Length(text string) (int, error) {
	length, err := sourceposition.UTF16Length(text)
	if err != nil {
		return 0, ErrInvalidRange
	}
	return length, nil
}
