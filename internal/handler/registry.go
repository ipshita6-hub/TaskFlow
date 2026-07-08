package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"taskflow/internal/models"
)

// HandlerFunc is the signature for task execution handlers.
type HandlerFunc func(ctx context.Context, payload json.RawMessage) (string, error)

// Registry maps task types to their handler functions.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

// NewRegistry creates a new Registry with the built-in "noop" handler pre-registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]HandlerFunc),
	}
	r.Register("noop", func(ctx context.Context, payload json.RawMessage) (string, error) {
		return "noop: ok", nil
	})
	return r
}

// Register adds a handler for the given taskType, overwriting any existing one.
func (r *Registry) Register(taskType string, fn HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = fn
}

// Execute looks up the handler for taskType and calls it with payload.
// Returns models.ErrUnknownTaskType if no handler is registered for taskType.
func (r *Registry) Execute(ctx context.Context, taskType string, payload json.RawMessage) (string, error) {
	r.mu.RLock()
	fn, ok := r.handlers[taskType]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("%w: %s", models.ErrUnknownTaskType, taskType)
	}
	return fn(ctx, payload)
}
