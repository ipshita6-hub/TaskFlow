package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"
)

// logger is the package-level structured logger for the api package.
var logger = slog.Default()

// FieldError represents a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteJSON sets Content-Type: application/json, writes the given status code,
// and encodes v as JSON into the response body.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point the header is already written; log the encoding failure.
		logger.Error("WriteJSON: failed to encode response", "error", err)
	}
}

// WriteError writes a standard {"error": message} JSON response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// WriteFieldErrors writes a validation-failure response that includes per-field detail.
//
//	{"error": "validation failed", "fields": [...]}
func WriteFieldErrors(w http.ResponseWriter, status int, fields []FieldError) {
	WriteJSON(w, status, map[string]interface{}{
		"error":  "validation failed",
		"fields": fields,
	})
}

// Write500 generates a unique error reference UUID, logs the full error together
// with the current goroutine stack, and writes an opaque 500 response so that
// internal details are never leaked to the caller.
//
//	{"error": "internal server error", "ref": "<uuid>"}
func Write500(w http.ResponseWriter, err error) {
	ref := uuid.New().String()
	stack := debug.Stack()
	logger.Error("internal server error",
		"ref", ref,
		"error", fmt.Sprintf("%v", err),
		"stack", string(stack),
	)
	WriteJSON(w, http.StatusInternalServerError, map[string]string{
		"error": "internal server error",
		"ref":   ref,
	})
}
