package indexmanifest

import (
	"path/filepath"
	"testing"

	indexpkg "tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

func TestDebugManifestDiagnostics(t *testing.T) {
	root := filepath.Join("..", "..")
	cases := []struct {
		name        string
		sourceFiles []string
		roots       []string
		context     FixtureContext
	}{
		{name: "missing-root", sourceFiles: []string{"testdata/index/missing-context/unknown-header.tops"}, context: manifestTestContext()},
		{name: "unknown-condition", sourceFiles: []string{"testdata/index/conditions/basic.tops"}, context: manifestTestContext()},
		{name: "condition-matrix", sourceFiles: []string{"testdata/index/conditions/matrix.tops"}, context: manifestTestContext()},
		{name: "include-cycle", sourceFiles: []string{"testdata/index/multi-file/cycle-main.tops", "testdata/index/headers/cycle-a.h", "testdata/index/headers/cycle-b.h"}, roots: []string{"testdata/index/headers"}, context: manifestTestContext()},
	}
	for _, fixture := range cases {
		t.Run(fixture.name, func(t *testing.T) {
			opened, primaryURI, primaryText, _, err := loadFixtureSources(root, fixture.sourceFiles)
			if err != nil {
				t.Fatal(err)
			}
			roots := make([]string, 0, len(fixture.roots))
			for _, value := range fixture.roots {
				roots = append(roots, filepath.Join(root, value))
			}
			provider := indexpkg.NewLocalSourceProvider(opened, roots)
			_, available := provider.IncludeRoots()
			parseContext := fixture.context.parseContext(primaryURI, "tops", roots, available, 1)
			snapshot := indexpkg.BuildWithProvider(primaryText, parser.Parse(primaryText, parseContext), parseContext, provider)
			for _, diagnostic := range snapshot.Diagnostics {
				t.Logf("code=%q source=%q range=%d:%d condition=%#v", diagnostic.Code, diagnostic.SourceURI, diagnostic.Range.Start, diagnostic.Range.End, diagnostic.Condition)
			}
		})
	}
}
