package decorators

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// RetryDecorator implements the @retry decorator for retrying failed command execution
type RetryDecorator struct{}

// Template for retry execution code generation
const retryExecutionTemplate = `return func() error {
	maxAttempts := {{.MaxAttempts}}
	delay, err := time.ParseDuration({{printf "%q" .Delay}})
	if err != nil {
		return fmt.Errorf("invalid retry delay '{{.Delay}}': %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("Retry attempt %d/%d\n", attempt, maxAttempts)

		// Execute commands
		execErr := func() error {
			{{range $i, $cmd := .Commands}}
			// Execute command {{$i}}
			if err := func() error {
				{{executeCommand $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}()

		if execErr == nil {
			fmt.Printf("Commands succeeded on attempt %d\n", attempt)
			return nil
		}

		lastErr = execErr
		fmt.Printf("Attempt %d failed: %v\n", attempt, execErr)

		// Don't delay after the last attempt
		if attempt < maxAttempts {
			fmt.Printf("Waiting %s before next attempt...\n", delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr)
}()`

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
func (r *RetryDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
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
func (r *RetryDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) == 0 {
		return fmt.Errorf("@retry requires at least 1 parameter (attempts), got 0")
	}
	if len(params) > 2 {
		return fmt.Errorf("@retry accepts at most 2 parameters (attempts, delay), got %d", len(params))
	}

	// Validate the required attempts parameter
	if err := ValidateRequiredParameter(params, "attempts", ast.NumberType, "retry"); err != nil {
		return err
	}

	// Validate the optional delay parameter
	if err := ValidateOptionalParameter(params, "delay", ast.DurationType, "retry"); err != nil {
		return err
	}

	return nil
}

// Run executes the decorator at runtime with retry logic
func (r *RetryDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) error {
	if err := r.Validate(ctx, params); err != nil {
		return err
	}

	// Parse parameters
	maxAttempts := ast.GetIntParam(params, "attempts", 3)
	delay := ast.GetDurationParam(params, "delay", 1*time.Second)

	// Validate attempts is positive
	if maxAttempts <= 0 {
		return fmt.Errorf("retry attempts must be positive, got %d", maxAttempts)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("Retry attempt %d/%d\n", attempt, maxAttempts)

		// TODO: Execute commands using execution engine
		// For now, simulate execution
		execErr := r.executeCommands(ctx, content, attempt)

		if execErr == nil {
			fmt.Printf("Commands succeeded on attempt %d\n", attempt)
			return nil // Success!
		}

		lastErr = execErr
		fmt.Printf("Attempt %d failed: %v\n", attempt, execErr)

		// Don't delay after the last attempt
		if attempt < maxAttempts {
			fmt.Printf("Waiting %s before next attempt...\n", delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("all %d retry attempts failed, last error: %w", maxAttempts, lastErr)
}

// executeCommands simulates command execution (TODO: replace with actual execution engine)
func (r *RetryDecorator) executeCommands(ctx *ExecutionContext, content []ast.CommandContent, attempt int) error {
	for i, cmd := range content {
		fmt.Printf("  Executing command %d: %+v\n", i, cmd)

		// Simulate some commands failing on first attempts
		if attempt == 1 && i == 0 {
			return fmt.Errorf("simulated failure for command %d", i)
		}
	}
	return nil
}

// Generate produces Go code for the decorator in compiled mode using templates
func (r *RetryDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error) {
	if err := r.Validate(ctx, params); err != nil {
		return "", err
	}

	// Parse parameters for code generation
	maxAttempts := ast.GetIntParam(params, "attempts", 3)
	defaultDelay := "1s"
	if delayParam := ast.FindParameter(params, "delay"); delayParam != nil {
		if durLit, ok := delayParam.Value.(*ast.DurationLiteral); ok {
			defaultDelay = durLit.Value
		}
	}

	// Prepare template data
	templateData := RetryTemplateData{
		MaxAttempts: maxAttempts,
		Delay:       defaultDelay,
		Commands:    content,
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("retry").Funcs(ctx.GetTemplateFunctions()).Parse(retryExecutionTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse retry template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute retry template: %w", err)
	}

	return result.String(), nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (r *RetryDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (plan.PlanElement, error) {
	if err := r.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Parse parameters with defaults
	maxAttempts := 3 // Default: 3 attempts
	delayStr := "1s" // Default: 1 second delay

	maxAttempts = ast.GetIntParam(params, "attempts", maxAttempts)
	if delayParam := ast.FindParameter(params, "delay"); delayParam != nil {
		if durLit, ok := delayParam.Value.(*ast.DurationLiteral); ok {
			delayStr = durLit.Value
		}
	}

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

	return element, nil
}

// ImportRequirements returns the dependencies needed for code generation
func (r *RetryDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"time", "fmt"}, // Retry needs time for delays and fmt for errors
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the retry decorator
func init() {
	RegisterBlock(&RetryDecorator{})
}
