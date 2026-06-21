package server

import (
	"encoding/json"
	"image/png"
	"net/http"
	"os"
	"path/filepath"

	"github.com/boombuler/barcode/qr"
	"github.com/victorhdchagas/secrethub/internal/auth"
	"github.com/victorhdchagas/secrethub/internal/vault"
)

type setupResponse struct {
	TOTPURL       string   `json:"totp_url"`
	TOTPSecret    string   `json:"totp_secret"`
	RecoveryCodes []string `json:"recovery_codes"`
}

func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	if s.isConfigured() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.setupTmpl.Execute(w, nil) // intentionally discarded
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if s.isConfigured() {
		http.Error(w, "Already set up", http.StatusConflict)
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 4 {
		http.Error(w, "Password too short", http.StatusBadRequest)
		return
	}

	dir := s.config.DataDir
	ctx := r.Context()

	hash, err := s.hasher.Hash(ctx, body.Password)
	if err != nil {
		http.Error(w, "Hash error", http.StatusInternalServerError)
		return
	}

	key, err := s.totp.Generate(ctx, "secrethub")
	if err != nil {
		http.Error(w, "TOTP error", http.StatusInternalServerError)
		return
	}

	recovery := auth.NewRecoveryHandler(nil)
	plainCodes, err := recovery.Generate(ctx)
	if err != nil {
		http.Error(w, "Recovery error", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(filepath.Join(dir, "vaults"), 0700); err != nil {
		http.Error(w, "Directory error", http.StatusInternalServerError)
		return
	}

	salt, err := vault.NewSalt()
	if err != nil {
		http.Error(w, "Salt error", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "salt"), salt, 0600); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	// derive vault key and encrypt TOTP secret
	vk := vault.DeriveKey(body.Password, salt)
	encSecret, err := vault.Encrypt([]byte(key.Secret), vk)
	if err != nil {
		http.Error(w, "Encryption error", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "totp.secret"), encSecret, 0600); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	var hashData []byte
	for _, h := range recovery.Hashes() {
		hashData = append(hashData, []byte(h+"\n")...)
	}
	if err := os.WriteFile(filepath.Join(dir, "recovery.hashes"), hashData, 0600); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	// master.hash por último — se algo falhar acima, o setup ainda pode ser refeito
	if err := os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600); err != nil {
		http.Error(w, "Write error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(setupResponse{ // intentionally discarded
		TOTPURL:       key.URL,
		TOTPSecret:    key.Secret,
		RecoveryCodes: plainCodes,
	})
}

func (s *Server) handleSetupVerifyTOTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code     string `json:"code"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	dir := s.config.DataDir

	saltBytes, err := os.ReadFile(filepath.Join(dir, "salt"))
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Setup required", http.StatusPreconditionFailed)
		} else {
			http.Error(w, "Setup incomplete", http.StatusInternalServerError)
		}
		return
	}

	vk := vault.DeriveKey(body.Password, saltBytes)

	encSecret, err := os.ReadFile(filepath.Join(dir, "totp.secret"))
	if err != nil {
		http.Error(w, "Setup required", http.StatusPreconditionFailed)
		return
	}

	decSecret, err := vault.Decrypt(encSecret, vk)
	if err != nil {
		http.Error(w, "Password mismatch with setup step 1", http.StatusUnauthorized)
		return
	}

	if !s.totp.Validate(r.Context(), string(decSecret), body.Code) {
		http.Error(w, "Invalid TOTP code", http.StatusUnauthorized)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"}) // intentionally discarded
}

func (s *Server) handleSetupQR(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	code, err := qr.Encode(url, qr.M, qr.Auto)
	if err != nil {
		http.Error(w, "QR generation error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, code) // intentionally discarded
}
