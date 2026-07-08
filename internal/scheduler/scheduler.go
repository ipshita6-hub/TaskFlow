package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"taskflow/internal/models"
	"taskflow/internal/task"
)

// JobRepository defines the job persistence operations needed by the Scheduler.
// Defined locally to avoid a circular import with internal/job.
type JobRepository interface {
	Insert(ctx context.Context, job *models.Job) error
	ResetStaleJobs(ctx context.Context, threshold time.Time) error
}

// Scheduler polls for due tasks, enqueues jobs, and reclaims stale jobs.
type Scheduler struct {
	taskRepo       task.TaskRepository
	jobRepo        JobRepository
	cronParser     *CronParser
	tickInterval   time.Duration
	staleThreshold time.Duration
}

// NewScheduler constructs a Scheduler with the given dependencies.
func NewScheduler(
	taskRepo task.TaskRepository,
	jobRepo JobRepository,
	cronParser *CronParser,
	tickInterval time.Duration,
	staleThreshold time.Duration,
) *Scheduler {
	return &Scheduler{
		taskRepo:       taskRepo,
		jobRepo:        jobRepo,
		cronParser:     cronParser,
		tickInterval:   tickInterval,
		staleThreshold: staleThreshold,
	}
}

// Start launches two background goroutines that run until ctx is cancelled:
//   - An enqueue loop that ticks at tickInterval, calling EnqueueDueJobs.
//   - A reclaim loop that ticks every 30 seconds, calling ReclaimStaleJobs.
func (s *Scheduler) Start(ctx context.Context) {
	// Enqueue due jobs loop
	go func() {
		ticker := time.NewTicker(s.tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.EnqueueDueJobs(ctx); err != nil {
					slog.Error("scheduler: enqueue due jobs failed", "error", err)
				}
			}
		}
	}()

	// Reclaim stale jobs loop
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.ReclaimStaleJobs(ctx); err != nil {
					slog.Error("scheduler: reclaim stale jobs failed", "error", err)
				}
			}
		}
	}()
}

// EnqueueDueJobs queries for active tasks whose next_run_at is at or before now,
// inserts a queued job for each, and advances (or completes) the task schedule.
func (s *Scheduler) EnqueueDueJobs(ctx context.Context) error {
	now := time.Now().UTC()

	tasks, err := s.taskRepo.FindDueActive(ctx, now)
	if err != nil {
		return err
	}

	for _, t := range tasks {
		job := models.Job{
			ID:         uuid.New(),
			TaskID:     t.ID,
			UserID:     t.UserID,
			Status:     "queued",
			Attempt:    1,
			EnqueuedAt: now,
			RunAfter:   now,
		}

		if err := s.jobRepo.Insert(ctx, &job); err != nil {
			slog.Error("scheduler: insert job failed",
				"task_id", t.ID,
				"error", err,
			)
			continue
		}

		if t.CronExpr != nil {
			// Recurring task: advance to the next scheduled time.
			nextTime, err := s.cronParser.NextTime(*t.CronExpr, now)
			if err != nil {
				slog.Error("scheduler: compute next cron time failed",
					"task_id", t.ID,
					"cron_expr", *t.CronExpr,
					"error", err,
				)
				continue
			}
			if err := s.taskRepo.UpdateNextRunAt(ctx, t.ID, &nextTime, "active"); err != nil {
				slog.Error("scheduler: update next_run_at failed",
					"task_id", t.ID,
					"error", err,
				)
			}
		} else {
			// One-time task: mark as completed.
			if err := s.taskRepo.UpdateNextRunAt(ctx, t.ID, nil, "completed"); err != nil {
				slog.Error("scheduler: mark task completed failed",
					"task_id", t.ID,
					"error", err,
				)
			}
		}
	}

	return nil
}

// ReclaimStaleJobs resets running jobs whose last heartbeat predates the stale
// threshold back to queued so they can be re-claimed by a healthy worker.
func (s *Scheduler) ReclaimStaleJobs(ctx context.Context) error {
	threshold := time.Now().UTC().Add(-s.staleThreshold)
	if err := s.jobRepo.ResetStaleJobs(ctx, threshold); err != nil {
		slog.Error("scheduler: reset stale jobs failed", "error", err)
		return err
	}
	return nil
}

// ComputeNextTimes delegates to the underlying CronParser to return the next n
// UTC times that the given cron expression fires, starting from now.
func (s *Scheduler) ComputeNextTimes(expr string, n int) ([]time.Time, error) {
	return s.cronParser.ComputeNextTimes(expr, n)
}
