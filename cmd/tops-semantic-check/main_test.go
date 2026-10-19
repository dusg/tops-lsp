package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tops-lsp/internal/semantic"
)

func TestRunValidManifest(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "testdata", "semantic", "manifest.json")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-manifest", manifestPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, stderr = %s", code, stderr.String())
	}
	var report semantic.FixtureManifestReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if report.Total != 43 || report.Passed != 43 || report.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestRunInvalidManifestReturnsFailure(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"cases":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-manifest", manifestPath}, &stdout, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1; stdout = %s; stderr = %s", code, stdout.String(), stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("run() did not write a failure report")
	}
}
