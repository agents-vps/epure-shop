package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RateLimiter is an in-memory rate limiter keyed by IP or IP+email.
// Limits: 5 requests per 15 minutes per key.  Background goroutine
// periodically cleans expired entries.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]*rateEntry
	log      *slog.Logger
	maxReqs  int
	window   time.Duration
}

type rateEntry struct {
	count    int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter with maxReqs per window.
// Defaults: maxReqs=5, window=15 minutes.
func NewRateLimiter(log *slog.Logger) *RateLimiter {
	return NewRateLimiterConfig(log, 5, 15*time.Minute)
}

// NewRateLimiterConfig creates a rate limiter with custom limits.
func NewRateLimiterConfig(log *slog.Logger, maxReqs int, window time.Duration) *RateLimiter {
	if log == nil {
		log = slog.Default()
	}
	rl := &RateLimiter{
		entries:  make(map[string]*rateEntry),
		log:      log,
		maxReqs:  maxReqs,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

// Middleware returns the rate-limiting http.Handler middleware.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build key: IP or IP+email when available.
		key := r.RemoteAddr
		if email := r.Header.Get("X-RateLimit-Email"); email != "" {
			key = fmt.Sprintf("%s:%s", r.RemoteAddr, email)
		}

		rl.mu.Lock()
		now := time.Now()
		e, ok := rl.entries[key]
		if !ok || now.Sub(e.windowStart) >= rl.window {
			// New window.
			rl.entries[key] = &rateEntry{count: 1, windowStart: now}
			rl.mu.Unlock()
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.maxReqs))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.maxReqs-1))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(rl.window).Unix()))
			next.ServeHTTP(w, r)
			return
		}

		e.count++
		remaining := rl.maxReqs - e.count
		if remaining < 0 {
			remaining = 0
		}
		resetTime := e.windowStart.Add(rl.window)

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.maxReqs))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))

		if e.count > rl.maxReqs {
			rl.mu.Unlock()
			rl.log.Warn("rate limit exceeded",
				slog.String("key", redactKey(key)),
				slog.String("path", r.URL.Path),
			)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}
		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// cleanupLoop periodically purges expired entries.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for k, e := range rl.entries {
			if e.windowStart.Before(cutoff) {
				delete(rl.entries, k)
			}
		}
		rl.mu.Unlock()
	}
}

// redactKey shows only the first 4 chars of the key for logging.
func redactKey(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[:4] + "..."
}
