package index

import "sync"

// HeaderSnapshotStore caches built header snapshots keyed by root URI, context
// version and source version, and tracks which root URIs depend on which header
// so a changed header can invalidate every cached snapshot that included it.
//
// A zero value is usable only after NewHeaderSnapshotStore; the cache maps are
// created by the constructor.
type HeaderSnapshotStore struct {
	mu         sync.RWMutex
	headers    map[string]map[int]map[string]IndexSnapshot
	dependents map[string]map[string]struct{}
}

func NewHeaderSnapshotStore() *HeaderSnapshotStore {
	return &HeaderSnapshotStore{
		headers:    make(map[string]map[int]map[string]IndexSnapshot),
		dependents: make(map[string]map[string]struct{}),
	}
}

func (store *HeaderSnapshotStore) Get(uri, sourceVersion string, contextVersion int) (IndexSnapshot, bool) {
	if store == nil {
		return IndexSnapshot{}, false
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	byContext := store.headers[uri]
	if byContext == nil {
		return IndexSnapshot{}, false
	}
	byVersion := byContext[contextVersion]
	if byVersion == nil {
		return IndexSnapshot{}, false
	}
	snapshot, ok := byVersion[sourceVersion]
	return snapshot, ok
}

func (store *HeaderSnapshotStore) Put(snapshot IndexSnapshot) {
	if store == nil || snapshot.RootURI == "" {
		return
	}
	sourceVersion := snapshot.SourceVersion.SourceVersion
	if text, ok := snapshot.Sources[snapshot.RootURI]; ok {
		sourceVersion = SourceDigest(text)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	byContext := store.headers[snapshot.RootURI]
	if byContext == nil {
		byContext = make(map[int]map[string]IndexSnapshot)
		store.headers[snapshot.RootURI] = byContext
	}
	byVersion := byContext[snapshot.ContextVersion]
	if byVersion == nil {
		byVersion = make(map[string]IndexSnapshot)
		byContext[snapshot.ContextVersion] = byVersion
	}
	byVersion[sourceVersion] = snapshot
	store.removeDependenciesLocked(snapshot.RootURI)
	for _, byContext := range store.headers[snapshot.RootURI] {
		for _, cached := range byContext {
			store.addDependenciesLocked(cached)
		}
	}
}

// Invalidate drops the cached snapshot for changedURI and every root URI that
// transitively depends on it.
func (store *HeaderSnapshotStore) Invalidate(changedURI string) {
	if store == nil {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.invalidateLocked(changedURI)
}

func (store *HeaderSnapshotStore) invalidateLocked(changedURI string) {
	queue := []string{changedURI}
	visited := make(map[string]struct{})
	for len(queue) > 0 {
		uri := queue[0]
		queue = queue[1:]
		if _, ok := visited[uri]; ok {
			continue
		}
		visited[uri] = struct{}{}
		for dependent := range store.dependents[uri] {
			queue = append(queue, dependent)
		}
		delete(store.dependents, uri)
		delete(store.headers, uri)
		store.removeDependenciesLocked(uri)
	}
}

func (store *HeaderSnapshotStore) removeDependenciesLocked(requesterURI string) {
	for target, dependents := range store.dependents {
		delete(dependents, requesterURI)
		if len(dependents) == 0 {
			delete(store.dependents, target)
		}
	}
}

func (store *HeaderSnapshotStore) addDependenciesLocked(snapshot IndexSnapshot) {
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		dependents := store.dependents[dependency.TargetURI]
		if dependents == nil {
			dependents = make(map[string]struct{})
			store.dependents[dependency.TargetURI] = dependents
		}
		dependents[snapshot.RootURI] = struct{}{}
	}
}
