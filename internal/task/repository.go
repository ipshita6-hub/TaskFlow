package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"taskflow/internal/models"
)

// TaskRepository defines persistence operations for Task entities.
type TaskRepository interface {
	// Insert persists a new task row.
	Insert(ctx context.Context, task *models.Task) error

	// FindByID retrieves a task by its primary key.
	// Returns models.ErrNotFound when no row exists.
	FindByID(ctx context.Context, id uuid.UUID) (*models.Task, error)

	// FindAllByUser returns a paginated, optionally-filtered list of tasks for a
	// user together with the total count across all pages.
	FindAllByUser(ctx context.Context, userID uuid.UUID, filter models.TaskFilter, page models.Pagination) ([]*models.Task, int, error)

	// Update writes changed fields (name, payload, cron_expr, scheduled_at,
	// max_attempts, backoff_seconds, next_run_at, status, updated_at) back to
	// the database for the given task.
	Update(ctx context.Context, task *models.Task) error

	// SoftDelete marks the task as deleted and cancels all queued jobs for it
	// inside a single transaction.
	SoftDelete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// FindDueActive returns all active tasks whose next_run_at is at or before now.
	FindDueActive(ctx context.Context, now time.Time) ([]*models.Task, error)

	// UpdateNextRunAt updates the next_run_at and status fields of a task.
	UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRunAt *time.Time, status string) error
}

// postgresTaskRepository is a sqlx-backed implementation of TaskRepository.
type postgresTaskRepository struct {
	db *sqlx.DB
}

// NewPostgresTaskRepository constructs a TaskRepository backed by PostgreSQL.
func NewPostgresTaskRepository(db *sqlx.DB) TaskRepository {
	return &postgresTaskRepository{db: db}
}

// Insert persists a new task row.
func (r *postgresTaskRepository) Insert(ctx context.Context, task *models.Task) error {
	const q = `
		INSERT INTO tasks (
			id, user_id, name, task_type, payload, schedule_type,
			scheduled_at, cron_expr, next_run_at, status,
			max_attempts, backoff_seconds, created_at, updated_at
		) VALUES (
			:id, :user_id, :name, :task_type, :payload, :schedule_type,
			:scheduled_at, :cron_expr, :next_run_at, :status,
			:max_attempts, :backoff_seconds, :created_at, :updated_at
		)`
	if _, err := r.db.NamedExecContext(ctx, q, task); err != nil {
		return fmt.Errorf("task insert: %w", err)
	}
	return nil
}

// FindByID retrieves a single task by primary key.
func (r *postgresTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	const q = `SELECT * FROM tasks WHERE id = $1`
	var task models.Task
	if err := r.db.GetContext(ctx, &task, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("task find by id: %w", err)
	}
	return &task, nil
}

// FindAllByUser returns a paginated list of tasks owned by userID, optionally
// filtered by status, along with the total row count.
func (r *postgresTaskRepository) FindAllByUser(
	ctx context.Context,
	userID uuid.UUID,
	filter models.TaskFilter,
	page models.Pagination,
) ([]*models.Task, int, error) {
	// Build the WHERE clause dynamically.
	conditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIdx := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total matching rows.
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM tasks WHERE %s", where)
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, args...); err != nil {
		return nil, 0, fmt.Errorf("task count: %w", err)
	}

	// Fetch the requested page.
	dataQ := fmt.Sprintf(
		`SELECT * FROM tasks WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, page.Limit(), page.Offset())

	var tasks []*models.Task
	if err := r.db.SelectContext(ctx, &tasks, dataQ, args...); err != nil {
		return nil, 0, fmt.Errorf("task list: %w", err)
	}

	return tasks, total, nil
}

// Update writes the mutable fields of task back to the database.
func (r *postgresTaskRepository) Update(ctx context.Context, task *models.Task) error {
	const q = `
		UPDATE tasks SET
			name            = :name,
			payload         = :payload,
			cron_expr       = :cron_expr,
			scheduled_at    = :scheduled_at,
			max_attempts    = :max_attempts,
			backoff_seconds = :backoff_seconds,
			next_run_at     = :next_run_at,
			status          = :status,
			updated_at      = :updated_at
		WHERE id = :id`
	if _, err := r.db.NamedExecContext(ctx, q, task); err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	return nil
}

// SoftDelete marks the task as deleted and cancels all queued jobs for that
// task inside a single transaction.
func (r *postgresTaskRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("soft delete begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const deleteTask = `
		UPDATE tasks SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND user_id = $2`
	if _, err = tx.ExecContext(ctx, deleteTask, id, userID); err != nil {
		return fmt.Errorf("soft delete task: %w", err)
	}

	const cancelJobs = `
		UPDATE jobs SET status = 'cancelled'
		WHERE task_id = $1 AND status = 'queued'`
	if _, err = tx.ExecContext(ctx, cancelJobs, id); err != nil {
		return fmt.Errorf("cancel queued jobs: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("soft delete commit: %w", err)
	}
	return nil
}

// FindDueActive returns all active tasks whose next_run_at is at or before now.
func (r *postgresTaskRepository) FindDueActive(ctx context.Context, now time.Time) ([]*models.Task, error) {
	const q = `SELECT * FROM tasks WHERE next_run_at <= $1 AND status = 'active'`
	var tasks []*models.Task
	if err := r.db.SelectContext(ctx, &tasks, q, now); err != nil {
		return nil, fmt.Errorf("find due active tasks: %w", err)
	}
	return tasks, nil
}

// UpdateNextRunAt sets the next_run_at and status fields for a task.
func (r *postgresTaskRepository) UpdateNextRunAt(ctx context.Context, id uuid.UUID, nextRunAt *time.Time, status string) error {
	const q = `
		UPDATE tasks SET next_run_at = $1, status = $2, updated_at = NOW()
		WHERE id = $3`
	if _, err := r.db.ExecContext(ctx, q, nextRunAt, status, id); err != nil {
		return fmt.Errorf("update next_run_at: %w", err)
	}
	return nil
}
