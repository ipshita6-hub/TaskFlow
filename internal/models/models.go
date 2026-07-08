package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors used across the service layer.
var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrValidation      = errors.New("validation error")
	ErrUnknownTaskType = errors.New("unknown task type")
)

// ─── Domain structs ────────────────────────────────────────────────────────────

// User represents an authenticated application user.
type User struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

// Task represents a scheduled unit of work created by a user.
type Task struct {
	ID             uuid.UUID       `db:"id"              json:"id"`
	UserID         uuid.UUID       `db:"user_id"         json:"-"`
	Name           string          `db:"name"            json:"name"`
	TaskType       string          `db:"task_type"       json:"task_type"`
	Payload        json.RawMessage `db:"payload"         json:"payload"`
	ScheduleType   string          `db:"schedule_type"   json:"schedule_type"`
	ScheduledAt    *time.Time      `db:"scheduled_at"    json:"scheduled_at,omitempty"`
	CronExpr       *string         `db:"cron_expr"       json:"cron_expr,omitempty"`
	NextRunAt      *time.Time      `db:"next_run_at"     json:"next_run_at,omitempty"`
	Status         string          `db:"status"          json:"status"`
	MaxAttempts    int             `db:"max_attempts"    json:"max_attempts"`
	BackoffSeconds int             `db:"backoff_seconds" json:"backoff_seconds"`
	CreatedAt      time.Time       `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"      json:"updated_at"`
}

// Job represents a single execution attempt of a task enqueued by the scheduler.
type Job struct {
	ID              uuid.UUID  `db:"id"                json:"id"`
	TaskID          uuid.UUID  `db:"task_id"           json:"task_id"`
	UserID          uuid.UUID  `db:"user_id"           json:"-"`
	Status          string     `db:"status"            json:"status"`
	Attempt         int        `db:"attempt"           json:"attempt"`
	EnqueuedAt      time.Time  `db:"enqueued_at"       json:"enqueued_at"`
	RunAfter        time.Time  `db:"run_after"         json:"run_after"`
	ClaimedAt       *time.Time `db:"claimed_at"        json:"claimed_at,omitempty"`
	CompletedAt     *time.Time `db:"completed_at"      json:"completed_at,omitempty"`
	WorkerID        *uuid.UUID `db:"worker_id"         json:"worker_id,omitempty"`
	LastHeartbeatAt *time.Time `db:"last_heartbeat_at" json:"last_heartbeat_at,omitempty"`
	ErrorMessage    *string    `db:"error_message"     json:"error_message,omitempty"`
}

// ExecutionLog records the outcome of a single job execution attempt.
type ExecutionLog struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	JobID        uuid.UUID `db:"job_id"        json:"job_id"`
	TaskID       uuid.UUID `db:"task_id"       json:"task_id"`
	UserID       uuid.UUID `db:"user_id"       json:"-"`
	Attempt      int       `db:"attempt"       json:"attempt"`
	Status       string    `db:"status"        json:"status"`
	StartedAt    time.Time `db:"started_at"    json:"started_at"`
	EndedAt      time.Time `db:"ended_at"      json:"ended_at"`
	DurationMs   int64     `db:"duration_ms"   json:"duration_ms"`
	Output       *string   `db:"output"        json:"output,omitempty"`
	ErrorMessage *string   `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
}

// WorkerRegistration tracks a running worker process in the worker registry.
type WorkerRegistration struct {
	ID              uuid.UUID `db:"id"                json:"id"`
	Hostname        string    `db:"hostname"          json:"hostname"`
	StartedAt       time.Time `db:"started_at"        json:"started_at"`
	LastHeartbeatAt time.Time `db:"last_heartbeat_at" json:"last_heartbeat_at"`
	Status          string    `db:"status"            json:"status"`
}

// RetryPolicy holds the retry configuration for a task (JSON only, no db tags).
type RetryPolicy struct {
	MaxAttempts    int `json:"max_attempts"`
	BackoffSeconds int `json:"backoff_seconds"`
}

// ─── Request / Response DTOs ───────────────────────────────────────────────────

// RegisterRequest is the payload for POST /api/v1/auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the payload for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateTaskRequest is the payload for POST /api/v1/tasks.
type CreateTaskRequest struct {
	Name         string          `json:"name"`
	TaskType     string          `json:"task_type"`
	Payload      json.RawMessage `json:"payload"`
	ScheduleType string          `json:"schedule_type"`
	ScheduledAt  *time.Time      `json:"scheduled_at,omitempty"`
	CronExpr     *string         `json:"cron_expr,omitempty"`
	RetryPolicy  *RetryPolicy    `json:"retry_policy,omitempty"`
}

// UpdateTaskRequest is the payload for PATCH /api/v1/tasks/:id.
type UpdateTaskRequest struct {
	Name        *string         `json:"name,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	CronExpr    *string         `json:"cron_expr,omitempty"`
	ScheduledAt *time.Time      `json:"scheduled_at,omitempty"`
	RetryPolicy *RetryPolicy    `json:"retry_policy,omitempty"`
}

// TaskFilter carries optional filter criteria for listing tasks.
type TaskFilter struct {
	Status *string `json:"status,omitempty"`
}

// Pagination carries page / page_size parameters.
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Offset returns the row offset for the current page (0-based).
// If Page is less than 1 it defaults to 1.
func (p Pagination) Offset() int {
	page := p.Page
	if page < 1 {
		page = 1
	}
	return (page - 1) * p.Limit()
}

// Limit returns the effective page size, defaulting to 20 when unset.
func (p Pagination) Limit() int {
	if p.PageSize <= 0 {
		return 20
	}
	return p.PageSize
}

// LogFilter carries optional filter criteria for listing execution logs.
type LogFilter struct {
	Status *string    `json:"status,omitempty"`
	From   *time.Time `json:"from,omitempty"`
	To     *time.Time `json:"to,omitempty"`
}

// ValidationResult is the response from GET /api/v1/schedule/validate.
type ValidationResult struct {
	Valid      bool        `json:"valid"`
	NextTimes  []time.Time `json:"next_times,omitempty"`
	Error      *ParseError `json:"error,omitempty"`
}

// ParseError describes a cron expression parse failure with field-level detail.
type ParseError struct {
	Field    string `json:"field"`
	Position int    `json:"position"`
	Message  string `json:"message"`
}

// PaginatedResponse wraps a page of results with metadata.
type PaginatedResponse struct {
	Data     interface{} `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
