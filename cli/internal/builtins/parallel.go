package decorators

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// ParallelDecorator implements the @parallel decorator for concurrent command execution
type ParallelDecorator struct{}

// Template for parallel execution code generation (unified contract: statement blocks)
const parallelExecutionTemplate = `// Parallel execution setup
{{if .FailOnFirstError}}parallelCtx, cancelParallel := context.WithCancel(ctx)
defer cancelParallel(){{end}}

parallelSemaphore := make(chan struct{}, {{.Concurrency}})
var parallelWg sync.WaitGroup
parallelErrChan := make(chan error, {{.CommandCount}})

{{range $i, $cmd := .Commands}}
// Parallel command {{$i}}
parallelWg.Add(1)
go func() {
	defer parallelWg.Done()
	
	// Acquire semaphore
	parallelSemaphore <- struct{}{}
	defer func() { <-parallelSemaphore }()

	{{if $.FailOnFirstError}}// Check cancellation
	select {
	case <-parallelCtx.Done():
		parallelErrChan <- parallelCtx.Err()
		return
	default:
	}{{end}}

	// Execute command using template function
	if err := func() error {
		{{generateShellCode $cmd}}
		return nil
	}(); err != nil {
		parallelErrChan <- err
		return
	}
	parallelErrChan <- nil
}()
{{end}}

// Wait for parallel completion
go func() {
	parallelWg.Wait()
	close(parallelErrChan)
}()

// Collect parallel errors
var parallelErrors []string
for err := range parallelErrChan {
	if err != nil {
		parallelErrors = append(parallelErrors, err.Error())
		{{if .FailOnFirstError}}cancelParallel() // Cancel remaining tasks
		break{{end}}
	}
}

if len(parallelErrors) > 0 {
	return fmt.Errorf("parallel execution failed: %s", strings.Join(parallelErrors, "; "))
}`

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
func (p *ParallelDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
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

// Execute provides unified execution for all modes using the execution package
func (p *ParallelDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Parse parameters with defaults
	concurrency := len(content) // Default: no limit (run all at once)
	failOnFirstError := false   // Default: continue on errors

	concurrency = ast.GetIntParam(params, "concurrency", concurrency)
	failOnFirstError = ast.GetBoolParam(params, "failOnFirstError", failOnFirstError)

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return p.executeInterpreter(ctx, concurrency, failOnFirstError, content)
	case execution.GeneratorMode:
		return p.executeGenerator(ctx, concurrency, failOnFirstError, content)
	case execution.PlanMode:
		return p.executePlan(ctx, concurrency, failOnFirstError, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes commands concurrently in interpreter mode
func (p *ParallelDecorator) executeInterpreter(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
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
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: execCtx.Err(),
			}
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

			// Create isolated execution context for this parallel task
			// This gives each task its own working directory state and environment
			isolatedCtx := p.createIsolatedContext(execCtx)
			
			// Execute the command content in isolated environment
			// Handle different content types appropriately
			var err error
			switch cmd := command.(type) {
			case *ast.BlockDecorator:
				// For block decorators, look up and execute the decorator directly
				blockDecorator, lookupErr := decorators.GetBlock(cmd.Name)
				if lookupErr != nil {
					err = fmt.Errorf("block decorator @%s not found: %w", cmd.Name, lookupErr)
				} else {
					// Execute the block decorator with the isolated context
					result := blockDecorator.Execute(isolatedCtx, cmd.Args, cmd.Content)
					err = result.Error
				}
			default:
				// For other content types (like ShellContent), use the standard executor
				err = isolatedCtx.ExecuteCommandContent(command)
			}
			
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

	var finalError error
	if len(errors) > 0 {
		finalError = fmt.Errorf("parallel execution failed: %s", strings.Join(errors, "; "))
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  nil,
		Error: finalError,
	}
}

// executeGenerator generates Go code for parallel execution
func (p *ParallelDecorator) executeGenerator(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
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
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse parallel template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute parallel template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (p *ParallelDecorator) executePlan(ctx *execution.ExecutionContext, concurrency int, failOnFirstError bool, content []ast.CommandContent) *execution.ExecutionResult {
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
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Mode:  execution.PlanMode,
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
			// Execute nested decorator in plan mode to get its proper plan structure
			blockDecorator, err := decorators.GetBlock(c.Name)
			if err != nil {
				// Fallback to placeholder if decorator not found
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription(fmt.Sprintf("Unknown decorator: %s", c.Name))
				element = element.AddChild(childElement)
			} else {
				// Execute the nested decorator in plan mode
				result := blockDecorator.Execute(ctx, c.Args, c.Content)
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

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// createIsolatedContext creates a copy of the execution context for isolated parallel execution
// Each parallel task gets its own context with independent working directory state
func (p *ParallelDecorator) createIsolatedContext(parentCtx *execution.ExecutionContext) *execution.ExecutionContext {
	// Create a new execution context with the same program and parent context
	isolatedCtx := execution.NewExecutionContext(parentCtx.Context, parentCtx.Program)
	
	// Copy the execution mode
	isolatedCtx = isolatedCtx.WithMode(parentCtx.Mode())
	
	// Copy variables (each task gets the same variable values but independent state)
	for name, value := range parentCtx.Variables {
		isolatedCtx.SetVariable(name, value)
	}
	
	
	// Copy execution state
	isolatedCtx.WorkingDir = parentCtx.WorkingDir
	isolatedCtx.Debug = parentCtx.Debug
	isolatedCtx.DryRun = parentCtx.DryRun
	
	// Copy the essential function references from parent context
	// These allow the isolated context to execute commands and look up decorators
	isolatedCtx.SetTemplateFunctions(parentCtx.GetTemplateFunctions())
	
	// Copy the function executors - we need to access the parent's function fields directly
	// since the SetContentExecutor method expects a function, not the bound method
	// This creates a closure that captures the parent's state
	isolatedCtx.SetContentExecutor(func(content ast.CommandContent) error {
		return parentCtx.ExecuteCommandContent(content)
	})
	
	// Set up function decorator lookup by copying from parent
	// Note: We can't easily access the parent's functionDecoratorLookup directly,
	// but SetFunctionDecoratorLookup exists, so this should work through reflection or shared state
	
	// Copy the function executors that enable decorator execution
	// These methods are always available on ExecutionContext, so we can set them directly
	isolatedCtx.SetCommandExecutor(func(cmd *ast.CommandDecl) error {
		return parentCtx.ExecuteCommand(cmd.Name)
	})
	
	isolatedCtx.SetCommandPlanGenerator(func(cmd *ast.CommandDecl) (*execution.ExecutionResult, error) {
		return parentCtx.GenerateCommandPlan(cmd.Name)
	})
	
	return isolatedCtx
}

// ImportRequirements returns the dependencies needed for code generation
func (p *ParallelDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"context", "sync", "fmt", "strings"}, // Parallel needs sync, context, etc.
		ThirdParty:      []string{},                                    // Plan import removed - only needed for dry-run which isn't implemented in generated binaries yet
		GoModules:       map[string]string{},
	}
}

// init registers the parallel decorator
func init() {
	decorators.RegisterBlock(&ParallelDecorator{})
}
