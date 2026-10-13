package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNavigationResultsDistinguishEmptyArrayAndNull(t *testing.T) {
	emptyLocations, err := json.Marshal([]Location{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(emptyLocations, []byte("[]")) {
		t.Fatalf("empty locations = %s", emptyLocations)
	}
	var noLocations []Location
	nullLocations, err := json.Marshal(noLocations)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nullLocations, []byte("null")) {
		t.Fatalf("null locations = %s", nullLocations)
	}
	if _, err := json.Marshal([]DocumentSymbol{}); err != nil {
		t.Fatal(err)
	}
}
