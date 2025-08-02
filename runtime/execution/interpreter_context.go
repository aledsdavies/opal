package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// InterpreterExecutionContext implements InterpreterContext for direct command execution
type InterpreterExecutionContext struct {
	*BaseExecutionContext
}


// ================================================================================================
// INTERPRETER-SPECIFIC FUNCTIONALITY
// ================================================================================================

// ExecuteShell executes shell content directly
func (c *InterpreterExecutionContext) ExecuteShell(content *ast.ShellContent) *ExecutionResult {
	// Compose the command string from parts
	cmdStr, err := c.composeShellCommand(content)
	if err != nil {
		return &ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("failed to compose shell command: %w", err),
		}
	}

	// Execute the command
	cmd := exec.CommandContext(c.Context, "sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.WorkingDir != "" {
		cmd.Dir = c.WorkingDir
	}

	err = cmd.Run()
	return &ExecutionResult{
		Data:  nil,
		Error: err,
	}
}

// ExecuteCommandContent executes command content using the engine's executor (used by decorators)
func (c *InterpreterExecutionContext) ExecuteCommandContent(content ast.CommandContent) error {
	// This method is no longer needed with the new architecture
	// Commands should be executed directly through the registry patterns
	return fmt.Errorf("ExecuteCommandContent is deprecated - use direct decorator registry execution")
}

// ExecuteCommand executes a full command by name (used by decorators like @cmd)
func (c *InterpreterExecutionContext) ExecuteCommand(commandName string) error {
	// This method is no longer needed with the new architecture
	// Commands should be executed directly through the engine patterns
	return fmt.Errorf("ExecuteCommand is deprecated - use engine command execution patterns")
}

// ================================================================================================
// CONTEXT MANAGEMENT WITH TYPE SAFETY
// ================================================================================================

// Child creates a child interpreter context that inherits from the parent but can be modified independently
func (c *InterpreterExecutionContext) Child() InterpreterContext {
	// Increment child counter to ensure unique variable naming across parallel contexts
	c.childCounter++
	childID := c.childCounter
	
	childBase := &BaseExecutionContext{
		Context:   c.Context,
		Program:   c.Program,
		Variables: make(map[string]string),
		env:       c.env, // Share the same immutable environment reference
		
		// Copy execution state
		WorkingDir:    c.WorkingDir,
		Debug:         c.Debug,
		DryRun:        c.DryRun,
		currentCommand: c.currentCommand,
		
		// Initialize unique counter space for this child to avoid variable name conflicts
		// Each child gets a unique counter space based on parent's counter and child ID
		shellCounter: c.shellCounter + (childID * 1000), // Give each child 1000 numbers of space
		childCounter: 0, // Reset child counter for this context's children
		
		// Environment variable tracking for generators
		trackedEnvVars: make(map[string]string),
	}
	
	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childBase.Variables[name] = value
	}
	
	return &InterpreterExecutionContext{
		BaseExecutionContext: childBase,
	}
}

// WithTimeout creates a new interpreter context with timeout
func (c *InterpreterExecutionContext) WithTimeout(timeout time.Duration) (InterpreterContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithCancel creates a new interpreter context with cancellation
func (c *InterpreterExecutionContext) WithCancel() (InterpreterContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newBase := *c.BaseExecutionContext
	newBase.Context = ctx
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}, cancel
}

// WithWorkingDir creates a new interpreter context with the specified working directory
func (c *InterpreterExecutionContext) WithWorkingDir(workingDir string) InterpreterContext {
	newBase := *c.BaseExecutionContext
	newBase.WorkingDir = workingDir
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}
}

// WithCurrentCommand creates a new interpreter context with the specified current command name
func (c *InterpreterExecutionContext) WithCurrentCommand(commandName string) InterpreterContext {
	newBase := *c.BaseExecutionContext
	newBase.currentCommand = commandName
	return &InterpreterExecutionContext{BaseExecutionContext: &newBase}
}

// ================================================================================================
// SHELL COMMAND COMPOSITION
// ================================================================================================

// composeShellCommand composes the shell command string from AST parts
func (c *InterpreterExecutionContext) composeShellCommand(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		result, err := c.processShellPart(part)
		if err != nil {
			return "", err
		}

		if value, ok := result.(string); ok {
			parts = append(parts, value)
		} else {
			return "", fmt.Errorf("shell part returned non-string result: %T", result)
		}
	}

	return strings.Join(parts, ""), nil
}

// processShellPart processes any shell part (text, value decorator, action decorator) for interpreter mode
func (c *InterpreterExecutionContext) processShellPart(part ast.ShellPart) (interface{}, error) {
	switch p := part.(type) {
	case *ast.TextPart:
		return p.Text, nil

	case *ast.ValueDecorator:
		return c.processValueDecorator(p)

	case *ast.ActionDecorator:
		return c.processActionDecorator(p)

	default:
		return nil, fmt.Errorf("unsupported shell part type: %T", part)
	}
}

// processValueDecorator handles value decorators in interpreter mode
func (c *InterpreterExecutionContext) processValueDecorator(decorator *ast.ValueDecorator) (interface{}, error) {
	// This method is deprecated - use direct decorator registry access
	return nil, fmt.Errorf("processValueDecorator is deprecated - use decorator registry directly")
}

// processActionDecorator handles action decorators in interpreter mode
func (c *InterpreterExecutionContext) processActionDecorator(decorator *ast.ActionDecorator) (interface{}, error) {
	// This method is deprecated - use direct decorator registry access
	return nil, fmt.Errorf("processActionDecorator is deprecated - use decorator registry directly")
}

