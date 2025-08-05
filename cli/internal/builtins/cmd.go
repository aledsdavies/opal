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

// GetCommandDependencies returns the command names this @cmd decorator depends on
func (d *CmdDecorator) GetCommandDependencies(params []ast.NamedParameter) []string {
	if len(params) == 0 {
		return []string{}
	}

	// Extract the command name from the first parameter
	if ident, ok := params[0].Value.(*ast.Identifier); ok {
		// Keep the original command name format (don't convert hyphens to underscores)
		return []string{ident.Name}
	}

	return []string{}
}

// ExpandInterpreter executes the command reference returning output for shell chaining
func (d *CmdDecorator) ExpandInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult {
	return d.ExecuteInterpreter(ctx, params)
}

// ExpandGenerator generates Go code for action chaining
func (d *CmdDecorator) ExpandGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter) *execution.ExecutionResult {
	return d.ExecuteGenerator(ctx, params)
}

// ExpandPlan creates a plan element for the command reference
func (d *CmdDecorator) ExpandPlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult {
	return d.ExecutePlan(ctx, params)
}

// ImportRequirements returns the dependencies needed for code generation
func (d *CmdDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"fmt", "bytes", "context", "os/exec", "os", "io"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ExecuteInterpreter executes the command reference in interpreter mode
func (d *CmdDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter) *execution.ExecutionResult {
	cmdName, err := d.extractCommandName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	// Find the command in the program
	program := ctx.GetProgram()
	var command *ast.CommandDecl
	for _, cmd := range program.Commands {
		if cmd.Name == cmdName {
			command = &cmd
			break
		}
	}

	if command == nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("command '%s' not found", cmdName),
		}
	}

	// Execute the command's content directly
	for _, content := range command.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("failed to execute command '%s': %w", cmdName, result.Error),
				}
			}
		default:
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("unsupported command content type in command '%s': %T", cmdName, content),
			}
		}
	}

	return &execution.ExecutionResult{
		Data:  "true", // Return "true" for shell chaining
		Error: nil,
	}
}

// ExecuteGenerator generates Go code for the command reference
func (d *CmdDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter) *execution.ExecutionResult {
	cmdName, err := d.extractCommandName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	// Generate function call that returns CommandResult for chaining
	// This allows @cmd to be used both standalone and in action chains
	functionName := strings.Title(toCamelCase(cmdName))
	code := fmt.Sprintf("execute%s()", functionName)

	return &execution.ExecutionResult{
		Data:  code,
		Error: nil,
	}
}

// ExecutePlan creates a plan element for the command reference
func (d *CmdDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter) *execution.ExecutionResult {
	cmdName, err := d.extractCommandName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	// Find the command in the program
	program := ctx.GetProgram()
	var command *ast.CommandDecl
	for _, cmd := range program.Commands {
		if cmd.Name == cmdName {
			command = &cmd
			break
		}
	}

	if command == nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("command '%s' not found", cmdName),
		}
	}

	// Create a simple plan element that references the command
	// For now, we'll create a basic plan element - this could be enhanced
	// to actually generate the nested plan for the referenced command
	planData := map[string]interface{}{
		"type":        "command_reference",
		"command":     cmdName,
		"description": fmt.Sprintf("Execute command: %s", cmdName),
	}

	return &execution.ExecutionResult{
		Data:  planData,
		Error: nil,
	}
}

// extractCommandName extracts the command name from decorator parameters
func (d *CmdDecorator) extractCommandName(params []ast.NamedParameter) (string, error) {
	// Get the command name parameter using the same pattern as var decorator
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}

	if nameParam == nil {
		return "", fmt.Errorf("@cmd decorator requires a command name parameter")
	}

	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		return ident.Name, nil
	} else {
		return "", fmt.Errorf("@cmd parameter must be an identifier, got %T", nameParam.Value)
	}
}

// toCamelCase converts a command name to camelCase for function naming
// This matches the engine's toCamelCase function exactly
func toCamelCase(name string) string {
	// Handle different separators: hyphens, underscores, and spaces
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	if len(parts) == 0 {
		return name
	}

	// First part stays lowercase, subsequent parts get title case
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += capitalizeFirst(parts[i])
	}

	return result
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// init registers the cmd decorator
func init() {
	decorators.RegisterAction(&CmdDecorator{})
}
