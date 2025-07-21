package decorators

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// ParallelDecorator implements the @parallel decorator for concurrent command execution
type ParallelDecorator struct{}

// Template for parallel execution code generation
const parallelExecutionTemplate = `func() error {
	{{if .FailOnFirstError}}ctx, cancel := context.WithCancel(context.Background())
	defer cancel(){{end}}

	semaphore := make(chan struct{}, {{.Concurrency}})
	var wg sync.WaitGroup
	errChan := make(chan error, {{.CommandCount}})

	{{range $i, $cmd := .Commands}}
	// Command {{$i}}
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Acquire semaphore
		semaphore <- struct{}{}
		defer func() { <-semaphore }()

		{{if $.FailOnFirstError}}// Check cancellation
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			return
		default:
		}{{end}}

		// Execute command using template function
		if err := func() error {
			{{executeCommand $cmd}}
		}(); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()
	{{end}}

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
			{{if .FailOnFirstError}}cancel() // Cancel remaining tasks
			break{{end}}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed: %s", strings.Join(errors, "; "))
	}
	return nil
}()`

// TemplateData holds data for template execution
type ParallelTemplateData struct {
	Concurrency      int
	FailOnFirstError bool
	CommandCount     int
	Commands         []ast.CommandContent
}

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

// executeCommandContent executes different types of command content in interpreter mode
func (p *ParallelDecorator) executeCommandContent(ctx *ExecutionContext, content ast.CommandContent) error {
	// Use the engine's content executor for full decorator support
	return ctx.ExecuteCommandContent(content)
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

			// Execute the actual command content
			err := p.executeCommandContent(execCtx, command)
			errChan <- err
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

// Generate produces Go code for the decorator in compiled mode using templates
func (p *ParallelDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error) {
	if err := p.Validate(ctx, params); err != nil {
		return "", err
	}

	// Parse parameters with defaults for code generation
	concurrency := len(content)
	failOnFirstError := false

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

	// Prepare template data
	templateData := ParallelTemplateData{
		Concurrency:      concurrency,
		FailOnFirstError: failOnFirstError,
		CommandCount:     len(content),
		Commands:         content, // Pass raw AST content
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("parallel").Funcs(ctx.GetTemplateFunctions()).Parse(parallelExecutionTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse parallel template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute parallel template: %w", err)
	}

	return result.String(), nil
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
