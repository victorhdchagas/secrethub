package server

import (
	"fmt"
	"net/http"
	"time"
)

type Config struct {
	Host string
	Port int
}

type Server struct {
	config Config
	mux    *http.ServeMux
}

func New(cfg Config) *Server {
	s := &Server{
		config: cfg,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
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
