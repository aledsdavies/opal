package decorators

import (
	"context"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// ExecutionContext provides execution context for decorators and implements context.Context
type ExecutionContext struct {
	context.Context
	
	// Core data
	Program   *ast.Program
	Variables map[string]string // Resolved variable values
	Env       map[string]string // Environment variables
	
	// Execution state
	WorkingDir string
	Debug      bool
	DryRun     bool
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(parent context.Context, program *ast.Program) *ExecutionContext {
	if parent == nil {
		parent = context.Background()
	}
	
	return &ExecutionContext{
		Context:   parent,
		Program:   program,
		Variables: make(map[string]string),
		Env:       make(map[string]string),
		Debug:     false,
		DryRun:    false,
	}
}

// WithTimeout creates a new context with timeout
func (c *ExecutionContext) WithTimeout(timeout time.Duration) (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// WithCancel creates a new context with cancellation
func (c *ExecutionContext) WithCancel() (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// GetVariable retrieves a variable value
func (c *ExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetVariable sets a variable value
func (c *ExecutionContext) SetVariable(name, value string) {
	c.Variables[name] = value
}

// GetEnv retrieves an environment variable
func (c *ExecutionContext) GetEnv(name string) (string, bool) {
	value, exists := c.Env[name]
	return value, exists
}

// SetEnv sets an environment variable
func (c *ExecutionContext) SetEnv(name, value string) {
	c.Env[name] = value
}