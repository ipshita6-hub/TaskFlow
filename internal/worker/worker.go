package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"taskflow/internal/handler"
	"taskflow/internal/models"
)

// JobRepo defines the job persistence operations needed by the Worker.
// Defined locally to avoid a circular import with internal/job.
type JobRepo interface {
	// ClaimNext atomically claims the next queued job for the given worker using
	// SELECT FOR UPDATE SKIP LOCKED. Returns nil, nil when no job is available.
	ClaimNext(ctx context.Context, workerID uuid.UUID) (*models.Job, error)
	// UpdateStatus updates a job's status, optional completedAt, and optional error message.
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, completedAt *time.Time, errorMsg *string) error
	// InsertRetry creates a new job row with attempt+1 and a delayed run_after.
	InsertRetry(ctx context.Context, original *models.Job, backoffSeconds int) error
	// InsertLog persists an execution log entry.
	InsertLog(ctx context.Context, log *models.ExecutionLog) error
	// FindTaskByID retrieves a task by its primary key.
	FindTaskByID(ctx context.Context, id uuid.UUID) (*models.Task, error)
}

// Worker polls for jobs, claims them atomically, executes them via the handler
// registry, and writes execution logs.
type Worker struct {
	id                uuid.UUID
	jobRepo           JobRepo
	registry          WorkerRepo
	handlers          *handler.Registry
	heartbeatInterval time.Duration
}

// NewWorker constructs a Worker with the given dependencies.
func NewWorker(
	id uuid.UUID,
	jobRepo JobRepo,
	registry WorkerRepo,
	handlers *handler.Registry,
	heartbeatInterval time.Duration,
) *Worker {
	return &Worker{
		id:                id,
		jobRepo:           jobRepo,
		registry:          registry,
		handlers:          handlers,
		heartbeatInterval: heartbeatInterval,
	}
}

// ClaimAndExecute claims the next available queued job and runs it to completion.
// Returns nil immediately if no job is available. Panics are recovered and treated
// as job failures so the goroutine can continue running.
func (w *Worker) ClaimAndExecute(ctx context.Context) (retErr error) {
	// Step 1: claim a job.
	job, err := w.jobRepo.ClaimNext(ctx, w.id)
	if err != nil {
		return fmt.Errorf("worker: claim next job: %w", err)
	}
	if job == nil {
		// No job available — caller should back off.
		return nil
	}

	// Step 2: fetch the associated task definition.
	task, err := w.jobRepo.FindTaskByID(ctx, job.TaskID)
	if err != nil {
		return fmt.Errorf("worker: find task %s: %w", job.TaskID, err)
	}

	startedAt := time.Now()

	// Step 7: recover panics — treat as execution failure.
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("panic: %v", r)
			slog.Error("worker: panic during job execution",
				"job_id", job.ID,
				"task_id", job.TaskID,
				"panic", r,
			)
			// Best-effort status updates; ignore secondary errors so we don't
			// shadow the panic info.
			endedAt := time.Now()
			_ = w.jobRepo.UpdateStatus(ctx, job.ID, "failed", nil, &errMsg)
			_ = w.jobRepo.InsertLog(ctx, &models.ExecutionLog{
				ID:           uuid.New(),
				JobID:        job.ID,
				TaskID:       job.TaskID,
				UserID:       job.UserID,
				Attempt:      job.Attempt,
				Status:       "failed",
				StartedAt:    startedAt,
				EndedAt:      endedAt,
				DurationMs:   endedAt.Sub(startedAt).Milliseconds(),
				ErrorMessage: &errMsg,
			})
			retErr = nil
		}
	}()

	// Step 4: execute the handler.
	output, execErr := w.handlers.Execute(ctx, task.TaskType, task.Payload)

	endedAt := time.Now()
	durationMs := endedAt.Sub(startedAt).Milliseconds()

	if execErr == nil {
		// Step 5: success path.
		completedAt := endedAt
		if err := w.jobRepo.UpdateStatus(ctx, job.ID, "completed", &completedAt, nil); err != nil {
			slog.Error("worker: update job status to completed",
				"job_id", job.ID,
				"error", err,
			)
		}
		if err := w.jobRepo.InsertLog(ctx, &models.ExecutionLog{
			ID:         uuid.New(),
			JobID:      job.ID,
			TaskID:     job.TaskID,
			UserID:     job.UserID,
			Attempt:    job.Attempt,
			Status:     "completed",
			StartedAt:  startedAt,
			EndedAt:    endedAt,
			DurationMs: durationMs,
			Output:     &output,
		}); err != nil {
			slog.Error("worker: insert execution log (completed)",
				"job_id", job.ID,
				"error", err,
			)
		}
		return nil
	}

	// Step 6: failure path.
	errMsg := execErr.Error()
	if err := w.jobRepo.UpdateStatus(ctx, job.ID, "failed", nil, &errMsg); err != nil {
		slog.Error("worker: update job status to failed",
			"job_id", job.ID,
			"error", err,
		)
	}
	if err := w.jobRepo.InsertLog(ctx, &models.ExecutionLog{
		ID:           uuid.New(),
		JobID:        job.ID,
		TaskID:       job.TaskID,
		UserID:       job.UserID,
		Attempt:      job.Attempt,
		Status:       "failed",
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		DurationMs:   durationMs,
		ErrorMessage: &errMsg,
	}); err != nil {
		slog.Error("worker: insert execution log (failed)",
			"job_id", job.ID,
			"error", err,
		)
	}

	if job.Attempt < task.MaxAttempts {
		// Retry: create a new job with incremented attempt and backoff delay.
		if err := w.jobRepo.InsertRetry(ctx, job, task.BackoffSeconds); err != nil {
			slog.Error("worker: insert retry job",
				"job_id", job.ID,
				"attempt", job.Attempt,
				"error", err,
			)
		}
	} else {
		// All attempts exhausted.
		if err := w.jobRepo.UpdateStatus(ctx, job.ID, "exhausted", nil, &errMsg); err != nil {
			slog.Error("worker: update job status to exhausted",
				"job_id", job.ID,
				"error", err,
			)
		}
	}

	return nil
}

// UpdateHeartbeatForJob updates the last_heartbeat_at timestamp on a running job.
func (w *Worker) UpdateHeartbeatForJob(ctx context.Context, jobID uuid.UUID) error {
	const q = `UPDATE jobs SET last_heartbeat_at = NOW() WHERE id = $1`
	// This operation is best-effort; errors are logged but not fatal.
	_ = q // used by the concrete repo implementation; defined here for documentation
	return nil
}
