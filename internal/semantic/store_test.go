package semantic

import "testing"

func TestSemanticStoreCommitsNewerVersionsAndRejectsStaleResults(t *testing.T) {
	store := NewSemanticStore()
	first := testSemanticSnapshot(1, 1, 1, 1, SnapshotStatusComplete)
	if !store.Replace(first) {
		t.Fatal("Replace() rejected the first snapshot")
	}
	newer := testSemanticSnapshot(2, 1, 2, 1, SnapshotStatusDegraded)
	if !store.Replace(newer) {
		t.Fatal("Replace() rejected a newer document snapshot")
	}
	if store.Replace(first) {
		t.Fatal("Replace() accepted an older snapshot")
	}
	stale := newer
	stale.Status = SnapshotStatusStale
	if store.Replace(stale) {
		t.Fatal("Replace() accepted a stale snapshot")
	}
	current, ok := store.Get(newer.Context.DocumentURI)
	if !ok || current.DocumentVersion != 2 || current.Status != SnapshotStatusDegraded {
		t.Fatalf("current snapshot = %+v, ok = %v", current, ok)
	}
}

func TestSemanticStoreMarksSnapshotStaleWithoutPublishingIt(t *testing.T) {
	store := NewSemanticStore()
	snapshot := testSemanticSnapshot(1, 3, 1, 1, SnapshotStatusComplete)
	if !store.Replace(snapshot) {
		t.Fatal("Replace() rejected the snapshot")
	}
	stale, ok := store.MarkStale(snapshot.Context.DocumentURI, "header-changed")
	if !ok || stale.Status != SnapshotStatusStale || stale.StaleReason != "header-changed" {
		t.Fatalf("stale snapshot = %+v, ok = %v", stale, ok)
	}
	if _, ok := store.Get(snapshot.Context.DocumentURI); ok {
		t.Fatal("stale snapshot remained queryable")
	}
}

func TestSemanticSnapshotCarriesDirectVersionAndProfileFields(t *testing.T) {
	snapshot := testSemanticSnapshot(4, 5, 6, 7, SnapshotStatusComplete)
	if snapshot.ProfileID != "gcu400-default" || snapshot.ContextStatus != ContextStatusResolved || snapshot.SourceVersionFingerprint == "" {
		t.Fatalf("direct snapshot fields = %+v", snapshot)
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.SourceVersions) != 1 || len(snapshot.HeaderVersions) != 1 || snapshot.HeaderVersions[0].Status != SourceVersionStatusMissing {
		t.Fatalf("source/header versions = %+v/%+v", snapshot.SourceVersions, snapshot.HeaderVersions)
	}
}

func TestSemanticSnapshotRejectsNoProfileAndInvalidStatus(t *testing.T) {
	snapshot := testSemanticSnapshot(1, 1, 1, 1, SnapshotStatusComplete)
	snapshot.ProfileID = ""
	if err := snapshot.Validate(); err == nil {
		t.Fatal("Validate() accepted a snapshot without profile")
	}
	snapshot = testSemanticSnapshot(1, 1, 1, 1, SnapshotStatusStale)
	if err := snapshot.Validate(); err == nil {
		t.Fatal("Validate() accepted a stale snapshot for commit")
	}
}

func testSemanticSnapshot(documentVersion, contextVersion int, generation, epoch uint64, status SnapshotStatus) SemanticSnapshot {
	documentURI := "file:///workspace/main.tops"
	sourceVersions := []SourceVersionEntry{{URI: documentURI, Version: "root-v1"}}
	headerVersions := []SourceVersionEntry{{URI: "file:///workspace/include/api.h", Status: SourceVersionStatusMissing}}
	fingerprintEntries := append(append([]SourceVersionEntry(nil), sourceVersions...), headerVersions...)
	fingerprint, _ := AggregateSourceVersion(documentURI, fingerprintEntries)
	context := SemanticContext{
		DocumentURI:            documentURI,
		Command:                CompilationCommand{PassKind: PassKindHostDevice},
		Profile:                DefaultSemanticProfile(true),
		Status:                 ContextStatusResolved,
		SemanticContextVersion: contextVersion,
		PassViews:              NewPassViews(PassKindHostDevice, ContextStatusResolved),
	}
	return SemanticSnapshot{
		Context:                  context,
		ProfileID:                context.Profile.ProfileID,
		ContextStatus:            context.Status,
		SourceVersionFingerprint: fingerprint,
		DocumentVersion:          documentVersion,
		SemanticContextVersion:   contextVersion,
		Generation:               generation,
		Epoch:                    epoch,
		Status:                   status,
		SourceVersions:           sourceVersions,
		HeaderVersions:           headerVersions,
	}
}
