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

// RetryDecorator implements the @retry decorator for retrying failed command execution
type RetryDecorator struct{}

// Template for retry execution code generation (unified contract: statement blocks)
const retryExecutionTemplate = `// Retry execution setup
maxAttempts := {{.MaxAttempts}}
delay, err := time.ParseDuration({{printf "%q" .Delay}})
if err != nil {
	return fmt.Errorf("invalid retry delay '{{.Delay}}': %w", err)
}

var lastErr error
for attempt := 1; attempt <= maxAttempts; attempt++ {
	fmt.Printf("Retry attempt %d/%d\n", attempt, maxAttempts)

	// Execute commands in child context
	execErr := func() error {
		{{range $i, $cmd := .Commands}}
		{{generateShellCode $cmd}}
		{{end}}
		return nil
	}()

	if execErr == nil {
		fmt.Printf("Commands succeeded on attempt %d\n", attempt)
		break
	}

	lastErr = execErr
	fmt.Printf("Attempt %d failed: %v\n", attempt, execErr)

	// Don't delay after the last attempt
	if attempt < maxAttempts {
		fmt.Printf("Waiting %s before next attempt...\n", delay)
		time.Sleep(delay)
	}
}

if lastErr != nil {
	return fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr)
}`

// RetryTemplateData holds data for template execution
type RetryTemplateData struct {
	MaxAttempts int
	Delay       string
	Commands    []ast.CommandContent
}

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

	// Validate delay parameter if present (1ms to 1 hour range)
	if err := decorators.ValidateDuration(params, "delay", 1*time.Millisecond, 1*time.Hour, "retry"); err != nil {
		return 0, 0, err
	}

	// Parse parameters (validation passed, so these should be safe)
	maxAttempts := ast.GetIntParam(params, "attempts", 3)
	delay := ast.GetDurationParam(params, "delay", 1*time.Second)

	return maxAttempts, delay, nil
}

// executeInterpreterImpl executes retry logic in interpreter mode
func (r *RetryDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create child context for each retry attempt
		retryCtx := ctx.Child()
		
		// Execute commands using the unified execution engine
		execErr := r.executeCommands(retryCtx, content)

		if execErr == nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: nil, // Success!
			}
		}

		lastErr = execErr

		// Don't delay after the last attempt
		if attempt < maxAttempts {
			time.Sleep(delay)
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr),
	}
}

// executeGeneratorImpl generates Go code for retry logic
func (r *RetryDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, maxAttempts int, delay time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	// Create child context for isolated execution
	retryCtx := ctx.Child()
	
	// Parse delay for code generation
	defaultDelay := delay.String()

	// Prepare template data
	templateData := RetryTemplateData{
		MaxAttempts: maxAttempts,
		Delay:       defaultDelay,
		Commands:    content,
	}

	// Parse and execute template with child context functions
	tmpl, err := template.New("retry").Funcs(retryCtx.GetTemplateFunctions()).Parse(retryExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse retry template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute retry template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
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

// executeCommands executes commands using direct execution
func (r *RetryDecorator) executeCommands(ctx execution.InterpreterContext, content []ast.CommandContent) error {
	for _, cmd := range content {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return result.Error
			}
		default:
			return fmt.Errorf("unsupported command content type in retry: %T", cmd)
		}
	}
	return nil
}

// ImportRequirements returns the dependencies needed for code generation
func (r *RetryDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"time", "fmt"}, // Retry needs time for delays and fmt for errors
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the retry decorator
func init() {
	decorators.RegisterBlock(&RetryDecorator{})
}
