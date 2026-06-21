package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type tokenCreateResponse struct {
	Token     string `json:"token"`
	Prefix    string `json:"prefix"`
	CreatedAt string `json:"created_at"`
}

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	sess := SessionFromContext(r.Context())
	if sess == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := s.tokens.Generate(r.Context(), sess.VaultKey)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenCreateResponse{ // intentionally discarded
		Token:     token,
		Prefix:    token[:8],
		CreatedAt: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (s *Server) handleListTokens(w http.ResponseWriter, r *http.Request) {
	infos, err := s.tokens.List(r.Context())
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	type tokenView struct {
		Prefix    string `json:"prefix"`
		CreatedAt string `json:"created_at"`
	}

	views := make([]tokenView, 0, len(infos))
	for _, info := range infos {
		views = append(views, tokenView{
			Prefix:    info.Prefix,
			CreatedAt: info.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tokens": views}) // intentionally discarded
}

func (s *Server) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	prefix := r.PathValue("prefix")

	if err := s.tokens.Revoke(r.Context(), prefix); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
