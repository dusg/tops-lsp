package server

import (
	"context"
	"encoding/json"
	"sync"
)

type requestState struct {
	mu             sync.Mutex
	cancel         context.CancelFunc
	cancelled      bool
	finished       bool
	freshnessCheck func() bool
	staleObserver  func()
	staleObserved  bool
}

func newRequestState(cancel context.CancelFunc) *requestState {
	return &requestState{cancel: cancel}
}

func (state *requestState) Cancel() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.finished || state.cancelled {
		return false
	}
	state.cancelled = true
	state.cancel()
	return true
}

func (state *requestState) Complete() (cancelled bool, ok bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.finished {
		return false, false
	}
	state.finished = true
	return state.cancelled, true
}

func (state *requestState) SetFreshnessCheck(check func() bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.freshnessCheck = check
}

func (state *requestState) IsFresh() bool {
	state.mu.Lock()
	check := state.freshnessCheck
	state.mu.Unlock()
	return check == nil || check()
}

func (state *requestState) SetStaleObserver(observer func()) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.staleObserver = observer
}

func (state *requestState) ObserveStale() bool {
	state.mu.Lock()
	if state.staleObserved {
		state.mu.Unlock()
		return true
	}
	observer := state.staleObserver
	state.staleObserved = observer != nil
	state.mu.Unlock()
	if observer == nil {
		return false
	}
	observer()
	return true
}

func (state *requestState) MarkStaleObserved() bool {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.staleObserved {
		return false
	}
	state.staleObserved = true
	return true
}

type requestRegistry struct {
	mu     sync.Mutex
	active map[string]*requestState
}

func newRequestRegistry() *requestRegistry {
	return &requestRegistry{active: make(map[string]*requestState)}
}

func (registry *requestRegistry) Add(id json.RawMessage, state *requestState) bool {
	key := string(id)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.active[key]; exists {
		return false
	}
	registry.active[key] = state
	return true
}

func (registry *requestRegistry) Get(id json.RawMessage) *requestState {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.active[string(id)]
}

func (registry *requestRegistry) Remove(id json.RawMessage) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.active, string(id))
}

func (registry *requestRegistry) Cancel(id json.RawMessage) bool {
	state := registry.Get(id)
	if state == nil {
		return false
	}
	return state.Cancel()
}

func (registry *requestRegistry) CancelAll() {
	registry.mu.Lock()
	states := make([]*requestState, 0, len(registry.active))
	for _, state := range registry.active {
		states = append(states, state)
	}
	registry.mu.Unlock()
	for _, state := range states {
		state.Cancel()
	}
}
