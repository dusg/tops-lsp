package index

import (
	"sync"
	"testing"
)

func TestStoreReplacesOnlyNewerSnapshot(t *testing.T) {
	store := NewStore()
	first := IndexSnapshot{RootURI: "file:///workspace/main.tops", SourceVersion: SourceVersion{DocumentVersion: 1, ContextVersion: 1}}
	newer := first
	newer.SourceVersion.DocumentVersion = 2
	older := first
	if !store.Replace(first) {
		t.Fatal("first snapshot was not stored")
	}
	if !store.Replace(newer) {
		t.Fatal("newer snapshot was not stored")
	}
	if store.Replace(older) {
		t.Fatal("older snapshot replaced newer snapshot")
	}
	current, ok := store.Get(first.RootURI)
	if !ok || current.SourceVersion.DocumentVersion != 2 {
		t.Fatalf("current snapshot = %+v, ok = %v", current, ok)
	}
}

func TestStoreMarksHeaderDependentsStale(t *testing.T) {
	store := NewStore()
	headerURI := "file:///workspace/include/api.h"
	root := IndexSnapshot{
		RootURI:       "file:///workspace/main.tops",
		SourceVersion: SourceVersion{DocumentVersion: 1, ContextVersion: 1},
		Dependencies:  []HeaderDependency{{RequesterURI: "file:///workspace/main.tops", TargetURI: headerURI, Status: HeaderDependencyResolved}},
	}
	if !store.Replace(root) {
		t.Fatal("root snapshot was not stored")
	}
	stale := store.MarkHeaderStale(headerURI)
	if len(stale) != 1 || stale[0] != root.RootURI {
		t.Fatalf("stale URIs = %v", stale)
	}
	if _, ok := store.Get(root.RootURI); ok {
		t.Fatal("stale snapshot remained queryable")
	}
}

func TestStoreMarksIndirectHeaderDependentsStale(t *testing.T) {
	store := NewStore()
	childURI := "file:///workspace/include/child.h"
	parentURI := "file:///workspace/include/parent.h"
	rootURI := "file:///workspace/main.tops"
	parent := IndexSnapshot{
		RootURI:       parentURI,
		SourceVersion: SourceVersion{URI: parentURI, DocumentVersion: 1, ContextVersion: 1},
		Dependencies:  []HeaderDependency{{RequesterURI: parentURI, TargetURI: childURI, Status: HeaderDependencyResolved}},
	}
	root := IndexSnapshot{
		RootURI:       rootURI,
		SourceVersion: SourceVersion{URI: rootURI, DocumentVersion: 1, ContextVersion: 1},
		Dependencies:  []HeaderDependency{{RequesterURI: rootURI, TargetURI: parentURI, Status: HeaderDependencyResolved}},
	}
	if !store.Replace(parent) || !store.Replace(root) {
		t.Fatal("header dependency snapshots were not stored")
	}
	stale := store.MarkHeaderStale(childURI)
	if len(stale) != 2 || stale[0] != parentURI || stale[1] != rootURI {
		t.Fatalf("indirect stale URIs = %v", stale)
	}
	if _, ok := store.Get(parentURI); ok {
		t.Fatal("parent header snapshot remained queryable")
	}
	if _, ok := store.Get(rootURI); ok {
		t.Fatal("root snapshot remained queryable")
	}
}

func TestStoreSupportsConcurrentReadsAndWrites(t *testing.T) {
	store := NewStore()
	const uri = "file:///workspace/main.tops"
	var group sync.WaitGroup
	for version := 1; version <= 8; version++ {
		version := version
		group.Add(1)
		go func() {
			defer group.Done()
			store.Replace(IndexSnapshot{RootURI: uri, SourceVersion: SourceVersion{DocumentVersion: version, ContextVersion: 1}})
			store.Get(uri)
		}()
	}
	group.Wait()
	current, ok := store.Get(uri)
	if !ok || current.SourceVersion.DocumentVersion != 8 {
		t.Fatalf("current snapshot = %+v, ok = %v", current, ok)
	}
}
