package auth

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTokenHandler_GenerateAndValidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	th := NewTokenHandler(filepath.Join(dir, "tokens.json"))
	ctx := context.Background()

	if err := th.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}

	var vk [32]byte
	vk[0] = 0xDE
	vk[31] = 0xAD

	token, err := th.Generate(ctx, &vk)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if len(token) != 64 {
		t.Errorf("expected 64-char token, got %d", len(token))
	}

	got, err := th.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if got[0] != 0xDE || got[31] != 0xAD {
		t.Error("vault key mismatch after validate")
	}
}

func TestTokenHandler_ValidateWrongToken(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	th := NewTokenHandler(filepath.Join(dir, "tokens.json"))
	ctx := context.Background()
	th.Load(ctx)

	var vk [32]byte
	th.Generate(ctx, &vk)

	_, err := th.Validate(ctx, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err == nil {
		t.Error("expected error for wrong token")
	}
}

func TestTokenHandler_Revoke(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	th := NewTokenHandler(filepath.Join(dir, "tokens.json"))
	ctx := context.Background()
	th.Load(ctx)

	var vk [32]byte
	token, _ := th.Generate(ctx, &vk)

	infos, _ := th.List(ctx)
	if len(infos) != 1 {
		t.Fatalf("expected 1 token, got %d", len(infos))
	}

	if err := th.Revoke(ctx, infos[0].Prefix); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, err := th.Validate(ctx, token)
	if err == nil {
		t.Error("expected validate to fail after revoke")
	}

	infos, _ = th.List(ctx)
	if len(infos) != 0 {
		t.Errorf("expected 0 tokens after revoke, got %d", len(infos))
	}
}

func TestTokenHandler_RevokeUnknown(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	th := NewTokenHandler(filepath.Join(dir, "tokens.json"))
	ctx := context.Background()
	th.Load(ctx)

	if err := th.Revoke(ctx, "unknown"); err == nil {
		t.Error("expected error for unknown prefix")
	}
}

func TestTokenHandler_Persistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	th1 := NewTokenHandler(path)
	ctx := context.Background()
	th1.Load(ctx)

	var vk [32]byte
	vk[0] = 0x42
	token, _ := th1.Generate(ctx, &vk)

	th2 := NewTokenHandler(path)
	th2.Load(ctx)

	got, err := th2.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate after reload: %v", err)
	}
	if got[0] != 0x42 {
		t.Error("vault key mismatch after reload")
	}
}

func TestTokenHandler_LoadNonExistent(t *testing.T) {
	t.Parallel()

	th := NewTokenHandler("/nonexistent/path/tokens.json")
	ctx := context.Background()

	if err := th.Load(ctx); err != nil {
		t.Fatalf("Load of nonexistent file should succeed: %v", err)
	}

	infos, _ := th.List(ctx)
	if len(infos) != 0 {
		t.Error("expected empty list after load of nonexistent file")
	}
}

func TestTokenHandler_RejectsCorruptKeyCipher(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	th := NewTokenHandler(filepath.Join(dir, "tokens.json"))
	ctx := context.Background()
	th.Load(ctx)

	var vk [32]byte
	token, _ := th.Generate(ctx, &vk)

	// corrupt the stored entry on disk
	th.mu.Lock()
	th.entries[0].KeyCipher = []byte{0, 1, 2, 3}
	th.save()
	th.mu.Unlock()

	// reload and validate
	th2 := NewTokenHandler(th.path)
	th2.Load(ctx)
	_, err := th2.Validate(ctx, token)
	if err == nil {
		t.Error("expected error for corrupt key cipher")
	}
}
