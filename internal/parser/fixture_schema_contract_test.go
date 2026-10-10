package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureGoldenSchemaContainsAuditableMetadata(t *testing.T) {
	root := standardFixtureRoot(t)
	manifest := loadFixtureManifest(t, filepath.Join(root, "manifest.json"))
	body, err := os.ReadFile(filepath.Join(root, "expected", "parser_goldens.json"))
	if err != nil {
		t.Fatalf("read parser goldens: %v", err)
	}
	var document struct {
		Fixtures []map[string]json.RawMessage `json:"fixtures"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("decode parser goldens: %v", err)
	}
	byID := make(map[string]map[string]json.RawMessage, len(document.Fixtures))
	for _, fixture := range document.Fixtures {
		var id string
		if err := json.Unmarshal(fixture["id"], &id); err != nil {
			t.Fatalf("golden id: %v", err)
		}
		byID[id] = fixture
	}
	for _, fixture := range manifest.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			golden, ok := byID[fixture.ID]
			if !ok {
				t.Fatalf("missing golden %q", fixture.ID)
			}
			for _, field := range []string{"context", "context_hash", "source_hash", "conditional_regions", "consumed_bytes"} {
				if len(golden[field]) == 0 {
					t.Fatalf("golden %q is missing field %q", fixture.ID, field)
				}
			}
			var nodes []map[string]json.RawMessage
			if err := json.Unmarshal(golden["nodes"], &nodes); err != nil {
				t.Fatalf("nodes: %v", err)
			}
			for index, node := range nodes {
				for _, field := range []string{"path", "depth", "conditional_state", "callee", "config_tokens", "argument_tokens"} {
					if len(node[field]) == 0 {
						t.Fatalf("node %d is missing field %q", index, field)
					}
				}
			}
			var tokens []map[string]json.RawMessage
			if err := json.Unmarshal(golden["tokens"], &tokens); err != nil {
				t.Fatalf("tokens: %v", err)
			}
			for index, token := range tokens {
				for _, field := range []string{"conditional_state", "unterminated", "line_start"} {
					if len(token[field]) == 0 {
						t.Fatalf("token %d is missing field %q", index, field)
					}
				}
			}
			var diagnostics []map[string]json.RawMessage
			if err := json.Unmarshal(golden["diagnostics"], &diagnostics); err != nil {
				t.Fatalf("diagnostics: %v", err)
			}
			for index, diagnostic := range diagnostics {
				for _, field := range []string{"message", "recoverable", "incomplete", "document_version", "context_version"} {
					if len(diagnostic[field]) == 0 {
						t.Fatalf("diagnostic %d is missing field %q", index, field)
					}
				}
			}
		})
	}
}

func TestFixtureManifestContainsIndependentExpectedData(t *testing.T) {
	root := standardFixtureRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var document struct {
		Fixtures []map[string]json.RawMessage `json:"fixtures"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(document.Fixtures) == 0 {
		t.Fatal("manifest has no fixtures")
	}
	for _, fixture := range document.Fixtures {
		var id string
		if err := json.Unmarshal(fixture["id"], &id); err != nil {
			t.Fatalf("fixture id: %v", err)
		}
		t.Run(id, func(t *testing.T) {
			var expected map[string]json.RawMessage
			if err := json.Unmarshal(fixture["expected"], &expected); err != nil {
				t.Fatalf("expected data: %v", err)
			}
			for _, field := range []string{"status", "required_nodes", "required_tokens", "diagnostics", "conditional_states", "context"} {
				if len(expected[field]) == 0 {
					t.Fatalf("expected data is missing field %q", field)
				}
			}
		})
	}
}
