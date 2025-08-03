package decorators

import (
	"fmt"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)



// TimeoutDecorator implements the @timeout decorator for command execution with time limits
type TimeoutDecorator struct{}

// Name returns the decorator name
func (t *TimeoutDecorator) Name() string {
	return "timeout"
}

// Description returns a human-readable description
func (t *TimeoutDecorator) Description() string {
	return "Execute commands with a time limit, cancelling on timeout"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TimeoutDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "duration",
			Type:        ast.DurationType,
			Required:    true,
			Description: "Maximum execution time (e.g., '30s', '5m', '1h')",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes commands with timeout in interpreter mode
func (t *TimeoutDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	timeout, err := t.extractTimeout(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return t.executeInterpreterImpl(ctx, timeout, content)
}

// ExecuteGenerator generates Go code for timeout logic
func (t *TimeoutDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	timeout, err := t.extractTimeout(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	return t.executeGeneratorImpl(ctx, timeout, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (t *TimeoutDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	timeout, err := t.extractTimeout(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return t.executePlanImpl(ctx, timeout, content)
}

// extractTimeout extracts and validates the timeout duration from parameters
func (t *TimeoutDecorator) extractTimeout(params []ast.NamedParameter) (time.Duration, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 0, 1, "timeout"); err != nil {
		return 0, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, t.ParameterSchema(), "timeout"); err != nil {
		return 0, err
	}

	// Validate duration parameter if present (1ms to 24 hours range)
	if err := decorators.ValidateDuration(params, "duration", 1*time.Millisecond, 24*time.Hour, "timeout"); err != nil {
		return 0, err
	}
	
	// Enhanced security validation for timeout safety
	if err := decorators.ValidateTimeoutSafety(params, "duration", 24*time.Hour, "timeout"); err != nil {
		return 0, err
	}

	// Parse parameters (validation passed, so these should be safe)
	// If no duration parameter provided, use default of 30 seconds
	return ast.GetDurationParam(params, "duration", 30*time.Second), nil
}

// executeInterpreterImpl executes commands with timeout in interpreter mode using utilities
func (t *TimeoutDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create TimeoutExecutor with specified timeout
	timeoutExecutor := decorators.NewTimeoutExecutor(timeout)
	defer timeoutExecutor.Cleanup()

	// Execute all commands within the timeout using the utility
	err := timeoutExecutor.Execute(func() error {
		// Execute commands sequentially with isolated context
		childCtx := ctx.Child()
		
		// Use CommandExecutor utility to handle all commands
		commandExecutor := decorators.NewCommandExecutor()
		defer commandExecutor.Cleanup()
		
		return commandExecutor.ExecuteCommandsWithInterpreter(childCtx, content)
	})

	return &execution.ExecutionResult{
		Data:  nil,
		Error: err,
	}
}


// executeGeneratorImpl generates Go code for timeout logic using new utilities
func (t *TimeoutDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Convert AST commands to a single operation containing all sequential commands
	operations, err := decorators.ConvertCommandsToOperations(ctx, content)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to convert commands to operations: %w", err),
		}
	}

	// Combine all operations into a single sequential operation for timeout wrapping
	if len(operations) == 0 {
		return &execution.ExecutionResult{
			Data:  "// No operations to execute",
			Error: nil,
		}
	}

	var combinedCode string
	if len(operations) == 1 {
		combinedCode = operations[0].Code
	} else {
		// Use TemplateBuilder to create sequential execution, then wrap with timeout
		sequentialBuilder := decorators.NewTemplateBuilder()
		sequentialBuilder.WithSequentialExecution(operations, true) // Stop on error
		
		sequentialCode, err := sequentialBuilder.BuildTemplate()
		if err != nil {
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("failed to build sequential template: %w", err),
			}
		}
		combinedCode = sequentialCode
	}

	// Create a single operation from the combined code and wrap with timeout
	operation := decorators.Operation{Code: combinedCode}
	
	// Use TemplateBuilder to create timeout pattern
	builder := decorators.NewTemplateBuilder()
	builder.WithTimeout(timeout.String(), operation)

	// Build the template
	generatedCode, err := builder.BuildTemplate()
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to build timeout template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  generatedCode,
		Error: nil,
	}
}

// executePlanImpl creates a plan element for dry-run mode
func (t *TimeoutDecorator) executePlanImpl(ctx execution.PlanContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	durationStr := timeout.String()
	description := fmt.Sprintf("Execute %d commands with %s timeout (cancel if exceeded)", len(content), durationStr)

	element := plan.Decorator("timeout").
		WithType("block").
		WithTimeout(timeout).
		WithParameter("duration", durationStr).
		WithDescription(description)

	// Build child plan elements for each command in the timeout block
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			// Create plan element for shell command
			result := ctx.GenerateShellPlan(c)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("failed to create plan for shell content: %w", result.Error),
				}
			}

			// Extract command string from result
			if planData, ok := result.Data.(map[string]interface{}); ok {
				if cmdStr, ok := planData["command"].(string); ok {
					childDesc := "Execute shell command"
					if desc, ok := planData["description"].(string); ok {
						childDesc = desc
					}
					childElement := plan.Command(cmdStr).WithDescription(childDesc)
					element = element.AddChild(childElement)
				}
			}
		case *ast.BlockDecorator:
			// For nested decorators, just add a placeholder - they will be handled by the engine
			childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription("Nested decorator")
			element = element.AddChild(childElement)
		}
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TimeoutDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"context", "fmt", "time"}, // Required by TimeoutPattern
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the timeout decorator
func init() {
	decorators.RegisterBlock(&TimeoutDecorator{})
}
