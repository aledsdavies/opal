package types

import "sync"

// Value represents a runtime value in Opal
// This is a placeholder - will be expanded as we implement the runtime
type Value interface{}

// Block represents a code block passed to a decorator
// This is a placeholder - will be expanded when we implement block execution
type Block interface {
	// Execute runs the block in the given context
	Execute(ctx Context) error
}

// Context holds the execution context for decorator handlers
type Context struct {
	Variables  map[string]Value  // Variable bindings: var x = "value"
	Env        map[string]string // Environment variables
	WorkingDir string            // Current working directory
}

// Args holds the arguments passed to a decorator handler
type Args struct {
	Primary *Value           // Primary property: @env.HOME → "HOME"
	Params  map[string]Value // Named parameters: (default="", times=3)
	Block   *Block           // Lambda/block for execution decorators
}

// ValueHandler is a function that implements a value decorator
// Returns data with no side effects
type ValueHandler func(ctx Context, args Args) (Value, error)

// ExecutionHandler is a function that implements an execution decorator
// Performs actions with side effects
type ExecutionHandler func(ctx Context, args Args) error

// DecoratorKind represents the type of decorator
type DecoratorKind int

const (
	// DecoratorKindValue returns data with no side effects (can be interpolated in strings)
	DecoratorKindValue DecoratorKind = iota
	// DecoratorKindExecution performs actions with side effects (cannot be interpolated)
	DecoratorKindExecution
)

// DecoratorInfo holds metadata about a registered decorator
type DecoratorInfo struct {
	Path             string           // Full path: "var", "env", "file.read", "aws.instance.data"
	Kind             DecoratorKind    // Value or Execution
	ValueHandler     ValueHandler     // Handler for value decorators (nil for execution)
	ExecutionHandler ExecutionHandler // Handler for execution decorators (nil for value)
}

// Registry holds registered decorator paths and their metadata
type Registry struct {
	mu         sync.RWMutex
	decorators map[string]DecoratorInfo
}

// NewRegistry creates a new decorator registry
func NewRegistry() *Registry {
	return &Registry{
		decorators: make(map[string]DecoratorInfo),
	}
}

// RegisterValue registers a value decorator (returns data, no side effects)
// Can be used in string interpolation
func (r *Registry) RegisterValue(path string, handler ValueHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[path] = DecoratorInfo{
		Path:         path,
		Kind:         DecoratorKindValue,
		ValueHandler: handler,
	}
}

// RegisterExecution registers an execution decorator (performs actions with side effects)
// Cannot be used in string interpolation
func (r *Registry) RegisterExecution(path string, handler ExecutionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[path] = DecoratorInfo{
		Path:             path,
		Kind:             DecoratorKindExecution,
		ExecutionHandler: handler,
	}
}

// GetValueHandler retrieves the handler for a value decorator
func (r *Registry) GetValueHandler(path string) (ValueHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	if !exists || info.Kind != DecoratorKindValue {
		return nil, false
	}
	return info.ValueHandler, true
}

// GetExecutionHandler retrieves the handler for an execution decorator
func (r *Registry) GetExecutionHandler(path string) (ExecutionHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	if !exists || info.Kind != DecoratorKindExecution {
		return nil, false
	}
	return info.ExecutionHandler, true
}

// Register adds a decorator (defaults to value for backward compatibility)
// Deprecated: Use RegisterValue or RegisterExecution instead
func (r *Registry) Register(name string) {
	// For backward compatibility, register with a nil handler
	// This allows existing tests to pass while we migrate to the new pattern
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[name] = DecoratorInfo{
		Path: name,
		Kind: DecoratorKindValue,
	}
}

// IsRegistered checks if a decorator path is registered
func (r *Registry) IsRegistered(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.decorators[path]
	return exists
}

// IsValueDecorator checks if a decorator path is registered as a value decorator
func (r *Registry) IsValueDecorator(path string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, exists := r.decorators[path]
	return exists && info.Kind == DecoratorKindValue
}

// Global registry instance
var globalRegistry = NewRegistry()

// Global returns the global decorator registry
func Global() *Registry {
	return globalRegistry
}
