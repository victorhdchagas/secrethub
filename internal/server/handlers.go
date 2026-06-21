package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/publiquei/secrethub/internal/auth"
	"github.com/publiquei/secrethub/internal/vault"
)

type loginPageData struct {
	SetupRequired bool
	Error         string
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	masterHash := filepath.Join(s.config.DataDir, "master.hash")
	_, err := os.Stat(masterHash)

	data := loginPageData{
		SetupRequired: os.IsNotExist(err),
	}

	s.loginTmpl.Execute(w, data) // intentionally discarded
}

func (s *Server) handleDashboardPage(w http.ResponseWriter, r *http.Request) {
	s.dashTmpl.Execute(w, nil) // intentionally discarded
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	password := r.FormValue("password")
	totpCode := r.FormValue("totp")
	dir := s.config.DataDir

	hashBytes, err := os.ReadFile(filepath.Join(dir, "master.hash"))
	if err != nil {
		http.Error(w, "Setup required", http.StatusPreconditionFailed)
		return
	}

	if err := s.hasher.Verify(r.Context(), password, string(hashBytes)); err != nil {
		data := loginPageData{Error: "Invalid password or TOTP code"}
		s.loginTmpl.Execute(w, data) // intentionally discarded
		return
	}

	secretBytes, err := os.ReadFile(filepath.Join(dir, "totp.secret"))
	if err != nil {
		http.Error(w, "Setup required", http.StatusPreconditionFailed)
		return
	}
	totpSecret := strings.TrimSpace(string(secretBytes))

	if !s.totp.Validate(r.Context(), totpSecret, totpCode) {
		if !s.tryRecoveryCode(r.Context(), dir, totpCode, w) {
			return
		}
	}

	saltBytes, err := os.ReadFile(filepath.Join(dir, "salt"))
	if err != nil {
		http.Error(w, "Corrupt setup: salt file missing", http.StatusInternalServerError)
		return
	}

	vk := vault.DeriveKey(password, saltBytes)

	session, err := s.sessions.Create(vk, saltBytes)
	if err != nil {
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

func (s *Server) tryRecoveryCode(ctx context.Context, dir, code string, w http.ResponseWriter) bool {
	recoveryBytes, err := os.ReadFile(filepath.Join(dir, "recovery.hashes"))
	if err != nil {
		data := loginPageData{Error: "Invalid password or TOTP code"}
		s.loginTmpl.Execute(w, data) // intentionally discarded
		return false
	}

	hashes := strings.Split(strings.TrimSpace(string(recoveryBytes)), "\n")
	rec := auth.NewRecoveryHandler(hashes)

	if !rec.Validate(ctx, code) {
		data := loginPageData{Error: "Invalid password or TOTP code"}
		s.loginTmpl.Execute(w, data) // intentionally discarded
		return false
	}

	rec.Use(ctx, code)
	var hashData []byte
	for _, h := range rec.Hashes() {
		hashData = append(hashData, []byte(h+"\n")...)
	}
	os.WriteFile(filepath.Join(dir, "recovery.hashes"), hashData, 0600)
	return true
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		s.sessions.Destroy(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleListVaults(w http.ResponseWriter, r *http.Request) {
	names, err := s.store.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if names == nil {
		names = []string{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"vaults": names}) // intentionally discarded
}
