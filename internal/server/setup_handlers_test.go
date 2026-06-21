package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/victorhdchagas/secrethub/internal/auth"
)

func TestHandleSetup_Password(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		status   int
		check    func(*testing.T, *httptest.ResponseRecorder, string)
	}{
		{
			name:   "success",
			body:   `{"password":"mypassword"}`,
			status: http.StatusOK,
			check: func(t *testing.T, rec *httptest.ResponseRecorder, dir string) {
				if rec.Header().Get("Content-Type") != "application/json" {
					t.Error("expected JSON response")
				}
				body := rec.Body.String()
				if !strings.Contains(body, "totp_url") {
					t.Error("expected totp_url in response")
				}
				if !strings.Contains(body, "totp_secret") {
					t.Error("expected totp_secret in response")
				}
				if !strings.Contains(body, "recovery_codes") {
					t.Error("expected recovery_codes in response")
				}
				// verify files on disk
				if _, err := os.Stat(filepath.Join(dir, "master.hash")); err != nil {
					t.Error("master.hash not written")
				}
				if _, err := os.Stat(filepath.Join(dir, "totp.secret")); err != nil {
					t.Error("totp.secret not written")
				}
				if _, err := os.Stat(filepath.Join(dir, "recovery.hashes")); err != nil {
					t.Error("recovery.hashes not written")
				}
				if _, err := os.Stat(filepath.Join(dir, "salt")); err != nil {
					t.Error("salt not written")
				}
				if _, err := os.Stat(filepath.Join(dir, "vaults")); err != nil {
					t.Error("vaults dir not created")
				}
			},
		},
		{
			name:   "short password",
			body:   `{"password":"ab"}`,
			status: http.StatusBadRequest,
			check: func(t *testing.T, rec *httptest.ResponseRecorder, dir string) {
				if !strings.Contains(rec.Body.String(), "too short") {
					t.Error("expected 'too short' error")
				}
			},
		},
		{
			name:   "invalid json",
			body:   `not-json`,
			status: http.StatusBadRequest,
			check: func(t *testing.T, rec *httptest.ResponseRecorder, dir string) {
				if !strings.Contains(rec.Body.String(), "Invalid JSON") {
					t.Error("expected Invalid JSON error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})

			req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("expected %d, got %d: %s", tt.status, rec.Code, rec.Body.String())
			}
			tt.check(t, rec, dir)
		})
	}
}

func TestHandleSetup_AlreadyConfigured(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})

	// create master.hash to simulate configured state
	os.WriteFile(filepath.Join(dir, "master.hash"), []byte("hash"), 0600)

	req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"test123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Already set up") {
		t.Error("expected 'Already set up' error")
	}
}

func TestHandleSetupVerifyTOTP(t *testing.T) {
	t.Parallel()

	hasher := auth.NewBCryptHasher(4)
	totpHandler := auth.NewTOTPHandler()
	salt := []byte("test-salt-value!")

	// generate a TOTP key + valid code once for the "valid code" test case
	totpKey, err := totpHandler.Generate(context.Background(), "test")
	if err != nil {
		t.Fatalf("generate TOTP key: %v", err)
	}

	tests := []struct {
		name     string
		setupDir func(*testing.T, string)
		body     string
		status   int
		check    func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid code",
			setupDir: func(t *testing.T, dir string) {
				ctx := context.Background()
				hash, _ := hasher.Hash(ctx, "mypassword")
				os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600)
				os.WriteFile(filepath.Join(dir, "totp.secret"), []byte(totpKey.Secret), 0600)
				os.WriteFile(filepath.Join(dir, "salt"), salt, 0600)
				rec := auth.NewRecoveryHandler(nil)
				rec.Generate(ctx)
				var hb []byte
				for _, h := range rec.Hashes() {
					hb = append(hb, []byte(h+"\n")...)
				}
				os.WriteFile(filepath.Join(dir, "recovery.hashes"), hb, 0600)
				os.MkdirAll(filepath.Join(dir, "vaults"), 0700)
			},
			body: func() string {
				code, _ := totp.GenerateCode(totpKey.Secret, time.Now())
				return `{"code":"` + code + `","password":"mypassword"}`
			}(),
			status: http.StatusOK,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				found := false
				for _, c := range cookies {
					if c.Name == "session" && c.Value != "" {
						found = true
						break
					}
				}
				if !found {
					t.Error("expected session cookie")
				}
				if !strings.Contains(rec.Body.String(), "ok") {
					t.Error("expected ok response")
				}
			},
		},
		{
			name: "no setup done",
			setupDir: func(t *testing.T, dir string) {
			},
			body:   `{"code":"123456","password":"test"}`,
			status: http.StatusPreconditionFailed,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if !strings.Contains(rec.Body.String(), "Setup required") {
					t.Error("expected 'Setup required' error")
				}
			},
		},
		{
			name: "invalid totp code",
			setupDir: func(t *testing.T, dir string) {
				ctx := context.Background()
				hash, _ := hasher.Hash(ctx, "pw")
				os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600)
				os.WriteFile(filepath.Join(dir, "totp.secret"), []byte(totpKey.Secret), 0600)
				os.WriteFile(filepath.Join(dir, "salt"), salt, 0600)
			},
			body:   `{"code":"000000","password":"pw"}`,
			status: http.StatusUnauthorized,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if !strings.Contains(rec.Body.String(), "Invalid TOTP code") {
					t.Error("expected 'Invalid TOTP code' error")
				}
			},
		},
		{
			name: "invalid json",
			setupDir: func(t *testing.T, dir string) {
				hash, _ := hasher.Hash(context.Background(), "pw")
				os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600)
			},
			body:   `bad-json`,
			status: http.StatusBadRequest,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if !strings.Contains(rec.Body.String(), "Invalid JSON") {
					t.Error("expected Invalid JSON error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})
			tt.setupDir(t, dir)

			req := httptest.NewRequest("POST", "/api/setup/verify-totp", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("expected %d, got %d: %s", tt.status, rec.Code, rec.Body.String())
			}
			tt.check(t, rec)
		})
	}
}

func TestSetupQR(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := New(Config{Host: "127.0.0.1", Port: 0, DataDir: dir})

	tests := []struct {
		name   string
		url    string
		status int
	}{
		{
			name:   "valid url",
			url:    "otpauth://totp/SecretHub:secrethub?secret=TEST&issuer=SecretHub",
			status: http.StatusOK,
		},
		{
			name:   "empty url",
			url:    "",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/setup/qr?url="+tt.url, nil)
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("expected %d, got %d", tt.status, rec.Code)
			}

			if tt.status == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if ct != "image/png" {
					t.Errorf("expected image/png, got %s", ct)
				}
				if len(rec.Body.Bytes()) == 0 {
					t.Error("expected non-empty PNG body")
				}
			}
		})
	}
}
