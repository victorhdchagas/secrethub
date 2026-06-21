package vault

import "sync"

type Vault struct {
	mu   sync.RWMutex
	Name string
	keys map[string]string
}

func New(name string) *Vault {
	return &Vault{
		Name: name,
		keys: make(map[string]string),
	}
}

func (v *Vault) Set(key, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys[key] = value
}

func (v *Vault) Get(key string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	val, ok := v.keys[key]
	return val, ok
}

func (v *Vault) Delete(key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.keys, key)
}

func (v *Vault) Keys() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	keys := make([]string, 0, len(v.keys))
	for k := range v.keys {
		keys = append(keys, k)
	}
	return keys
}

func (v *Vault) Export() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	var out string
	for k, val := range v.keys {
		out += k + "=" + val + "\n"
	}
	return out
}

func (v *Vault) All() map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	cp := make(map[string]string, len(v.keys))
	for k, val := range v.keys {
		cp[k] = val
	}
	return cp
}
