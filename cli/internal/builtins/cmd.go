package decorators

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// CmdDecorator implements the @cmd decorator for referencing other commands
type CmdDecorator struct{}

// Name returns the decorator name
func (d *CmdDecorator) Name() string {
	return "cmd"
}

// Description returns a human-readable description
func (d *CmdDecorator) Description() string {
	return "References another defined command by name for reuse"
}

// ParameterSchema returns the expected parameters
func (d *CmdDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "name",
			Type:        ast.IdentifierType,
			Required:    true,
			Description: "Name of the command to reference",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (d *CmdDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"fmt"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// Expand provides unified execution for all modes
func (d *CmdDecorator) Expand(ctx *execution.ExecutionContext, params []ast.NamedParameter) *execution.ExecutionResult {
	// Get the command name parameter using the same pattern as var decorator
	var cmdName string
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}

	if nameParam == nil {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@cmd decorator requires a command name parameter"),
		}
	}

	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		cmdName = ident.Name
	} else {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@cmd parameter must be an identifier, got %T", nameParam.Value),
		}
	}

	switch ctx.Mode() {
	case execution.PlanMode:
		return d.executePlan(ctx, cmdName)
	case execution.InterpreterMode:
		return d.executeInterpreter(ctx, cmdName)
	case execution.GeneratorMode:
		return d.executeGenerator(ctx, cmdName)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// executePlan creates a plan element for the command reference
func (d *CmdDecorator) executePlan(ctx *execution.ExecutionContext, cmdName string) *execution.ExecutionResult {
	// Generate the plan for the referenced command
	planResult, err := ctx.GenerateCommandPlan(cmdName)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.PlanMode,
			Data:  nil,
			Error: fmt.Errorf("failed to generate plan for command '%s': %w", cmdName, err),
		}
	}

	// Return the nested plan
	return planResult
}

// executeInterpreter executes the command reference in interpreter mode
func (d *CmdDecorator) executeInterpreter(ctx *execution.ExecutionContext, cmdName string) *execution.ExecutionResult {
	// Execute the referenced command
	err := ctx.ExecuteCommand(cmdName)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to execute command '%s': %w", cmdName, err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  "true", // Return "true" for shell chaining
		Error: nil,
	}
}

// executeGenerator generates Go code for the command reference
func (d *CmdDecorator) executeGenerator(ctx *execution.ExecutionContext, cmdName string) *execution.ExecutionResult {
	// Generate a function call to the referenced command
	// This assumes that each command becomes a function in the generated code
	functionName := fmt.Sprintf("cmd_%s", strings.ReplaceAll(cmdName, "-", "_"))
	code := fmt.Sprintf("%s()", functionName)

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  code,
		Error: nil,
	}
}

// init registers the cmd decorator
func init() {
	decorators.RegisterFunction(&CmdDecorator{})
}
