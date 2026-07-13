package middleware

import (
	"net/http"
)

// RequireAuth returns a middleware that redirects unauthenticated requests
// to /login (preserving the original URL via ?next= query parameter).
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := UserIDFromContext(r.Context()); !ok {
				// If Unpoly, return 401 so the client can handle it gracefully.
				if r.Header.Get("X-Up-Version") != "" {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/login?next="+r.URL.Path, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin returns a middleware that returns 403 Forbidden unless the
// authenticated user has the "admin" role.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok || role != "admin" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
