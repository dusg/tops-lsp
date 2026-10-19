package server

import (
	"context"
	"errors"
	"time"

	"tops-lsp/internal/index"
)

const analysisWaitTimeout = 2 * time.Second

// errAnalysisTimeout marks a navigation wait that exceeded the fixed limit.
var errAnalysisTimeout = errors.New("analysis did not finish in time")

// analysisInProgress reports whether the document has queued or running analysis.
func (server *Server) analysisInProgress(uri string) bool {
	if uri == "" {
		return false
	}
	server.mu.RLock()
	defer server.mu.RUnlock()
	queue := server.analysisQueues[uri]
	return queue != nil && (queue.pending != nil || queue.running)
}

// currentFreshSnapshot returns the committed snapshot only when it matches the
// current document version and context version. It is the single definition of
// freshness for completion, hover and navigation.
func (server *Server) currentFreshSnapshot(uri string) (index.IndexSnapshot, bool) {
	state, ok := server.documents.Get(uri)
	if !ok {
		return index.IndexSnapshot{}, false
	}
	snapshot, ok := server.currentIndexSnapshot(uri)
	if !ok || snapshot.IndexStatus == index.IndexStatusStale {
		return index.IndexSnapshot{}, false
	}
	if snapshot.SourceVersion.DocumentVersion != state.Version || snapshot.ContextVersion != server.currentContextVersion() {
		return index.IndexSnapshot{}, false
	}
	return snapshot, true
}

// hasFreshAnalysis reports whether a committed snapshot matches the current
// document version and context version.
func (server *Server) hasFreshAnalysis(uri string) bool {
	_, ok := server.currentFreshSnapshot(uri)
	return ok
}

// hasIndexedSnapshot reports whether any snapshot, fresh or stale, exists.
func (server *Server) hasIndexedSnapshot(uri string) bool {
	_, ok := server.currentIndexSnapshot(uri)
	return ok
}

// snapshotAvailability classifies the committed snapshot for a document. It is
// the single definition of the stale / missing / analyzing precedence shared by
// completion, hover and navigation.
type snapshotAvailability int

const (
	snapshotFresh snapshotAvailability = iota
	snapshotStale
	snapshotMissing
	snapshotAnalyzing
)

func (server *Server) snapshotAvailability(uri string) (index.IndexSnapshot, snapshotAvailability) {
	if snapshot, ok := server.currentFreshSnapshot(uri); ok {
		return snapshot, snapshotFresh
	}
	if server.hasIndexedSnapshot(uri) {
		return index.IndexSnapshot{}, snapshotStale
	}
	if server.analysisInProgress(uri) {
		return index.IndexSnapshot{}, snapshotAnalyzing
	}
	return index.IndexSnapshot{}, snapshotMissing
}

func (server *Server) registerAnalysisWaiter(uri string) chan struct{} {
	signal := make(chan struct{})
	server.analysisWaitMu.Lock()
	server.analysisWaiters[uri] = append(server.analysisWaiters[uri], signal)
	server.analysisWaitMu.Unlock()
	return signal
}

func (server *Server) removeAnalysisWaiter(uri string, signal chan struct{}) {
	server.analysisWaitMu.Lock()
	defer server.analysisWaitMu.Unlock()
	waiters := server.analysisWaiters[uri]
	for index, waiter := range waiters {
		if waiter == signal {
			waiters = append(waiters[:index], waiters[index+1:]...)
			break
		}
	}
	if len(waiters) == 0 {
		delete(server.analysisWaiters, uri)
		return
	}
	server.analysisWaiters[uri] = waiters
}

// broadcastAnalysis wakes every waiter registered for the document. It must run
// without holding server.mu so waiters can re-check state safely.
func (server *Server) broadcastAnalysis(uri string) {
	if uri == "" || server.analysisWaiters == nil {
		return
	}
	server.analysisWaitMu.Lock()
	waiters := server.analysisWaiters[uri]
	delete(server.analysisWaiters, uri)
	server.analysisWaitMu.Unlock()
	for _, waiter := range waiters {
		close(waiter)
	}
}

// waitForAnalysis waits until deadline for a fresh snapshot. It returns
// errAnalysisTimeout on expiry and ctx.Err() when the request is cancelled.
// Callers must re-check freshness after it returns nil.
func (server *Server) waitForAnalysis(ctx context.Context, uri string, deadline time.Time) error {
	for {
		if server.hasFreshAnalysis(uri) || !server.analysisInProgress(uri) {
			return nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return errAnalysisTimeout
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		signal := server.registerAnalysisWaiter(uri)
		waitContext, cancel := context.WithTimeout(ctx, remaining)
		select {
		case <-signal:
		case <-waitContext.Done():
		}
		cancel()
		server.removeAnalysisWaiter(uri, signal)
		if err := ctx.Err(); err != nil {
			return err
		}
	}
}
