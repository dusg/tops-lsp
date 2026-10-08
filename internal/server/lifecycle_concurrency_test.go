package server

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"tops-lsp/internal/protocol"
	"tops-lsp/internal/transport"
)

func TestConcurrentInitializeHasOneSuccessfulTransition(t *testing.T) {
	instance := New(nil)
	folders := make([]protocol.WorkspaceFolder, 512)
	for index := range folders {
		folders[index] = protocol.WorkspaceFolder{URI: "file:///workspace/" + strconv.Itoa(index), Name: "workspace"}
	}
	params, err := json.Marshal(protocol.InitializeParams{WorkspaceFolders: folders})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	responses := dispatchConcurrentRequests(t, instance, "initialize", params, 64)
	successes := 0
	invalid := 0
	for _, response := range responses {
		if response.Error == nil {
			successes++
		} else if response.Error.Code == protocol.InvalidRequest {
			invalid++
		}
	}
	if successes != 1 || invalid != len(responses)-1 {
		t.Fatalf("initialize successes=%d invalid=%d responses=%d", successes, invalid, len(responses))
	}
}

func TestConcurrentShutdownHasOneSuccessfulTransition(t *testing.T) {
	instance := New(nil)
	instance.Handle(context.Background(), protocol.Message{Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "initialize"})
	responses := dispatchConcurrentRequests(t, instance, "shutdown", nil, 64)
	successes := 0
	invalid := 0
	for _, response := range responses {
		if response.Error == nil {
			successes++
		} else if response.Error.Code == protocol.InvalidRequest {
			invalid++
		}
	}
	if successes != 1 || invalid != len(responses)-1 {
		t.Fatalf("shutdown successes=%d invalid=%d responses=%d", successes, invalid, len(responses))
	}
}

type testResponse struct {
	Error *protocol.ErrorObject
}

func dispatchConcurrentRequests(t *testing.T, instance *Server, method string, params json.RawMessage, count int) []testResponse {
	t.Helper()
	var output bytes.Buffer
	writer := transport.NewWriter(&output)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for index := 0; index < count; index++ {
		id := json.RawMessage(strconv.Itoa(index + 1))
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			instance.dispatchRequest(context.Background(), protocol.Message{
				Kind: protocol.RequestMessage, JSONRPC: "2.0", ID: id, Method: method, Params: params,
			}, writer)
		}()
	}
	close(start)
	workers.Wait()
	instance.waitGroup.Wait()

	reader := transport.NewReader(bytes.NewReader(output.Bytes()))
	responses := make([]testResponse, 0, count)
	for index := 0; index < count; index++ {
		body, err := reader.ReadFrame()
		if err != nil {
			t.Fatalf("response %d ReadFrame() error = %v", index, err)
		}
		var response testResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("response %d json.Unmarshal() error = %v", index, err)
		}
		responses = append(responses, response)
	}
	return responses
}
