package auth

import (
	"context"
	"net/http"
	"strings"

	"taskflow/internal/api"
)

// contextKeyType is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type contextKeyType int

const (
	// ContextKeyUserClaims is the context key under which *Claims is stored.
	// This is exported so other packages can retrieve claims from the context.
	ContextKeyUserClaims contextKeyType = iota
)

// JWTMiddleware returns a chi-compatible middleware that validates the
// Authorization: Bearer <token> header on every request.
//
// On success, the parsed *Claims are injected into the request context.
// On failure (missing header, malformed token, expired token), a 401 response
// is written and the handler chain is terminated.
func JWTMiddleware(svc AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				api.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				api.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			claims, err := svc.ValidateToken(parts[1])
			if err != nil {
				api.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the *Claims injected by JWTMiddleware from ctx.
// Returns (nil, false) if no claims are present.
func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ContextKeyUserClaims).(*Claims)
	return claims, ok
}
