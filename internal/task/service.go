package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"taskflow/internal/models"
)

// CronParser is the interface the TaskService uses for cron expression handling.
// It is satisfied by the scheduler/cron package implementation.
type CronParser interface {
	// Parse validates and parses a cron expression, returning a NextScheduler.
	Parse(expr string) (NextScheduler, error)
	// NextTime returns the first time after `from` that the expression fires.
	NextTime(expr string, from time.Time) (time.Time, error)
}

// NextScheduler can compute the next scheduled time after a given time.Time.
type NextScheduler interface {
	Next(t time.Time) time.Time
}

// TaskService defines the business logic operations for managing tasks.
type TaskService interface {
	Create(ctx context.Context, userID uuid.UUID, req models.CreateTaskRequest) (*models.Task, error)
	List(ctx context.Context, userID uuid.UUID, filter models.TaskFilter, page models.Pagination) ([]*models.Task, int, error)
	Get(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*models.Task, error)
	Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req models.UpdateTaskRequest) (*models.Task, error)
	Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
}

// taskService is the concrete implementation of TaskService.
type taskService struct {
	repo       TaskRepository
	cronParser CronParser
}

// NewTaskService constructs a TaskService backed by the given repository and
// cron parser.
func NewTaskService(repo TaskRepository, cronParser CronParser) TaskService {
	return &taskService{
		repo:       repo,
		cronParser: cronParser,
	}
}

// Create validates the request, builds a Task, and persists it.
func (s *taskService) Create(ctx context.Context, userID uuid.UUID, req models.CreateTaskRequest) (*models.Task, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", models.ErrValidation)
	}

	now := time.Now().UTC()

	task := &models.Task{
		ID:           uuid.New(),
		UserID:       userID,
		Name:         req.Name,
		TaskType:     req.TaskType,
		Payload:      req.Payload,
		ScheduleType: req.ScheduleType,
		ScheduledAt:  req.ScheduledAt,
		CronExpr:     req.CronExpr,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Apply default retry policy when the caller omits it.
	if req.RetryPolicy != nil {
		task.MaxAttempts = req.RetryPolicy.MaxAttempts
		task.BackoffSeconds = req.RetryPolicy.BackoffSeconds
	} else {
		task.MaxAttempts = 3
		task.BackoffSeconds = 60
	}

	switch req.ScheduleType {
	case "one_time":
		if req.ScheduledAt == nil {
			return nil, fmt.Errorf("%w: scheduled_at is required for one_time tasks", models.ErrValidation)
		}
		if !req.ScheduledAt.After(now) {
			return nil, fmt.Errorf("%w: scheduled_at must be in the future", models.ErrValidation)
		}
		task.Status = "pending"
		task.NextRunAt = req.ScheduledAt

	case "recurring":
		if req.CronExpr == nil {
			return nil, fmt.Errorf("%w: cron_expr is required for recurring tasks", models.ErrValidation)
		}
		if _, err := s.cronParser.Parse(*req.CronExpr); err != nil {
			return nil, fmt.Errorf("%w: invalid cron_expr: %v", models.ErrValidation, err)
		}
		nextRun, err := s.cronParser.NextTime(*req.CronExpr, now)
		if err != nil {
			return nil, fmt.Errorf("%w: computing next_run_at: %v", models.ErrValidation, err)
		}
		task.Status = "active"
		task.NextRunAt = &nextRun

	default:
		return nil, fmt.Errorf("%w: schedule_type must be 'one_time' or 'recurring'", models.ErrValidation)
	}

	if err := s.repo.Insert(ctx, task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	return task, nil
}

// List returns a paginated, optionally-filtered collection of tasks for the user.
func (s *taskService) List(ctx context.Context, userID uuid.UUID, filter models.TaskFilter, page models.Pagination) ([]*models.Task, int, error) {
	tasks, total, err := s.repo.FindAllByUser(ctx, userID, filter, page)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, total, nil
}

// Get retrieves a single task, verifying ownership.
// Returns models.ErrNotFound if the task does not exist or belongs to another user.
func (s *taskService) Get(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*models.Task, error) {
	task, err := s.repo.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("fetching task: %w", err)
	}
	// Do not leak existence of tasks owned by other users.
	if task.UserID != userID {
		return nil, models.ErrNotFound
	}
	return task, nil
}

// Update applies partial changes from req to the task after verifying ownership.
// Returns models.ErrConflict when the task is currently running.
func (s *taskService) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.Get(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}

	if task.Status == "running" {
		return nil, fmt.Errorf("%w: task is currently running", models.ErrConflict)
	}

	now := time.Now().UTC()

	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Payload != nil {
		task.Payload = req.Payload
	}
	if req.ScheduledAt != nil {
		task.ScheduledAt = req.ScheduledAt
		task.NextRunAt = req.ScheduledAt
	}
	if req.CronExpr != nil {
		// Re-validate and recompute next_run_at when the cron expression changes.
		if _, err := s.cronParser.Parse(*req.CronExpr); err != nil {
			return nil, fmt.Errorf("%w: invalid cron_expr: %v", models.ErrValidation, err)
		}
		nextRun, err := s.cronParser.NextTime(*req.CronExpr, now)
		if err != nil {
			return nil, fmt.Errorf("%w: computing next_run_at: %v", models.ErrValidation, err)
		}
		task.CronExpr = req.CronExpr
		task.NextRunAt = &nextRun
	}
	if req.RetryPolicy != nil {
		task.MaxAttempts = req.RetryPolicy.MaxAttempts
		task.BackoffSeconds = req.RetryPolicy.BackoffSeconds
	}

	task.UpdatedAt = now

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("updating task: %w", err)
	}

	return task, nil
}

// Delete enforces ownership then soft-deletes the task and its queued jobs.
func (s *taskService) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	// Get validates ownership; if not owned it returns ErrNotFound.
	if _, err := s.Get(ctx, userID, taskID); err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, taskID, userID); err != nil {
		return fmt.Errorf("deleting task: %w", err)
	}

	return nil
}
