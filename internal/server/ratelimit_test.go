package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(time.Minute, 5)

	for i := 0; i < 5; i++ {
		if !rl.Allow("test-key") {
			t.Fatalf("expected allow at attempt %d", i+1)
		}
	}

	if rl.Allow("test-key") {
		t.Error("expected deny after exceeding limit")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(time.Minute, 3)

	for i := 0; i < 3; i++ {
		rl.Allow("key-a")
	}

	if !rl.Allow("key-b") {
		t.Error("expected allow for different key")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(50*time.Millisecond, 2)

	rl.Allow("key")
	rl.Allow("key")
	if rl.Allow("key") {
		t.Error("expected deny before reset")
	}

	time.Sleep(60 * time.Millisecond)

	if !rl.Allow("key") {
		t.Error("expected allow after window reset")
	}
}

func TestRateLimiter_MiddlewareBlocks(t *testing.T) {
	t.Parallel()

	rl := NewRateLimiter(time.Minute, 1)
	handler := rl.Middleware(func(r *http.Request) string {
		return r.RemoteAddr
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// first request passes
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// second request blocked
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec2.Code)
	}
}
