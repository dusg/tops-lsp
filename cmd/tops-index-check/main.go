package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"tops-lsp/internal/index"
	"tops-lsp/internal/indexmanifest"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/position"
	"tops-lsp/internal/protocol"
	serverpkg "tops-lsp/internal/server"
)

func main() {
	manifestPath := flag.String("manifest", "testdata/index/manifest.json", "index fixture manifest path")
	root := flag.String("root", ".", "workspace root used to load fixture source files")
	flag.Parse()
	manifest, err := indexmanifest.LoadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := indexmanifest.ValidateManifestAtWithMetricsExecutor(*root, manifest, executeManifestQueryWithMetrics)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(body))
	if report.Failed > 0 {
		os.Exit(1)
	}
}

func executeManifestQueryWithMetrics(root string, fixture indexmanifest.FixtureCase, snapshot index.IndexSnapshot, query indexmanifest.ManifestQuery) ([]byte, int, error) {
	roots := make([]string, 0, len(fixture.IncludeRoots))
	for _, includeRoot := range fixture.IncludeRoots {
		if filepath.IsAbs(includeRoot) {
			roots = append(roots, includeRoot)
		} else {
			roots = append(roots, filepath.Join(root, includeRoot))
		}
	}
	originalTexts, err := loadManifestSourceTexts(root, fixture.SourceFiles)
	if err != nil {
		return nil, 0, err
	}
	opened := make(map[string]index.SourceDocument, 1)
	if text, ok := originalTexts[snapshot.RootURI]; ok {
		opened[snapshot.RootURI] = index.SourceDocument{URI: snapshot.RootURI, Text: text, Version: 1, SourceVersion: index.SourceDigest(text)}
	}
	compilationContext := manifestCompilationContext(fixture.Context)
	if fixture.Category == "version-race" {
		return executeManifestVersionRaceQuery(root, fixture, snapshot, query, originalTexts, roots, compilationContext)
	}
	server := serverpkg.NewWithOptions(nil, serverpkg.ServerOptions{
		SourceProvider:     index.NewLocalSourceProvider(opened, roots),
		CompilationContext: &compilationContext,
	})
	initializeBody, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "initialize", Params: json.RawMessage(`{}`)})
	if len(initializeBody) == 0 {
		return nil, 0, fmt.Errorf("initialize returned an empty response")
	}
	if err := replayManifestLifecycle(server, fixture, snapshot, originalTexts, compilationContext); err != nil {
		return nil, server.OldVersionOverwriteCount(), err
	}
	if query.ExpectedNull {
		state, ok := server.Documents().Get(snapshot.RootURI)
		if !ok {
			return nil, server.OldVersionOverwriteCount(), fmt.Errorf("query source %q is not loaded", snapshot.RootURI)
		}
		changeParams, err := fullDocumentChange(snapshot.RootURI, state.Version+1, state.Text, state.Text+"\n// stale query probe\n")
		if err != nil {
			return nil, server.OldVersionOverwriteCount(), err
		}
		cancelledContext, cancel := context.WithCancel(context.Background())
		cancel()
		server.HandleNotificationAsync(cancelledContext, protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didChange", Params: changeParams})
	}
	params, err := manifestQueryParams(snapshot, query)
	if err != nil {
		return nil, server.OldVersionOverwriteCount(), err
	}
	method := "textDocument/" + query.Operation
	body, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("2"), Method: method, Params: params})
	if len(body) == 0 {
		return nil, server.OldVersionOverwriteCount(), fmt.Errorf("query %s returned an empty response", query.Operation)
	}
	return body, server.OldVersionOverwriteCount(), nil
}

func manifestCompilationContext(context indexmanifest.FixtureContext) serverpkg.CompilationContext {
	return serverpkg.CompilationContext{
		LanguageStandard:      context.LanguageStandard,
		DriverKind:            context.DriverKind,
		CompilerContextStatus: context.CompilerContextStatus,
		ArgumentProvenance:    context.ArgumentProvenance,
		PredefinedMacros:      context.PredefinedMacros,
		TargetProfile:         context.TargetProfile,
		PassKind:              context.PassKind,
		ContextVersion:        context.ContextVersion,
	}
}

func executeManifestVersionRaceQuery(root string, fixture indexmanifest.FixtureCase, snapshot index.IndexSnapshot, query indexmanifest.ManifestQuery, originalTexts map[string]string, roots []string, compilationContext serverpkg.CompilationContext) ([]byte, int, error) {
	versions := append([]int(nil), fixture.DocumentVersions...)
	if len(versions) < 2 {
		return nil, 0, fmt.Errorf("case %s: version-race requires at least two versions", fixture.CaseID)
	}
	rootText := originalTexts[snapshot.RootURI]
	texts := make([]string, len(versions))
	for index := range texts {
		texts[index] = rootText
		if len(fixture.VersionTexts) > 0 {
			texts[index] = fixture.VersionTexts[index]
		}
	}
	finalVersion := snapshot.SourceVersion.DocumentVersion
	if finalVersion <= versions[0] {
		return nil, 0, fmt.Errorf("case %s: final version %d is not newer than initial version %d", fixture.CaseID, finalVersion, versions[0])
	}

	parseStarted := make(chan struct{})
	releaseParse := make(chan struct{})
	var parseOnce sync.Once
	parse := func(text string, context parser.ParseContext) parser.ParseResult {
		if context.DocumentURI == snapshot.RootURI && context.DocumentVersion == versions[0] {
			parseOnce.Do(func() { close(parseStarted) })
			<-releaseParse
		}
		return parser.Parse(text, context)
	}
	parseServer := serverpkg.NewWithOptions(nil, serverpkg.ServerOptions{
		SourceProvider:     index.NewLocalSourceProvider(nil, roots),
		CompilationContext: &compilationContext,
		Parse:              parse,
	})
	if err := initializeManifestServer(parseServer); err != nil {
		return nil, parseServer.OldVersionOverwriteCount(), err
	}
	if err := openManifestDocument(parseServer, snapshot.RootURI, versions[0], texts[0]); err != nil {
		return nil, parseServer.OldVersionOverwriteCount(), err
	}
	if err := waitManifestSignal(parseStarted, "initial parse"); err != nil {
		return nil, parseServer.OldVersionOverwriteCount(), err
	}
	state, ok := parseServer.Documents().Get(snapshot.RootURI)
	if !ok {
		return nil, parseServer.OldVersionOverwriteCount(), fmt.Errorf("case %s: initial document was not opened", fixture.CaseID)
	}
	if err := replayManifestVersions(parseServer, snapshot.RootURI, versions, texts); err != nil {
		return nil, parseServer.OldVersionOverwriteCount(), err
	}
	close(releaseParse)
	parseServer.WaitForAnalysis()
	parsed, ok := parseServer.CurrentIndexSnapshot(snapshot.RootURI)
	if !ok || parsed.SourceVersion.DocumentVersion != finalVersion {
		overwriteCount := parseServer.OldVersionOverwriteCount() + 1
		return nil, overwriteCount, fmt.Errorf("case %s: late parse did not publish final version", fixture.CaseID)
	}
	for name := range parsedSymbolNames(parsed) {
		if _, expected := parsedSymbolNames(snapshot)[name]; !expected {
			overwriteCount := parseServer.OldVersionOverwriteCount() + 1
			return nil, overwriteCount, fmt.Errorf("case %s: stale parse published symbol %q", fixture.CaseID, name)
		}
	}

	queryStarted := make(chan struct{})
	releaseQuery := make(chan struct{})
	var queryOnce sync.Once
	gate := func(ctx context.Context, operation string) error {
		if operation != query.Operation {
			return nil
		}
		queryOnce.Do(func() { close(queryStarted) })
		select {
		case <-releaseQuery:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	queryServer := serverpkg.NewWithOptions(nil, serverpkg.ServerOptions{
		SourceProvider:     index.NewLocalSourceProvider(nil, roots),
		CompilationContext: &compilationContext,
		NavigationGate:     gate,
	})
	if err := initializeManifestServer(queryServer); err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	if err := openManifestDocument(queryServer, snapshot.RootURI, versions[0], texts[0]); err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	queryServer.WaitForAnalysis()
	oldSnapshot, ok := queryServer.CurrentIndexSnapshot(snapshot.RootURI)
	if !ok {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), fmt.Errorf("case %s: old query snapshot was not published", fixture.CaseID)
	}
	probeQuery := query
	if probeQuery.Operation != "documentSymbol" {
		probeQuery.Name = firstManifestSymbolName(oldSnapshot)
	}
	probeParams, err := manifestQueryParams(oldSnapshot, probeQuery)
	if err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	probeBody := make(chan []byte, 1)
	go func() {
		body, _ := queryServer.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("3"), Method: "textDocument/" + query.Operation, Params: probeParams})
		probeBody <- body
	}()
	if err := waitManifestSignal(queryStarted, "initial navigation query"); err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	state, ok = queryServer.Documents().Get(snapshot.RootURI)
	if !ok {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), fmt.Errorf("case %s: query document was not opened", fixture.CaseID)
	}
	if err := replayManifestVersions(queryServer, snapshot.RootURI, versions, texts); err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	queryServer.WaitForAnalysis()
	close(releaseQuery)
	staleBody, err := receiveManifestResponse(probeBody, "stale navigation query")
	if err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	if !manifestResponseIsNull(staleBody) {
		overwriteCount := parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount() + 1
		return nil, overwriteCount, fmt.Errorf("case %s: stale %s response was published: %s", fixture.CaseID, query.Operation, staleBody)
	}

	if query.ExpectedNull {
		state, ok = queryServer.Documents().Get(snapshot.RootURI)
		if !ok {
			return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), fmt.Errorf("case %s: final query document was not opened", fixture.CaseID)
		}
		changeParams, err := fullDocumentChange(snapshot.RootURI, state.Version+1, state.Text, state.Text+"\n// stale query probe\n")
		if err != nil {
			return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
		}
		cancelledContext, cancel := context.WithCancel(context.Background())
		cancel()
		queryServer.HandleNotificationAsync(cancelledContext, protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didChange", Params: changeParams})
	}
	params, err := manifestQueryParams(snapshot, query)
	if err != nil {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), err
	}
	body, _ := queryServer.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("4"), Method: "textDocument/" + query.Operation, Params: params})
	if len(body) == 0 {
		return nil, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), fmt.Errorf("case %s: final %s query returned an empty response", fixture.CaseID, query.Operation)
	}
	return body, parseServer.OldVersionOverwriteCount() + queryServer.OldVersionOverwriteCount(), nil
}

func replayManifestVersions(server *serverpkg.Server, uri string, versions []int, texts []string) error {
	state, ok := server.Documents().Get(uri)
	if !ok {
		return fmt.Errorf("version replay document %q is not open", uri)
	}
	for index := 1; index < len(versions); index++ {
		changeParams, err := fullDocumentChange(uri, versions[index], state.Text, texts[index])
		if err != nil {
			return err
		}
		server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didChange", Params: changeParams})
		updated, ok := server.Documents().Get(uri)
		if !ok {
			return fmt.Errorf("version replay document %q was closed", uri)
		}
		state = updated
	}
	return nil
}

func initializeManifestServer(server *serverpkg.Server) error {
	body, _ := server.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, ID: json.RawMessage("1"), Method: "initialize", Params: json.RawMessage(`{}`)})
	if len(body) == 0 {
		return fmt.Errorf("initialize returned an empty response")
	}
	return nil
}

func openManifestDocument(server *serverpkg.Server, uri string, version int, text string) error {
	body, err := json.Marshal(protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: uri, LanguageID: "tops", Version: version, Text: text}})
	if err != nil {
		return err
	}
	server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didOpen", Params: body})
	return nil
}

func waitManifestSignal(signal <-chan struct{}, name string) error {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-signal:
		return nil
	case <-timer.C:
		return fmt.Errorf("timed out waiting for %s", name)
	}
}

func receiveManifestResponse(response <-chan []byte, name string) ([]byte, error) {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case body := <-response:
		if len(body) == 0 {
			return nil, fmt.Errorf("%s returned an empty response", name)
		}
		return body, nil
	case <-timer.C:
		return nil, fmt.Errorf("timed out waiting for %s", name)
	}
}

func manifestResponseIsNull(body []byte) bool {
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &response) != nil || len(response.Error) > 0 && string(response.Error) != "null" {
		return false
	}
	return string(response.Result) == "null"
}

func parsedSymbolNames(snapshot index.IndexSnapshot) map[string]struct{} {
	names := make(map[string]struct{}, len(snapshot.Symbols))
	for _, symbol := range snapshot.Symbols {
		names[symbol.Name] = struct{}{}
	}
	return names
}

func firstManifestSymbolName(snapshot index.IndexSnapshot) string {
	names := make([]string, 0, len(snapshot.Symbols))
	for _, symbol := range snapshot.Symbols {
		names = append(names, symbol.Name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func replayManifestLifecycle(server *serverpkg.Server, fixture indexmanifest.FixtureCase, snapshot index.IndexSnapshot, originalTexts map[string]string, compilationContext serverpkg.CompilationContext) error {
	rootText, ok := originalTexts[snapshot.RootURI]
	if !ok {
		return fmt.Errorf("root source %q is not available for lifecycle replay", snapshot.RootURI)
	}
	versions := append([]int(nil), fixture.DocumentVersions...)
	if len(versions) == 0 {
		versions = []int{1}
	}
	texts := append([]string(nil), fixture.VersionTexts...)
	if len(texts) == 0 {
		texts = make([]string, len(versions))
		for index := range texts {
			texts[index] = rootText
		}
	}
	if len(texts) != len(versions) {
		return fmt.Errorf("case %s: lifecycle version/text count mismatch", fixture.CaseID)
	}
	openParams, err := json.Marshal(protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: snapshot.RootURI, LanguageID: "tops", Version: versions[0], Text: texts[0]}})
	if err != nil {
		return err
	}
	server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didOpen", Params: openParams})
	for index := 1; index < len(versions); index++ {
		state, ok := server.Documents().Get(snapshot.RootURI)
		if !ok {
			return fmt.Errorf("case %s: root document closed during replay", fixture.CaseID)
		}
		changeParams, err := fullDocumentChange(snapshot.RootURI, versions[index], state.Text, texts[index])
		if err != nil {
			return err
		}
		server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didChange", Params: changeParams})
	}
	server.WaitForAnalysis()
	if fixture.Category == "version-race" || fixture.Category == "header-change" {
		server.UpdateCompilationContext(compilationContext)
	}
	if fixture.Category == "header-change" {
		headerURI := manifestHeaderTarget(snapshot)
		if headerURI == "" {
			return fmt.Errorf("case %s: header-change has no dependency target", fixture.CaseID)
		}
		headerText, ok := originalTexts[headerURI]
		if !ok {
			return fmt.Errorf("case %s: header source %q is not available", fixture.CaseID, headerURI)
		}
		languageID := "tops-header"
		headerOpen, err := json.Marshal(protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: headerURI, LanguageID: languageID, Version: 1, Text: headerText}})
		if err != nil {
			return err
		}
		server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didOpen", Params: headerOpen})
		server.WaitForAnalysis()
		changedText := snapshot.Sources[headerURI]
		state, ok := server.Documents().Get(headerURI)
		if !ok || changedText == "" || changedText == state.Text {
			return fmt.Errorf("case %s: header target %q did not produce changed text", fixture.CaseID, headerURI)
		}
		changeParams, err := fullDocumentChange(headerURI, state.Version+1, state.Text, changedText)
		if err != nil {
			return err
		}
		server.HandleNotificationAsync(context.Background(), protocol.Message{Kind: protocol.NotificationMessage, Method: "textDocument/didChange", Params: changeParams})
		server.WaitForAnalysis()
	}
	current, ok := server.CurrentIndexSnapshot(snapshot.RootURI)
	if !ok {
		return fmt.Errorf("case %s: server did not publish root index", fixture.CaseID)
	}
	if current.SourceVersion.DocumentVersion != snapshot.SourceVersion.DocumentVersion {
		return fmt.Errorf("case %s: server final document version %d, want %d", fixture.CaseID, current.SourceVersion.DocumentVersion, snapshot.SourceVersion.DocumentVersion)
	}
	for _, expected := range fixture.ExpectedSymbols {
		found := false
		for _, symbol := range current.Symbols {
			if symbol.Name == expected.Name && (expected.Kind == "" || string(symbol.Kind) == expected.Kind) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("case %s: server final result is missing symbol %q", fixture.CaseID, expected.Name)
		}
	}
	if fixture.Category == "version-race" {
		for _, staleName := range []string{"old_name", "stale_name"} {
			for _, symbol := range current.Symbols {
				if symbol.Name == staleName {
					return fmt.Errorf("case %s: stale symbol %q survived lifecycle replay", fixture.CaseID, staleName)
				}
			}
		}
	}
	if fixture.Category == "header-change" {
		foundDependency := false
		for _, dependency := range current.Dependencies {
			if dependency.TargetURI == manifestHeaderTarget(snapshot) && dependency.SourceVersion.SourceVersion == index.SourceDigest(snapshot.Sources[dependency.TargetURI]) {
				foundDependency = true
				break
			}
		}
		if !foundDependency {
			return fmt.Errorf("case %s: server dependent snapshot has stale header version", fixture.CaseID)
		}
	}
	return nil
}

func loadManifestSourceTexts(root string, paths []string) (map[string]string, error) {
	texts := make(map[string]string, len(paths))
	for _, sourcePath := range paths {
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(root, sourcePath)
		}
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, err
		}
		absolutePath, err := filepath.Abs(sourcePath)
		if err != nil {
			return nil, err
		}
		uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}).String()
		texts[uri] = string(body)
	}
	return texts, nil
}

func manifestHeaderTarget(snapshot index.IndexSnapshot) string {
	for _, dependency := range snapshot.Dependencies {
		if dependency.TargetURI == "" {
			continue
		}
		if strings.HasSuffix(strings.ToLower(dependency.TargetURI), ".h") || strings.HasSuffix(strings.ToLower(dependency.TargetURI), ".hpp") {
			return dependency.TargetURI
		}
	}
	return ""
}

func fullDocumentChange(uri string, version int, oldText, newText string) ([]byte, error) {
	end, err := position.New(oldText).PositionAt(len(oldText))
	if err != nil {
		return nil, err
	}
	return json.Marshal(protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{URI: uri, Version: version},
		ContentChanges: []protocol.ContentChange{{
			Range: &protocol.Range{Start: protocol.Position{}, End: protocol.Position{Line: end.Line, Character: end.Character}},
			Text:  newText,
		}},
	})
}

func manifestQueryParams(snapshot index.IndexSnapshot, query indexmanifest.ManifestQuery) ([]byte, error) {
	if query.Malformed {
		switch query.Operation {
		case "definition":
			return []byte(`{"textDocument":{},"position":{}}`), nil
		case "references":
			return []byte(`{"textDocument":{},"position":{},"context":{}}`), nil
		case "documentSymbol":
			return []byte(`{"textDocument":{}}`), nil
		default:
			return nil, fmt.Errorf("unsupported operation %q", query.Operation)
		}
	}
	queryURI, positionValue, err := indexmanifest.ManifestQueryLocation(snapshot, query)
	if err != nil && query.Operation != "documentSymbol" {
		return nil, err
	}
	if queryURI == "" {
		queryURI = snapshot.RootURI
	}
	identifier := protocol.TextDocumentIdentifier{URI: queryURI}
	switch query.Operation {
	case "definition":
		return json.Marshal(protocol.TextDocumentPositionParams{TextDocument: identifier, Position: positionValue})
	case "references":
		includeDeclaration := true
		if query.IncludeDeclaration != nil {
			includeDeclaration = *query.IncludeDeclaration
		}
		return json.Marshal(protocol.ReferenceParams{TextDocument: identifier, Position: positionValue, Context: protocol.ReferenceContext{IncludeDeclaration: includeDeclaration}})
	case "documentSymbol":
		return json.Marshal(protocol.DocumentSymbolParams{TextDocument: identifier})
	default:
		return nil, fmt.Errorf("unsupported operation %q", query.Operation)
	}
}
