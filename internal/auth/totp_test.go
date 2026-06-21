package auth

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestTOTPHandler_GenerateAndValidate(t *testing.T) {
	t.Parallel()

	h := NewTOTPHandler()
	ctx := context.Background()

	key, err := h.Generate(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if key.Secret == "" {
		t.Error("expected non-empty secret")
	}

	code, err := totp.GenerateCode(key.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	if !h.Validate(ctx, key.Secret, code) {
		t.Error("expected valid code to validate")
	}
}

func TestTOTPHandler_ValidateWrongCode(t *testing.T) {
	t.Parallel()

	h := NewTOTPHandler()
	ctx := context.Background()

	key, _ := h.Generate(ctx, "test@example.com")

	if h.Validate(ctx, key.Secret, "000000") {
		t.Error("expected invalid code to fail")
	}
}

func TestTOTPHandler_ValidateEmptyCode(t *testing.T) {
	t.Parallel()

	h := NewTOTPHandler()
	ctx := context.Background()

	key, _ := h.Generate(ctx, "test@example.com")

	if h.Validate(ctx, key.Secret, "") {
		t.Error("expected empty code to fail")
	}
}
