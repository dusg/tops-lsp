package semantic

import "sync"

type SemanticStore struct {
	mu        sync.RWMutex
	snapshots map[string]SemanticSnapshot
}

func NewSemanticStore() *SemanticStore {
	return &SemanticStore{snapshots: make(map[string]SemanticSnapshot)}
}

func (store *SemanticStore) Get(uri string) (SemanticSnapshot, bool) {
	if store == nil {
		return SemanticSnapshot{}, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	snapshot, ok := store.snapshots[uri]
	return snapshot, ok
}

func (store *SemanticStore) Replace(snapshot SemanticSnapshot) bool {
	if store == nil || snapshot.Context.DocumentURI == "" || snapshot.Validate() != nil {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	uri := snapshot.Context.DocumentURI
	if current, ok := store.snapshots[uri]; ok && !semanticSnapshotNewer(snapshot, current) {
		return false
	}
	store.snapshots[uri] = snapshot
	return true
}

func (store *SemanticStore) MarkStale(uri, reason string) (SemanticSnapshot, bool) {
	if store == nil {
		return SemanticSnapshot{}, false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	snapshot, ok := store.snapshots[uri]
	if !ok {
		return SemanticSnapshot{}, false
	}
	delete(store.snapshots, uri)
	snapshot.Status = SnapshotStatusStale
	snapshot.StaleReason = reason
	return snapshot, true
}

func (store *SemanticStore) Remove(uri string) {
	if store == nil {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.snapshots, uri)
}

func (store *SemanticStore) Clear() {
	if store == nil {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.snapshots = make(map[string]SemanticSnapshot)
}

func semanticSnapshotNewer(candidate, current SemanticSnapshot) bool {
	if candidate.SemanticContextVersion != current.SemanticContextVersion {
		return candidate.SemanticContextVersion > current.SemanticContextVersion
	}
	if candidate.DocumentVersion != current.DocumentVersion {
		return candidate.DocumentVersion > current.DocumentVersion
	}
	if candidate.Generation != current.Generation {
		return candidate.Generation > current.Generation
	}
	if candidate.Epoch != current.Epoch {
		return candidate.Epoch > current.Epoch
	}
	return candidate.SourceVersionFingerprint != current.SourceVersionFingerprint
}
