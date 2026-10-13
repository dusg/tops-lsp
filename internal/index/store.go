package index

import (
	"sort"
	"sync"
)

type Store struct {
	mu         sync.RWMutex
	snapshots  map[string]IndexSnapshot
	dependents map[string]map[string]struct{}
}

func NewStore() *Store {
	return &Store{
		snapshots:  make(map[string]IndexSnapshot),
		dependents: make(map[string]map[string]struct{}),
	}
}

func (store *Store) Get(uri string) (IndexSnapshot, bool) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	snapshot, ok := store.snapshots[uri]
	return snapshot, ok
}

func (store *Store) Replace(snapshot IndexSnapshot) bool {
	if snapshot.RootURI == "" {
		return false
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if current, ok := store.snapshots[snapshot.RootURI]; ok && !isNewer(snapshot.SourceVersion, current.SourceVersion) {
		return false
	}
	store.removeDependenciesLocked(snapshot.RootURI)
	store.snapshots[snapshot.RootURI] = snapshot
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		requesters := store.dependents[dependency.TargetURI]
		if requesters == nil {
			requesters = make(map[string]struct{})
			store.dependents[dependency.TargetURI] = requesters
		}
		requesters[snapshot.RootURI] = struct{}{}
	}
	return true
}

func (store *Store) Remove(uri string) {
	store.mu.Lock()
	defer store.mu.Unlock()
	delete(store.snapshots, uri)
	store.removeDependenciesLocked(uri)
}

func (store *Store) Clear() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.snapshots = make(map[string]IndexSnapshot)
	store.dependents = make(map[string]map[string]struct{})
}

func (store *Store) MarkHeaderStale(headerURI string) []string {
	store.mu.Lock()
	defer store.mu.Unlock()
	stale := make([]string, 0)
	queue := []string{headerURI}
	visited := make(map[string]struct{})
	for len(queue) > 0 {
		target := queue[0]
		queue = queue[1:]
		if _, ok := visited[target]; ok {
			continue
		}
		visited[target] = struct{}{}
		requesters := store.dependents[target]
		delete(store.dependents, target)
		for requester := range requesters {
			if _, ok := store.snapshots[requester]; !ok {
				continue
			}
			stale = append(stale, requester)
			delete(store.snapshots, requester)
			store.removeDependenciesLocked(requester)
			queue = append(queue, requester)
		}
	}
	sort.Strings(stale)
	return stale
}

func (store *Store) removeDependenciesLocked(requester string) {
	for target, requesters := range store.dependents {
		delete(requesters, requester)
		if len(requesters) == 0 {
			delete(store.dependents, target)
		}
	}
}

func isNewer(candidate, current SourceVersion) bool {
	if candidate.ContextVersion != current.ContextVersion {
		return candidate.ContextVersion > current.ContextVersion
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
	return candidate.SourceVersion != current.SourceVersion
}
