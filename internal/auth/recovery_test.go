package auth

import (
	"context"
	"testing"
)

func TestRecoveryHandler_GenerateAndValidate(t *testing.T) {
	t.Parallel()

	r := NewRecoveryHandler(nil)
	ctx := context.Background()

	codes, err := r.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}

	for _, code := range codes {
		if !r.Validate(ctx, code) {
			t.Errorf("expected code %q to be valid", code)
		}
	}
}

func TestRecoveryHandler_UseOnce(t *testing.T) {
	t.Parallel()

	r := NewRecoveryHandler(nil)
	ctx := context.Background()

	codes, _ := r.Generate(ctx)
	code := codes[0]

	if !r.Use(ctx, code) {
		t.Fatal("expected Use to return true on first use")
	}

	if r.Validate(ctx, code) {
		t.Error("expected code to be invalid after use")
	}

	if r.Use(ctx, code) {
		t.Error("expected Use to return false for already used code")
	}
}

func TestRecoveryHandler_ValidateUnknownCode(t *testing.T) {
	t.Parallel()

	r := NewRecoveryHandler(nil)
	ctx := context.Background()

	if r.Validate(ctx, "unknown-recovery-code") {
		t.Error("expected unknown code to be invalid")
	}
}
