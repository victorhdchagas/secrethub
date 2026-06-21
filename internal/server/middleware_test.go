package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionCreateAndGet(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	var key [32]byte
	key[0] = 1
	salt := []byte("test-salt-1234567")

	s, err := sm.Create(&key, salt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if s.Token == "" {
		t.Error("expected non-empty token")
	}

	got := sm.Get(s.Token)
	if got == nil {
		t.Fatal("expected session to exist")
	}
	if got.Token != s.Token {
		t.Errorf("token mismatch: got %q, want %q", got.Token, s.Token)
	}
}

func TestSessionExpired(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(1 * time.Microsecond)
	var key [32]byte
	salt := []byte("test-salt-1234567")

	s, _ := sm.Create(&key, salt)
	time.Sleep(10 * time.Millisecond)

	got := sm.Get(s.Token)
	if got != nil {
		t.Error("expected session to be expired")
	}
}

func TestSessionDestroy(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	var key [32]byte
	salt := []byte("test-salt-1234567")

	s, _ := sm.Create(&key, salt)
	sm.Destroy(s.Token)

	got := sm.Get(s.Token)
	if got != nil {
		t.Error("expected session to be destroyed")
	}
}

func TestMiddlewareNoCookie(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/vaults", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddlewareInvalidCookie(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/vaults", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "invalid-token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestSessionRefresh(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	var key [32]byte
	salt := []byte("test-salt-1234567")

	s, _ := sm.Create(&key, salt)
	original := s.Expires()
	time.Sleep(time.Microsecond)

	sm.Refresh(s.Token)
	refreshed := s.Expires()

	if !refreshed.After(original) {
		t.Error("expected refreshed expiry to be later")
	}
}

func TestMiddlewareValidCookie(t *testing.T) {
	t.Parallel()

	sm := NewSessionManager(15 * time.Minute)
	var key [32]byte
	salt := []byte("test-salt-1234567")

	s, _ := sm.Create(&key, salt)
	handler := sm.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := SessionFromContext(r.Context())
		if sess == nil {
			t.Error("expected session in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/vaults", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: s.Token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	expiry := rec.Header().Get("X-Session-Expires")
	if expiry == "" {
		t.Error("expected X-Session-Expires header")
	}
}
