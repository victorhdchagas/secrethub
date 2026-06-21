package server

import (
	"net/http"
	"sync"
	"time"
)

type rateEntry struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu     sync.Mutex
	limits map[string]*rateEntry
	window time.Duration
	max    int
}

func NewRateLimiter(window time.Duration, max int) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*rateEntry),
		window: window,
		max:    max,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// lazy cleanup of expired entries
	for k, e := range rl.limits {
		if now.After(e.resetAt) {
			delete(rl.limits, k)
		}
	}

	e, ok := rl.limits[key]
	if !ok || now.After(e.resetAt) {
		rl.limits[key] = &rateEntry{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if e.count >= rl.max {
		return false
	}

	e.count++
	return true
}

// RateLimitMiddleware retorna um middleware HTTP que limita por IP (ou chave customizada).
func (rl *RateLimiter) Middleware(keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if !rl.Allow(key) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
