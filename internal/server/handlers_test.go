package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/publiquei/secrethub/internal/auth"
	"github.com/publiquei/secrethub/internal/vault"
)

func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()

	hasher := auth.NewBCryptHasher(4)
	hash, _ := hasher.Hash(ctx, "password")
	os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600)

	totp := auth.NewTOTPHandler()
	key, _ := totp.Generate(ctx, "test")
	os.WriteFile(filepath.Join(dir, "totp.secret"), []byte(key.Secret), 0600)

	salt := []byte("test-salt-value!")
	os.WriteFile(filepath.Join(dir, "salt"), salt, 0600)

	os.MkdirAll(filepath.Join(dir, "vaults"), 0700)

	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})
	return s, dir
}

func TestLoginPageRedirectsToSetup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302 redirect to /setup, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc != "/setup" {
		t.Errorf("expected Location /setup, got %s", loc)
	}
}

func TestSetupPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})

	req := httptest.NewRequest("GET", "/setup", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "SecretHub Setup") {
		t.Error("expected setup page content")
	}
}

func TestLoginPageWithSetup(t *testing.T) {
	t.Parallel()

	s, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "SecretHub is not yet configured") {
		t.Error("should not show setup instructions when configured")
	}
}

func TestProtectedRouteWithoutCookie(t *testing.T) {
	t.Parallel()

	s, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/vaults", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetVaultWithSession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})
	ctx := context.Background()

	var key [32]byte
	copy(key[:], []byte("0123456789abcdef0123456789abcdef"))
	salt := []byte("test-salt-value!")

	session, err := s.sessions.Create(&key, salt)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	v := vault.New("testapp")
	v.Set("DB_HOST", "localhost")
	v.Set("DB_PORT", "5432")

	plaintext, _ := vault.SerializeVault(v)
	ciphertext, _ := vault.Encrypt(plaintext, &key)
	s.store.Save(ctx, "testapp", ciphertext)

	req := httptest.NewRequest("GET", "/api/vault/testapp", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: session.Token})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), "localhost") {
		t.Errorf("expected vault content, got %s", rec.Body.String())
	}
}

func TestExportVault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})
	ctx := context.Background()

	var key [32]byte
	copy(key[:], []byte("0123456789abcdef0123456789abcdef"))
	salt := []byte("test-salt-value!")

	session, _ := s.sessions.Create(&key, salt)

	v := vault.New("myapp")
	v.Set("KEY", "VALUE")
	plaintext, _ := vault.SerializeVault(v)
	ciphertext, _ := vault.Encrypt(plaintext, &key)
	s.store.Save(ctx, "myapp", ciphertext)

	req := httptest.NewRequest("GET", "/api/vault/myapp/export", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: session.Token})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "KEY=VALUE") {
		t.Errorf("expected KEY=VALUE, got %s", body)
	}
}
