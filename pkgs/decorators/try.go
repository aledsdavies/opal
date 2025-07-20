package decorators

import (
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// TryDecorator implements the @try decorator for error handling with pattern matching
type TryDecorator struct{}

// Name returns the decorator name
func (t *TryDecorator) Name() string {
	return "try"
}

// Description returns a human-readable description
func (t *TryDecorator) Description() string {
	return "Execute commands with error handling via pattern matching (main required, error/finally optional but at least one required)"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TryDecorator) ParameterSchema() []ParameterSchema {
	return []ParameterSchema{} // @try takes no parameters
}

// Validate checks if the decorator usage is correct during parsing
func (t *TryDecorator) Validate(ctx *ExecutionContext, params []ast.NamedParameter) error {
	if len(params) > 0 {
		return fmt.Errorf("@try takes no parameters, got %d", len(params))
	}
	return nil
}

// Run executes the decorator at runtime with error handling patterns
func (t *TryDecorator) Run(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) error {
	if err := t.Validate(ctx, params); err != nil {
		return err
	}

	// Find pattern branches
	var mainBranch, errorBranch, finallyBranch *ast.PatternBranch
	
	for i := range patterns {
		pattern := &patterns[i]
		patternStr := t.patternToString(pattern.Pattern)
		
		switch patternStr {
		case "main":
			mainBranch = pattern
		case "error":
			errorBranch = pattern
		case "finally":
			finallyBranch = pattern
		default:
			return fmt.Errorf("@try only supports 'main', 'error', and 'finally' patterns, got '%s'", patternStr)
		}
	}

	// Validate required patterns
	if mainBranch == nil {
		return fmt.Errorf("@try requires a 'main' pattern")
	}
	if errorBranch == nil && finallyBranch == nil {
		return fmt.Errorf("@try requires at least one of 'error' or 'finally' patterns")
	}

	var mainErr error

	// Execute main block
	fmt.Printf("Executing 'main' pattern with %d commands\n", len(mainBranch.Commands))
	mainErr = t.executeCommands(ctx, mainBranch.Commands)

	// Execute error block if main failed and error pattern exists
	if mainErr != nil && errorBranch != nil {
		fmt.Printf("Main execution failed (%v), executing 'error' pattern with %d commands\n", mainErr, len(errorBranch.Commands))
		if errorErr := t.executeCommands(ctx, errorBranch.Commands); errorErr != nil {
			// If error handler also fails, we still want to run finally
			fmt.Printf("Error handler also failed: %v\n", errorErr)
		}
	}

	// Always execute finally block if it exists
	if finallyBranch != nil {
		fmt.Printf("Executing 'finally' pattern with %d commands\n", len(finallyBranch.Commands))
		if finallyErr := t.executeCommands(ctx, finallyBranch.Commands); finallyErr != nil {
			// Finally block errors are logged but don't override main error
			fmt.Printf("Finally block failed: %v\n", finallyErr)
		}
	}

	// Return the original main error (if any)
	return mainErr
}

// executeCommands simulates command execution (TODO: replace with actual execution engine)
func (t *TryDecorator) executeCommands(ctx *ExecutionContext, commands []ast.CommandContent) error {
	for i, cmd := range commands {
		fmt.Printf("  Executing command %d: %+v\n", i, cmd)
		
		// Simulate some main commands failing for testing
		if shellCmd, ok := cmd.(*ast.ShellContent); ok {
			cmdText := shellCmd.String()
			if strings.Contains(cmdText, "fail") {
				return fmt.Errorf("command failed: %s", cmdText)
			}
		}
	}
	return nil
}

// patternToString converts a pattern to its string representation
func (t *TryDecorator) patternToString(pattern ast.Pattern) string {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return p.Name
	default:
		return "unknown"
	}
}

// Generate produces Go code for the decorator in compiled mode
func (t *TryDecorator) Generate(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (string, error) {
	if err := t.Validate(ctx, params); err != nil {
		return "", err
	}

	// Find pattern branches for code generation
	var mainBranch, errorBranch, finallyBranch *ast.PatternBranch
	
	for i := range patterns {
		pattern := &patterns[i]
		patternStr := t.patternToString(pattern.Pattern)
		
		switch patternStr {
		case "main":
			mainBranch = pattern
		case "error":
			errorBranch = pattern
		case "finally":
			finallyBranch = pattern
		}
	}

	// Validate patterns for code generation
	if mainBranch == nil {
		return "", fmt.Errorf("@try requires a 'main' pattern")
	}
	if errorBranch == nil && finallyBranch == nil {
		return "", fmt.Errorf("@try requires at least one of 'error' or 'finally' patterns")
	}

	var builder strings.Builder
	builder.WriteString("func() error {\n")
	builder.WriteString("\tvar mainErr error\n")
	builder.WriteString("\n")

	// Generate main block execution
	builder.WriteString("\t// Execute main block\n")
	builder.WriteString("\tmainErr = func() error {\n")
	for i, cmd := range mainBranch.Commands {
		builder.WriteString(fmt.Sprintf("\t\t// Execute main command %d: %+v\n", i, cmd))
		builder.WriteString("\t\t// TODO: Generate actual command execution code\n")
		builder.WriteString("\t\t// if err := executeCommand(...); err != nil { return err }\n")
	}
	builder.WriteString("\t\treturn nil\n")
	builder.WriteString("\t}()\n")
	builder.WriteString("\n")

	// Generate error block execution if it exists
	if errorBranch != nil {
		builder.WriteString("\t// Execute error block if main failed\n")
		builder.WriteString("\tif mainErr != nil {\n")
		builder.WriteString("\t\terrorErr := func() error {\n")
		for i, cmd := range errorBranch.Commands {
			builder.WriteString(fmt.Sprintf("\t\t\t// Execute error command %d: %+v\n", i, cmd))
			builder.WriteString("\t\t\t// TODO: Generate actual command execution code\n")
			builder.WriteString("\t\t\t// if err := executeCommand(...); err != nil { return err }\n")
		}
		builder.WriteString("\t\t\treturn nil\n")
		builder.WriteString("\t\t}()\n")
		builder.WriteString("\t\tif errorErr != nil {\n")
		builder.WriteString("\t\t\tfmt.Printf(\"Error handler also failed: %v\\n\", errorErr)\n")
		builder.WriteString("\t\t}\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\n")
	}

	// Generate finally block execution if it exists
	if finallyBranch != nil {
		builder.WriteString("\t// Always execute finally block\n")
		builder.WriteString("\tfinallyErr := func() error {\n")
		for i, cmd := range finallyBranch.Commands {
			builder.WriteString(fmt.Sprintf("\t\t// Execute finally command %d: %+v\n", i, cmd))
			builder.WriteString("\t\t// TODO: Generate actual command execution code\n")
			builder.WriteString("\t\t// if err := executeCommand(...); err != nil { return err }\n")
		}
		builder.WriteString("\t\treturn nil\n")
		builder.WriteString("\t}()\n")
		builder.WriteString("\tif finallyErr != nil {\n")
		builder.WriteString("\t\tfmt.Printf(\"Finally block failed: %v\\n\", finallyErr)\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\n")
	}

	builder.WriteString("\t// Return the original main error\n")
	builder.WriteString("\treturn mainErr\n")
	builder.WriteString("}()")

	return builder.String(), nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (t *TryDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (plan.PlanElement, error) {
	if err := t.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Find pattern branches
	var mainBranch, errorBranch, finallyBranch *ast.PatternBranch
	
	for i := range patterns {
		pattern := &patterns[i]
		patternStr := t.patternToString(pattern.Pattern)
		
		switch patternStr {
		case "main":
			mainBranch = pattern
		case "error":
			errorBranch = pattern
		case "finally":
			finallyBranch = pattern
		}
	}

	description := "Try-catch execution: "
	if mainBranch != nil {
		description += fmt.Sprintf("execute main (%d commands)", len(mainBranch.Commands))
	}
	if errorBranch != nil {
		description += fmt.Sprintf(", on error execute fallback (%d commands)", len(errorBranch.Commands))
	}
	if finallyBranch != nil {
		description += fmt.Sprintf(", always execute finally (%d commands)", len(finallyBranch.Commands))
	}

	return plan.Decorator("try").
		WithType("pattern").
		WithDescription(description), nil
}

// ImportRequirements returns the dependencies needed for code generation
func (t *TryDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"fmt"}, // Try decorator needs fmt for error handling
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the try decorator
func init() {
	RegisterPattern(&TryDecorator{})
}