package types

import "sync"

// Registry holds registered decorator names
type Registry struct {
	mu         sync.RWMutex
	decorators map[string]bool
}

// NewRegistry creates a new decorator registry
func NewRegistry() *Registry {
	return &Registry{
		decorators: make(map[string]bool),
	}
}

// Register adds a decorator name to the registry
func (r *Registry) Register(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[name] = true
}

// IsRegistered checks if a decorator name is registered
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.decorators[name]
}

// Global registry instance
var globalRegistry = NewRegistry()

// Global returns the global decorator registry
func Global() *Registry {
	return globalRegistry
}
