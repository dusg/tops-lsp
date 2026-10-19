package server

import (
	"context"
	"testing"

	"tops-lsp/internal/document"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/semantic"
)

func TestSemanticContextCommitRejectsStaleGenerationAndEpoch(t *testing.T) {
	server := New(nil)
	state, err := server.openDocument("file:///workspace/stale.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := server.newAnalysisSnapshot(state)
	context := server.resolveSemanticContext(state, snapshot.context.version)
	server.invalidateAnalysis(state.URI)
	if server.commitSemanticContext(snapshot, context) {
		t.Fatal("semantic context with stale generation was committed")
	}
	newSnapshot := server.newAnalysisSnapshot(state)
	newContext := server.resolveSemanticContext(state, newSnapshot.context.version)
	server.analysisEpoch++
	if server.commitSemanticContext(newSnapshot, newContext) {
		t.Fatal("semantic context with stale epoch was committed")
	}
}

func TestSemanticContextAlwaysCarriesProfileWhenResolverReturns(t *testing.T) {
	context := New(nil).resolveSemanticContext(
		document.DocumentState{URI: "file:///workspace/main.cpp", LanguageID: "cpp", Version: 1, Text: "int value;"},
		1,
	)
	if context.Profile.ProfileID == "" || context.Profile.ProfileID != "cpp-default" {
		t.Fatalf("context profile = %+v", context.Profile)
	}
	if context.Status != semantic.ContextStatusResolved {
		t.Fatalf("context status = %q", context.Status)
	}
}

func TestSemanticPipelineDropsResultCanceledDuringParse(t *testing.T) {
	var cancelDuringParse context.CancelFunc
	server := NewWithOptions(nil, ServerOptions{Parse: func(text string, context parser.ParseContext) parser.ParseResult {
		if cancelDuringParse != nil {
			cancelDuringParse()
		}
		return parser.Parse(text, context)
	}})
	state, err := server.openDocument("file:///workspace/canceled.tops", "tops", 1, "int value;")
	if err != nil {
		t.Fatal(err)
	}
	requestContext, cancel := context.WithCancel(context.Background())
	cancelDuringParse = cancel
	server.parseSnapshot(requestContext, nil, server.newAnalysisSnapshot(state))
	cancelDuringParse = nil
	if _, ok := server.CurrentIndexSnapshot(state.URI); ok {
		t.Fatal("canceled parse committed an index snapshot")
	}
	if _, ok := server.CurrentSemanticContext(state.URI); ok {
		t.Fatal("canceled parse committed a semantic snapshot")
	}
}
