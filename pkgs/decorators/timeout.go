package decorators

import (
	"fmt"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
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
func (t *TimeoutDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "duration",
			Type:        ast.DurationType,
			Required:    true,
			Description: "Maximum execution time (e.g., '30s', '5m', '1h')",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (t *TimeoutDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) != 1 {
		return fmt.Errorf("@timeout requires exactly 1 parameter (duration), got %d", len(params))
	}

	// Validate the required duration parameter
	if err := ValidateRequiredParameter(params, "duration", ast.DurationType, "timeout"); err != nil {
		return err
	}

	return nil
}

// Run executes the decorator at runtime with timeout
func (t *TimeoutDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) error {
	if err := t.Validate(ctx, params); err != nil {
		return err
	}

	// Get the timeout duration
	var timeout time.Duration
	durationParam := ast.FindParameter(params, "duration")
	if durationParam == nil && len(params) > 0 {
		durationParam = &params[0]
	}
	if durLit, ok := durationParam.Value.(*ast.DurationLiteral); ok {
		var err error
		timeout, err = time.ParseDuration(durLit.Value)
		if err != nil {
			return fmt.Errorf("invalid duration '%s': %w", durLit.Value, err)
		}
	}

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

		// TODO: Execute commands using execution engine
		// For now, simulate execution
		fmt.Printf("Executing with timeout %s: %d commands\n", timeout, len(content))
		for i, cmd := range content {
			// Check for cancellation before each command
			select {
			case <-timeoutCtx.Done():
				done <- timeoutCtx.Err()
				return
			default:
			}

			fmt.Printf("  Command %d: %+v\n", i, cmd)
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
		}
		done <- nil
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("command execution failed: %w", err)
		}
		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("command execution timed out after %s", timeout)
	}
}

// Generate produces Go code for the decorator in compiled mode
func (t *TimeoutDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error) {
	if err := t.Validate(ctx, params); err != nil {
		return "", err
	}

	// Get the timeout duration
	var durationStr string
	durationParam := ast.FindParameter(params, "duration")
	if durationParam == nil && len(params) > 0 {
		durationParam = &params[0]
	}
	if durLit, ok := durationParam.Value.(*ast.DurationLiteral); ok {
		durationStr = durLit.Value
	}

	var builder strings.Builder
	builder.WriteString("func() error {\n")
	builder.WriteString(fmt.Sprintf("\ttimeout, err := time.ParseDuration(%q)\n", durationStr))
	builder.WriteString("\tif err != nil {\n")
	builder.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"invalid timeout duration '%s': %%w\", err)\n", durationStr))
	builder.WriteString("\t}\n")
	builder.WriteString("\n")
	builder.WriteString("\tctx, cancel := context.WithTimeout(context.Background(), timeout)\n")
	builder.WriteString("\tdefer cancel()\n")
	builder.WriteString("\n")
	builder.WriteString("\tdone := make(chan error, 1)\n")
	builder.WriteString("\n")
	builder.WriteString("\tgo func() {\n")
	builder.WriteString("\t\tdefer func() {\n")
	builder.WriteString("\t\t\tif r := recover(); r != nil {\n")
	builder.WriteString("\t\t\t\tdone <- fmt.Errorf(\"panic during execution: %v\", r)\n")
	builder.WriteString("\t\t\t}\n")
	builder.WriteString("\t\t}()\n")
	builder.WriteString("\n")

	// Generate execution for each command
	for i, cmd := range content {
		builder.WriteString("\t\t// Check for cancellation\n")
		builder.WriteString("\t\tselect {\n")
		builder.WriteString("\t\tcase <-ctx.Done():\n")
		builder.WriteString("\t\t\tdone <- ctx.Err()\n")
		builder.WriteString("\t\t\treturn\n")
		builder.WriteString("\t\tdefault:\n")
		builder.WriteString("\t\t}\n")
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("\t\t// Execute command %d: %+v\n", i, cmd))
		builder.WriteString("\t\t// TODO: Generate actual command execution code\n")
		builder.WriteString("\n")
	}

	builder.WriteString("\t\tdone <- nil\n")
	builder.WriteString("\t}()\n")
	builder.WriteString("\n")
	builder.WriteString("\tselect {\n")
	builder.WriteString("\tcase err := <-done:\n")
	builder.WriteString("\t\tif err != nil {\n")
	builder.WriteString("\t\t\treturn fmt.Errorf(\"command execution failed: %w\", err)\n")
	builder.WriteString("\t\t}\n")
	builder.WriteString("\t\treturn nil\n")
	builder.WriteString("\tcase <-ctx.Done():\n")
	builder.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"command execution timed out after %s\")\n", durationStr))
	builder.WriteString("\t}\n")
	builder.WriteString("}()")

	return builder.String(), nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (t *TimeoutDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (plan.PlanElement, error) {
	if err := t.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Get the timeout duration
	var durationStr string
	var timeout time.Duration
	durationParam := ast.FindParameter(params, "duration")
	if durationParam == nil && len(params) > 0 {
		durationParam = &params[0]
	}
	if durLit, ok := durationParam.Value.(*ast.DurationLiteral); ok {
		durationStr = durLit.Value
		if d, err := time.ParseDuration(durationStr); err == nil {
			timeout = d
		}
	}

	description := fmt.Sprintf("Execute %d commands with %s timeout (cancel if exceeded)", len(content), durationStr)

	return plan.Decorator("timeout").
		WithType("block").
		WithTimeout(timeout).
		WithParameter("duration", durationStr).
		WithDescription(description), nil
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TimeoutDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"time", "context", "fmt"}, // Timeout needs time and context packages
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the timeout decorator
func init() {
	RegisterBlock(&TimeoutDecorator{})
}
