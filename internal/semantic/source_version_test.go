package semantic

import (
	"path/filepath"
	"strings"
	"testing"

	"tops-lsp/internal/index"
)

func TestAggregateSourceVersionDeduplicatesAndSorts(t *testing.T) {
	rootURI := "file:///workspace/main.tops"
	fingerprint, err := AggregateSourceVersion(rootURI, []SourceVersionEntry{
		{URI: rootURI, Version: "root-v1"},
		{URI: "file:///workspace/include/a.h", Version: "header-v1"},
		{URI: "file:///workspace/include/a.h", Version: "header-v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != "sha256:5afd972eeaa921f299042a727fb27a9dacb1201e1a32cda4a618dcf82c7144b9" {
		t.Fatalf("fingerprint = %q", fingerprint)
	}
}

func TestAggregateSourceVersionRecordsConflictsWithoutChoosingVersion(t *testing.T) {
	fingerprint, err := AggregateSourceVersion("file:///workspace/main.tops", []SourceVersionEntry{
		{URI: "file:///workspace/main.tops", Version: "root-v1"},
		{URI: "file:///workspace/main.tops", Version: "root-v2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint == "" || fingerprint == "sha256:5afd972eeaa921f299042a727fb27a9dacb1201e1a32cda4a618dcf82c7144b9" {
		t.Fatalf("conflicting fingerprint = %q", fingerprint)
	}
	if strings.Contains(fingerprint, "root-v1") || strings.Contains(fingerprint, "root-v2") {
		t.Fatalf("fingerprint exposed a source version: %q", fingerprint)
	}
}

func TestAggregateSourceVersionRejectsInvalidRootAndEntries(t *testing.T) {
	tests := []struct {
		name    string
		docURI  string
		entries []SourceVersionEntry
	}{
		{name: "missing root", docURI: "file:///workspace/main.tops", entries: []SourceVersionEntry{{URI: "file:///workspace/other.tops", Version: "v1"}}},
		{name: "root status", docURI: "file:///workspace/main.tops", entries: []SourceVersionEntry{{URI: "file:///workspace/main.tops", Status: SourceVersionStatusMissing}}},
		{name: "both version and status", docURI: "file:///workspace/main.tops", entries: []SourceVersionEntry{{URI: "file:///workspace/main.tops", Version: "v1", Status: SourceVersionStatusStale}}},
		{name: "invalid status", docURI: "file:///workspace/main.tops", entries: []SourceVersionEntry{{URI: "file:///workspace/main.tops", Status: "bad"}}},
		{name: "invalid URI", docURI: "file:///workspace/main.tops", entries: []SourceVersionEntry{{URI: "", Version: "v1"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := AggregateSourceVersion(test.docURI, test.entries); err == nil {
				t.Fatal("AggregateSourceVersion() error = nil")
			}
		})
	}
}

func TestBuildSemanticSnapshotRequiresCurrentRootFingerprint(t *testing.T) {
	snapshot := semanticTestIndexSnapshot()
	context := semanticTestContext(ContextStatusResolved)
	context.SourceVersions = []SourceVersionEntry{{URI: snapshot.RootURI, Version: snapshot.SourceVersion.SourceVersion}}
	fingerprint, err := AggregateSourceVersion(context.DocumentURI, context.SourceVersions)
	if err != nil {
		t.Fatal(err)
	}
	context.SourceVersionFingerprint = fingerprint
	if _, err := BuildSemanticSnapshot(snapshot, context); err != nil {
		t.Fatalf("valid source fingerprint rejected: %v", err)
	}

	context.SourceVersionFingerprint = "sha256:forged"
	if _, err := BuildSemanticSnapshot(snapshot, context); err == nil {
		t.Fatal("forged source fingerprint was accepted")
	}

	context.SourceVersions = []SourceVersionEntry{{URI: "file:///workspace/other.tops", Version: "other-v1"}}
	context.SourceVersionFingerprint = "sha256:forged"
	if _, err := BuildSemanticSnapshot(snapshot, context); err == nil {
		t.Fatal("snapshot without root source was accepted")
	}
}

func TestNormalizeSourceURIIsPublicAndStable(t *testing.T) {
	path := filepath.Join(string(filepath.Separator), "workspace", "main.tops")
	got := index.NormalizeSourceURI(path)
	if got == "" || !strings.HasPrefix(got, "file://") || !strings.HasSuffix(got, "/workspace/main.tops") {
		t.Fatalf("NormalizeSourceURI(%q) = %q", path, got)
	}
}
