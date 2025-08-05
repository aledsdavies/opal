package decorators

import (
	"fmt"
	"runtime"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ParallelDecorator implements the @parallel decorator for concurrent command execution
type ParallelDecorator struct{}

// Name returns the decorator name
func (p *ParallelDecorator) Name() string {
	return "parallel"
}

// Description returns a human-readable description
func (p *ParallelDecorator) Description() string {
	return "Execute commands concurrently with optional concurrency limit and fail-fast behavior"
}

// ParameterSchema returns the expected parameters for this decorator
func (p *ParallelDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "concurrency",
			Type:        ast.NumberType,
			Required:    false,
			Description: "Maximum number of commands to run concurrently (default: CPU cores * 2, capped for safety)",
		},
		{
			Name:        "failOnFirstError",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Cancel remaining tasks on first error (default: false)",
		},
		{
			Name:        "uncapped",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Disable CPU-based concurrency capping (default: false, use with caution)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes commands concurrently in interpreter mode
func (p *ParallelDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return execution.NewErrorResult(err)
	}

	return p.executeInterpreterImpl(ctx, concurrency, failOnFirstError, content)
}

// ExecuteGenerator generates Go code for parallel execution
func (p *ParallelDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	return p.executeGeneratorImpl(ctx, concurrency, failOnFirstError, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (p *ParallelDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	concurrency, failOnFirstError, err := p.extractParallelParams(params, len(content))
	if err != nil {
		return execution.NewErrorResult(err)
	}

	return p.executePlanImpl(ctx, concurrency, failOnFirstError, content)
}

// extractParallelParams extracts and validates parallel parameters
func (p *ParallelDecorator) extractParallelParams(params []ast.NamedParameter, contentLength int) (int, bool, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 0, 3, "parallel"); err != nil {
		return 0, false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, p.ParameterSchema(), "parallel"); err != nil {
		return 0, false, err
	}

	// Enhanced security validation for concurrency parameter
	if err := decorators.ValidatePositiveInteger(params, "concurrency", "parallel"); err != nil {
		// ValidatePositiveInteger returns error if parameter is invalid, but not if missing
		// Check if the parameter exists first
		if ast.FindParameter(params, "concurrency") != nil {
			return 0, false, err
		}
	}

	// Validate resource limits for concurrency to prevent DoS attacks
	if err := decorators.ValidateResourceLimits(params, "concurrency", 1000, "parallel"); err != nil {
		return 0, false, err
	}

	// Parse parameters with defaults (validation passed, so these should be safe)
	defaultConcurrency := contentLength
	if defaultConcurrency == 0 {
		defaultConcurrency = 1 // Always have a positive default
	}

	concurrency := ast.GetIntParam(params, "concurrency", defaultConcurrency)
	failOnFirstError := ast.GetBoolParam(params, "failOnFirstError", false)
	uncapped := ast.GetBoolParam(params, "uncapped", false)

	// Apply intelligent CPU-based concurrency capping for production robustness
	// This prevents resource exhaustion on systems with limited CPU cores
	if !uncapped {
		cpuCount := runtime.NumCPU()
		maxRecommendedConcurrency := cpuCount * 2 // Allow some over-subscription for I/O bound tasks

		if concurrency > maxRecommendedConcurrency {
			// Cap concurrency but don't error - just limit to reasonable bounds
			// This provides good defaults while still allowing explicit override via uncapped=true
			concurrency = maxRecommendedConcurrency
		}
	}

	return concurrency, failOnFirstError, nil
}

// executeInterpreterImpl executes commands concurrently in interpreter mode using performance-optimized utilities
func (p *ParallelDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Set up logging with decorator context
	logger := decorators.GetLogger("parallel").WithDecorator("parallel").WithField("mode", "interpreter").WithField("concurrency", concurrency)
	logger.Infof("Starting parallel execution with %d commands, concurrency=%d, failOnFirstError=%v", len(content), concurrency, failOnFirstError)

	start := time.Now()
	defer func() {
		logger.LogDuration(decorators.LogLevelInfo, "Parallel execution completed", time.Since(start))
	}()

	// Use performance tracking for interpreter execution
	perfExecutor := decorators.NewPerformanceOptimizedExecutor("parallel", "interpreter", false)
	defer perfExecutor.Cleanup()

	var execError error
	err := perfExecutor.Execute(func() error {
		logger.Debug("Creating concurrent executor")
		// Create ConcurrentExecutor with proper concurrency limit
		concurrentExecutor := decorators.NewConcurrentExecutor(concurrency)
		defer concurrentExecutor.Cleanup()

		logger.Debug("Getting pooled command executor")
		// Get pooled command executor for better resource management
		pooledExecutor := decorators.GetPooledCommandExecutor()
		defer pooledExecutor.Cleanup()

		logger.Debugf("Converting %d commands to execution functions", len(content))
		// Convert AST commands to execution functions using the utility
		functions := make([]decorators.ExecutionFunction, len(content))
		for i, cmd := range content {
			cmd := cmd // Capture loop variable
			cmdIndex := i
			functions[i] = func() error {
				cmdLogger := logger.WithField("command_index", cmdIndex)
				cmdLogger.Trace("Starting command execution")

				// Create isolated context for each parallel command
				isolatedCtx := ctx.Child()

				// Use CommandExecutor utility to handle the switch logic
				commandExecutor := decorators.NewCommandExecutor()
				defer commandExecutor.Cleanup()

				err := commandExecutor.ExecuteCommandWithInterpreter(isolatedCtx, cmd)
				if err != nil {
					cmdLogger.ErrorWithErr("Command execution failed", err)
					// Record error for diagnostics
					decorators.RecordError("parallel", err.Error(), []string{}, fmt.Sprintf("command %d", cmdIndex))
				} else {
					cmdLogger.Trace("Command execution succeeded")
				}

				return err
			}
		}

		logger.Debug("Executing functions concurrently")
		// Execute all functions concurrently using the utility
		execError = concurrentExecutor.Execute(functions)
		if execError != nil {
			logger.ErrorWithErr("Concurrent execution failed", execError)
		} else {
			logger.Debug("Concurrent execution succeeded")
		}
		return execError
	})

	// Return the execution error if performance tracking succeeded
	if err == nil {
		err = execError
	}

	if err != nil {
		logger.ErrorWithErr("Parallel execution failed", err)
		return execution.NewErrorResult(err)
	} else {
		logger.Info("Parallel execution completed successfully")
		return execution.NewSuccessResult(nil)
	}
}

// executeGeneratorImpl generates Go code for parallel execution using performance-optimized utilities
func (p *ParallelDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Use performance optimization if enabled
	optimizer := decorators.GetASTOptimizer()
	optimizedSequence, err := optimizer.OptimizeCommandSequence(ctx, content)
	if err != nil {
		// Use generator utilities for consistent CommandResult handling
		executor := decorators.NewCommandResultExecutor(ctx)
		operations, err := executor.ConvertCommandsToCommandResultOperations(content)
		if err != nil {
			return execution.NewFormattedErrorResult("failed to convert commands to operations: %w", err)
		}

		// Generate concurrent execution with proper CommandResult handling
		generatedCode, err := executor.GenerateConcurrentExecution(operations, concurrency)
		if err != nil {
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("failed to build parallel template: %w", err),
			}
		}

		return &execution.ExecutionResult{
			Data:  generatedCode,
			Error: nil,
		}
	}

	// Use optimized operations
	operations := make([]decorators.Operation, len(optimizedSequence.Commands))
	for i, optimizedOp := range optimizedSequence.Commands {
		operations[i] = optimizedOp.Operation
	}

	// Use cached template builder with performance tracking
	perfExecutor := decorators.NewPerformanceOptimizedExecutor("parallel", "generator", true)

	var generatedCode string
	err = perfExecutor.Execute(func() error {
		// Use generator utilities for consistent CommandResult handling
		executor := decorators.NewCommandResultExecutor(ctx)
		code, buildErr := executor.GenerateConcurrentExecution(operations, concurrency)
		if buildErr != nil {
			return buildErr
		}
		generatedCode = code
		return nil
	})
	if err != nil {
		return execution.NewFormattedErrorResult("failed to build parallel template: %w", err)
	}

	return execution.NewSuccessResult(generatedCode)
}

// executePlanImpl creates a plan element for dry-run mode
func (p *ParallelDecorator) executePlanImpl(ctx execution.PlanContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
	description := fmt.Sprintf("Execute %d commands concurrently", len(content))
	if concurrency < len(content) {
		description += fmt.Sprintf(" (max %d at a time)", concurrency)
	}
	if failOnFirstError {
		description += ", stop on first error"
	} else {
		description += ", continue on errors"
	}

	element := plan.Decorator("parallel").
		WithType("block").
		WithDescription(description)

	if concurrency < len(content) {
		element = element.WithParameter("concurrency", fmt.Sprintf("%d", concurrency))
	}
	if failOnFirstError {
		element = element.WithParameter("failOnFirstError", "true")
	}

	// Build child plan elements for each command in the parallel block
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			// Create plan element for shell command
			result := ctx.GenerateShellPlan(c)
			if result.Error != nil {
				return execution.NewFormattedErrorResult("failed to create plan for shell content: %w", result.Error)
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
			// Execute nested decorator in plan mode to get its proper plan structure
			blockDecorator, err := decorators.GetBlock(c.Name)
			if err != nil {
				// Fallback to placeholder if decorator not found
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription(fmt.Sprintf("Unknown decorator: %s", c.Name))
				element = element.AddChild(childElement)
			} else {
				// Execute the nested decorator in plan mode
				result := blockDecorator.ExecutePlan(ctx, c.Args, c.Content)
				if result.Error != nil {
					// Fallback to placeholder if plan execution fails
					childElement := plan.Command(fmt.Sprintf("@%s{error}", c.Name)).WithDescription(fmt.Sprintf("Error in %s: %v", c.Name, result.Error))
					element = element.AddChild(childElement)
				} else if planElement, ok := result.Data.(plan.PlanElement); ok {
					// Add the nested decorator's plan element as a child
					element = element.AddChild(planElement)
				} else {
					// Fallback if result format is unexpected
					childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription(fmt.Sprintf("Nested decorator: %s", c.Name))
					element = element.AddChild(childElement)
				}
			}
		}
	}

	return execution.NewSuccessResult(element)
}

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.StandardImportRequirement(decorators.CoreImports, decorators.StringImports, decorators.ConcurrencyImports)
}

// init registers the parallel decorator
func init() {
	decorators.RegisterBlock(&ParallelDecorator{})
}
