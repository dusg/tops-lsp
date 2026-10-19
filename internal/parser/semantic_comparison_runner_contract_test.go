package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// semanticComparisonRunner resolves the semantic offline comparison manifest and
// golden paths from the repository root. It exists only for the offline
// comparison contract test; server runtime must never construct it.
type semanticComparisonRunner struct {
	root string
}

func (runner semanticComparisonRunner) manifestPath() string {
	return filepath.Join(runner.root, "testdata", "semantic", "comparison_manifest.json")
}

func (runner semanticComparisonRunner) goldenPath() string {
	return filepath.Join(runner.root, "testdata", "semantic", "expected", "semantic_comparison.json")
}

func (runner semanticComparisonRunner) run(transform func([]byte) ([]byte, error), writeGolden bool) ([]byte, error) {
	if transform == nil {
		return nil, fmt.Errorf("semantic comparison transform is nil")
	}
	manifest, err := os.ReadFile(runner.manifestPath())
	if err != nil {
		return nil, err
	}
	golden, err := transform(manifest)
	if err != nil {
		return nil, err
	}
	if writeGolden {
		if err := os.WriteFile(runner.goldenPath(), golden, 0o644); err != nil {
			return nil, err
		}
	}
	return golden, nil
}

func TestSemanticComparisonRunnerUsesDedicatedManifestAndGolden(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	runner := semanticComparisonRunner{root: filepath.Join(filepath.Dir(file), "..", "..")}
	manifest, err := runner.run(func(input []byte) ([]byte, error) { return bytes.ToUpper(input), nil }, false)
	if err != nil || len(manifest) == 0 {
		t.Fatalf("manifest = %d bytes, err = %v", len(manifest), err)
	}
	if runner.manifestPath() == runner.goldenPath() || filepath.Base(runner.manifestPath()) != "comparison_manifest.json" || filepath.Base(runner.goldenPath()) != "semantic_comparison.json" {
		t.Fatalf("runner paths = %q / %q", runner.manifestPath(), runner.goldenPath())
	}
}
