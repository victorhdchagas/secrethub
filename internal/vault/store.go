package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	BaseDir string
}

func NewStore(baseDir string) *Store {
	return &Store{BaseDir: baseDir}
}

func (s *Store) Save(ctx context.Context, name string, data []byte) error {
	if err := os.MkdirAll(s.BaseDir, 0700); err != nil {
		return fmt.Errorf("create vault dir: %w", err)
	}

	path := s.path(name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write vault %s: %w", name, err)
	}
	return nil
}

func (s *Store) Load(ctx context.Context, name string) ([]byte, error) {
	path := s.path(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault %s not found", name)
		}
		return nil, fmt.Errorf("read vault %s: %w", name, err)
	}
	return data, nil
}

func (s *Store) Delete(ctx context.Context, name string) error {
	path := s.path(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vault %s not found", name)
		}
		return fmt.Errorf("delete vault %s: %w", name, err)
	}
	return nil
}

func (s *Store) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.BaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list vaults: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".enc" {
			name := e.Name()[:len(e.Name())-4]
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *Store) path(name string) string {
	return filepath.Join(s.BaseDir, name+".enc")
}

// SerializeVault serializa o vault para JSON.
func SerializeVault(v *Vault) ([]byte, error) {
	return json.Marshal(v.All())
}

// DeserializeVault restaura o vault a partir de JSON.
func DeserializeVault(name string, data []byte) (*Vault, error) {
	var keys map[string]string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("deserialize vault: %w", err)
	}
	v := New(name)
	for k, val := range keys {
		v.Set(k, val)
	}
	return v, nil
}
