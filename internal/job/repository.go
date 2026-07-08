package job

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"taskflow/internal/models"
)

// JobRepository defines job persistence operations.
type JobRepository interface {
	Insert(ctx context.Context, job *models.Job) error
	ClaimNext(ctx context.Context, workerID uuid.UUID) (*models.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, completedAt *time.Time, errorMsg *string) error
	InsertRetry(ctx context.Context, original *models.Job, backoffSeconds int) error
	InsertLog(ctx context.Context, log *models.ExecutionLog) error
	FindTaskByID(ctx context.Context, taskID uuid.UUID) (*models.Task, error)
	ResetStaleJobs(ctx context.Context, threshold time.Time) error
	FindExhaustedByUser(ctx context.Context, userID uuid.UUID, page models.Pagination) ([]*models.Job, int, error)
}

// LogRepository defines execution log persistence operations.
type LogRepository interface {
	Insert(ctx context.Context, log *models.ExecutionLog) error
	FindByTask(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, filter models.LogFilter, page models.Pagination) ([]*models.ExecutionLog, int, error)
}

// postgresJobRepository is a sqlx-backed implementation of JobRepository.
type postgresJobRepository struct {
	db *sqlx.DB
}

// NewPostgresJobRepository constructs a JobRepository backed by PostgreSQL.
func NewPostgresJobRepository(db *sqlx.DB) JobRepository {
	return &postgresJobRepository{db: db}
}

// Insert persists a new job row.
func (r *postgresJobRepository) Insert(ctx context.Context, job *models.Job) error {
	const q = `INSERT INTO jobs (id, task_id, user_id, status, attempt, enqueued_at, run_after)
	           VALUES (:id, :task_id, :user_id, :status, :attempt, :enqueued_at, :run_after)`
	if _, err := r.db.NamedExecContext(ctx, q, job); err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

// ClaimNext atomically claims a single queued job available for execution.
// Uses SELECT FOR UPDATE SKIP LOCKED to avoid duplicate claims across concurrent workers.
// Returns the claimed job with status updated to 'running' and claimed_at set.
// Returns nil, nil if no job is available.
func (r *postgresJobRepository) ClaimNext(ctx context.Context, workerID uuid.UUID) (*models.Job, error) {
	const q = `WITH claimed AS (
		SELECT id FROM jobs
		WHERE status = 'queued' AND run_after <= NOW()
		ORDER BY enqueued_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	)
	UPDATE jobs SET status = 'running', claimed_at = NOW(), worker_id = $1, last_heartbeat_at = NOW()
	WHERE id = (SELECT id FROM claimed)
	RETURNING *`

	var job models.Job
	err := r.db.GetContext(ctx, &job, q, workerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("claim next job: %w", err)
	}
	return &job, nil
}

// UpdateStatus updates a job's status, optionally its completion timestamp, and optionally an error message.
func (r *postgresJobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, completedAt *time.Time, errorMsg *string) error {
	const q = `UPDATE jobs SET status = $1, completed_at = $2, error_message = $3 WHERE id = $4`
	if _, err := r.db.ExecContext(ctx, q, status, completedAt, errorMsg, id); err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

// InsertRetry creates a new queued job for the next retry attempt, with run_after
// calculated as: now + (backoff_seconds × attempt) seconds.
func (r *postgresJobRepository) InsertRetry(ctx context.Context, original *models.Job, backoffSeconds int) error {
	nextAttempt := original.Attempt + 1
	runAfter := time.Now().UTC().Add(time.Duration(backoffSeconds*nextAttempt) * time.Second)

	newJob := &models.Job{
		ID:         uuid.New(),
		TaskID:     original.TaskID,
		UserID:     original.UserID,
		Status:     "queued",
		Attempt:    nextAttempt,
		EnqueuedAt: time.Now().UTC(),
		RunAfter:   runAfter,
	}

	const q = `INSERT INTO jobs (id, task_id, user_id, status, attempt, enqueued_at, run_after)
	           VALUES (:id, :task_id, :user_id, :status, :attempt, :enqueued_at, :run_after)`
	if _, err := r.db.NamedExecContext(ctx, q, newJob); err != nil {
		return fmt.Errorf("insert retry job: %w", err)
	}
	return nil
}

// InsertLog persists an execution log entry.
func (r *postgresJobRepository) InsertLog(ctx context.Context, log *models.ExecutionLog) error {
	const q = `INSERT INTO execution_logs 
	           (id, job_id, task_id, user_id, attempt, status, started_at, ended_at, duration_ms, output, error_message, created_at)
	           VALUES (:id, :job_id, :task_id, :user_id, :attempt, :status, :started_at, :ended_at, :duration_ms, :output, :error_message, :created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, log); err != nil {
		return fmt.Errorf("insert execution log: %w", err)
	}
	return nil
}

// FindTaskByID retrieves a task by ID to fetch its max_attempts and backoff_seconds.
func (r *postgresJobRepository) FindTaskByID(ctx context.Context, taskID uuid.UUID) (*models.Task, error) {
	const q = `SELECT * FROM tasks WHERE id = $1`
	var task models.Task
	if err := r.db.GetContext(ctx, &task, q, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("find task by id: %w", err)
	}
	return &task, nil
}

// ResetStaleJobs resets running jobs whose last heartbeat is older than threshold
// back to queued status so they can be re-claimed by a healthy worker.
func (r *postgresJobRepository) ResetStaleJobs(ctx context.Context, threshold time.Time) error {
	const q = `UPDATE jobs 
	           SET status = 'queued', worker_id = NULL, last_heartbeat_at = NULL, claimed_at = NULL
	           WHERE status = 'running' AND last_heartbeat_at < $1`
	if _, err := r.db.ExecContext(ctx, q, threshold); err != nil {
		return fmt.Errorf("reset stale jobs: %w", err)
	}
	return nil
}

// FindExhaustedByUser retrieves all exhausted jobs belonging to tasks owned by userID,
// paginated and ordered by most recent first.
func (r *postgresJobRepository) FindExhaustedByUser(ctx context.Context, userID uuid.UUID, page models.Pagination) ([]*models.Job, int, error) {
	// Count total exhausted jobs for this user
	const countQ = `SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND status = 'exhausted'`
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, userID); err != nil {
		return nil, 0, fmt.Errorf("count exhausted jobs: %w", err)
	}

	// Fetch paginated results
	const q = `SELECT * FROM jobs 
	           WHERE user_id = $1 AND status = 'exhausted'
	           ORDER BY enqueued_at DESC
	           LIMIT $2 OFFSET $3`
	var jobs []*models.Job
	if err := r.db.SelectContext(ctx, &jobs, q, userID, page.Limit(), page.Offset()); err != nil {
		return nil, 0, fmt.Errorf("find exhausted jobs: %w", err)
	}
	return jobs, total, nil
}

// ─── PostgreSQL LogRepository ──────────────────────────────────────────────────

// postgresLogRepository is a sqlx-backed implementation of LogRepository.
type postgresLogRepository struct {
	db *sqlx.DB
}

// NewPostgresLogRepository constructs a LogRepository backed by PostgreSQL.
func NewPostgresLogRepository(db *sqlx.DB) LogRepository {
	return &postgresLogRepository{db: db}
}

// Insert persists a new execution log entry.
func (r *postgresLogRepository) Insert(ctx context.Context, log *models.ExecutionLog) error {
	const q = `INSERT INTO execution_logs 
	           (id, job_id, task_id, user_id, attempt, status, started_at, ended_at, duration_ms, output, error_message, created_at)
	           VALUES (:id, :job_id, :task_id, :user_id, :attempt, :status, :started_at, :ended_at, :duration_ms, :output, :error_message, :created_at)`
	if _, err := r.db.NamedExecContext(ctx, q, log); err != nil {
		return fmt.Errorf("insert execution log: %w", err)
	}
	return nil
}

// FindByTask retrieves execution logs for a given task, with optional filtering by status and date range.
// Enforces user ownership by verifying user_id matches. Returns ErrNotFound if the task doesn't belong to userID.
func (r *postgresLogRepository) FindByTask(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, filter models.LogFilter, page models.Pagination) ([]*models.ExecutionLog, int, error) {
	// First verify the task belongs to the user
	const taskCheckQ = `SELECT user_id FROM tasks WHERE id = $1`
	var owner uuid.UUID
	if err := r.db.GetContext(ctx, &owner, taskCheckQ, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, models.ErrNotFound
		}
		return nil, 0, fmt.Errorf("find task owner: %w", err)
	}
	if owner != userID {
		return nil, 0, models.ErrNotFound
	}

	// Build the query dynamically based on filters
	baseQ := `SELECT * FROM execution_logs WHERE task_id = $1`
	args := []interface{}{taskID}
	argNum := 2

	if filter.Status != nil {
		baseQ += fmt.Sprintf(` AND status = $%d`, argNum)
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.From != nil {
		baseQ += fmt.Sprintf(` AND started_at >= $%d`, argNum)
		args = append(args, *filter.From)
		argNum++
	}
	if filter.To != nil {
		baseQ += fmt.Sprintf(` AND ended_at <= $%d`, argNum)
		args = append(args, *filter.To)
		argNum++
	}

	// Count total before pagination
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM execution_logs WHERE task_id = $1`)
	countArgs := []interface{}{taskID}
	if filter.Status != nil {
		countQ += ` AND status = $2`
		countArgs = append(countArgs, *filter.Status)
		if filter.From != nil {
			countQ += ` AND started_at >= $3`
			countArgs = append(countArgs, *filter.From)
			if filter.To != nil {
				countQ += ` AND ended_at <= $4`
				countArgs = append(countArgs, *filter.To)
			}
		} else if filter.To != nil {
			countQ += ` AND ended_at <= $3`
			countArgs = append(countArgs, *filter.To)
		}
	} else if filter.From != nil {
		countQ += ` AND started_at >= $2`
		countArgs = append(countArgs, *filter.From)
		if filter.To != nil {
			countQ += ` AND ended_at <= $3`
			countArgs = append(countArgs, *filter.To)
		}
	} else if filter.To != nil {
		countQ += ` AND ended_at <= $2`
		countArgs = append(countArgs, *filter.To)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQ, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count logs: %w", err)
	}

	// Add ordering and pagination
	baseQ += ` ORDER BY started_at DESC LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)
	args = append(args, page.Limit(), page.Offset())

	var logs []*models.ExecutionLog
	if err := r.db.SelectContext(ctx, &logs, baseQ, args...); err != nil {
		return nil, 0, fmt.Errorf("find logs: %w", err)
	}
	return logs, total, nil
}
