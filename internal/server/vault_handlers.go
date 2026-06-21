package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/publiquei/secrethub/internal/vault"
)

func (s *Server) handleGetVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sess := SessionFromContext(r.Context())
	data, err := s.store.Load(r.Context(), name)
	if err != nil {
		http.Error(w, "Vault not found", http.StatusNotFound)
		return
	}
	plaintext, err := vault.Decrypt(data, sess.VaultKey)
	if err != nil {
		http.Error(w, "Decryption error", http.StatusInternalServerError)
		return
	}
	v, err := vault.DeserializeVault(name, plaintext)
	if err != nil {
		http.Error(w, "Corrupt vault", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v.All()) // intentionally discarded
}

func (s *Server) handleSaveVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sess := SessionFromContext(r.Context())
	var keys map[string]string
	if err := json.NewDecoder(r.Body).Decode(&keys); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	v := vault.New(name)
	for k, val := range keys {
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
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Saved")
}

func (s *Server) handleDeleteVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.store.Delete(r.Context(), name); err != nil {
		http.Error(w, "Delete error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePutKey(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	key := r.PathValue("key")
	sess := SessionFromContext(r.Context())
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	v, err := loadDecryptedVault(sess.VaultKey, s.store, r.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Vault not found", http.StatusNotFound)
		} else {
			http.Error(w, "Vault error", http.StatusInternalServerError)
		}
		return
	}
	v.Set(key, body.Value)
	ciphertext, err := encryptVault(v, sess.VaultKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.Save(r.Context(), name, ciphertext); err != nil {
		http.Error(w, "Save error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func (s *Server) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	key := r.PathValue("key")
	sess := SessionFromContext(r.Context())
	v, err := loadDecryptedVault(sess.VaultKey, s.store, r.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Vault not found", http.StatusNotFound)
		} else {
			http.Error(w, "Vault error", http.StatusInternalServerError)
		}
		return
	}
	v.Delete(key)
	ciphertext, err := encryptVault(v, sess.VaultKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.store.Save(r.Context(), name, ciphertext); err != nil {
		http.Error(w, "Save error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleExportVault(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	sess := SessionFromContext(r.Context())
	v, err := loadDecryptedVault(sess.VaultKey, s.store, r.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Vault not found", http.StatusNotFound)
		} else {
			http.Error(w, "Vault error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, v.Export())
}

func loadDecryptedVault(key *[32]byte, store *vault.Store, ctx context.Context, name string) (*vault.Vault, error) {
	data, err := store.Load(ctx, name)
	if err != nil {
		return nil, err
	}
	plaintext, err := vault.Decrypt(data, key)
	if err != nil {
		return nil, err
	}
	return vault.DeserializeVault(name, plaintext)
}

func encryptVault(v *vault.Vault, key *[32]byte) ([]byte, error) {
	plaintext, err := vault.SerializeVault(v)
	if err != nil {
		return nil, fmt.Errorf("serialize error: %w", err)
	}
	ciphertext, err := vault.Encrypt(plaintext, key)
	if err != nil {
		return nil, fmt.Errorf("encryption error: %w", err)
	}
	return ciphertext, nil
}
