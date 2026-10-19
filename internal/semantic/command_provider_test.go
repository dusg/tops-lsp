package semantic

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestFileCommandCandidateProviderLoadsMatchingCompileCommand(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.tops")
	databasePath := filepath.Join(root, "compile_commands.json")
	writeCompileCommands(t, databasePath, []map[string]any{
		{
			"directory": root,
			"file":      sourcePath,
			"arguments": []string{"topscc", "-arch", "gcu400", "-std=c++17", "-x", "tops", sourcePath},
		},
		{
			"directory": root,
			"file":      filepath.Join(root, "other.tops"),
			"arguments": []string{"topscc", "-arch", "gcu500", "other.tops"},
		},
	})

	provider := NewFileCommandCandidateProvider(FileCommandCandidateProviderOptions{
		CompileCommandsPath: databasePath,
		WorkspaceFallback: WorkspaceFallbackFields{
			IncludeRoots: []string{root},
			Macros:       map[string]string{"PROJECT": "1"},
			Version:      "workspace-v1",
		},
		Toolchain: ToolchainProvenance{
			Identity:     "topscc",
			Version:      "toolchain-v1",
			IncludeRoots: []string{"/tops/include"},
			Macros:       map[string]string{"__GCU_ARCH__": "400"},
			MacroVersion: "toolchain-m1",
		},
	})

	output, err := provider.ProvideCommandCandidates(CommandCandidateProviderInput{
		DocumentURI:   fileURIForProviderTest(sourcePath),
		WorkspaceRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Candidates) != 2 {
		t.Fatalf("candidates = %+v", output.Candidates)
	}
	var candidate CommandCandidate
	for _, value := range output.Candidates {
		if value.SourceURI == fileURIForProviderTest(sourcePath) {
			candidate = value
			break
		}
	}
	if candidate.SourceKind != "compile_commands" || candidate.Arch != "gcu400" || candidate.Language != "TOPS" || candidate.PassKind != PassKindHostDevice {
		t.Fatalf("candidate = %+v", candidate)
	}
	if candidate.SourceURI != fileURIForProviderTest(sourcePath) || candidate.WorkingDirectory != root {
		t.Fatalf("candidate location = %+v", candidate)
	}
	if output.Toolchain.FieldResolution["toolchain_include_roots"] != "resolved" || output.WorkspaceFallback.FieldResolution["workspace_include_roots"] != "resolved" {
		t.Fatalf("source resolution = toolchain=%+v workspace=%+v", output.Toolchain.FieldResolution, output.WorkspaceFallback.FieldResolution)
	}
	if output.PassAvailability[PassKindHost] != PassAvailabilityAvailable || output.PassAvailability[PassKindDevice] != PassAvailabilityAvailable || output.PassProvenance[PassKindHost]["source"] != "compile_commands" || output.PassProvenance[PassKindDevice]["source"] != "compile_commands" {
		t.Fatalf("pass provenance = availability=%+v provenance=%+v", output.PassAvailability, output.PassProvenance)
	}
}

func TestFileCommandCandidateProviderReloadsChangedDatabase(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.tops")
	databasePath := filepath.Join(root, "compile_commands.json")
	provider := NewFileCommandCandidateProvider(FileCommandCandidateProviderOptions{CompileCommandsPath: databasePath})
	input := CommandCandidateProviderInput{DocumentURI: fileURIForProviderTest(sourcePath), WorkspaceRoot: root}

	writeCompileCommands(t, databasePath, []map[string]any{{
		"directory": root,
		"file":      sourcePath,
		"arguments": []string{"topscc", "-arch", "gcu400", sourcePath},
	}})
	first, err := provider.ProvideCommandCandidates(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Candidates) != 1 || first.Candidates[0].Arch != "gcu400" {
		t.Fatalf("first candidates = %+v", first.Candidates)
	}

	writeCompileCommands(t, databasePath, []map[string]any{{
		"directory": root,
		"file":      sourcePath,
		"arguments": []string{"topscc", "-arch", "gcu500", "-std=c++20", sourcePath},
	}})
	second, err := provider.ProvideCommandCandidates(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Candidates) != 1 || second.Candidates[0].Arch != "gcu500" || second.Candidates[0].LanguageStandard != "c++20" {
		t.Fatalf("reloaded candidates = %+v", second.Candidates)
	}
}

func TestFileCommandCandidateProviderKeepsClangCppAsOrdinaryCpp(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.tops")
	databasePath := filepath.Join(root, "compile_commands.json")
	writeCompileCommands(t, databasePath, []map[string]any{{
		"directory": root,
		"file":      sourcePath,
		"arguments": []string{"clang++", "-std=c++20", sourcePath},
	}})
	provider := NewFileCommandCandidateProvider(FileCommandCandidateProviderOptions{CompileCommandsPath: databasePath})
	output, err := provider.ProvideCommandCandidates(CommandCandidateProviderInput{DocumentURI: fileURIForProviderTest(sourcePath), WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Candidates) != 1 || output.Candidates[0].Language != "C++" {
		t.Fatalf("clang++ candidate = %+v", output.Candidates)
	}
	context := ResolveSemanticContext(ResolverInput{DocumentURI: fileURIForProviderTest(sourcePath), WorkspaceRoot: root, LanguageID: "tops", Candidates: output.Candidates})
	if context.Profile.ProfileID != "cpp-default" || context.Command.Language != "C++" || context.Profile.TargetRules {
		t.Fatalf("clang++ context = %+v", context)
	}
}

func writeCompileCommands(t *testing.T, path string, records []map[string]any) {
	t.Helper()
	body, err := json.Marshal(records)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func fileURIForProviderTest(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}
