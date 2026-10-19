package semantic

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalJSONNormalizesNumbersAndSortsObjectKeys(t *testing.T) {
	value := map[string]any{
		"z": json.Number("1e0"),
		"a": []any{json.Number("-0"), json.Number("1.2000")},
	}
	canonical, err := CanonicalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != `{"a":[0,1.2],"z":1}` {
		t.Fatalf("canonical JSON = %s", canonical)
	}
}

func TestCanonicalJSONRejectsDuplicateKeysAndNonFiniteNumbers(t *testing.T) {
	if _, err := CanonicalJSON(json.RawMessage(`{"a":1,"a":2}`)); err == nil {
		t.Fatal("CanonicalJSON() accepted duplicate object keys")
	}
	if _, err := CanonicalJSON(math.Inf(1)); err == nil {
		t.Fatal("CanonicalJSON() accepted an infinite number")
	}
}

func TestExpandResponseFilesUsesWorkingDirectoryAndDepthFirstOrder(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "outer.rsp"), []byte(`--outer "two words" @inner.rsp`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inner.rsp"), []byte(`--inner 'three words'`), 0o644); err != nil {
		t.Fatal(err)
	}
	arguments, err := ExpandResponseFiles([]string{"@outer.rsp", "tail"}, root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--outer", "two words", "--inner", "three words", "tail"}
	if strings.Join(arguments, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("arguments = %#v, want %#v", arguments, want)
	}
}

func TestExpandResponseFilesRejectsCyclesAndUnclosedQuotes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cycle.rsp"), []byte(`@cycle.rsp`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExpandResponseFiles([]string{"@cycle.rsp"}, root); err == nil {
		t.Fatal("ExpandResponseFiles() accepted a cycle")
	}
	if err := os.WriteFile(filepath.Join(root, "bad.rsp"), []byte(`"unterminated`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExpandResponseFiles([]string{"@bad.rsp"}, root); err == nil {
		t.Fatal("ExpandResponseFiles() accepted an unclosed quote")
	}
}
