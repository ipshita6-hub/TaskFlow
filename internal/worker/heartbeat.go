package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// StartHeartbeat launches a goroutine that calls registry.UpdateHeartbeat every
// interval until ctx is cancelled. It returns immediately; the goroutine runs
// in the background.
func StartHeartbeat(ctx context.Context, workerID uuid.UUID, registry WorkerRepo, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := registry.UpdateHeartbeat(ctx, workerID); err != nil {
					slog.Error("worker: heartbeat update failed",
						"worker_id", workerID,
						"error", err,
					)
				}
			}
		}
	}()
}
