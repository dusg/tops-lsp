package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/index"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
	"tops-lsp/internal/semantic"
	"tops-lsp/internal/transport"
)

type State uint8

const (
	Starting State = iota
	Initialized
	ShutdownPending
	Exited
)

type RequestHandler func(context.Context, protocol.Message) []byte
type parseDocumentFunc func(string, parser.ParseContext) parser.ParseResult

type CompilationContext struct {
	Language              string
	Mode                  string
	LanguageStandard      string
	DriverKind            string
	CompilerContextStatus string
	ArgumentProvenance    string
	PredefinedMacros      map[string]parser.MacroValue
	TargetProfile         string
	PassKind              string
	ContextVersion        int
}

type ServerOptions struct {
	SourceProvider           index.SourceProvider
	CommandCandidateProvider semantic.CommandCandidateProvider
	CompilationContext       *CompilationContext
	NavigationGate           func(context.Context, string) error
	Parse                    func(string, parser.ParseContext) parser.ParseResult
}

type frameResult struct {
	body []byte
	err  error
}

type requestMetadataKey struct{}
type requestStateKey struct{}

type requestMetadata struct {
	ID      string
	Method  string
	URI     string
	Started time.Time
}

func (state State) String() string {
	switch state {
	case Starting:
		return "starting"
	case Initialized:
		return "initialized"
	case ShutdownPending:
		return "shutdown_pending"
	case Exited:
		return "exited"
	default:
		return "unknown"
	}
}

type Server struct {
	mu                                sync.RWMutex
	state                             State
	exitStatus                        int
	documents                         *document.Store
	symbolIndex                       *index.Store
	sourceProvider                    index.SourceProvider
	commandCandidateProvider          semantic.CommandCandidateProvider
	logger                            *slog.Logger
	sessionID                         string
	requests                          *requestRegistry
	handlers                          map[string]RequestHandler
	analysis                          map[string]uint64
	analysisEpoch                     uint64
	contextVersion                    uint64
	parse                             parseDocumentFunc
	requestMu                         sync.Mutex
	freshnessMu                       sync.Mutex
	requestClosing                    bool
	waitGroup                         sync.WaitGroup
	analysisQueues                    map[string]*analysisQueue
	analysisWG                        sync.WaitGroup
	analysisStopping                  bool
	semanticStore                     *semantic.SemanticStore
	semanticSourceCache               *semantic.WorkspaceSourceCache
	semanticSourceAdapter             semantic.SourceProviderAdapter
	semanticMacroSources              map[string]semantic.SourceCacheValue
	fatalErrors                       chan error
	compilationContext                CompilationContext
	navigationGate                    func(context.Context, string) error
	oldVersionOverwriteCount          atomic.Uint64
	staleResultSuppressionCount       atomic.Uint64
	diagnosticMu                      sync.Mutex
	publishedDiagnosticURIsByDocument map[string]map[string]struct{}
	semanticDiagnosticURIsByDocument  map[string]map[string]struct{}
	analysisWaitMu                    sync.Mutex
	analysisWaiters                   map[string][]chan struct{}
	hoverContentFormat                []string
}

func New(logger *slog.Logger) *Server {
	return NewWithOptions(logger, ServerOptions{})
}

func NewWithOptions(logger *slog.Logger, options ServerOptions) *Server {
	if logger == nil {
		logger = logging.New(io.Discard, slog.LevelInfo)
	}
	sourceProvider := options.SourceProvider
	if sourceProvider == nil {
		sourceProvider = index.EmptySourceProvider{}
	}
	compilationContext := defaultCompilationContext()
	if options.CompilationContext != nil {
		compilationContext = cloneCompilationContext(*options.CompilationContext)
	}
	contextVersion := compilationContext.ContextVersion
	if contextVersion <= 0 {
		contextVersion = 1
		compilationContext.ContextVersion = contextVersion
	}
	server := &Server{
		documents:                         document.NewStore(),
		symbolIndex:                       index.NewStore(),
		sourceProvider:                    sourceProvider,
		commandCandidateProvider:          options.CommandCandidateProvider,
		logger:                            logger,
		sessionID:                         strconv.FormatInt(time.Now().UnixNano(), 10),
		requests:                          newRequestRegistry(),
		handlers:                          make(map[string]RequestHandler),
		analysis:                          make(map[string]uint64),
		analysisQueues:                    make(map[string]*analysisQueue),
		semanticStore:                     semantic.NewSemanticStore(),
		semanticSourceCache:               semantic.NewWorkspaceSourceCache(),
		semanticMacroSources:              make(map[string]semantic.SourceCacheValue),
		contextVersion:                    uint64(contextVersion),
		parse:                             options.Parse,
		fatalErrors:                       make(chan error, 1),
		compilationContext:                compilationContext,
		navigationGate:                    options.NavigationGate,
		publishedDiagnosticURIsByDocument: make(map[string]map[string]struct{}),
		semanticDiagnosticURIsByDocument:  make(map[string]map[string]struct{}),
		analysisWaiters:                   make(map[string][]chan struct{}),
	}
	server.semanticSourceAdapter = semantic.SourceProviderAdapter{
		Cache:      server.semanticSourceCache,
		ReadHeader: server.readSemanticHeader,
		ReadMacros: server.readSemanticMacros,
	}
	if server.commandCandidateProvider == nil {
		if provider, ok := sourceProvider.(semantic.CommandCandidateProvider); ok {
			server.commandCandidateProvider = provider
		}
	}
	server.sourceProvider = &serverSourceProvider{documents: server.documents, delegate: sourceProvider, sourceVersions: make(map[string]string), headerStore: index.NewHeaderSnapshotStore()}
	return server
}

func (server *Server) State() State {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.state
}

func (server *Server) ExitStatus() int {
	server.mu.RLock()
	defer server.mu.RUnlock()
	return server.exitStatus
}

func (server *Server) Documents() *document.Store {
	return server.documents
}

func (server *Server) CurrentIndexSnapshot(uri string) (index.IndexSnapshot, bool) {
	return server.currentIndexSnapshot(uri)
}

func (server *Server) CurrentSemanticContext(uri string) (semantic.SemanticContext, bool) {
	if server.semanticStore == nil {
		return semantic.SemanticContext{}, false
	}
	snapshot, ok := server.semanticStore.Get(uri)
	if !ok {
		return semantic.SemanticContext{}, false
	}
	return snapshot.Context, true
}

func (server *Server) OldVersionOverwriteCount() int {
	return int(server.oldVersionOverwriteCount.Load())
}

func (server *Server) StaleResultSuppressionCount() int {
	return int(server.staleResultSuppressionCount.Load())
}

func (server *Server) Handle(ctx context.Context, message protocol.Message) ([]byte, bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	if message.Kind == protocol.RequestMessage {
		var output bytes.Buffer
		done := server.dispatchRequest(ctx, message, transport.NewWriter(&output))
		<-done
		body, err := transport.NewReader(bytes.NewReader(output.Bytes())).ReadFrame()
		if err != nil {
			return nil, false
		}
		return body, false
	}
	return nil, server.handleNotification(ctx, message)
}

func (server *Server) HandleNotificationAsync(ctx context.Context, message protocol.Message) bool {
	if ctx == nil {
		ctx = context.Background()
	}
	return server.handleNotificationWithPublisherAsync(ctx, message, nil)
}

func (server *Server) WaitForAnalysis() {
	server.analysisWG.Wait()
}

func (server *Server) Run(ctx context.Context, reader *transport.Reader, writer *transport.Writer) int {
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		_ = reader.Close()
	}()
	server.log(ctx, slog.LevelInfo, "server_started", "state", server.State().String())
	server.log(ctx, slog.LevelInfo, "transport_ready", "transport", "stdio", "status", "ready")
	diagnosticPublisher := &frameDiagnosticPublisher{writer: writer}
	for {
		frame := make(chan frameResult, 1)
		go func() {
			body, err := reader.ReadFrame()
			frame <- frameResult{body: body, err: err}
		}()
		select {
		case <-ctx.Done():
			return server.terminate(ctx, 1, slog.LevelWarn, "server_context_cancelled")
		case fatalErr := <-server.fatalErrors:
			return server.terminate(ctx, 1, slog.LevelError, "transport_fatal", "error", fatalErr.Error())
		case result := <-frame:
			body, err := result.body, result.err
			if err != nil {
				if errors.Is(err, io.EOF) {
					return server.terminate(ctx, 1, slog.LevelInfo, "stdin_eof")
				}
				return server.terminate(ctx, 1, slog.LevelError, "frame_rejected", "error", err.Error())
			}

			message, err := protocol.DecodeMessage(body)
			if err != nil {
				decodeErr, ok := err.(*protocol.DecodeError)
				if !ok {
					decodeErr = &protocol.DecodeError{Code: protocol.InvalidRequest, Message: err.Error()}
				}
				response, marshalErr := protocol.MarshalError(decodeErr.ID, protocol.NewError(decodeErr.Code, decodeErr.Message, nil))
				if marshalErr != nil {
					return server.terminate(ctx, 1, slog.LevelError, "error_response_marshal_failed", "body_length", len(body), "error", marshalErr.Error())
				}
				if err := server.writeFrame(ctx, writer, response, "response_write_failed"); err != nil {
					return server.terminate(ctx, 1, slog.LevelError, "response_write_failed", "body_length", len(body), "error", err.Error())
				}
				server.log(ctx, slog.LevelWarn, "message_rejected", "body_length", len(body), "error_code", decodeErr.Code, "error", decodeErr.Message)
				continue
			}

			if message.Kind == protocol.RequestMessage {
				server.dispatchRequest(ctx, message, writer)
				continue
			}

			exit := server.handleNotificationWithPublisherAsync(ctx, message, diagnosticPublisher)
			if exit {
				server.stopAnalysisAndWait()
				server.stopActiveRequests()
				return server.ExitStatus()
			}
		}
	}
}

func (server *Server) dispatchRequest(ctx context.Context, message protocol.Message, writer *transport.Writer) <-chan struct{} {
	done := make(chan struct{})
	started := time.Now()
	requestContext, cancel := context.WithCancel(context.WithValue(ctx, requestMetadataKey{}, requestMetadata{
		ID:      protocol.IDKey(message.ID),
		Method:  message.Method,
		URI:     requestDocumentURI(message),
		Started: started,
	}))
	state := newRequestState(cancel)
	requestContext = context.WithValue(requestContext, requestStateKey{}, state)
	server.requestMu.Lock()
	if server.requestClosing {
		server.requestMu.Unlock()
		_ = server.writeFrame(requestContext, writer, server.errorResponse(message.ID, protocol.InvalidRequest, "server is shutting down"), "response_write_failed")
		close(done)
		cancel()
		return done
	}
	if !server.requests.Add(message.ID, state) {
		server.requestMu.Unlock()
		response := server.errorResponse(message.ID, protocol.InvalidRequest, "request id is already active")
		_ = server.writeFrame(requestContext, writer, response, "response_write_failed")
		server.log(requestContext, slog.LevelWarn, "request_rejected", "result", "error", "error_code", protocol.InvalidRequest)
		cancel()
		close(done)
		return done
	}

	server.waitGroup.Add(1)
	server.requestMu.Unlock()
	go func() {
		defer server.waitGroup.Done()
		defer server.requests.Remove(message.ID)
		defer close(done)

		var response []byte
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					server.log(requestContext, slog.LevelError, "request_handler_panic", "error", fmt.Sprintf("%v", recovered), "diagnostic", string(debug.Stack()))
					response = server.errorResponse(message.ID, protocol.InternalError, "request handler failed")
				}
			}()
			response = server.handleRequest(requestContext, message)
		}()
		cancelled, ok := state.Complete()
		if !ok {
			return
		}
		if cancelled {
			response = server.errorResponse(message.ID, protocol.RequestCancelled, "request was cancelled")
		}
		if len(response) == 0 {
			response = server.errorResponse(message.ID, protocol.InternalError, "request handler returned no response")
		}
		server.freshnessMu.Lock()
		if !cancelled && !state.IsFresh() {
			response = server.successResponse(message.ID, nil)
			if !state.ObserveStale() {
				metadata, _ := requestContext.Value(requestMetadataKey{}).(requestMetadata)
				var snapshot *index.IndexSnapshot
				if current, ok := server.currentIndexSnapshot(metadata.URI); ok {
					snapshot = &current
				}
				server.observeStaleResult(requestContext, metadata.Method, metadata.URI, snapshot, "request-freshness-check", metadata.Started)
			}
		}
		server.logRequestCompletion(requestContext, response)
		_ = server.writeFrame(requestContext, writer, response, "response_write_failed")
		server.freshnessMu.Unlock()
	}()
	return done
}

func requestDocumentURI(message protocol.Message) string {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if len(message.Params) == 0 || json.Unmarshal(message.Params, &params) != nil {
		return ""
	}
	return params.TextDocument.URI
}

func (server *Server) handleRequest(ctx context.Context, message protocol.Message) []byte {
	server.mu.RLock()
	handler := server.handlers[message.Method]
	server.mu.RUnlock()
	if handler != nil {
		return handler(ctx, message)
	}

	if _, requiresInitialized := navigationRequestMethods[message.Method]; requiresInitialized {
		if response := server.navigationStateError(message.ID); response != nil {
			return response
		}
	}

	switch message.Method {
	case "initialize":
		var params protocol.InitializeParams
		if err := protocol.DecodeParams(message, &params); err != nil {
			server.log(ctx, slog.LevelWarn, "initialize_failed", "error_code", protocol.InvalidParams, "reason", "invalid_params")
			return server.errorResponse(message.ID, protocol.InvalidParams, "initialize params are invalid")
		}
		if !server.tryInitialize() {
			server.log(ctx, slog.LevelWarn, "initialize_failed", "error_code", protocol.InvalidRequest, "reason", "invalid_state")
			return server.errorResponse(message.ID, protocol.InvalidRequest, "initialize was already called")
		}
		server.mu.Lock()
		server.hoverContentFormat = append([]string(nil), params.Capabilities.TextDocument.Hover.ContentFormat...)
		server.mu.Unlock()
		result := protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				TextDocumentSync: &protocol.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    protocol.TextDocumentSyncIncremental,
				},
				DefinitionProvider:     true,
				ReferencesProvider:     true,
				DocumentSymbolProvider: true,
				CompletionProvider:     &protocol.CompletionOptions{},
				HoverProvider:          true,
			},
			ServerInfo: &protocol.ServerInfo{Name: "tops-lsp", Version: "dev"},
		}
		initializeFields := []any{"workspace_count", len(params.WorkspaceFolders)}
		if params.ProcessID != nil {
			initializeFields = append(initializeFields, "process_id", *params.ProcessID)
		}
		if params.ClientInfo != nil {
			initializeFields = append(initializeFields, "client_name", params.ClientInfo.Name, "client_version", params.ClientInfo.Version)
		}
		server.log(ctx, slog.LevelInfo, "initialize_succeeded", initializeFields...)
		return server.successResponse(message.ID, result)
	case "shutdown":
		previousState := server.tryShutdown()
		if previousState != Initialized {
			if previousState == Starting {
				return server.errorResponse(message.ID, protocol.ServerNotInitialized, "server is not initialized")
			}
			return server.errorResponse(message.ID, protocol.InvalidRequest, "server is shutting down")
		}
		server.log(ctx, slog.LevelInfo, "shutdown_requested")
		return server.successResponse(message.ID, nil)
	case "textDocument/definition":
		return server.handleDefinition(ctx, message)
	case "textDocument/references":
		return server.handleReferences(ctx, message)
	case "textDocument/documentSymbol":
		return server.handleDocumentSymbol(ctx, message)
	case "textDocument/completion":
		return server.handleCompletion(ctx, message)
	case "textDocument/hover":
		return server.handleHover(ctx, message)
	default:
		if server.State() == Starting {
			return server.errorResponse(message.ID, protocol.ServerNotInitialized, "server is not initialized")
		}
		if server.State() == ShutdownPending {
			return server.errorResponse(message.ID, protocol.InvalidRequest, "server is shutting down")
		}
		return server.errorResponse(message.ID, protocol.MethodNotFound, "method is not supported")
	}
}

// navigationRequestMethods lists the requests that require an initialized
// server. They share one state guard in handleRequest.
var navigationRequestMethods = map[string]struct{}{
	"textDocument/definition":     {},
	"textDocument/references":     {},
	"textDocument/documentSymbol": {},
	"textDocument/completion":     {},
	"textDocument/hover":          {},
}

func (server *Server) navigationStateError(id json.RawMessage) []byte {
	switch server.State() {
	case Starting:
		return server.errorResponse(id, protocol.ServerNotInitialized, "server is not initialized")
	case ShutdownPending, Exited:
		return server.errorResponse(id, protocol.InvalidRequest, "server is shutting down")
	default:
		return nil
	}
}

func (server *Server) handleNotification(ctx context.Context, message protocol.Message) bool {
	return server.handleNotificationWithPublisher(ctx, message, nil)
}

func (server *Server) handleNotificationWithPublisher(ctx context.Context, message protocol.Message, publisher diagnosticPublisher) bool {
	return server.handleNotificationWithPublisherMode(ctx, message, publisher, false)
}

func (server *Server) handleNotificationWithPublisherAsync(ctx context.Context, message protocol.Message, publisher diagnosticPublisher) bool {
	return server.handleNotificationWithPublisherMode(ctx, message, publisher, true)
}

func (server *Server) handleNotificationWithPublisherMode(ctx context.Context, message protocol.Message, publisher diagnosticPublisher, asynchronous bool) bool {
	if message.Method == "exit" {
		status := server.exit()
		server.log(ctx, slog.LevelInfo, "exit", "status", status)
		return true
	}
	if message.Method == "$/cancelRequest" {
		server.handleCancel(ctx, message)
		return false
	}
	if server.State() != Initialized {
		server.log(ctx, slog.LevelDebug, "notification_ignored", "method", message.Method, "state", server.State().String())
		return false
	}

	switch message.Method {
	case "textDocument/didOpen":
		server.handleDidOpen(ctx, message, publisher, asynchronous)
	case "textDocument/didChange":
		server.handleDidChange(ctx, message, publisher, asynchronous)
	case "textDocument/didClose":
		server.handleDidClose(ctx, message, publisher, asynchronous)
	default:
		server.log(ctx, slog.LevelDebug, "unknown_notification", "method", message.Method)
	}
	return false
}

func (server *Server) handleDidOpen(ctx context.Context, message protocol.Message, publisher diagnosticPublisher, asynchronous bool) {
	var params protocol.DidOpenTextDocumentParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.log(ctx, slog.LevelWarn, "document_open_rejected", "error", err.Error())
		return
	}
	documentItem := params.TextDocument
	state, err := server.openDocument(documentItem.URI, documentItem.LanguageID, documentItem.Version, documentItem.Text)
	if err != nil {
		server.log(ctx, slog.LevelWarn, "document_open_rejected", "error", err.Error())
		return
	}
	server.log(ctx, slog.LevelInfo, "document_opened", "document_id", logging.DocumentID(documentItem.URI), "document_version", documentItem.Version)
	server.syncSourceDocument(state)
	if documentItem.LanguageID == "tops-header" || index.IsHeaderURI(documentItem.URI) {
		server.markHeaderDependentsStale(documentItem.URI)
		server.semanticStore.Clear()
		if err := server.clearAllPublishedDiagnostics(ctx, publisher); err != nil {
			server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "header-open", "error_kind", "publisher_write_failed")
			server.reportFatal(err)
		}
		server.requeueOpenDocuments(ctx, publisher, asynchronous)
		return
	}
	server.scheduleAnalysis(ctx, publisher, state, asynchronous)
}

func (server *Server) handleDidChange(ctx context.Context, message protocol.Message, publisher diagnosticPublisher, asynchronous bool) {
	var params protocol.DidChangeTextDocumentParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.log(ctx, slog.LevelWarn, "document_change_rejected", "error", err.Error())
		return
	}
	changes := make([]document.Change, len(params.ContentChanges))
	for index, change := range params.ContentChanges {
		converted := document.Change{Text: change.Text, RangeLength: change.RangeLength}
		if change.Range != nil {
			converted.Range = &document.Range{
				Start: document.Position{Line: change.Range.Start.Line, Character: change.Range.Start.Character},
				End:   document.Position{Line: change.Range.End.Line, Character: change.Range.End.Character},
			}
		}
		changes[index] = converted
	}
	server.freshnessMu.Lock()
	state, err := server.changeDocumentLocked(params.TextDocument.URI, params.TextDocument.Version, changes)
	if err != nil {
		server.freshnessMu.Unlock()
		server.log(ctx, slog.LevelWarn, "document_change_rejected", "document_id", logging.DocumentID(params.TextDocument.URI), "document_version", params.TextDocument.Version, "error", err.Error())
		return
	}
	server.syncSourceDocument(state)
	staleDependents := server.markHeaderDependentsStaleLocked(state.URI)
	if err := server.clearPublishedDiagnostics(ctx, publisher, state.URI, state.Version); err != nil {
		server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "document-change", "error_kind", "publisher_write_failed")
		server.reportFatal(err)
	}
	for _, uri := range staleDependents {
		if dependent, ok := server.documents.Get(uri); ok {
			if err := server.clearPublishedDiagnostics(ctx, publisher, uri, dependent.Version); err != nil {
				server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "header-change", "error_kind", "publisher_write_failed")
				server.reportFatal(err)
			}
		}
	}
	server.freshnessMu.Unlock()
	server.log(ctx, slog.LevelInfo, "document_changed", "document_id", logging.DocumentID(params.TextDocument.URI), "document_version", params.TextDocument.Version, "change_count", len(changes))
	for _, uri := range staleDependents {
		if dependent, ok := server.documents.Get(uri); ok {
			server.scheduleAnalysis(ctx, publisher, dependent, asynchronous)
		}
	}
	server.scheduleAnalysis(ctx, publisher, state, asynchronous)
}

func (server *Server) scheduleAnalysis(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState, asynchronous bool) {
	if asynchronous {
		server.enqueueAnalysis(ctx, publisher, state)
		return
	}
	server.parseAndPublish(ctx, publisher, state)
}

func (server *Server) handleDidClose(ctx context.Context, message protocol.Message, publisher diagnosticPublisher, asynchronous bool) {
	var params protocol.DidCloseTextDocumentParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.log(ctx, slog.LevelWarn, "document_close_rejected", "error", err.Error())
		return
	}
	header := index.IsHeaderURI(params.TextDocument.URI)
	closedVersion := 0
	if state, ok := server.documents.Get(params.TextDocument.URI); ok {
		closedVersion = state.Version
		if state.LanguageID == "tops-header" {
			header = true
		}
	}
	if !server.closeDocument(params.TextDocument.URI) {
		server.log(ctx, slog.LevelWarn, "document_close_rejected", "document_id", logging.DocumentID(params.TextDocument.URI), "error", "document is not open")
		return
	}
	server.broadcastAnalysis(params.TextDocument.URI)
	if header {
		server.markHeaderDependentsStale(params.TextDocument.URI)
		server.semanticStore.Clear()
		if err := server.clearAllPublishedDiagnostics(ctx, publisher); err != nil {
			server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "header-close", "error_kind", "publisher_write_failed")
			server.reportFatal(err)
		}
		server.requeueOpenDocuments(ctx, publisher, asynchronous)
	} else {
		if err := server.clearDocumentDiagnostics(ctx, publisher, params.TextDocument.URI, closedVersion); err != nil {
			server.log(ctx, slog.LevelError, "semantic_diagnostics_clear_failed", "operation", "document-close", "error_kind", "publisher_write_failed")
			server.reportFatal(err)
		}
		server.invalidateHeaderDependents(ctx, publisher, params.TextDocument.URI, asynchronous)
	}
	server.log(ctx, slog.LevelInfo, "document_closed", "document_id", logging.DocumentID(params.TextDocument.URI))
}

func (server *Server) openDocument(uri, languageID string, version int, text string) (document.DocumentState, error) {
	server.freshnessMu.Lock()
	defer server.freshnessMu.Unlock()
	server.mu.Lock()
	defer server.mu.Unlock()
	if err := server.documents.Open(uri, languageID, version, text); err != nil {
		return document.DocumentState{}, err
	}
	state, ok := server.documents.Get(uri)
	if !ok {
		return document.DocumentState{}, document.ErrDocumentNotOpen
	}
	server.analysis[state.URI]++
	server.analysisEpoch++
	return state, nil
}

func (server *Server) changeDocument(uri string, version int, changes []document.Change) (document.DocumentState, error) {
	server.freshnessMu.Lock()
	defer server.freshnessMu.Unlock()
	state, err := server.changeDocumentLocked(uri, version, changes)
	if err == nil {
		server.syncSourceDocument(state)
	}
	return state, err
}

func (server *Server) changeDocumentLocked(uri string, version int, changes []document.Change) (document.DocumentState, error) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if err := server.documents.Change(uri, version, changes); err != nil {
		return document.DocumentState{}, err
	}
	state, ok := server.documents.Get(uri)
	if !ok {
		return document.DocumentState{}, document.ErrDocumentNotOpen
	}
	server.semanticStore.Remove(state.URI)
	server.analysis[state.URI]++
	server.analysisEpoch++
	return state, nil
}

func (server *Server) closeDocument(uri string) bool {
	server.freshnessMu.Lock()
	defer server.freshnessMu.Unlock()
	server.mu.Lock()
	defer server.mu.Unlock()
	state, ok := server.documents.Get(uri)
	if !ok || !server.documents.Close(uri) {
		return false
	}
	if provider, ok := server.sourceProvider.(*serverSourceProvider); ok {
		provider.clearOpenedDocument(state.URI)
	}
	server.analysis[state.URI]++
	server.analysisEpoch++
	if queue := server.analysisQueues[state.URI]; queue != nil {
		queue.pending = nil
	}
	if server.symbolIndex != nil {
		server.symbolIndex.Remove(state.URI)
	}
	server.semanticStore.Remove(state.URI)
	return true
}

func (server *Server) handleCancel(ctx context.Context, message protocol.Message) {
	var params protocol.CancelRequestParams
	if err := protocol.DecodeParams(message, &params); err != nil || !protocol.IsValidRequestID(params.ID) {
		server.log(ctx, slog.LevelWarn, "cancel_rejected", "error", "cancel request id is invalid")
		return
	}
	if server.requests.Cancel(params.ID) {
		server.log(ctx, slog.LevelInfo, "request_cancelled", "request_id", protocol.IDKey(params.ID))
		return
	}
	server.log(ctx, slog.LevelDebug, "cancel_ignored", "request_id", protocol.IDKey(params.ID))
}

func (server *Server) registerHandler(method string, handler RequestHandler) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.handlers[method] = handler
}

func (server *Server) stopActiveRequests() {
	server.requestMu.Lock()
	server.requestClosing = true
	server.requestMu.Unlock()
	server.requests.CancelAll()
	server.waitGroup.Wait()
}

func (server *Server) reportFatal(err error) {
	select {
	case server.fatalErrors <- err:
	default:
	}
}

func (server *Server) writeFrame(ctx context.Context, writer *transport.Writer, body []byte, event string) error {
	if err := writer.WriteFrame(body); err != nil {
		server.log(ctx, slog.LevelError, event, "error", err.Error(), "result", "write_error")
		server.reportFatal(err)
		return err
	}
	return nil
}

func (server *Server) terminate(ctx context.Context, status int, level slog.Level, event string, args ...any) int {
	server.stopAnalysisAndWait()
	server.markExited(status)
	server.stopActiveRequests()
	server.log(ctx, level, event, args...)
	return status
}

func (server *Server) successResponse(id json.RawMessage, result any) []byte {
	response, err := protocol.MarshalSuccess(id, result)
	if err != nil {
		server.log(context.Background(), slog.LevelError, "response_marshal_failed", "error", err.Error())
		return nil
	}
	return response
}

func (server *Server) errorResponse(id json.RawMessage, code protocol.ErrorCode, message string) []byte {
	response, err := protocol.MarshalError(id, protocol.NewError(code, message, nil))
	if err != nil {
		server.log(context.Background(), slog.LevelError, "error_response_marshal_failed", "error", err.Error())
		return nil
	}
	return response
}

func (server *Server) markExited(status int) {
	server.mu.Lock()
	defer server.mu.Unlock()
	server.state = Exited
	server.exitStatus = status
}

func (server *Server) tryInitialize() bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.state != Starting {
		return false
	}
	server.state = Initialized
	return true
}

func (server *Server) tryShutdown() State {
	server.mu.Lock()
	defer server.mu.Unlock()
	previous := server.state
	if previous == Initialized {
		server.state = ShutdownPending
	}
	return previous
}

func (server *Server) exit() int {
	server.mu.Lock()
	defer server.mu.Unlock()
	status := 1
	if server.state == ShutdownPending {
		status = 0
	}
	server.state = Exited
	server.exitStatus = status
	return status
}

func (server *Server) log(ctx context.Context, level slog.Level, message string, args ...any) {
	fields := make([]any, 0, len(args)+10)
	fields = append(fields, "event", message, "session_id", server.sessionID)
	if metadata, ok := ctx.Value(requestMetadataKey{}).(requestMetadata); ok {
		fields = append(fields, "request_id", metadata.ID, "method", metadata.Method, "duration_ms", time.Since(metadata.Started).Milliseconds())
	}
	fields = append(fields, args...)
	server.logger.Log(ctx, level, message, fields...)
}

func (server *Server) logRequestCompletion(ctx context.Context, response []byte) {
	metadata, _ := ctx.Value(requestMetadataKey{}).(requestMetadata)
	fields := []any{
		"document_id", logging.DocumentID(metadata.URI),
		"document_version", server.currentDocumentVersion(metadata.URI),
	}
	var envelope struct {
		Error  *protocol.ErrorObject `json:"error"`
		Result json.RawMessage       `json:"result"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		server.log(ctx, slog.LevelInfo, "request_completed", append(fields, "result", "success")...)
		return
	}
	if envelope.Error != nil {
		result := "error"
		switch envelope.Error.Code {
		case protocol.InvalidParams:
			result = "invalid-params"
		case protocol.RequestCancelled:
			result = "cancelled"
		case protocol.ContentModified:
			result = "navigation-timeout"
		case protocol.MethodNotFound:
			result = "method-not-found"
		}
		server.log(ctx, slog.LevelWarn, "request_completed", append(fields, "result", result, "error_code", envelope.Error.Code)...)
		return
	}
	result := "success"
	if isEmptyRequestResult(envelope.Result) {
		result = "empty"
	}
	server.log(ctx, slog.LevelInfo, "request_completed", append(fields, "result", result)...)
}

func isEmptyRequestResult(result json.RawMessage) bool {
	trimmed := bytes.TrimSpace(result)
	if len(trimmed) == 0 {
		return true
	}
	switch string(trimmed) {
	case "null", "[]", "{}":
		return true
	}
	return false
}
