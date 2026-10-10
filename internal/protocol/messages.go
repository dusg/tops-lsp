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
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return Message{}, ParseErrorFrom(err)
	}
	if _, ok := value.(map[string]any); !ok {
		return Message{}, InvalidRequestError("message must be a JSON object")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return Message{}, ParseErrorFrom(err)
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
	params := bytes.TrimSpace(message.Params)
	if len(params) == 0 || bytes.Equal(params, []byte("null")) {
		if _, required := any(target).(requiredParams); required {
			return fmt.Errorf("params are required")
		}
		return nil
	}
	if err := json.Unmarshal(message.Params, target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

type requiredParams interface {
	requireParams()
}

func (*DidOpenTextDocumentParams) requireParams()   {}
func (*DidChangeTextDocumentParams) requireParams() {}
func (*DidCloseTextDocumentParams) requireParams()  {}

func decodeObject(data []byte, target any) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, fmt.Errorf("params must be a JSON object: %w", err)
	}
	if fields == nil {
		return nil, fmt.Errorf("params must be a JSON object")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return nil, err
	}
	return fields, nil
}

func requireField(fields map[string]json.RawMessage, name string) (json.RawMessage, error) {
	value, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return nil, fmt.Errorf("field %q is required", name)
	}
	return value, nil
}

func requireNonEmptyString(value json.RawMessage, name string) error {
	var text string
	if err := json.Unmarshal(value, &text); err != nil || text == "" {
		return fmt.Errorf("field %q must be a non-empty string", name)
	}
	return nil
}

func requireNonNegativeInt(value json.RawMessage, name string) error {
	var number int
	if err := json.Unmarshal(value, &number); err != nil || number < 0 {
		return fmt.Errorf("field %q must be a non-negative integer", name)
	}
	return nil
}

func (item *TextDocumentItem) UnmarshalJSON(data []byte) error {
	type alias TextDocumentItem
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	uri, err := requireField(fields, "uri")
	if err != nil {
		return err
	}
	if err := requireNonEmptyString(uri, "uri"); err != nil {
		return err
	}
	languageID, err := requireField(fields, "languageId")
	if err != nil {
		return err
	}
	if err := requireNonEmptyString(languageID, "languageId"); err != nil {
		return err
	}
	if _, err := requireField(fields, "text"); err != nil {
		return err
	}
	version, err := requireField(fields, "version")
	if err != nil {
		return err
	}
	if err := requireNonNegativeInt(version, "version"); err != nil {
		return err
	}
	*item = TextDocumentItem(value)
	return nil
}

func (identifier *VersionedTextDocumentIdentifier) UnmarshalJSON(data []byte) error {
	type alias VersionedTextDocumentIdentifier
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	uri, err := requireField(fields, "uri")
	if err != nil {
		return err
	}
	if err := requireNonEmptyString(uri, "uri"); err != nil {
		return err
	}
	version, err := requireField(fields, "version")
	if err != nil {
		return err
	}
	if err := requireNonNegativeInt(version, "version"); err != nil {
		return err
	}
	*identifier = VersionedTextDocumentIdentifier(value)
	return nil
}

func (identifier *TextDocumentIdentifier) UnmarshalJSON(data []byte) error {
	type alias TextDocumentIdentifier
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	uri, err := requireField(fields, "uri")
	if err != nil {
		return err
	}
	if err := requireNonEmptyString(uri, "uri"); err != nil {
		return err
	}
	*identifier = TextDocumentIdentifier(value)
	return nil
}

func (change *ContentChange) UnmarshalJSON(data []byte) error {
	type alias ContentChange
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "text"); err != nil {
		return err
	}
	*change = ContentChange(value)
	return nil
}

func (params *DidOpenTextDocumentParams) UnmarshalJSON(data []byte) error {
	type alias DidOpenTextDocumentParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return err
	}
	*params = DidOpenTextDocumentParams(value)
	return nil
}

func (params *DidChangeTextDocumentParams) UnmarshalJSON(data []byte) error {
	type alias DidChangeTextDocumentParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return err
	}
	if len(value.ContentChanges) == 0 {
		return fmt.Errorf("field %q must not be empty", "contentChanges")
	}
	*params = DidChangeTextDocumentParams(value)
	return nil
}

func (params *DidCloseTextDocumentParams) UnmarshalJSON(data []byte) error {
	type alias DidCloseTextDocumentParams
	var value alias
	fields, err := decodeObject(data, &value)
	if err != nil {
		return err
	}
	if _, err := requireField(fields, "textDocument"); err != nil {
		return err
	}
	*params = DidCloseTextDocumentParams(value)
	return nil
}
