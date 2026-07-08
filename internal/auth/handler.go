package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"taskflow/internal/api"
	"taskflow/internal/models"
)

// Handler exposes HTTP handlers for authentication endpoints.
type Handler struct {
	svc            AuthService
	jwtExpiryHours int
}

// NewHandler constructs an auth Handler.
// jwtExpiryHours must match the value used by the AuthService so that
// the expires_at field in the login response is accurate.
func NewHandler(svc AuthService, jwtExpiryHours int) *Handler {
	return &Handler{svc: svc, jwtExpiryHours: jwtExpiryHours}
}

// registerResponse is the JSON body returned on a successful registration.
type registerResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// loginResponse is the JSON body returned on a successful login.
type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Register handles POST /api/v1/auth/register.
//
//   - 201 on success with {id, email, created_at}
//   - 409 when the email is already registered
//   - 422 when validation fails (email empty, password too short)
//   - 500 on unexpected errors
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrConflict):
			api.WriteError(w, http.StatusConflict, "email already registered")
		case errors.Is(err, models.ErrValidation):
			api.WriteFieldErrors(w, http.StatusUnprocessableEntity, buildValidationFields(err))
		default:
			api.Write500(w, err)
		}
		return
	}

	api.WriteJSON(w, http.StatusCreated, registerResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

// Login handles POST /api/v1/auth/login.
//
//   - 200 on success with {token, expires_at}
//   - 401 on invalid credentials
//   - 500 on unexpected errors
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrUnauthorized):
			api.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			api.Write500(w, err)
		}
		return
	}

	expiresAt := time.Now().UTC().Add(time.Duration(h.jwtExpiryHours) * time.Hour)
	api.WriteJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})
}

// buildValidationFields extracts a human-readable field error list from err.
// Since the service wraps ErrValidation with a single descriptive message, we
// surface it as a single "request" field error for now.
func buildValidationFields(err error) []api.FieldError {
	return []api.FieldError{
		{Field: "request", Message: err.Error()},
	}
}
