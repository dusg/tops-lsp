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
	"time"

	"tops-lsp/internal/document"
	"tops-lsp/internal/logging"
	"tops-lsp/internal/parser"
	"tops-lsp/internal/protocol"
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

type frameResult struct {
	body []byte
	err  error
}

type requestMetadataKey struct{}

type requestMetadata struct {
	ID      string
	Method  string
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
	mu               sync.RWMutex
	state            State
	exitStatus       int
	documents        *document.Store
	logger           *slog.Logger
	sessionID        string
	requests         *requestRegistry
	handlers         map[string]RequestHandler
	analysis         map[string]uint64
	analysisEpoch    uint64
	contextVersion   uint64
	parse            parseDocumentFunc
	requestMu        sync.Mutex
	requestClosing   bool
	waitGroup        sync.WaitGroup
	analysisQueues   map[string]*analysisQueue
	analysisWG       sync.WaitGroup
	analysisStopping bool
	fatalErrors      chan error
}

func New(logger *slog.Logger) *Server {
	if logger == nil {
		logger = logging.New(io.Discard, slog.LevelInfo)
	}
	return &Server{
		documents:      document.NewStore(),
		logger:         logger,
		sessionID:      strconv.FormatInt(time.Now().UnixNano(), 10),
		requests:       newRequestRegistry(),
		handlers:       make(map[string]RequestHandler),
		analysis:       make(map[string]uint64),
		analysisQueues: make(map[string]*analysisQueue),
		contextVersion: 1,
		parse:          parser.Parse,
		fatalErrors:    make(chan error, 1),
	}
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
		Started: started,
	}))
	state := newRequestState(cancel)
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
		server.logRequestCompletion(requestContext, response)
		_ = server.writeFrame(requestContext, writer, response, "response_write_failed")
	}()
	return done
}

func (server *Server) handleRequest(ctx context.Context, message protocol.Message) []byte {
	server.mu.RLock()
	handler := server.handlers[message.Method]
	server.mu.RUnlock()
	if handler != nil {
		return handler(ctx, message)
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
		result := protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				TextDocumentSync: &protocol.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    protocol.TextDocumentSyncIncremental,
				},
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
		server.handleDidClose(ctx, message)
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
	state, err := server.changeDocument(params.TextDocument.URI, params.TextDocument.Version, changes)
	if err != nil {
		server.log(ctx, slog.LevelWarn, "document_change_rejected", "document_id", logging.DocumentID(params.TextDocument.URI), "document_version", params.TextDocument.Version, "error", err.Error())
		return
	}
	server.log(ctx, slog.LevelInfo, "document_changed", "document_id", logging.DocumentID(params.TextDocument.URI), "document_version", params.TextDocument.Version, "change_count", len(changes))
	server.scheduleAnalysis(ctx, publisher, state, asynchronous)
}

func (server *Server) scheduleAnalysis(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState, asynchronous bool) {
	if publisher == nil {
		return
	}
	if asynchronous {
		server.enqueueAnalysis(ctx, publisher, state)
		return
	}
	server.parseAndPublish(ctx, publisher, state)
}

func (server *Server) handleDidClose(ctx context.Context, message protocol.Message) {
	var params protocol.DidCloseTextDocumentParams
	if err := protocol.DecodeParams(message, &params); err != nil {
		server.log(ctx, slog.LevelWarn, "document_close_rejected", "error", err.Error())
		return
	}
	if !server.closeDocument(params.TextDocument.URI) {
		server.log(ctx, slog.LevelWarn, "document_close_rejected", "document_id", logging.DocumentID(params.TextDocument.URI), "error", "document is not open")
		return
	}
	server.log(ctx, slog.LevelInfo, "document_closed", "document_id", logging.DocumentID(params.TextDocument.URI))
}

func (server *Server) openDocument(uri, languageID string, version int, text string) (document.DocumentState, error) {
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
	server.mu.Lock()
	defer server.mu.Unlock()
	if err := server.documents.Change(uri, version, changes); err != nil {
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

func (server *Server) closeDocument(uri string) bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	state, ok := server.documents.Get(uri)
	if !ok || !server.documents.Close(uri) {
		return false
	}
	server.analysis[state.URI]++
	server.analysisEpoch++
	if queue := server.analysisQueues[state.URI]; queue != nil {
		queue.pending = nil
	}
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
	var envelope struct {
		Error *protocol.ErrorObject `json:"error"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil || envelope.Error == nil {
		server.log(ctx, slog.LevelInfo, "request_completed", "result", "success")
		return
	}
	server.log(ctx, slog.LevelWarn, "request_completed", "result", "error", "error_code", envelope.Error.Code)
}
