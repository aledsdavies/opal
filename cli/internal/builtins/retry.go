package decorators

import (
	"fmt"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// RetryDecorator implements the @retry decorator for retrying failed command execution
type RetryDecorator struct{}

// Name returns the decorator name
func (r *RetryDecorator) Name() string {
	return "retry"
}

// Description returns a human-readable description
func (r *RetryDecorator) Description() string {
	return "Retry command execution on failure with configurable attempts and delay"
}

// ParameterSchema returns the expected parameters for this decorator
func (r *RetryDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "attempts",
			Type:        ast.NumberType,
			Required:    true,
			Description: "Maximum number of retry attempts",
		},
		{
			Name:        "delay",
			Type:        ast.DurationType,
			Required:    false,
			Description: "Delay between retry attempts (default: 1s)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes retry logic in interpreter mode
func (r *RetryDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	maxAttempts, delay, err := r.extractRetryParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return r.executeInterpreterImpl(ctx, maxAttempts, delay, content)
}

// ExecuteGenerator generates Go code for retry logic
func (r *RetryDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	maxAttempts, delay, err := r.extractRetryParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	return r.executeGeneratorImpl(ctx, maxAttempts, delay, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (r *RetryDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	maxAttempts, delay, err := r.extractRetryParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return r.executePlanImpl(ctx, maxAttempts, delay, content)
}

// extractRetryParams extracts and validates retry parameters
func (r *RetryDecorator) extractRetryParams(params []ast.NamedParameter) (int, time.Duration, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 2, "retry"); err != nil {
		return 0, 0, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, r.ParameterSchema(), "retry"); err != nil {
		return 0, 0, err
	}

	// Validate attempts parameter is positive
	if err := decorators.ValidatePositiveInteger(params, "attempts", "retry"); err != nil {
		return 0, 0, err
	}

	// Enhanced security validation for attempts to prevent resource exhaustion
	if err := decorators.ValidateResourceLimits(params, "attempts", 100, "retry"); err != nil {
		return 0, 0, err
	}

	// Validate delay parameter if present (1ms to 1 hour range)
	if err := decorators.ValidateDuration(params, "delay", 1*time.Millisecond, 1*time.Hour, "retry"); err != nil {
		return 0, 0, err
	}

	// Enhanced security validation for timeout safety
	if err := decorators.ValidateTimeoutSafety(params, "delay", 1*time.Hour, "retry"); err != nil {
		return 0, 0, err
	}

	// Parse parameters (validation passed, so these should be safe)
	maxAttempts := ast.GetIntParam(params, "attempts", 3)
	delay := ast.GetDurationParam(params, "delay", 1*time.Second)

	return maxAttempts, delay, nil
}

// executeInterpreterImpl executes retry logic in interpreter mode using utilities
func (r *RetryDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create RetryExecutor with specified attempts and delay
	retryExecutor := decorators.NewRetryExecutor(maxAttempts, delay)
	defer retryExecutor.Cleanup()

	// Execute all commands within the retry logic using the utility
	err := retryExecutor.Execute(func() error {
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

// executeGeneratorImpl generates Go code for retry logic using new utilities
func (r *RetryDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Convert AST commands to a single operation containing all sequential commands
	executor := decorators.NewCommandResultExecutor(ctx)
	operations, err := executor.ConvertCommandsToCommandResultOperations(content)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to convert commands to operations: %w", err),
		}
	}

	// Combine all operations into a single sequential operation for retry wrapping
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
		// Use TemplateBuilder to create sequential execution, then wrap with retry
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

	// Create a single operation from the combined code and wrap with retry
	operation := decorators.Operation{Code: combinedCode}

	// Use TemplateBuilder to create retry pattern with pre-validated delay
	builder := decorators.NewTemplateBuilder()
	delayExpr := decorators.DurationToGoExpr(delay)
	builder.WithRetryExpr(maxAttempts, delayExpr, operation)

	// Build the template
	generatedCode, err := builder.BuildTemplate()
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to build retry template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  generatedCode,
		Error: nil,
	}
}

// executePlanImpl creates a plan element for dry-run mode
func (r *RetryDecorator) executePlanImpl(ctx execution.PlanContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	delayStr := delay.String()

	description := fmt.Sprintf("Execute %d commands with up to %d attempts", len(content), maxAttempts)
	if delayStr != "" && delayStr != "0s" {
		description += fmt.Sprintf(", %s delay between retries", delayStr)
	}

	element := plan.Decorator("retry").
		WithType("block").
		WithParameter("attempts", fmt.Sprintf("%d", maxAttempts)).
		WithDescription(description)

	if delayStr != "" && delayStr != "0s" {
		element = element.WithParameter("delay", delayStr)
	}

	// Build child plan elements for each command in the retry block
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
func (r *RetryDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"fmt", "time"}, // Required by RetryPattern
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the retry decorator
func init() {
	decorators.RegisterBlock(&RetryDecorator{})
}
