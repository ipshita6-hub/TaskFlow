package task

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"taskflow/internal/api"
	"taskflow/internal/auth"
	"taskflow/internal/models"
)

// Handler holds the HTTP handlers for the task endpoints.
type Handler struct {
	Service TaskService
}

// NewHandler constructs a task Handler.
func NewHandler(svc TaskService) *Handler {
	return &Handler{Service: svc}
}

// Create handles POST /tasks.
// Decodes a CreateTaskRequest, calls the service, and returns 201 with the
// created task.  Validation failures produce 422; other errors produce 500.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.Service.Create(r.Context(), claims.UserID, req)
	if err != nil {
		if errors.Is(err, models.ErrValidation) {
			api.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		api.Write500(w, err)
		return
	}

	api.WriteJSON(w, http.StatusCreated, task)
}

// List handles GET /tasks.
// Parses optional query params: status, page (default 1), page_size (default 20,
// max 100).  Returns a paginated response envelope.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()

	var filter models.TaskFilter
	if s := q.Get("status"); s != "" {
		filter.Status = &s
	}

	page := models.Pagination{
		Page:     parseIntParam(q.Get("page"), 1),
		PageSize: parseIntParam(q.Get("page_size"), 20),
	}
	if page.PageSize > 100 {
		page.PageSize = 100
	}

	tasks, total, err := h.Service.List(r.Context(), claims.UserID, filter, page)
	if err != nil {
		api.Write500(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, models.PaginatedResponse{
		Data:     tasks,
		Total:    total,
		Page:     page.Page,
		PageSize: page.PageSize,
	})
}

// Get handles GET /tasks/{id}.
// Extracts the id path parameter, verifies ownership, and returns the task.
// Missing or non-owned tasks produce 404.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "task not found")
		return
	}

	task, err := h.Service.Get(r.Context(), claims.UserID, taskID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		api.Write500(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, task)
}

// Update handles PATCH /tasks/{id}.
// Decodes an UpdateTaskRequest, applies partial updates, and returns the updated task.
// ErrNotFound → 404, ErrConflict → 409, ErrValidation → 422.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "task not found")
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.Service.Update(r.Context(), claims.UserID, taskID, req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			api.WriteError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, models.ErrConflict):
			api.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, models.ErrValidation):
			api.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			api.Write500(w, err)
		}
		return
	}

	api.WriteJSON(w, http.StatusOK, task)
}

// Delete handles DELETE /tasks/{id}.
// Soft-deletes the task and cancels its queued jobs.  Returns 204 on success.
// ErrNotFound → 404.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r.Context())
	if !ok {
		api.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "task not found")
		return
	}

	if err := h.Service.Delete(r.Context(), claims.UserID, taskID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			api.WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		api.Write500(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseIntParam parses s as an integer, returning defaultVal on any failure.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
