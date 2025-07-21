package decorators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
)

// WhenDecorator implements the @when decorator for conditional execution based on patterns
type WhenDecorator struct{}

// Template for when execution code generation
const whenExecutionTemplate = `func() error {
	// Pattern matching for variable: {{.VariableName}}
	value := os.Getenv({{printf "%q" .VariableName}})
	switch value {
	{{range $pattern := .Patterns}}
	{{if $pattern.IsDefault}}
	default:
	{{else}}
	case {{printf "%q" $pattern.Name}}:
	{{end}}
		// Execute commands for pattern: {{$pattern.Name}}
		if err := func() error {
			{{range $i, $cmd := $pattern.Commands}}
			if err := func() error {
				{{executeCommand $cmd}}
			}(); err != nil {
				return err
			}
			{{end}}
			return nil
		}(); err != nil {
			return err
		}
	{{end}}
	}
	return nil
}()`

// WhenPatternData holds data for a single pattern branch
type WhenPatternData struct {
	Name      string
	IsDefault bool
	Commands  []ast.CommandContent
}

// WhenTemplateData holds data for template execution
type WhenTemplateData struct {
	VariableName string
	Patterns     []WhenPatternData
}

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
			// Execute the commands in the matching pattern using unified execution engine
			return w.executeCommands(ctx, pattern.Commands)
		}
	}

	// No pattern matched - this is not an error, just no action needed
	return nil
}

// executeCommands executes commands using the unified execution engine
func (w *WhenDecorator) executeCommands(ctx *ExecutionContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		if err := ctx.ExecuteCommandContent(cmd); err != nil {
			return err
		}
	}
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
		return "default"
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

	// Convert patterns to template data
	var patternData []WhenPatternData
	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		isDefault := false
		if _, ok := pattern.Pattern.(*ast.WildcardPattern); ok {
			isDefault = true
		}

		patternData = append(patternData, WhenPatternData{
			Name:      patternStr,
			IsDefault: isDefault,
			Commands:  pattern.Commands,
		})
	}

	// Prepare template data
	templateData := WhenTemplateData{
		VariableName: varName,
		Patterns:     patternData,
	}

	// Parse and execute template with context functions
	tmpl, err := template.New("when").Funcs(ctx.GetTemplateFunctions()).Parse(whenExecutionTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse when template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute when template: %w", err)
	}

	return result.String(), nil
}

// Plan creates a plan element describing what this decorator would do in dry run mode
func (w *WhenDecorator) Plan(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (plan.PlanElement, error) {
	if err := w.Validate(ctx, params); err != nil {
		return nil, err
	}

	// Get the variable name to match against
	varName := ast.GetStringParam(params, "variable", "")
	if varName == "" && len(params) > 0 {
		// Fallback to positional if no named parameter
		if varLiteral, ok := params[0].Value.(*ast.StringLiteral); ok {
			varName = varLiteral.Value
		}
	}

	// Get current value from context or environment
	currentValue := ""
	if value, exists := ctx.GetVariable(varName); exists {
		currentValue = value
	} else {
		currentValue = os.Getenv(varName)
	}

	// Find matching pattern
	selectedPattern := "default"
	var selectedCommands []ast.CommandContent

	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		if patternStr == currentValue {
			selectedPattern = patternStr
			break
		}
		if patternStr == "default" {
			selectedCommands = pattern.Commands
		}
	}

	description := fmt.Sprintf("Evaluate %s = %q → execute '%s' branch (%d commands)",
		varName, currentValue, selectedPattern, len(selectedCommands))

	return plan.Decorator("when").
		WithType("pattern").
		WithParameter("variable", varName).
		WithDescription(description), nil
}

// ImportRequirements returns the dependencies needed for code generation
func (w *WhenDecorator) ImportRequirements() ImportRequirement {
	return ImportRequirement{
		StandardLibrary: []string{"os"}, // When decorator may need os for environment variables
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the when decorator
func init() {
	RegisterPattern(&WhenDecorator{})
}
