package vault

import (
	"testing"
)

func TestVault_SetAndGet(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("DB_HOST", "localhost")

	val, ok := v.Get("DB_HOST")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "localhost" {
		t.Errorf("expected 'localhost', got %q", val)
	}
}

func TestVault_GetNonExistent(t *testing.T) {
	t.Parallel()

	v := New("test")
	_, ok := v.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent key")
	}
}

func TestVault_Overwrite(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("KEY", "old")
	v.Set("KEY", "new")

	val, _ := v.Get("KEY")
	if val != "new" {
		t.Errorf("expected 'new', got %q", val)
	}
}

func TestVault_Delete(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("KEY", "value")
	v.Delete("KEY")

	_, ok := v.Get("KEY")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestVault_DeleteNonExistent(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Delete("nonexistent")
}

func TestVault_Keys(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("A", "1")
	v.Set("B", "2")

	keys := v.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestVault_Export(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("A", "1")
	v.Set("B", "2")

	export := v.Export()
	if export != "A=1\nB=2\n" && export != "B=2\nA=1\n" {
		t.Errorf("unexpected export format: %q", export)
	}
}

func TestVault_All(t *testing.T) {
	t.Parallel()

	v := New("test")
	v.Set("A", "1")

	all := v.All()
	if len(all) != 1 || all["A"] != "1" {
		t.Errorf("unexpected All() result: %v", all)
	}

	v.Set("A", "2")
	if all["A"] != "1" {
		t.Error("All() should return a copy, not reference")
	}
}
