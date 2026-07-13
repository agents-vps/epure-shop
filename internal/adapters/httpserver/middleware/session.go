package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/agents-vps/epure-shop/internal/core/ports"
)

// ─── Context key types (unexported to prevent collisions) ───

type ctxKeyUserID int
type ctxKeyRole int

const (
	keyUserID ctxKeyUserID = iota
	keyRole   ctxKeyRole   = iota
)

// UserIDFromContext extracts the authenticated user ID from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(keyUserID).(string)
	return id, ok
}

// RoleFromContext extracts the authenticated user's role from context.
func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(keyRole).(string)
	return role, ok
}

// WithUserID injects a user ID into the context (for testing or programmatic use).
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, keyUserID, userID)
}

// WithRole injects a role into the context (for testing or programmatic use).
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, keyRole, role)
}

// SessionAuth reads the "sid" cookie, hashes it with SHA-256, looks up the
// session via SessionStore, and injects userID and role into the request context.
type SessionAuth struct {
	Store  ports.SessionStore
	Logger *slog.Logger
}

// NewSessionAuth creates a SessionAuth middleware.
func NewSessionAuth(store ports.SessionStore, log *slog.Logger) *SessionAuth {
	if log == nil {
		log = slog.Default()
	}
	return &SessionAuth{Store: store, Logger: log}
}

// Middleware returns an http.Handler that performs session authentication.
func (sa *SessionAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("sid")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimSpace(cookie.Value)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Pass the raw token to the store; it handles hashing internally.
		userID, role, err := sa.Store.Get(r.Context(), token)
		if err != nil {
			sa.Logger.Warn("session lookup failed",
				slog.String("error", err.Error()),
			)
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, keyUserID, userID)
		ctx = context.WithValue(ctx, keyRole, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
