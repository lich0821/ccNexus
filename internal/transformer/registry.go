package transformer

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Transformer)
	mu       sync.RWMutex
)

// Register registers a transformer
func Register(t Transformer) {
	mu.Lock()
	defer mu.Unlock()
	registry[t.Name()] = t
}

// Get retrieves a transformer by name
func Get(name string) (Transformer, error) {
	mu.RLock()
	defer mu.RUnlock()

	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("transformer not found: %s", name)
	}
	return t, nil
}

// IsRegistered checks if a transformer is registered
func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := registry[name]
	return ok
}

// List returns all registered transformer names
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
