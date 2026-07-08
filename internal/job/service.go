package job

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"taskflow/internal/models"
)

// JobService defines business logic operations for managing jobs and execution logs.
type JobService interface {
	ListDLQ(ctx context.Context, userID uuid.UUID, page models.Pagination) ([]*models.Job, int, error)
	ListLogs(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, filter models.LogFilter, page models.Pagination) ([]*models.ExecutionLog, int, error)
}

// jobService is the concrete implementation of JobService.
type jobService struct {
	jobRepo JobRepository
	logRepo LogRepository
}

// NewJobService constructs a JobService backed by the given repositories.
func NewJobService(jobRepo JobRepository, logRepo LogRepository) JobService {
	return &jobService{
		jobRepo: jobRepo,
		logRepo: logRepo,
	}
}

// ListDLQ returns all exhausted (dead-letter) jobs for tasks belonging to userID.
// The jobs are paginated and ordered by most recent first.
func (s *jobService) ListDLQ(ctx context.Context, userID uuid.UUID, page models.Pagination) ([]*models.Job, int, error) {
	jobs, total, err := s.jobRepo.FindExhaustedByUser(ctx, userID, page)
	if err != nil {
		return nil, 0, fmt.Errorf("listing DLQ: %w", err)
	}
	return jobs, total, nil
}

// ListLogs returns execution logs for a specific task belonging to userID,
// with optional filtering by status and date range. Enforces ownership.
func (s *jobService) ListLogs(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, filter models.LogFilter, page models.Pagination) ([]*models.ExecutionLog, int, error) {
	logs, total, err := s.logRepo.FindByTask(ctx, taskID, userID, filter, page)
	if err != nil {
		return nil, 0, fmt.Errorf("listing logs: %w", err)
	}
	return logs, total, nil
}
