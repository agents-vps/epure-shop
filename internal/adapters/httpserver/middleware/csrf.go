package middleware

import (
	"net/http"
)

// CSRF returns a middleware that wraps http.NewCrossOriginProtection.
// This rejects requests with unsafe methods (POST/PUT/DELETE/PATCH) that
// do not include a valid Origin or Referer header matching the host.
func CSRF() func(http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	return func(next http.Handler) http.Handler {
		return cop.Handler(next)
	}
}
