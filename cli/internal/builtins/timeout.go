package decorators

import (
	"fmt"
	"strings"
	"text/template"
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

	// Parse parameters (validation passed, so these should be safe)
	// If no duration parameter provided, use default of 30 seconds
	return ast.GetDurationParam(params, "duration", 30*time.Second), nil
}

// executeInterpreterImpl executes commands with timeout in interpreter mode
func (t *TimeoutDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create context with timeout
	timeoutCtx, cancel := ctx.WithTimeout(timeout)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan error, 1)

	// Execute commands in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic during execution: %v", r)
			}
		}()

		// Execute commands using the unified execution engine
		for _, cmd := range content {
			// Check for cancellation before each command
			select {
			case <-timeoutCtx.Done():
				done <- timeoutCtx.Err()
				return
			default:
			}

			// Execute the command content directly
			switch c := cmd.(type) {
			case *ast.ShellContent:
				result := timeoutCtx.ExecuteShell(c)
				if result.Error != nil {
					done <- result.Error
					return
				}
			default:
				done <- fmt.Errorf("unsupported command content type in timeout: %T", cmd)
				return
			}
		}
		done <- nil
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("command execution failed: %w", err),
			}
		}
		return &execution.ExecutionResult{
			Data:  nil,
			Error: nil,
		}
	case <-timeoutCtx.Done():
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("command execution timed out after %s", timeout),
		}
	}
}

// Template for timeout execution code generation (unified contract: statement blocks)
const timeoutExecutionTemplate = `// Timeout execution setup
timeoutDuration, err := time.ParseDuration({{printf "%q" .Timeout}})
if err != nil {
	return fmt.Errorf("invalid timeout duration '{{.Timeout}}': %w", err)
}

timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), timeoutDuration)
defer timeoutCancel()

timeoutDone := make(chan error, 1)

go func() {
	defer func() {
		if r := recover(); r != nil {
			timeoutDone <- fmt.Errorf("panic during execution: %v", r)
		}
	}()

	// Execute commands in timeout context
	{{range $i, $cmd := .Commands}}
	// Check for timeout cancellation before command {{add $i 1}}
	select {
	case <-timeoutCtx.Done():
		timeoutDone <- timeoutCtx.Err()
		return
	default:
	}

	// Execute timeout command {{add $i 1}}
	if err := func() error {
		{{generateShellCode $cmd}}
		return nil
	}(); err != nil {
		timeoutDone <- err
		return
	}
	{{end}}

	timeoutDone <- nil
}()

select {
case err := <-timeoutDone:
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}
case <-timeoutCtx.Done():
	return fmt.Errorf("command execution timed out after {{.Timeout}}")
}`

// TimeoutTemplateData holds data for template execution
type TimeoutTemplateData struct {
	Timeout  string
	Commands []ast.CommandContent
}

// executeGeneratorImpl generates Go code for timeout logic using templates
func (t *TimeoutDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create child context for isolated execution
	timeoutCtx := ctx.Child()
	
	// Prepare template data
	templateData := TimeoutTemplateData{
		Timeout:  timeout.String(),
		Commands: content,
	}

	// Parse and execute template with child context functions and add helper function
	tmpl, err := template.New("timeout").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Funcs(timeoutCtx.GetTemplateFunctions()).Parse(timeoutExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse timeout template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute timeout template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
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
		StandardLibrary: []string{"time", "context", "fmt"}, // Timeout needs time and context packages
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the timeout decorator
func init() {
	decorators.RegisterBlock(&TimeoutDecorator{})
}
