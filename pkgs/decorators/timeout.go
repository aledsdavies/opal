package decorators

import (
	"fmt"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/execution"
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

// Execute provides unified execution for all modes using the execution package
func (t *TimeoutDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Validate parameters first

	// Parse timeout duration
	var timeout time.Duration
	durationParam := ast.FindParameter(params, "duration")
	if durationParam == nil && len(params) > 0 {
		durationParam = &params[0]
	}
	if durationParam != nil {
		if durLit, ok := durationParam.Value.(*ast.DurationLiteral); ok {
			var err error
			timeout, err = time.ParseDuration(durLit.Value)
			if err != nil {
				return &execution.ExecutionResult{
					Mode:  ctx.Mode(),
					Data:  nil,
					Error: fmt.Errorf("invalid duration '%s': %w", durLit.Value, err),
				}
			}
		} else {
			return &execution.ExecutionResult{
				Mode:  ctx.Mode(),
				Data:  nil,
				Error: fmt.Errorf("duration parameter must be a duration literal"),
			}
		}
	} else {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("timeout decorator requires a duration parameter"),
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		return t.executeInterpreter(ctx, timeout, content)
	case execution.GeneratorMode:
		return t.executeGenerator(ctx, timeout, content)
	case execution.PlanMode:
		return t.executePlan(ctx, timeout, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executeInterpreter executes commands with timeout in interpreter mode
func (t *TimeoutDecorator) executeInterpreter(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
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

			// Execute the command using the engine's content executor
			if err := timeoutCtx.ExecuteCommandContent(cmd); err != nil {
				done <- err
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
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: fmt.Errorf("command execution failed: %w", err),
			}
		}
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: nil,
		}
	case <-timeoutCtx.Done():
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("command execution timed out after %s", timeout),
		}
	}
}

// executeGenerator generates Go code for timeout logic
func (t *TimeoutDecorator) executeGenerator(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	durationStr := timeout.String()

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
		builder.WriteString(fmt.Sprintf("\t\t// Execute command %d\n", i))

		// For shell content, use the execution helpers
		if shellContent, ok := cmd.(*ast.ShellContent); ok {
			result := ctx.WithMode(execution.GeneratorMode).ExecuteShell(shellContent)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Mode:  execution.GeneratorMode,
					Data:  "",
					Error: fmt.Errorf("failed to generate shell command %d: %w", i, result.Error),
				}
			}
			if code, ok := result.Data.(string); ok {
				builder.WriteString(fmt.Sprintf("\t\tif err := func() error {\n%s\n\t\t\treturn nil\n\t\t}(); err != nil {\n", code))
				builder.WriteString("\t\t\tdone <- err\n")
				builder.WriteString("\t\t\treturn\n")
				builder.WriteString("\t\t}\n")
			}
		} else {
			builder.WriteString("\t\t// TODO: Generate execution for non-shell command content\n")
		}
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

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  builder.String(),
		Error: nil,
	}
}

// executePlan creates a plan element for dry-run mode
func (t *TimeoutDecorator) executePlan(ctx *execution.ExecutionContext, timeout time.Duration, content []ast.CommandContent) *execution.ExecutionResult {
	durationStr := timeout.String()
	description := fmt.Sprintf("Execute %d commands with %s timeout (cancel if exceeded)", len(content), durationStr)

	element := plan.Decorator("timeout").
		WithType("block").
		WithTimeout(timeout).
		WithParameter("duration", durationStr).
		WithDescription(description)

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
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
