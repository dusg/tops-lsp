package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tops-lsp/internal/parser"
)

func TestLocalSourceProviderUsesOpenedDocumentsAndRoots(t *testing.T) {
	root := t.TempDir()
	headerPath := filepath.Join(root, "api.h")
	if err := os.WriteFile(headerPath, []byte("int api_add(int left, int right);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	openedURI := fileURI(filepath.Join(root, "opened.h"))
	provider := NewLocalSourceProvider(map[string]SourceDocument{
		openedURI: {URI: openedURI, Text: "int opened;", Version: 4},
	}, []string{root})
	if roots, available := provider.IncludeRoots(); !available || len(roots) != 1 || roots[0] != root {
		t.Fatalf("roots = %v, available = %v", roots, available)
	}
	opened, status := provider.ResolveInclude("file:///workspace/main.tops", "opened.h")
	if status != HeaderDependencyResolved || opened.Version != 4 {
		t.Fatalf("opened = %+v, status = %q", opened, status)
	}
	loaded, status := provider.ResolveInclude("file:///workspace/main.tops", "api.h")
	if status != HeaderDependencyResolved || loaded.Text == "" || loaded.URI == "" || loaded.SourceVersion == "" {
		t.Fatalf("loaded = %+v, status = %q", loaded, status)
	}
}

func TestNormalizeSourceURIProvidesStableFileIdentity(t *testing.T) {
	if got := NormalizeSourceURI("file:///workspace/./src/../main.tops"); got != "file:///workspace/main.tops" {
		t.Fatalf("normalized file URI = %q", got)
	}
	if got := NormalizeSourceURI("  file:///workspace/main.tops  "); got != "file:///workspace/main.tops" {
		t.Fatalf("trimmed file URI = %q", got)
	}
	if NormalizeSourceURI("") != "" {
		t.Fatal("empty URI was not rejected")
	}
}

func TestLocalSourceProviderDistinguishesMissingAndUnresolved(t *testing.T) {
	opened := NewLocalSourceProvider(map[string]SourceDocument{"file:///workspace/include/opened.h": {URI: "file:///workspace/include/opened.h", Text: "int opened;", Version: 2}}, nil)
	if source, status := opened.ResolveInclude("file:///workspace/main.tops", "opened.h"); status != HeaderDependencyResolved || source.Version != 2 {
		t.Fatalf("opened without roots = %+v, status = %q", source, status)
	}
	provider := NewLocalSourceProvider(nil, nil)
	_, status := provider.ResolveInclude("file:///workspace/main.tops", "api.h")
	if status != HeaderDependencyUnresolved {
		t.Fatalf("status without roots = %q", status)
	}
	root := t.TempDir()
	provider = NewLocalSourceProvider(nil, []string{root})
	_, status = provider.ResolveInclude("file:///workspace/main.tops", "missing.h")
	if status != HeaderDependencyMissing {
		t.Fatalf("status for missing header = %q", status)
	}
}

func TestLocalSourceProviderPrefersRequesterRelativeHeader(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	includeDir := filepath.Join(root, "include")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	localURI := fileURI(filepath.Join(sourceDir, "api.h"))
	rootURI := fileURI(filepath.Join(includeDir, "api.h"))
	provider := NewLocalSourceProvider(map[string]SourceDocument{
		localURI: {URI: localURI, Text: "int local_api;", Version: 1},
		rootURI:  {URI: rootURI, Text: "int root_api;", Version: 1},
	}, []string{includeDir})
	requester := fileURI(filepath.Join(sourceDir, "main.tops"))
	source, status := provider.ResolveInclude(requester, "api.h")
	if status != HeaderDependencyResolved || source.URI != localURI {
		t.Fatalf("requester-relative source = %+v, status = %q", source, status)
	}
}

func TestLocalSourceProviderRejectsAmbiguousOpenedBasename(t *testing.T) {
	provider := NewLocalSourceProvider(map[string]SourceDocument{
		"file:///workspace/one/api.h": {URI: "file:///workspace/one/api.h", Text: "int one;", Version: 1},
		"file:///workspace/two/api.h": {URI: "file:///workspace/two/api.h", Text: "int two;", Version: 1},
	}, nil)
	if _, status := provider.ResolveInclude("file:///workspace/other/main.tops", "api.h"); status != HeaderDependencyUnresolved {
		t.Fatalf("ambiguous status = %q", status)
	}
}

func TestLocalSourceProviderDistinguishesQuotedAndAngleIncludes(t *testing.T) {
	root := t.TempDir()
	requesterDir := filepath.Join(root, "src")
	includeDir := filepath.Join(root, "include")
	if err := os.MkdirAll(requesterDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(requesterDir, "api.h")
	rootPath := filepath.Join(includeDir, "api.h")
	if err := os.WriteFile(localPath, []byte("int local_api;"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootPath, []byte("int root_api;"), 0o644); err != nil {
		t.Fatal(err)
	}
	provider := NewLocalSourceProvider(nil, []string{includeDir})
	requester := fileURI(filepath.Join(requesterDir, "main.tops"))
	quoted, status := provider.ResolveIncludeForm(requester, "api.h", IncludeFormQuoted)
	if status != HeaderDependencyResolved || quoted.URI != fileURI(localPath) {
		t.Fatalf("quoted include = %+v, status = %q", quoted, status)
	}
	angle, status := provider.ResolveIncludeForm(requester, "api.h", IncludeFormAngle)
	if status != HeaderDependencyResolved || angle.URI != fileURI(rootPath) {
		t.Fatalf("angle include = %+v, status = %q", angle, status)
	}
}

func TestLocalSourceProviderCachesHeaderSnapshotsByVersionAndContext(t *testing.T) {
	provider := NewLocalSourceProvider(nil, nil)
	headerURI := "file:///workspace/include/api.h"
	snapshot := IndexSnapshot{
		RootURI:        headerURI,
		ContextVersion: 3,
		SourceVersion:  SourceVersion{URI: headerURI, SourceVersion: "aggregate-v1"},
		Sources:        map[string]string{headerURI: "int value;"},
	}
	provider.PutHeaderSnapshot(snapshot)
	if cached, ok := provider.GetHeaderSnapshot(headerURI, SourceDigest("int value;"), 3); !ok || cached.RootURI != headerURI {
		t.Fatalf("cached header = %+v, ok = %v", cached, ok)
	}
	if _, ok := provider.GetHeaderSnapshot(headerURI, SourceDigest("int value;"), 4); ok {
		t.Fatal("header snapshot from a different context was reused")
	}
	if _, ok := provider.GetHeaderSnapshot(headerURI, SourceDigest("int other;"), 3); ok {
		t.Fatal("header snapshot from a different source version was reused")
	}
}

func TestLocalSourceProviderInvalidatesIndirectHeaderSnapshotCache(t *testing.T) {
	provider := NewLocalSourceProvider(nil, nil)
	childURI := "file:///workspace/include/child.h"
	parentURI := "file:///workspace/include/parent.h"
	rootURI := "file:///workspace/main.tops"
	parentText := "#include \"child.h\"\nint parent_value;\n"
	rootText := "#include \"parent.h\"\nint root_value;\n"
	provider.PutHeaderSnapshot(IndexSnapshot{
		RootURI:        parentURI,
		ContextVersion: 1,
		SourceVersion:  SourceVersion{URI: parentURI, SourceVersion: SourceDigest(parentText)},
		Sources:        map[string]string{parentURI: parentText},
		Dependencies:   []HeaderDependency{{RequesterURI: parentURI, TargetURI: childURI, Status: HeaderDependencyResolved}},
	})
	provider.PutHeaderSnapshot(IndexSnapshot{
		RootURI:        rootURI,
		ContextVersion: 1,
		SourceVersion:  SourceVersion{URI: rootURI, SourceVersion: SourceDigest(rootText)},
		Sources:        map[string]string{rootURI: rootText},
		Dependencies:   []HeaderDependency{{RequesterURI: rootURI, TargetURI: parentURI, Status: HeaderDependencyResolved}},
	})
	provider.ClearOpenDocument(childURI)
	if _, ok := provider.GetHeaderSnapshot(parentURI, SourceDigest(parentText), 1); ok {
		t.Fatal("parent header cache remained after child invalidation")
	}
	if _, ok := provider.GetHeaderSnapshot(rootURI, SourceDigest(rootText), 1); ok {
		t.Fatal("root header cache remained after indirect child invalidation")
	}
}

func TestLocalSourceProviderDoesNotVerifyUntrustedIncludedHeader(t *testing.T) {
	root := t.TempDir()
	headerURI := fileURI(filepath.Join(root, "ordinary.h"))
	mainURI := fileURI(filepath.Join(root, "main.tops"))
	opened := map[string]SourceDocument{
		headerURI: {URI: headerURI, Text: "int threadIdx;\n", Version: 1},
	}
	context := parser.ParseContext{
		DocumentURI:           mainURI,
		LanguageID:            "tops",
		CompilerContextStatus: "resolved",
		TargetProfile:         "gcu400",
		PassKind:              "device",
		IncludeRoots:          []string{root},
		IncludeRootsAvailable: true,
		DocumentVersion:       1,
		ContextVersion:        1,
	}
	source := "#include \"ordinary.h\"\nint use() { return threadIdx; }\n"
	snapshot := BuildWithProvider(source, parser.Parse(source, context), context, NewLocalSourceProvider(opened, []string{root}))
	builtin := symbolByName(snapshot, "threadIdx")
	if builtin == nil || builtin.Builtin.Availability != "unresolved" {
		t.Fatalf("untrusted header builtin = %+v", builtin)
	}
	usage := strings.LastIndex(source, "threadIdx")
	if definitions := snapshot.DefinitionAtURI(mainURI, usage); len(definitions) != 0 {
		t.Fatalf("untrusted header definition = %+v", definitions)
	}
}
