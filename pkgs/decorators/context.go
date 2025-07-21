package decorators

import (
	"context"
	"fmt"
	"text/template"
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

	// Template functions for code generation (populated by engine)
	templateFunctions template.FuncMap

	// Command content executor for nested command execution (populated by engine)
	contentExecutor func(ast.CommandContent) error
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(parent context.Context, program *ast.Program) *ExecutionContext {
	if parent == nil {
		parent = context.Background()
	}

	return &ExecutionContext{
		Context:           parent,
		Program:           program,
		Variables:         make(map[string]string),
		Env:               make(map[string]string),
		Debug:             false,
		DryRun:            false,
		templateFunctions: make(template.FuncMap),
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

// InitializeVariables processes and sets all variables from the program
func (c *ExecutionContext) InitializeVariables() error {
	if c.Program == nil {
		return nil
	}

	// Process individual variables
	for _, variable := range c.Program.Variables {
		value, err := c.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		c.SetVariable(variable.Name, value)
	}

	// Process variable groups
	for _, group := range c.Program.VarGroups {
		for _, variable := range group.Variables {
			value, err := c.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
			}
			c.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// GetTemplateFunctions returns the template function map for code generation
func (c *ExecutionContext) GetTemplateFunctions() template.FuncMap {
	return c.templateFunctions
}

// SetTemplateFunctions sets the template function map (used by engine)
func (c *ExecutionContext) SetTemplateFunctions(funcs template.FuncMap) {
	c.templateFunctions = funcs
}

// ExecuteCommandContent executes command content using the engine's executor (used by decorators)
func (c *ExecutionContext) ExecuteCommandContent(content ast.CommandContent) error {
	if c.contentExecutor == nil {
		return fmt.Errorf("command content executor not available (engine not properly initialized)")
	}
	return c.contentExecutor(content)
}

// SetContentExecutor sets the command content executor (used by engine)
func (c *ExecutionContext) SetContentExecutor(executor func(ast.CommandContent) error) {
	c.contentExecutor = executor
}

// resolveVariableValue converts an AST expression to its string value
func (c *ExecutionContext) resolveVariableValue(expr ast.Expression) (string, error) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value, nil
	case *ast.NumberLiteral:
		return v.Value, nil
	case *ast.BooleanLiteral:
		if v.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.DurationLiteral:
		return v.Value, nil
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}
