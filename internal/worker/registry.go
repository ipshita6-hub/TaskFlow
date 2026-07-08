package worker

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"taskflow/internal/models"
)

// WorkerRepo defines the persistence operations for worker registry management.
type WorkerRepo interface {
	Register(ctx context.Context, w *models.WorkerRegistration) error
	UpdateHeartbeat(ctx context.Context, id uuid.UUID) error
	Deregister(ctx context.Context, id uuid.UUID) error
	ListAll(ctx context.Context) ([]*models.WorkerRegistration, error)
}

type postgresWorkerRepo struct {
	db *sqlx.DB
}

// NewPostgresWorkerRepo returns a WorkerRepo backed by the given PostgreSQL connection.
func NewPostgresWorkerRepo(db *sqlx.DB) WorkerRepo {
	return &postgresWorkerRepo{db: db}
}

// Register inserts a new worker registration row into worker_registry.
func (r *postgresWorkerRepo) Register(ctx context.Context, w *models.WorkerRegistration) error {
	const q = `
		INSERT INTO worker_registry (id, hostname, started_at, last_heartbeat_at, status)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, q,
		w.ID,
		w.Hostname,
		w.StartedAt,
		w.LastHeartbeatAt,
		w.Status,
	)
	return err
}

// UpdateHeartbeat updates the last_heartbeat_at timestamp for the given worker ID.
func (r *postgresWorkerRepo) UpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE worker_registry SET last_heartbeat_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// Deregister marks the given worker as stale.
func (r *postgresWorkerRepo) Deregister(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE worker_registry SET status = 'stale' WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ListAll returns all worker registrations ordered by started_at descending.
func (r *postgresWorkerRepo) ListAll(ctx context.Context) ([]*models.WorkerRegistration, error) {
	const q = `SELECT * FROM worker_registry ORDER BY started_at DESC`
	var workers []*models.WorkerRegistration
	if err := r.db.SelectContext(ctx, &workers, q); err != nil {
		return nil, err
	}
	return workers, nil
}
