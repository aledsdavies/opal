package execution

import (
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// GeneratorExecutionContext implements GeneratorContext for Go code generation
type GeneratorExecutionContext struct {
	*BaseExecutionContext

	// Decorator lookup functions (set by engine during initialization)
	blockDecoratorLookup   func(name string) (interface{}, bool)
	patternDecoratorLookup func(name string) (interface{}, bool)
	valueDecoratorLookup   func(name string) (interface{}, bool)
}

// ================================================================================================
// GENERATOR-SPECIFIC FUNCTIONALITY
// ================================================================================================

// GenerateShellCode generates Go code for shell execution using unified decorator model
func (c *GeneratorExecutionContext) GenerateShellCode(content *ast.ShellContent) *ExecutionResult {
	// Use the unified shell code builder
	shellBuilder := NewShellCodeBuilder(c)
	code, err := shellBuilder.GenerateShellCode(content)
	if err != nil {
		return &ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to generate shell execution code: %w", err),
		}
	}

	return &ExecutionResult{
		Data:  code,
		Error: nil,
	}
}

// GenerateDirectActionCode generates Go code for ActionDecorator direct execution with CommandResult chaining
func (c *GeneratorExecutionContext) GenerateDirectActionCode(content *ast.ShellContent) *ExecutionResult {
	// Use the unified shell code builder
	shellBuilder := NewShellCodeBuilder(c)
	code, err := shellBuilder.GenerateDirectActionTemplate(content)
	if err != nil {
		return &ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to generate action chain code: %w", err),
		}
	}

	return &ExecutionResult{
		Data:  code,
		Error: nil,
	}
}

// GetTemplateFunctions returns the template function map for code generation
func (c *GeneratorExecutionContext) GetTemplateFunctions() template.FuncMap {
	// Create a unified shell code builder and get its template functions
	shellBuilder := NewShellCodeBuilder(c)
	return shellBuilder.GetTemplateFunctions()
}

// SetTemplateFunctions sets the template function map (used by engine)
func (c *GeneratorExecutionContext) SetTemplateFunctions(funcs template.FuncMap) {
	// Template functions are now handled by the unified shell code builder
	// This method is deprecated but kept for interface compatibility
}

// ================================================================================================
// INTERNAL ACCESS METHODS FOR TEMPLATE GENERATION
// ================================================================================================

// GetShellCounter returns the current shell counter for unique variable naming
func (c *GeneratorExecutionContext) GetShellCounter() int {
	return c.shellCounter
}

// IncrementShellCounter increments the shell counter for unique variable naming
func (c *GeneratorExecutionContext) IncrementShellCounter() {
	c.shellCounter++
}

// GetCurrentCommand returns the current command name for variable generation
func (c *GeneratorExecutionContext) GetCurrentCommand() string {
	return c.currentCommand
}

// GetBlockDecoratorLookup returns the block decorator lookup function
func (c *GeneratorExecutionContext) GetBlockDecoratorLookup() func(name string) (interface{}, bool) {
	// Block decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.blockDecoratorLookup
}

// GetPatternDecoratorLookup returns the pattern decorator lookup function
func (c *GeneratorExecutionContext) GetPatternDecoratorLookup() func(name string) (interface{}, bool) {
	// Pattern decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.patternDecoratorLookup
}

// GetValueDecoratorLookup returns the value decorator lookup function
func (c *GeneratorExecutionContext) GetValueDecoratorLookup() func(name string) (interface{}, bool) {
	// Value decorators are looked up through dependency injection to avoid import cycles
	// This will be set by the engine during initialization
	return c.valueDecoratorLookup
}

// SetBlockDecoratorLookup sets the block decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetBlockDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.blockDecoratorLookup = lookup
}

// SetPatternDecoratorLookup sets the pattern decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetPatternDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.patternDecoratorLookup = lookup
}

// SetValueDecoratorLookup sets the value decorator lookup function (called by engine during setup)
func (c *GeneratorExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.valueDecoratorLookup = lookup
}

// TrackEnvironmentVariableReference tracks which env vars are referenced for code generation
func (c *GeneratorExecutionContext) TrackEnvironmentVariableReference(key, defaultValue string) {
	// For now, generator context doesn't store these - they're handled by decorators
	// This method exists to satisfy calls from builtin decorators
}

// GetTrackedEnvironmentVariableReferences returns env var references for template generation
func (c *GeneratorExecutionContext) GetTrackedEnvironmentVariableReferences() map[string]string {
	// For now, return empty - the actual env var tracking happens in the engine
	// via decorator calls during code generation
	return make(map[string]string)
}

// ================================================================================================
// CONTEXT MANAGEMENT WITH TYPE SAFETY
// ================================================================================================

// Child creates a child generator context that inherits from the parent but can be modified independently
func (c *GeneratorExecutionContext) Child() GeneratorContext {
	// Increment child counter to ensure unique variable naming across parallel contexts
	c.childCounter++
	childID := c.childCounter

	childBase := &BaseExecutionContext{
		Context:   c.Context,
		Program:   c.Program,
		Variables: make(map[string]string),
		env:       c.env, // Share the same immutable environment reference

		// Copy execution state
		WorkingDir:     c.WorkingDir,
		Debug:          c.Debug,
		DryRun:         c.DryRun,
		currentCommand: c.currentCommand,

		// Initialize unique counter space for this child to avoid variable name conflicts
		// Each child gets a unique counter space based on parent's counter and child ID
		shellCounter: c.shellCounter + (childID * 1000), // Give each child 1000 numbers of space
		childCounter: 0,                                 // Reset child counter for this context's children
	}

	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childBase.Variables[name] = value
	}

	return &GeneratorExecutionContext{
		BaseExecutionContext: childBase,
		// Copy immutable configuration from parent to child
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
	}
}

// WithTimeout creates a new generator context with timeout
func (c *GeneratorExecutionContext) WithTimeout(timeout time.Duration) (GeneratorContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
	}, cancel
}

// WithCancel creates a new generator context with cancellation
func (c *GeneratorExecutionContext) WithCancel() (GeneratorContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
	}, cancel
}

// WithWorkingDir creates a new generator context with the specified working directory
func (c *GeneratorExecutionContext) WithWorkingDir(workingDir string) GeneratorContext {
	newBase := *c.BaseExecutionContext
	newBase.WorkingDir = workingDir
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
	}
}

// WithCurrentCommand creates a new generator context with the specified current command name
func (c *GeneratorExecutionContext) WithCurrentCommand(commandName string) GeneratorContext {
	newBase := *c.BaseExecutionContext
	newBase.currentCommand = commandName
	return &GeneratorExecutionContext{
		BaseExecutionContext: &newBase,
		// Copy decorator lookups from parent
		blockDecoratorLookup:   c.blockDecoratorLookup,
		patternDecoratorLookup: c.patternDecoratorLookup,
		valueDecoratorLookup:   c.valueDecoratorLookup,
	}
}
