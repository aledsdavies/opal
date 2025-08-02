package testing

import (
	"fmt"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// Simple test decorators for examples

// TestVarDecorator is a simple variable decorator for testing (ValueDecorator)
type TestVarDecorator struct{}

func (v *TestVarDecorator) Name() string        { return "var" }
func (v *TestVarDecorator) Description() string { return "Test variable decorator" }
func (v *TestVarDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "name", Type: ast.IdentifierType, Required: true, Description: "Variable name"},
	}
}

func (v *TestVarDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

func (v *TestVarDecorator) Expand(ctx *execution.ExecutionContext, params []ast.NamedParameter) *execution.ExecutionResult {
	// Get variable name
	var varName string
	if len(params) > 0 {
		if ident, ok := params[0].Value.(*ast.Identifier); ok {
			varName = ident.Name
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		if value, exists := ctx.GetVariable(varName); exists {
			return &execution.ExecutionResult{Mode: ctx.Mode(), Data: value, Error: nil}
		}
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("variable '%s' not defined", varName)}
	case execution.GeneratorMode:
		if _, exists := ctx.GetVariable(varName); exists {
			return &execution.ExecutionResult{Mode: ctx.Mode(), Data: varName, Error: nil}
		}
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("variable '%s' not defined", varName)}
	case execution.PlanMode:
		var description string
		if value, exists := ctx.GetVariable(varName); exists {
			description = fmt.Sprintf("Variable resolution: ${%s} → %q", varName, value)
		} else {
			description = fmt.Sprintf("Variable resolution: ${%s} → <undefined>", varName)
		}
		element := plan.Decorator("var").WithType("function").WithParameter("name", varName).WithDescription(description)
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: element, Error: nil}
	default:
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("unsupported mode")}
	}
}

// TestTimeoutDecorator is a simple timeout decorator for testing
type TestTimeoutDecorator struct{}

func (t *TestTimeoutDecorator) Name() string        { return "timeout" }
func (t *TestTimeoutDecorator) Description() string { return "Test timeout decorator" }
func (t *TestTimeoutDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "duration", Type: ast.DurationType, Required: true, Description: "Timeout duration"},
	}
}

func (t *TestTimeoutDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{StandardLibrary: []string{"time", "context"}}
}

func (t *TestTimeoutDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Parse timeout duration
	var timeout time.Duration = 30 * time.Second
	if len(params) > 0 {
		if durLit, ok := params[0].Value.(*ast.DurationLiteral); ok {
			if parsed, err := time.ParseDuration(durLit.Value); err == nil {
				timeout = parsed
			}
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		// Execute each command in the content
		for _, cmd := range content {
			if err := ctx.ExecuteCommandContent(cmd); err != nil {
				return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: err}
			}
		}
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: nil}
	case execution.GeneratorMode:
		code := fmt.Sprintf("// Timeout decorator with %s duration\n// Commands: %d", timeout, len(content))
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: code, Error: nil}
	case execution.PlanMode:
		description := fmt.Sprintf("Execute %d commands with %s timeout", len(content), timeout)
		element := plan.Decorator("timeout").WithType("block").WithTimeout(timeout).WithDescription(description)
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: element, Error: nil}
	default:
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("unsupported mode")}
	}
}

// TestWhenDecorator is a simple when decorator for testing
type TestWhenDecorator struct{}

func (w *TestWhenDecorator) Name() string        { return "when" }
func (w *TestWhenDecorator) Description() string { return "Test when decorator" }
func (w *TestWhenDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{Name: "variable", Type: ast.StringType, Required: true, Description: "Variable to check"},
	}
}

func (w *TestWhenDecorator) PatternSchema() decorators.PatternSchema {
	return decorators.PatternSchema{
		AllowsWildcard:      true,
		AllowsAnyIdentifier: true,
		Description:         "Test pattern matching",
	}
}

func (w *TestWhenDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{}
}

func (w *TestWhenDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Get variable name
	var varName string
	if len(params) > 0 {
		if strLit, ok := params[0].Value.(*ast.StringLiteral); ok {
			varName = strLit.Value
		}
	}

	switch ctx.Mode() {
	case execution.InterpreterMode:
		// For testing, just simulate success
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: nil}
	case execution.GeneratorMode:
		code := fmt.Sprintf("// When decorator checking variable %s\n// Patterns: %d", varName, len(patterns))
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: code, Error: nil}
	case execution.PlanMode:
		description := fmt.Sprintf("Conditional execution based on %s", varName)
		element := plan.Conditional(varName, "test_value", "matched_pattern").WithReason(description)
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: element, Error: nil}
	default:
		return &execution.ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("unsupported mode")}
	}
}