package decorators

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// TryDecorator implements the @try decorator for error handling with pattern matching
type TryDecorator struct{}

// Template for try execution code generation
const tryExecutionTemplate = `return func() error {
	var mainErr error

	// Execute main block
	mainErr = func() error {
		{{range $i, $cmd := .MainCommands}}
		if err := func() error {
			{{executeCommand $cmd}}
		}(); err != nil {
			return err
		}
		{{end}}
		return nil
	}()

	{{if .HasErrorBranch}}
	// Execute error block if main failed
	if mainErr != nil {
		errorErr := func() error {
			{{range $i, $cmd := .ErrorCommands}}
			if err := func() error {
				{{executeCommand $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}()
		if errorErr != nil {
			fmt.Printf("Error handler also failed: %v\n", errorErr)
		}
	}
	{{end}}

	{{if .HasFinallyBranch}}
	// Always execute finally block
	finallyErr := func() error {
		{{range $i, $cmd := .FinallyCommands}}
		if err := func() error {
			{{executeCommand $cmd}}
		}(); err != nil {
			return err
		}
		{{end}}
		return nil
	}()
	if finallyErr != nil {
		fmt.Printf("Finally block failed: %v\n", finallyErr)
	}
	{{end}}

	// Return the original main error
	return mainErr
}()`

// TryTemplateData holds data for template execution
type TryTemplateData struct {
	MainCommands     []ast.CommandContent
	ErrorCommands    []ast.CommandContent
	FinallyCommands  []ast.CommandContent
	HasErrorBranch   bool
	HasFinallyBranch bool
}

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
	mainErr = t.executeCommands(ctx, mainBranch.Commands)

	// Execute error block if main failed and error pattern exists
	if mainErr != nil && errorBranch != nil {
		// If error handler also fails, we still want to run finally
		t.executeCommands(ctx, errorBranch.Commands)
	}

	// Always execute finally block if it exists
	if finallyBranch != nil {
		// Finally block errors don't override main error
		t.executeCommands(ctx, finallyBranch.Commands)
	}

	// Return the original main error (if any)
	return mainErr
}

// executeCommands executes commands using the unified execution engine
func (t *TryDecorator) executeCommands(ctx *ExecutionContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return err
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

	// Prepare template data
	templateData := TryTemplateData{
		MainCommands:     mainBranch.Commands,
		HasErrorBranch:   errorBranch != nil,
		HasFinallyBranch: finallyBranch != nil,
	}

	if errorBranch != nil {
		templateData.ErrorCommands = errorBranch.Commands
	}

	if finallyBranch != nil {
		templateData.FinallyCommands = finallyBranch.Commands
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("try").Funcs(ctx.GetTemplateFunctions()).Parse(tryExecutionTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse try template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute try template: %w", err)
	}

	return result.String(), nil
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
