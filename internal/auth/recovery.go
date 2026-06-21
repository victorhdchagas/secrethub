package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type RecoveryHandler struct {
	hashes []string
}

func NewRecoveryHandler(hashes []string) *RecoveryHandler {
	return &RecoveryHandler{hashes: hashes}
}

func (r *RecoveryHandler) Generate(_ context.Context) ([]string, error) {
	codes := make([]string, 10)
	hashes := make([]string, 10)

	for i := range 10 {
		code, err := randomHex(8)
		if err != nil {
			return nil, err
		}
		codes[i] = code
		hash := sha256.Sum256([]byte(code))
		hashes[i] = hex.EncodeToString(hash[:])
	}

	r.hashes = hashes
	return codes, nil
}

func (r *RecoveryHandler) Validate(_ context.Context, code string) bool {
	hash := sha256.Sum256([]byte(code))
	encoded := hex.EncodeToString(hash[:])
	for _, h := range r.hashes {
		if h == encoded {
			return true
		}
	}
	return false
}

func (r *RecoveryHandler) Use(_ context.Context, code string) bool {
	hash := sha256.Sum256([]byte(code))
	encoded := hex.EncodeToString(hash[:])
	for i, h := range r.hashes {
		if h == encoded {
			r.hashes = append(r.hashes[:i], r.hashes[i+1:]...)
			return true
		}
	}
	return false
}

func (r *RecoveryHandler) Hashes() []string {
	return r.hashes
}

func randomHex(length int) (string, error) {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("random read: %w", err)
	}
	return hex.EncodeToString(bytes)[:length], nil
}
