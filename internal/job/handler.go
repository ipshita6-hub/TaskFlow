package job

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/internal/api"
	"taskflow/internal/auth"
	"taskflow/internal/models"
)

// Handler exposes HTTP handlers for job and log endpoints.
type Handler struct {
	svc JobService
}

// NewHandler constructs a job Handler.
func NewHandler(svc JobService) *Handler {
	return &Handler{svc: svc}
}

// GetLogs handles GET /api/v1/tasks/:id/logs.
// Returns paginated execution logs for a task with optional filtering by status and date range.
// Enforces user ownership of the task.
func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(auth.ContextKeyUserClaims).(*auth.Claims)
	taskIDStr := chi.URLParam(r, "id")

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid task ID format")
		return
	}

	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// Build filter
	filter := models.LogFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		if status == "completed" || status == "failed" {
			filter.Status = &status
		}
	}

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = &t
		}
	}

	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = &t
		}
	}

	logs, total, err := h.svc.ListLogs(r.Context(), claims.UserID, taskID, filter, models.Pagination{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "task not found")
		} else {
			api.Write500(w, err)
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, models.PaginatedResponse{
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetDLQ handles GET /api/v1/jobs/dlq.
// Returns paginated list of exhausted (dead-letter) jobs for the authenticated user's tasks.
func (h *Handler) GetDLQ(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(auth.ContextKeyUserClaims).(*auth.Claims)

	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	jobs, total, err := h.svc.ListDLQ(r.Context(), claims.UserID, models.Pagination{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		api.Write500(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, models.PaginatedResponse{
		Data:     jobs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
