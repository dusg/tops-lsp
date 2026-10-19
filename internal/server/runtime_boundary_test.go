package server

import (
	"context"
	"testing"
)

func TestServerSemanticPathDoesNotStartClang(t *testing.T) {
	t.Setenv("TOPS_LSP_CLANG", "/path/that/does/not/exist/clang")
	server := New(nil)
	state, err := server.openDocument("file:///workspace/runtime-boundary.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	server.parseAndPublish(context.Background(), nil, state)
	if _, ok := server.CurrentSemanticContext(state.URI); !ok {
		t.Fatal("semantic path did not complete without Clang")
	}
}
