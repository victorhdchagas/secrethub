package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/victorhdchagas/secrethub/internal/vault"
)

// handleImportVault faz merge de conteúdo .env (text/plain) em um vault existente.
// Cria o vault se não existir. Retorna contagem de chaves criadas/sobrescritas.
func (s *Server) handleImportVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sess := SessionFromContext(r.Context())

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "Body too large or unreadable", http.StatusBadRequest)
		return
	}

	parsed := vault.ParseEnv(string(body))
	if len(parsed) == 0 {
		http.Error(w, "No valid KEY=VALUE lines found", http.StatusBadRequest)
		return
	}

	v := loadOrNewVault(sess.VaultKey, s.store, r.Context(), name)
	created, overwritten := 0, 0
	for k, val := range parsed {
		if _, exists := v.Get(k); exists {
			overwritten++
		} else {
			created++
		}
		v.Set(k, val)
	}

	ciphertext, err := encryptVault(v, sess.VaultKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.Save(r.Context(), name, ciphertext); err != nil {
		http.Error(w, "Save error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{ // intentionally discarded
		"created":     created,
		"overwritten": overwritten,
	})
}

// loadOrNewVault carrega o vault descriptografado; em caso de ausência cria um novo.
func loadOrNewVault(key *[32]byte, store *vault.Store, ctx context.Context, name string) *vault.Vault {
	existing, err := loadDecryptedVault(key, store, ctx, name)
	if err == nil {
		return existing
	}
	return vault.New(name)
}
