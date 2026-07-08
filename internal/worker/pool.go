package worker

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskflow/internal/handler"
	"taskflow/internal/models"
)

// Pool manages a fixed-size set of worker goroutines, registers the process in
// the worker registry on startup, and deregisters it on clean shutdown.
type Pool struct {
	id                uuid.UUID
	hostname          string
	worker            *Worker
	registry          WorkerRepo
	concurrency       int
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	heartbeatInterval time.Duration
}

// NewPool constructs a Pool. The pool does not start until Start is called.
func NewPool(
	jobRepo JobRepo,
	registry WorkerRepo,
	handlers *handler.Registry,
	concurrency int,
	heartbeatInterval time.Duration,
) *Pool {
	id := uuid.New()

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	w := NewWorker(id, jobRepo, registry, handlers, heartbeatInterval)

	return &Pool{
		id:                id,
		hostname:          hostname,
		worker:            w,
		registry:          registry,
		concurrency:       concurrency,
		heartbeatInterval: heartbeatInterval,
	}
}

// Start registers this process, launches the heartbeat, and spawns concurrency
// worker goroutines. It returns immediately; all work runs in the background.
// The provided ctx governs the lifetime of the background goroutines.
func (p *Pool) Start(ctx context.Context) {
	// Build a cancellable child context so Stop() can cancel it independently.
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	now := time.Now().UTC()

	// Step 1: register this worker process.
	if err := p.registry.Register(workerCtx, &models.WorkerRegistration{
		ID:              p.id,
		Hostname:        p.hostname,
		StartedAt:       now,
		LastHeartbeatAt: now,
		Status:          "active",
	}); err != nil {
		slog.Error("worker pool: registration failed",
			"worker_id", p.id,
			"error", err,
		)
	}

	// Step 2: start the registry heartbeat goroutine.
	StartHeartbeat(workerCtx, p.id, p.registry, p.heartbeatInterval)

	// Step 3 & 4: spawn concurrency worker goroutines.
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-workerCtx.Done():
					return
				default:
					if err := p.worker.ClaimAndExecute(workerCtx); err != nil {
						slog.Error("worker: claim and execute failed", "error", err)
					}
					// Avoid busy-looping when the queue is empty.
					time.Sleep(500 * time.Millisecond)
				}
			}
		}()
	}
}

// Stop cancels the worker context, waits for all goroutines to finish, and then
// deregisters this worker from the registry.
func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	if err := p.registry.Deregister(context.Background(), p.id); err != nil {
		slog.Error("worker pool: deregistration failed",
			"worker_id", p.id,
			"error", err,
		)
	}
}
