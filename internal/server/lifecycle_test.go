package server_test

import (
	"encoding/json"
	"testing"

	"tops-lsp/internal/protocol"
	"tops-lsp/internal/server"
)

func requestWithRawID(t *testing.T, id string, method string, params any) protocol.Message {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		value, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		raw = value
	}
	return protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(id), Method: method, Params: raw}
}

func notificationMessage(t *testing.T, method string, params any) protocol.Message {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		value, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		raw = value
	}
	return protocol.Message{Kind: protocol.NotificationMessage, JSONRPC: "2.0", Method: method, Params: raw}
}

type responseEnvelope struct {
	JSONRPC string                `json:"jsonrpc"`
	ID      json.RawMessage       `json:"id"`
	Result  json.RawMessage       `json:"result"`
	Error   *protocol.ErrorObject `json:"error"`
}

func decodeResponse(t *testing.T, body []byte) responseEnvelope {
	t.Helper()
	var response responseEnvelope
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, body)
	}
	return response
}

func TestLifecycleInitializeShutdownExit(t *testing.T) {
	instance := server.New(nil)
	response, exit := instance.Handle(nil, requestWithRawID(t, "1", "initialize", protocol.InitializeParams{}))
	if exit || instance.State() != server.Initialized {
		t.Fatalf("initialize state = %s exit = %v", instance.State(), exit)
	}
	initialized := decodeResponse(t, response)
	if initialized.Error != nil || string(initialized.ID) != "1" {
		t.Fatalf("initialize response = %+v", initialized)
	}
	var result protocol.InitializeResult
	if err := json.Unmarshal(initialized.Result, &result); err != nil {
		t.Fatalf("initialize result error = %v", err)
	}
	if result.Capabilities.TextDocumentSync == nil || !result.Capabilities.TextDocumentSync.OpenClose || result.Capabilities.TextDocumentSync.Change != protocol.TextDocumentSyncIncremental {
		t.Fatalf("capabilities = %+v", result.Capabilities)
	}

	response, exit = instance.Handle(nil, requestWithRawID(t, "2", "shutdown", nil))
	shutdown := decodeResponse(t, response)
	if exit || shutdown.Error != nil || string(shutdown.Result) != "null" || instance.State() != server.ShutdownPending {
		t.Fatalf("shutdown response = %+v state = %s exit = %v", shutdown, instance.State(), exit)
	}

	response, exit = instance.Handle(nil, notificationMessage(t, "exit", nil))
	if response != nil || !exit || instance.State() != server.Exited || instance.ExitStatus() != 0 {
		t.Fatalf("exit response = %s exit = %v state = %s status = %d", response, exit, instance.State(), instance.ExitStatus())
	}
}

func TestLifecycleRejectsRequestsBeforeInitializeAndAfterShutdown(t *testing.T) {
	instance := server.New(nil)
	response, exit := instance.Handle(nil, requestWithRawID(t, "1", "shutdown", nil))
	beforeInit := decodeResponse(t, response)
	if exit || beforeInit.Error == nil || beforeInit.Error.Code != protocol.ServerNotInitialized {
		t.Fatalf("before initialize response = %+v exit = %v", beforeInit, exit)
	}

	instance.Handle(nil, requestWithRawID(t, "2", "initialize", nil))
	response, _ = instance.Handle(nil, requestWithRawID(t, "3", "initialize", nil))
	duplicate := decodeResponse(t, response)
	if duplicate.Error == nil || duplicate.Error.Code != protocol.InvalidRequest {
		t.Fatalf("duplicate initialize response = %+v", duplicate)
	}

	instance.Handle(nil, requestWithRawID(t, "4", "shutdown", nil))
	response, _ = instance.Handle(nil, requestWithRawID(t, "5", "textDocument/hover", nil))
	afterShutdown := decodeResponse(t, response)
	if afterShutdown.Error == nil || afterShutdown.Error.Code != protocol.InvalidRequest {
		t.Fatalf("after shutdown response = %+v", afterShutdown)
	}
}

func TestExitBeforeShutdownUsesAbnormalStatus(t *testing.T) {
	instance := server.New(nil)
	response, exit := instance.Handle(nil, notificationMessage(t, "exit", nil))
	if response != nil || !exit || instance.ExitStatus() == 0 {
		t.Fatalf("exit response = %s exit = %v status = %d", response, exit, instance.ExitStatus())
	}
}
