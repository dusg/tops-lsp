package semantic

import "testing"

func TestOutOfScopeManifestCasesRemainUnsupportedOrDegraded(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	seen := make(map[string]bool)
	for _, fixture := range manifest.Cases {
		if fixture.Capability != "out-of-scope" {
			continue
		}
		seen[fixture.OutOfScopeKind] = true
		if fixture.Scenario != "unsupported" || (fixture.ExpectedStatus != "unsupported" && fixture.ExpectedStatus != "degraded") {
			t.Fatalf("out-of-scope fixture = %+v", fixture)
		}
		for _, diagnostic := range fixture.ExpectedDiagnostics {
			if diagnostic.Status != "suppressed" || diagnostic.Reason == "" {
				t.Fatalf("out-of-scope diagnostic = %+v", diagnostic)
			}
		}
	}
	if len(seen) != 6 {
		t.Fatalf("out-of-scope kinds = %v", seen)
	}
}
