package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Token    string
	VaultKey *[32]byte
	Salt     []byte
	expires  time.Time
}

func (s *Session) Expired() bool {
	return time.Now().After(s.expires)
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	timeout  time.Duration
}

func NewSessionManager(timeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		timeout:  timeout,
	}
}

func (sm *SessionManager) Create(vaultKey *[32]byte, salt []byte) (*Session, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("session token: %w", err)
	}

	token := hex.EncodeToString(bytes)
	s := &Session{
		Token:    token,
		VaultKey: vaultKey,
		Salt:     salt,
		expires:  time.Now().Add(sm.timeout),
	}

	sm.mu.Lock()
	sm.sessions[tokenHash(token)] = s
	sm.mu.Unlock()
	return s, nil
}

func (sm *SessionManager) Get(token string) *Session {
	sm.mu.RLock()
	s, ok := sm.sessions[tokenHash(token)]
	sm.mu.RUnlock()
	if !ok || s.Expired() {
		if ok {
			sm.Destroy(token)
		}
		return nil
	}
	return s
}

func (sm *SessionManager) Destroy(token string) {
	sm.mu.Lock()
	delete(sm.sessions, tokenHash(token))
	sm.mu.Unlock()
}

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

type ctxKey string

const sessionCtxKey ctxKey = "session"

func SessionFromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionCtxKey).(*Session)
	return s
}

func (sm *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		s := sm.Get(cookie.Value)
		if s == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), sessionCtxKey, s)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
