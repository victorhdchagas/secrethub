package vault

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	SaltLen    = 16
	NonceLen   = 24
	KeyLen     = 32
	ArgonTime  = 3
	ArgonMem   = 64 * 1024
	ArgonThreads = 1
)

func DeriveKey(password string, salt []byte) *[KeyLen]byte {
	key := argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMem, ArgonThreads, KeyLen)
	var k [KeyLen]byte
	copy(k[:], key)
	return &k
}

func NewSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("salt generation: %w", err)
	}
	return salt, nil
}

func Encrypt(plaintext []byte, key *[KeyLen]byte) ([]byte, error) {
	var nonce [NonceLen]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("nonce generation: %w", err)
	}

	encrypted := secretbox.Seal(nil, plaintext, &nonce, key)
	out := make([]byte, NonceLen+len(encrypted))
	copy(out[:NonceLen], nonce[:])
	copy(out[NonceLen:], encrypted)
	return out, nil
}

func Decrypt(ciphertext []byte, key *[KeyLen]byte) ([]byte, error) {
	if len(ciphertext) < NonceLen {
		return nil, errors.New("ciphertext too short")
	}

	var nonce [NonceLen]byte
	copy(nonce[:], ciphertext[:NonceLen])

	plaintext, ok := secretbox.Open(nil, ciphertext[NonceLen:], &nonce, key)
	if !ok {
		return nil, errors.New("decryption failed")
	}
	return plaintext, nil
}


