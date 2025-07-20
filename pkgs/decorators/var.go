package decorators

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// VarDecorator implements the @var decorator for variable references
type VarDecorator struct{}

// Name returns the decorator name
func (v *VarDecorator) Name() string {
	return "var"
}

// Description returns a human-readable description
func (v *VarDecorator) Description() string {
	return "Reference variables defined in the CLI file"
}

// ParameterSchema returns the expected parameters for this decorator
func (v *VarDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "name",
			Type:        ast.IdentifierType,
			Required:    true,
			Description: "Variable name to reference",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (v *VarDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) != 1 {
		return fmt.Errorf("@var requires exactly 1 parameter (variable identifier), got %d", len(params))
	}

	// Get the variable name parameter
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		// If no named 'name', first parameter should be the variable name
		nameParam = &params[0]
	}
	if nameParam == nil {
		return fmt.Errorf("@var requires 'name' parameter")
	}

	// Parameter must be an identifier (variable name)
	if _, ok := nameParam.Value.(*ast.Identifier); !ok {
		return fmt.Errorf("@var 'name' parameter must be an identifier (variable name), got %T", nameParam.Value)
	}

	return nil
}

// Run executes the decorator at runtime and returns the variable value
func (v *VarDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter) (string, error) {
	if err := v.Validate(ctx, params); err != nil {
		return "", err
	}

	// Get the variable name
	var varName string
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}
	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		varName = ident.Name
	}

	// Look up the variable in the execution context
	if value, exists := ctx.GetVariable(varName); exists {
		return value, nil
	}

	return "", fmt.Errorf("variable '%s' not defined", varName)
}

// Generate produces Go code for the decorator in compiled mode
func (v *VarDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter) (string, error) {
	if err := v.Validate(ctx, params); err != nil {
		return "", err
	}

	// Get the variable name
	var varName string
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}
	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		varName = ident.Name
	}

	// Look up the variable value and return it as a string literal
	if value, exists := ctx.GetVariable(varName); exists {
		return fmt.Sprintf("%q", value), nil
	}

	return "", fmt.Errorf("variable '%s' not defined", varName)
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (v *VarDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter) (plan.PlanElement, error) {
	if err := v.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Get the variable name
	var varName string
	nameParam := ast.FindParameter(params, "name")
	if nameParam == nil && len(params) > 0 {
		nameParam = &params[0]
	}
	if ident, ok := nameParam.Value.(*ast.Identifier); ok {
		varName = ident.Name
	}

	// Look up the variable in the execution context
	var description string
	if value, exists := ctx.GetVariable(varName); exists {
		description = fmt.Sprintf("Variable resolution: ${%s} → %q", varName, value)
	} else {
		description = fmt.Sprintf("Variable resolution: ${%s} → <undefined>", varName)
	}

	return plan.Decorator("var").
		WithType("function").
		WithParameter("name", varName).
		WithDescription(description), nil
}

// ImportRequirements returns the dependencies needed for code generation
func (v *VarDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{}, // No additional imports needed
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the var decorator
func init() {
	RegisterFunction(&VarDecorator{})
}
