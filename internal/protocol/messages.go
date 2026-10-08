package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type MessageKind uint8

const (
	RequestMessage MessageKind = iota + 1
	NotificationMessage
)

type Message struct {
	Kind    MessageKind
	JSONRPC string
	ID      json.RawMessage
	Method  string
	Params  json.RawMessage
}

type InitializeParams struct {
	ProcessID        *int              `json:"processId,omitempty"`
	ClientInfo       *ClientInfo       `json:"clientInfo,omitempty"`
	RootURI          *string           `json:"rootUri,omitempty"`
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
}

type TextDocumentSyncOptions struct {
	OpenClose bool `json:"openClose"`
	Change    int  `json:"change"`
}

const TextDocumentSyncIncremental = 2

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type ContentChange struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []ContentChange                 `json:"contentChanges"`
}

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CancelRequestParams struct {
	ID json.RawMessage `json:"id"`
}

func DecodeMessage(body []byte) (Message, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return Message{}, ParseErrorFrom(err)
	}
	if fields == nil {
		return Message{}, InvalidRequestError("message must be a JSON object")
	}

	var version string
	versionRaw, ok := fields["jsonrpc"]
	if !ok || json.Unmarshal(versionRaw, &version) != nil || version != "2.0" {
		return Message{}, invalidRequestWithID(fields, "jsonrpc must be \"2.0\"")
	}

	methodRaw, ok := fields["method"]
	if !ok {
		return Message{}, invalidRequestWithID(fields, "message method is required")
	}
	var method string
	if err := json.Unmarshal(methodRaw, &method); err != nil || method == "" {
		return Message{}, invalidRequestWithID(fields, "method must be a non-empty string")
	}

	message := Message{JSONRPC: version, Method: method}
	if params, ok := fields["params"]; ok {
		message.Params = append(json.RawMessage(nil), params...)
	}

	if id, ok := fields["id"]; ok {
		if !IsValidRequestID(id) {
			return Message{}, InvalidRequestError("request id must be a string or number")
		}
		message.Kind = RequestMessage
		message.ID = append(json.RawMessage(nil), id...)
		return message, nil
	}

	message.Kind = NotificationMessage
	return message, nil
}

func invalidRequestWithID(fields map[string]json.RawMessage, message string) *DecodeError {
	decodeError := InvalidRequestError(message)
	if id, ok := fields["id"]; ok && IsValidRequestID(id) {
		decodeError.ID = append(json.RawMessage(nil), id...)
	}
	return decodeError
}

func DecodeParams[T any](message Message, target *T) error {
	if len(bytes.TrimSpace(message.Params)) == 0 || bytes.Equal(bytes.TrimSpace(message.Params), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(message.Params, target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}
