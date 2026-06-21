package vault

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}

	key := DeriveKey("mypassword", salt)
	plaintext := []byte("DB_PASSWORD=supersecret")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	t.Parallel()

	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	key := DeriveKey("correct", salt)
	ciphertext, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	wrongKey := DeriveKey("wrong", salt)
	_, err = Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecryptCorruptedData(t *testing.T) {
	t.Parallel()

	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	key := DeriveKey("password", salt)
	ciphertext, err := Encrypt([]byte("data"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	ciphertext[NonceLen+1] ^= 0xFF

	_, err = Decrypt(ciphertext, key)
	if err == nil {
		t.Error("expected error when decrypting corrupted data")
	}
}

func TestDecryptTooShort(t *testing.T) {
	t.Parallel()

	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	key := DeriveKey("password", salt)

	_, err = Decrypt([]byte("short"), key)
	if err == nil {
		t.Error("expected error for too-short ciphertext")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	t.Parallel()

	salt := []byte("0123456789abcdef")

	k1 := DeriveKey("password", salt)
	k2 := DeriveKey("password", salt)
	k3 := DeriveKey("different", salt)

	if *k1 != *k2 {
		t.Error("same password + salt should produce same key")
	}

	if *k1 == *k3 {
		t.Error("different password should produce different key")
	}
}
