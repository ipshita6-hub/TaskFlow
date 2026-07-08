package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// contextKey is an unexported type used for request-context keys in this package.
// Using a typed key prevents collisions with keys from other packages.
type contextKey int

const (
	// contextKeyRequestID is the context key under which the request ID is stored.
	contextKeyRequestID contextKey = iota
)

// GetRequestID retrieves the request ID injected by the RequestID middleware from ctx.
// Returns an empty string if no request ID is present.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(contextKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// RequestID is a chi-compatible middleware that:
//  1. Generates a UUID v4 for each request.
//  2. Injects it into the request context under contextKeyRequestID.
//  3. Sets the X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		ctx := context.WithValue(r.Context(), contextKeyRequestID, reqID)
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusResponseWriter wraps http.ResponseWriter to capture the written status code.
type statusResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (srw *statusResponseWriter) WriteHeader(code int) {
	if !srw.wroteHeader {
		srw.status = code
		srw.wroteHeader = true
		srw.ResponseWriter.WriteHeader(code)
	}
}

func (srw *statusResponseWriter) Write(b []byte) (int, error) {
	if !srw.wroteHeader {
		// Implicit 200 on first write.
		srw.WriteHeader(http.StatusOK)
	}
	return srw.ResponseWriter.Write(b)
}

// Logger is a chi-compatible middleware that logs the HTTP method, path, status
// code, and request latency after each request completes.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		srw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(srw, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", srw.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", GetRequestID(r.Context()),
		)
	})
}

// Recoverer is a chi-compatible middleware that recovers from panics, logs the
// panic value and stack trace, and responds with a 500 Internal Server Error via
// Write500 so that no panic details leak to the caller.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				var err error
				switch v := rec.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("panic: %v", v)
				}
				Write500(w, err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
