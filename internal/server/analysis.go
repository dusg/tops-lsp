package server

import (
	"context"

	"tops-lsp/internal/document"
)

type analysisSnapshot struct {
	state      document.DocumentState
	context    parserContextSnapshot
	generation uint64
	epoch      uint64
}

type parserContextSnapshot struct {
	version int
}

type analysisTask struct {
	ctx       context.Context
	publisher diagnosticPublisher
	snapshot  analysisSnapshot
}

type analysisQueue struct {
	pending *analysisTask
	running bool
}

func (server *Server) enqueueAnalysis(ctx context.Context, publisher diagnosticPublisher, state document.DocumentState) {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot := server.newAnalysisSnapshot(state)
	server.mu.Lock()
	if server.analysisStopping {
		server.mu.Unlock()
		return
	}
	queue := server.analysisQueues[state.URI]
	if queue == nil {
		queue = &analysisQueue{}
		server.analysisQueues[state.URI] = queue
	}
	queue.pending = &analysisTask{ctx: ctx, publisher: publisher, snapshot: snapshot}
	if queue.running {
		server.mu.Unlock()
		return
	}
	queue.running = true
	server.analysisWG.Add(1)
	server.mu.Unlock()
	go server.runAnalysisQueue(state.URI, queue)
}

func (server *Server) runAnalysisQueue(uri string, queue *analysisQueue) {
	defer server.analysisWG.Done()
	for {
		server.mu.Lock()
		if server.analysisStopping || queue.pending == nil {
			queue.running = false
			if server.analysisQueues[uri] == queue {
				delete(server.analysisQueues, uri)
			}
			server.mu.Unlock()
			return
		}
		task := queue.pending
		queue.pending = nil
		server.mu.Unlock()

		if task.ctx.Err() != nil || !server.analysisCurrentSnapshot(task.snapshot) {
			continue
		}
		server.parseSnapshot(task.ctx, task.publisher, task.snapshot)
	}
}

func (server *Server) stopAnalysis() {
	server.mu.Lock()
	server.analysisStopping = true
	for _, queue := range server.analysisQueues {
		queue.pending = nil
	}
	server.mu.Unlock()
}

func (server *Server) stopAnalysisAndWait() {
	server.stopAnalysis()
	server.analysisWG.Wait()
}
