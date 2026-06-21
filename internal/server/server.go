package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/publiquei/secrethub/internal/auth"
	"github.com/publiquei/secrethub/internal/templates"
	"github.com/publiquei/secrethub/internal/vault"
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
	loginTmpl *template.Template
	dashTmpl  *template.Template
}

func New(cfg Config) *Server {
	s := &Server{
		config:   cfg,
		mux:      http.NewServeMux(),
		store:    vault.NewStore(cfg.DataDir + "/vaults"),
		hasher:   auth.NewBCryptHasher(12),
		totp:     auth.NewTOTPHandler(),
		sessions: NewSessionManager(15 * time.Minute),
	}

	loginTmpl := template.Must(template.ParseFS(templates.FS, "login.html"))
	s.loginTmpl = loginTmpl
	dashTmpl := template.Must(template.ParseFS(templates.FS, "dashboard.html"))
	s.dashTmpl = dashTmpl

	s.routes()
	return s
}

func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return s.sessions.Middleware(h)
}

func (s *Server) routes() {
	staticFS := mustSubFS(templates.FS, ".")
	s.mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

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
	s.mux.Handle("GET /api/vault/{name}/export", s.protected(s.handleExportVault))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func mustSubFS(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

func Serve(cfg Config) error {
	s := New(cfg)
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	fmt.Printf("SecretHub listening on http://%s\n", addr)
	return httpServer.ListenAndServe()
}
