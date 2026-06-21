package auth

import (
	"context"
	"testing"
)

func TestBCryptHasher_HashAndVerify(t *testing.T) {
	t.Parallel()

	h := NewBCryptHasher(4)
	ctx := context.Background()

	hash, err := h.Hash(ctx, "my-secret-password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	if err := h.Verify(ctx, "my-secret-password", hash); err != nil {
		t.Errorf("Verify with correct password: %v", err)
	}
}

func TestBCryptHasher_VerifyWrongPassword(t *testing.T) {
	t.Parallel()

	h := NewBCryptHasher(4)
	ctx := context.Background()

	hash, _ := h.Hash(ctx, "correct-password")

	if err := h.Verify(ctx, "wrong-password", hash); err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestBCryptHasher_VerifyEmptyPassword(t *testing.T) {
	t.Parallel()

	h := NewBCryptHasher(4)
	ctx := context.Background()

	hash, _ := h.Hash(ctx, "some-password")

	if err := h.Verify(ctx, "", hash); err == nil {
		t.Error("expected error for empty password, got nil")
	}
}
