package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewStore(dir)
	ctx := context.Background()

	data := []byte("encrypted-blob")
	if err := s.Save(ctx, "myapp", data); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := s.Load(ctx, "myapp")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if string(loaded) != string(data) {
		t.Errorf("round-trip failed: got %q, want %q", loaded, data)
	}
}

func TestStoreLoadNonExistent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewStore(dir)
	ctx := context.Background()

	_, err := s.Load(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent vault")
	}
}

func TestStoreList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewStore(dir)
	ctx := context.Background()

	s.Save(ctx, "app1", []byte("data1"))
	s.Save(ctx, "app2", []byte("data2"))

	names, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(names))
	}
}

func TestStoreListEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewStore(dir)
	ctx := context.Background()

	names, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(names) != 0 {
		t.Errorf("expected 0 vaults, got %d", len(names))
	}
}

func TestStorePermissionDenied(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.Chmod(dir, 0000); err != nil {
		t.Skipf("cannot change permissions: %v", err)
	}
	defer os.Chmod(dir, 0700)

	s := NewStore(filepath.Join(dir, "subdir"))
	ctx := context.Background()

	err := s.Save(ctx, "test", []byte("data"))
	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}

func TestSerializeDeserializeVault(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("A", "1")
	v.Set("B", "2")

	data, err := SerializeVault(v)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	restored, err := DeserializeVault("test", data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}

	val, ok := restored.Get("A")
	if !ok || val != "1" {
		t.Errorf("expected A=1 after restore, got %q", val)
	}

	val, ok = restored.Get("B")
	if !ok || val != "2" {
		t.Errorf("expected B=2 after restore, got %q", val)
	}
}
