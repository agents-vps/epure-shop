package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger logs each request with duration, method, path, status, and optional
// Unpoly metadata.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			}
			if upVersion := r.Header.Get("X-Up-Version"); upVersion != "" {
				attrs = append(attrs, slog.String("up_version", upVersion))
			}
			if upTarget := r.Header.Get("X-Up-Target"); upTarget != "" {
				attrs = append(attrs, slog.String("up_target", upTarget))
			}

			log.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		})
	}
}
