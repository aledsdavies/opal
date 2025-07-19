package decorators

import (
	"fmt"
	"os"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// WhenDecorator implements the @when decorator for conditional execution based on patterns
type WhenDecorator struct{}

// Name returns the decorator name
func (w *WhenDecorator) Name() string {
	return "when"
}

// Description returns a human-readable description
func (w *WhenDecorator) Description() string {
	return "Conditionally execute commands based on pattern matching"
}

// ParameterSchema returns the expected parameters for this decorator
func (w *WhenDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "variable",
			Type:        ast.StringType,
			Required:    true,
			Description: "Variable name to match against",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (w *WhenDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) != 1 {
		return fmt.Errorf("@when requires exactly 1 parameter (variable name), got %d", len(params))
	}
	
	// Validate the required variable parameter
	if err := ValidateRequiredParameter(params, "variable", ast.StringType, "when"); err != nil {
		return err
	}
	
	return nil
}

// Run executes the decorator at runtime with pattern matching
func (w *WhenDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) error {
	if err := w.Validate(ctx, params); err != nil {
		return err
	}
	
	// Get the variable name to match against
	varName := ast.GetStringParam(params, "variable", "")
	if varName == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if varLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			varName = varLiteral.Value
		}
	}
	
	// Get the variable value (for now, check environment variables)
	// TODO: This should use the execution context to get variable values
	value := os.Getenv(varName)
	
	// Find matching pattern branch
	for _, pattern := range patterns {
		if w.matchesPattern(value, pattern.Pattern) {
			patternStr := w.patternToString(pattern.Pattern)
			fmt.Printf("Pattern '%s' matched value '%s', would execute %d commands\n", 
				patternStr, value, len(pattern.Commands))
			// TODO: Execute the commands in the matching pattern
			return nil
		}
	}
	
	fmt.Printf("No pattern matched value '%s'\n", value)
	return nil
}

// matchesPattern checks if a value matches a pattern
func (w *WhenDecorator) matchesPattern(value string, pattern ast.Pattern) bool {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return value == p.Name
	case *ast.WildcardPattern:
		return true // Wildcard matches everything
	default:
		return false
	}
}

// patternToString converts a pattern to its string representation
func (w *WhenDecorator) patternToString(pattern ast.Pattern) string {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return p.Name
	case *ast.WildcardPattern:
		return "*"
	default:
		return "unknown"
	}
}

// Generate produces Go code for the decorator in compiled mode
func (w *WhenDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (string, error) {
	if err := w.Validate(ctx, params); err != nil {
		return "", err
	}
	
	// Get the variable name
	varName := ast.GetStringParam(params, "variable", "")
	if varName == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if varLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			varName = varLiteral.Value
		}
	}
	
	// Generate Go code for pattern matching
	code := fmt.Sprintf(`
// Pattern matching for variable: %s
value := os.Getenv(%q)
switch value {`, varName, varName)
	
	// Add cases for each pattern
	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		if _, ok := pattern.Pattern.(*ast.WildcardPattern); ok {
			code += `
default:`
		} else {
			code += fmt.Sprintf(`
case %q:`, patternStr)
		}
		code += fmt.Sprintf(`
	// Execute commands for pattern: %s
	// TODO: Generate command execution code`, patternStr)
	}
	
	code += `
}`
	
	return code, nil
}

// init registers the when decorator
func init() {
	RegisterPattern(&WhenDecorator{})
}