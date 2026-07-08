package worker

import (
	"net/http"

	"taskflow/internal/api"
)

// Handler exposes HTTP handlers for worker-related endpoints.
type Handler struct {
	repo WorkerRepo
}

// NewHandler constructs a worker Handler.
func NewHandler(repo WorkerRepo) *Handler {
	return &Handler{repo: repo}
}

// ListWorkers handles GET /api/v1/workers.
// Returns a list of all registered workers. No authentication required (read-only diagnostic endpoint).
func (h *Handler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := h.repo.ListAll(r.Context())
	if err != nil {
		api.Write500(w, err)
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": workers,
	})
}
