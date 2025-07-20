package decorators

import (
	"fmt"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
)

// EnvDecorator implements the @env decorator for environment variable access
type EnvDecorator struct{}

// Name returns the decorator name
func (e *EnvDecorator) Name() string {
	return "env"
}

// Description returns a human-readable description
func (e *EnvDecorator) Description() string {
	return "Access environment variables with optional defaults"
}

// ParameterSchema returns the expected parameters for this decorator
func (e *EnvDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{
		{
			Name:        "key",
			Type:        ast.StringType,
			Required:    true,
			Description: "Environment variable name",
		},
		{
			Name:        "default",
			Type:        ast.StringType,
			Required:    false,
			Description: "Default value if environment variable is not set",
		},
	}
}

// Validate checks if the decorator usage is correct during parsing
func (e *EnvDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) == 0 {
		return fmt.Errorf("@env requires at least 1 parameter (key), got 0")
	}
	if len(params) > 2 {
		return fmt.Errorf("@env accepts at most 2 parameters (key, default), got %d", len(params))
	}
	
	// Check for required 'key' parameter
	keyParam := ast.FindParameter(params, "key")
	if keyParam == nil && len(params) > 0 {
		// If no named 'key', first parameter should be key
		keyParam = &params[0]
	}
	if keyParam == nil {
		return fmt.Errorf("@env requires 'key' parameter")
	}
	
	// Key parameter must be a string literal
	if _, ok := keyParam.Value.(*ast.StringLiteral); !ok {
		return fmt.Errorf("@env 'key' parameter must be a string literal (environment variable key)")
	}
	
	// Check optional 'default' parameter
	defaultParam := ast.FindParameter(params, "default")
	if defaultParam == nil && len(params) > 1 {
		// If no named 'default', second parameter should be default
		defaultParam = &params[1]
	}
	if defaultParam != nil {
		if _, ok := defaultParam.Value.(*ast.StringLiteral); !ok {
			return fmt.Errorf("@env 'default' parameter must be a string literal (default value)")
		}
	}
	
	return nil
}

// Run executes the decorator at runtime and returns the environment variable value
func (e *EnvDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter) (string, error) {
	if err := e.Validate(ctx, params); err != nil {
		return "", err
	}
	
	// Get the environment variable key using helper
	key := ast.GetStringParam(params, "key", "")
	if key == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if keyLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			key = keyLiteral.Value
		}
	}
	
	// Get the environment variable value
	value := os.Getenv(key)
	
	// If not found and default provided, use default
	if value == "" {
		value = ast.GetStringParam(params, "default", "")
	}
	
	return value, nil
}

// Generate produces Go code for the decorator in compiled mode
func (e *EnvDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter) (string, error) {
	if err := e.Validate(ctx, params); err != nil {
		return "", err
	}
	
	// Get the environment variable key using helper
	key := ast.GetStringParam(params, "key", "")
	if key == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if keyLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			key = keyLiteral.Value
		}
	}
	
	// Get default value if provided
	defaultValue := ast.GetStringParam(params, "default", "")
	
	// Generate Go code based on whether default is provided
	if defaultValue == "" {
		// No default value
		return fmt.Sprintf(`os.Getenv(%q)`, key), nil
	} else {
		// With default value
		var builder strings.Builder
		builder.WriteString("func() string {\n")
		builder.WriteString(fmt.Sprintf("\tif value := os.Getenv(%q); value != \"\" {\n", key))
		builder.WriteString("\t\treturn value\n")
		builder.WriteString("\t}\n")
		builder.WriteString(fmt.Sprintf("\treturn %q\n", defaultValue))
		builder.WriteString("}()")
		
		return builder.String(), nil
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (e *EnvDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"os"}, // Env decorator needs os package
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the env decorator
func init() {
	RegisterFunction(&EnvDecorator{})
}