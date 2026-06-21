package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/nacl/secretbox"
)

type TokenEntry struct {
	Hash      string    `json:"hash"`
	Prefix    string    `json:"prefix"`
	KeyCipher []byte    `json:"key_cipher"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenHandler struct {
	mu      sync.Mutex
	path    string
	entries []TokenEntry
}

func NewTokenHandler(path string) *TokenHandler {
	return &TokenHandler{path: path}
}

func (th *TokenHandler) Load(_ context.Context) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	data, err := os.ReadFile(th.path)
	if err != nil {
		if os.IsNotExist(err) {
			th.entries = nil
			return nil
		}
		return fmt.Errorf("read tokens: %w", err)
	}
	return json.Unmarshal(data, &th.entries)
}

func (th *TokenHandler) save() error {
	data, err := json.Marshal(th.entries)
	if err != nil {
		return fmt.Errorf("marshal tokens: %w", err)
	}
	return os.WriteFile(th.path, data, 0600)
}

// Generate cria um novo machine token e retorna o token plaintext (exibido 1x).
// O vaultKey fica criptografado no disco com o próprio token como chave.
func (th *TokenHandler) Generate(_ context.Context, vaultKey *[32]byte) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("token random: %w", err)
	}

	display := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(display))
	hash := hex.EncodeToString(sum[:])

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	keyCipher := secretbox.Seal(nil, vaultKey[:], &nonce, &sum)
	keyCipher = append(nonce[:], keyCipher...)

	entry := TokenEntry{
		Hash:      hash,
		Prefix:    display[:8],
		KeyCipher: keyCipher,
		CreatedAt: time.Now(),
	}

	th.mu.Lock()
	th.entries = append(th.entries, entry)
	if err := th.save(); err != nil {
		th.entries = th.entries[:len(th.entries)-1]
		th.mu.Unlock()
		return "", err
	}
	th.mu.Unlock()
	return display, nil
}

// Validate retorna a vault key decriptada se o token for válido.
func (th *TokenHandler) Validate(_ context.Context, token string) (*[32]byte, error) {
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])

	th.mu.Lock()
	defer th.mu.Unlock()

	for _, e := range th.entries {
		if e.Hash == hash {
			if len(e.KeyCipher) < 24 {
				return nil, fmt.Errorf("corrupt token entry")
			}
			var nonce [24]byte
			copy(nonce[:], e.KeyCipher[:24])

			plaintext, ok := secretbox.Open(nil, e.KeyCipher[24:], &nonce, &sum)
			if !ok {
				return nil, fmt.Errorf("token decryption failed")
			}
			var vk [32]byte
			copy(vk[:], plaintext)
			return &vk, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}

func (th *TokenHandler) Revoke(_ context.Context, prefix string) error {
	th.mu.Lock()
	defer th.mu.Unlock()

	for i, e := range th.entries {
		if e.Prefix == prefix {
			th.entries = append(th.entries[:i], th.entries[i+1:]...)
			return th.save()
		}
	}
	return fmt.Errorf("token %s not found", prefix)
}

type TokenInfo struct {
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

func (th *TokenHandler) List(_ context.Context) ([]TokenInfo, error) {
	th.mu.Lock()
	defer th.mu.Unlock()

	infos := make([]TokenInfo, len(th.entries))
	for i, e := range th.entries {
		infos[i] = TokenInfo{Prefix: e.Prefix, CreatedAt: e.CreatedAt}
	}
	return infos, nil
}
