package decorators

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
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
func (p *ParallelDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "concurrency",
			Type:        ast.NumberType,
			Required:    false,
			Description: "Maximum number of commands to run concurrently (default: unlimited)",
		},
		{
			Name:        "failOnFirstError",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Cancel remaining tasks on first error (default: false)",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (p *ParallelDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) > 2 {
		return fmt.Errorf("@parallel accepts at most 2 parameters (concurrency, failOnFirstError), got %d", len(params))
	}

	// Validate optional parameters
	if err := ValidateOptionalParameter(params, "concurrency", ast.NumberType, "parallel"); err != nil {
		return err
	}

	if err := ValidateOptionalParameter(params, "failOnFirstError", ast.BooleanType, "parallel"); err != nil {
		return err
	}

	return nil
}

// Run executes the decorator at runtime with concurrent command execution
func (p *ParallelDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) error {
	if err := p.Validate(ctx, params); err != nil {
		return err
	}

	// Parse parameters with defaults
	concurrency := len(content) // Default: no limit (run all at once)
	failOnFirstError := false   // Default: continue on errors

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

	// Create context for cancellation if failOnFirstError is true
	execCtx := ctx
	var cancel context.CancelFunc
	if failOnFirstError {
		execCtx, cancel = ctx.WithCancel()
		defer cancel()
	}

	// Use semaphore to limit concurrency
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(content))

	// Execute each command with concurrency control
	for i, cmd := range content {
		// Check if context is cancelled
		select {
		case <-execCtx.Done():
			return execCtx.Err()
		default:
		}

		wg.Add(1)
		go func(commandIndex int, command ast.CommandContent) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check cancellation again before executing
			select {
			case <-execCtx.Done():
				errChan <- execCtx.Err()
				return
			default:
			}

			// TODO: Execute the command using the execution engine
			// For now, just print what would be executed
			fmt.Printf("Executing in parallel (concurrency=%d): Command %d: %+v\n", concurrency, commandIndex, command)

			// Simulate potential error for testing
			// if some condition { errChan <- fmt.Errorf("command %d failed", commandIndex) }
			errChan <- nil
		}(i, cmd)
	}

	// Wait for all commands to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors and handle fail-fast behavior
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
			if failOnFirstError && cancel != nil {
				cancel() // Cancel remaining tasks
				break
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// Generate produces Go code for the decorator in compiled mode
func (p *ParallelDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error) {
	if err := p.Validate(ctx, params); err != nil {
		return "", err
	}

	// Parse parameters with defaults for code generation
	concurrency := len(content)
	failOnFirstError := false

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

	var builder strings.Builder
	builder.WriteString("func() error {\n")

	// Generate context setup if failOnFirstError is enabled
	if failOnFirstError {
		builder.WriteString("\tctx, cancel := context.WithCancel(context.Background())\n")
		builder.WriteString("\tdefer cancel()\n")
	}

	builder.WriteString(fmt.Sprintf("\tsemaphore := make(chan struct{}, %d)\n", concurrency))
	builder.WriteString("\tvar wg sync.WaitGroup\n")
	builder.WriteString(fmt.Sprintf("\terrChan := make(chan error, %d)\n", len(content)))
	builder.WriteString("\n")

	// Generate concurrent execution for each command
	for i, cmd := range content {
		builder.WriteString("\twg.Add(1)\n")
		builder.WriteString("\tgo func() {\n")
		builder.WriteString("\t\tdefer wg.Done()\n")
		builder.WriteString("\n")
		builder.WriteString("\t\t// Acquire semaphore\n")
		builder.WriteString("\t\tsemaphore <- struct{}{}\n")
		builder.WriteString("\t\tdefer func() { <-semaphore }()\n")
		builder.WriteString("\n")

		if failOnFirstError {
			builder.WriteString("\t\t// Check cancellation\n")
			builder.WriteString("\t\tselect {\n")
			builder.WriteString("\t\tcase <-ctx.Done():\n")
			builder.WriteString("\t\t\terrChan <- ctx.Err()\n")
			builder.WriteString("\t\t\treturn\n")
			builder.WriteString("\t\tdefault:\n")
			builder.WriteString("\t\t}\n")
			builder.WriteString("\n")
		}

		builder.WriteString(fmt.Sprintf("\t\t// Execute command %d: %+v\n", i, cmd))
		builder.WriteString("\t\t// TODO: Generate actual command execution code\n")
		builder.WriteString("\t\terrChan <- nil\n")
		builder.WriteString("\t}()\n")
		builder.WriteString("\n")
	}

	builder.WriteString("\tgo func() {\n")
	builder.WriteString("\t\twg.Wait()\n")
	builder.WriteString("\t\tclose(errChan)\n")
	builder.WriteString("\t}()\n")
	builder.WriteString("\n")

	builder.WriteString("\tvar errors []string\n")
	builder.WriteString("\tfor err := range errChan {\n")
	builder.WriteString("\t\tif err != nil {\n")
	builder.WriteString("\t\t\terrors = append(errors, err.Error())\n")

	if failOnFirstError {
		builder.WriteString("\t\t\tcancel() // Cancel remaining tasks\n")
		builder.WriteString("\t\t\tbreak\n")
	}

	builder.WriteString("\t\t}\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\n")

	builder.WriteString("\tif len(errors) > 0 {\n")
	builder.WriteString("\t\treturn fmt.Errorf(\"parallel execution failed: %s\", strings.Join(errors, \"; \"))\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\treturn nil\n")
	builder.WriteString("}()")

	return builder.String(), nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (p *ParallelDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (plan.PlanElement, error) {
	if err := p.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Parse parameters with defaults
	concurrency := len(content) // Default: no limit (run all at once)
	failOnFirstError := false   // Default: continue on errors

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

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

	return element, nil
}

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"context", "sync", "fmt", "strings"}, // Parallel needs sync, context, etc.
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the parallel decorator
func init() {
	RegisterBlock(&ParallelDecorator{})
}
