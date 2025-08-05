package execution

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ChainElement represents an element in an ActionDecorator command chain
type ChainElement struct {
	Type         string               // "action", "operator", "text"
	ActionName   string               // For ActionDecorator
	ActionArgs   []ast.NamedParameter // For ActionDecorator
	Operator     string               // "&&", "||", "|", ">>"
	Text         string               // For text parts
	VariableName string               // Generated variable name
	IsPipeTarget bool                 // True if this element receives piped input
	IsFileTarget bool                 // True if this element is a file for >> operation
}

// ChainOperator represents the type of chaining operator
type ChainOperator string

const (
	AndOperator    ChainOperator = "&&" // Execute next if current succeeds
	OrOperator     ChainOperator = "||" // Execute next if current fails
	PipeOperator   ChainOperator = "|"  // Pipe stdout to next command
	AppendOperator ChainOperator = ">>" // Append stdout to file
)

// BaseExecutionContext provides the common implementation for all execution contexts
type BaseExecutionContext struct {
	context.Context

	// Core data
	Program   *ast.Program
	Variables map[string]string // Resolved variable values
	env       map[string]string // Immutable environment variables captured at command start

	// Execution state
	WorkingDir string
	Debug      bool
	DryRun     bool

	// Current command name for generating meaningful variable names
	currentCommand string

	// Decorator lookup functions (set by engine during initialization)
	valueDecoratorLookup func(name string) (interface{}, bool)

	// Shell execution counter for unique variable naming
	shellCounter int

	// Child context counter for unique variable naming across parallel contexts
	childCounter int
}

// SetValueDecoratorLookup sets the value decorator lookup function (called by engine during setup)
func (c *BaseExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.valueDecoratorLookup = lookup
}

// newBaseExecutionContext creates a new base execution context
func newBaseExecutionContext(parent context.Context, program *ast.Program) *BaseExecutionContext {
	if parent == nil {
		parent = context.Background()
	}

	// Initialize working directory to current directory
	workingDir, err := os.Getwd()
	if err != nil {
		// Fallback to empty string if we can't get current directory
		workingDir = ""
	}

	// Capture environment variables immutably at command start for security
	// This prevents manipulation during execution and ensures consistent environment state
	capturedEnv := make(map[string]string)
	for _, envVar := range os.Environ() {
		if idx := strings.Index(envVar, "="); idx > 0 {
			key := envVar[:idx]
			value := envVar[idx+1:]
			capturedEnv[key] = value
		}
	}

	return &BaseExecutionContext{
		Context:    parent,
		Program:    program,
		Variables:  make(map[string]string),
		env:        capturedEnv, // Immutable captured environment
		WorkingDir: workingDir,
		Debug:      false,
		DryRun:     false,
	}
}

// GetVariable retrieves a variable value
func (c *BaseExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetVariable sets a variable value
func (c *BaseExecutionContext) SetVariable(name, value string) {
	c.Variables[name] = value
}

// GetEnv retrieves an environment variable from the immutable captured environment
func (c *BaseExecutionContext) GetEnv(name string) (string, bool) {
	value, exists := c.env[name]
	return value, exists
}

// GetProgram returns the AST program
func (c *BaseExecutionContext) GetProgram() *ast.Program {
	return c.Program
}

// GetWorkingDir returns the current working directory
func (c *BaseExecutionContext) GetWorkingDir() string {
	return c.WorkingDir
}

// IsDebug returns whether debug mode is enabled
func (c *BaseExecutionContext) IsDebug() bool {
	return c.Debug
}

// IsDryRun returns whether dry run mode is enabled
func (c *BaseExecutionContext) IsDryRun() bool {
	return c.DryRun
}

// InitializeVariables processes and sets all variables from the program
func (c *BaseExecutionContext) InitializeVariables() error {
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

// resolveVariableValue converts an AST expression to its string value
func (c *BaseExecutionContext) resolveVariableValue(expr ast.Expression) (string, error) {
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

// ================================================================================================
// SHARED UTILITY METHODS
// ================================================================================================

// getBaseName returns the base name for variable generation
func (c *BaseExecutionContext) getBaseName() string {
	if c.currentCommand != "" {
		return strings.Title(c.currentCommand)
	}
	return "Action"
}

// Helper function to format parameters for Go code generation
func formatParams(params []ast.NamedParameter) string {
	if len(params) == 0 {
		return "nil"
	}
	// For now, return simple representation - this needs to be expanded
	return "[]ast.NamedParameter{}"
}
