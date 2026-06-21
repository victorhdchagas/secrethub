package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/victorhdchagas/secrethub/internal/auth"
	"github.com/victorhdchagas/secrethub/internal/templates"
	"github.com/victorhdchagas/secrethub/internal/vault"
)

type Config struct {
	Host    string
	Port    int
	DataDir string
}

type Server struct {
	config    Config
	mux       *http.ServeMux
	store     *vault.Store
	hasher    *auth.BCryptHasher
	totp      *auth.TOTPHandler
	sessions  *SessionManager
	tokens    *auth.TokenHandler
	loginTmpl *template.Template
	setupTmpl *template.Template
	dashTmpl  *template.Template
	rateLimit *RateLimiter
}

func New(cfg Config) *Server {
	th := auth.NewTokenHandler(cfg.DataDir + "/machine.tokens")
	if err := th.Load(context.Background()); err != nil {
		fmt.Printf("Warning: failed to load machine tokens: %v\n", err)
	}

	s := &Server{
		config:    cfg,
		mux:       http.NewServeMux(),
		store:     vault.NewStore(cfg.DataDir + "/vaults"),
		hasher:    auth.NewBCryptHasher(12),
		totp:      auth.NewTOTPHandler(),
		sessions:  NewSessionManager(15 * time.Minute),
		tokens:    th,
		rateLimit: NewRateLimiter(1*time.Minute, 30),
	}

	loginTmpl := template.Must(template.ParseFS(templates.FS, "login.html"))
	s.loginTmpl = loginTmpl
	setupTmpl := template.Must(template.ParseFS(templates.FS, "setup.html"))
	s.setupTmpl = setupTmpl
	dashTmpl := template.Must(template.ParseFS(templates.FS, "dashboard.html"))
	s.dashTmpl = dashTmpl

	s.routes()
	return s
}

func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return s.sessions.Middleware(h)
}

func (s *Server) routes() {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(templates.FS))))

	s.mux.HandleFunc("GET /setup", s.handleSetupPage)
	s.mux.HandleFunc("POST /api/setup", s.handleSetup)
	s.mux.HandleFunc("POST /api/setup/verify-totp", s.handleSetupVerifyTOTP)
	s.mux.HandleFunc("GET /api/setup/qr", s.handleSetupQR)
	s.mux.HandleFunc("GET /", s.handleLoginPage)
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.Handle("GET /dashboard", s.protected(s.handleDashboardPage))
	s.mux.Handle("GET /api/logout", s.protected(s.handleLogout))
	s.mux.Handle("GET /api/vaults", s.protected(s.handleListVaults))
	s.mux.Handle("GET /api/vault/{name}", s.protected(s.handleGetVault))
	s.mux.Handle("POST /api/vault/{name}", s.protected(s.handleSaveVault))
	s.mux.Handle("DELETE /api/vault/{name}", s.protected(s.handleDeleteVault))
	s.mux.Handle("PUT /api/vault/{name}/keys/{key}", s.protected(s.handlePutKey))
	s.mux.Handle("DELETE /api/vault/{name}/keys/{key}", s.protected(s.handleDeleteKey))
	s.mux.HandleFunc("GET /api/vault/{name}/export", s.handleExportVault)
	s.mux.Handle("POST /api/tokens", s.protected(s.handleCreateToken))
	s.mux.Handle("GET /api/tokens", s.protected(s.handleListTokens))
	s.mux.Handle("DELETE /api/tokens/{prefix}", s.protected(s.handleRevokeToken))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.isConfigured() && !isSetupPath(r.URL.Path) {
		// allow through if a valid session cookie exists
		if cookie, err := r.Cookie("session"); err != nil || s.sessions.Get(cookie.Value) == nil {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

func isSetupPath(path string) bool {
	return path == "/setup" || strings.HasPrefix(path, "/api/setup") ||
		strings.HasPrefix(path, "/static/")
}

func Serve(cfg Config) error {
	s := New(cfg)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	fmt.Printf("SecretHub listening on http://%s\n", addr)
	return httpServer.ListenAndServe()
}
