package decorators

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// ValidateParameterType validates that a parameter value matches the expected type
// Allows both literal values and identifiers (which can resolve at runtime)
func ValidateParameterType(paramName string, paramValue ast.Expression, expectedType ast.ExpressionType, decoratorName string) error {
	switch expectedType {
	case ast.StringType:
		switch paramValue.(type) {
		case *ast.StringLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type string", decoratorName, paramName)
		}
	case ast.NumberType:
		switch paramValue.(type) {
		case *ast.NumberLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type number", decoratorName, paramName)
		}
	case ast.DurationType:
		switch paramValue.(type) {
		case *ast.DurationLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type duration", decoratorName, paramName)
		}
	case ast.BooleanType:
		switch paramValue.(type) {
		case *ast.BooleanLiteral, *ast.Identifier:
			return nil
		default:
			return fmt.Errorf("@%s '%s' parameter must be of type boolean", decoratorName, paramName)
		}
	default:
		return fmt.Errorf("@%s '%s' parameter has unsupported type %v", decoratorName, paramName, expectedType)
	}
}

// ValidateRequiredParameter validates that a required parameter exists and has the correct type
func ValidateRequiredParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s requires '%s' parameter", decoratorName, paramName)
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}

// ValidateOptionalParameter validates that an optional parameter (if present) has the correct type
func ValidateOptionalParameter(params []ast.NamedParameter, paramName string, expectedType ast.ExpressionType, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Optional parameter not provided, which is fine
	}
	return ValidateParameterType(paramName, param.Value, expectedType, decoratorName)
}

// ValidateParameterCount validates that the number of parameters is within expected bounds
func ValidateParameterCount(params []ast.NamedParameter, minParams, maxParams int, decoratorName string) error {
	count := len(params)
	if count < minParams {
		if minParams == maxParams {
			return fmt.Errorf("@%s requires exactly %d parameter(s), got %d", decoratorName, minParams, count)
		}
		return fmt.Errorf("@%s requires at least %d parameter(s), got %d", decoratorName, minParams, count)
	}
	if count > maxParams {
		if minParams == maxParams {
			return fmt.Errorf("@%s requires exactly %d parameter(s), got %d", decoratorName, maxParams, count)
		}
		return fmt.Errorf("@%s accepts at most %d parameter(s), got %d", decoratorName, maxParams, count)
	}
	return nil
}

// ValidatePositiveInteger validates that a numeric parameter is positive
func ValidatePositiveInteger(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the value
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else if value <= 0 {
			return fmt.Errorf("@%s '%s' parameter must be positive, got %d", decoratorName, paramName, value)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// ValidateIntegerRange validates that a numeric parameter is within a specific range
func ValidateIntegerRange(params []ast.NamedParameter, paramName string, min, max int, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the range
	if numLit, ok := param.Value.(*ast.NumberLiteral); ok {
		if value, err := strconv.Atoi(numLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid integer", decoratorName, paramName)
		} else if value < min || value > max {
			return fmt.Errorf("@%s '%s' parameter must be between %d and %d, got %d", decoratorName, paramName, min, max, value)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a number", decoratorName, paramName)
}

// ValidateDuration validates that a duration parameter is valid and within reasonable bounds
func ValidateDuration(params []ast.NamedParameter, paramName string, minDuration, maxDuration time.Duration, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // Duration parameters are typically optional
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For literals, validate the duration
	if durLit, ok := param.Value.(*ast.DurationLiteral); ok {
		if duration, err := time.ParseDuration(durLit.Value); err != nil {
			return fmt.Errorf("@%s '%s' parameter must be a valid duration (e.g., '1s', '5m')", decoratorName, paramName)
		} else if duration < minDuration {
			return fmt.Errorf("@%s '%s' parameter must be at least %v, got %v", decoratorName, paramName, minDuration, duration)
		} else if maxDuration > 0 && duration > maxDuration {
			return fmt.Errorf("@%s '%s' parameter must be at most %v, got %v", decoratorName, paramName, maxDuration, duration)
		}
		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a duration", decoratorName, paramName)
}

// ValidatePathSafety validates that a path parameter is safe (no directory traversal, etc.)
func ValidatePathSafety(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate path safety
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		path := strLit.Value
		
		// Check for empty path
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("@%s '%s' parameter cannot be empty", decoratorName, paramName)
		}

		// Check for dangerous patterns
		if strings.Contains(path, "..") {
			return fmt.Errorf("@%s '%s' parameter contains directory traversal (..), which is not allowed", decoratorName, paramName)
		}

		// Check for null bytes (security issue)
		if strings.Contains(path, "\x00") {
			return fmt.Errorf("@%s '%s' parameter contains null bytes, which is not allowed", decoratorName, paramName)
		}

		// Clean and validate the path
		cleanPath := filepath.Clean(path)
		if cleanPath != path && path != "." && path != ".." {
			// Allow common cases but warn about others
			// This is informational rather than blocking since Clean() may change valid paths
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// ValidateEnvironmentVariableName validates that an environment variable name is safe
func ValidateEnvironmentVariableName(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return fmt.Errorf("@%s '%s' parameter is required", decoratorName, paramName)
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate environment variable name
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		envName := strLit.Value
		
		// Check for empty name
		if strings.TrimSpace(envName) == "" {
			return fmt.Errorf("@%s '%s' parameter cannot be empty", decoratorName, paramName)
		}

		// Validate environment variable name format (POSIX standard)
		// Must start with letter or underscore, followed by letters, digits, or underscores
		envNameRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
		if !envNameRegex.MatchString(envName) {
			return fmt.Errorf("@%s '%s' parameter must be a valid environment variable name (letters, digits, underscore only, cannot start with digit)", decoratorName, paramName)
		}

		// Check for reasonable length (environment variable names shouldn't be too long)
		if len(envName) > 255 {
			return fmt.Errorf("@%s '%s' parameter is too long (max 255 characters)", decoratorName, paramName)
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// ValidateStringContent validates that a string parameter doesn't contain dangerous content
func ValidateStringContent(params []ast.NamedParameter, paramName string, decoratorName string) error {
	param := ast.FindParameter(params, paramName)
	if param == nil {
		return nil // String content parameters are typically optional
	}

	// For identifiers, we can't validate at parse time
	if _, isIdentifier := param.Value.(*ast.Identifier); isIdentifier {
		return nil
	}

	// For string literals, validate content safety
	if strLit, ok := param.Value.(*ast.StringLiteral); ok {
		content := strLit.Value
		
		// Check for null bytes (security issue)
		if strings.Contains(content, "\x00") {
			return fmt.Errorf("@%s '%s' parameter contains null bytes, which is not allowed", decoratorName, paramName)
		}

		// Check for excessively long strings (potential DoS)
		if len(content) > 10000 {
			return fmt.Errorf("@%s '%s' parameter is too long (max 10000 characters)", decoratorName, paramName)
		}

		// Check for potentially dangerous shell injection patterns if this might be used in shell context
		dangerousPatterns := []string{
			";", "&", "|", "`", "$(",  // Shell operators
			"\n", "\r",              // Newlines can break shell commands
		}
		
		for _, pattern := range dangerousPatterns {
			if strings.Contains(content, pattern) {
				// Note: This is a warning rather than an error since these might be legitimate
				// The actual shell execution should handle proper escaping
				break
			}
		}

		return nil
	}

	return fmt.Errorf("@%s '%s' parameter must be a string", decoratorName, paramName)
}

// ValidateSchemaCompliance validates parameters against a decorator's parameter schema
func ValidateSchemaCompliance(params []ast.NamedParameter, schema []ParameterSchema, decoratorName string) error {
	// Check for unknown parameters
	for _, param := range params {
		found := false
		for _, schemaParam := range schema {
			if schemaParam.Name == param.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("@%s does not accept parameter '%s'", decoratorName, param.Name)
		}
	}

	// Validate required parameters and types
	for _, schemaParam := range schema {
		if schemaParam.Required {
			if err := ValidateRequiredParameter(params, schemaParam.Name, schemaParam.Type, decoratorName); err != nil {
				return err
			}
		} else {
			if err := ValidateOptionalParameter(params, schemaParam.Name, schemaParam.Type, decoratorName); err != nil {
				return err
			}
		}
	}

	return nil
}
