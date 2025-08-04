package decorators

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// WhenDecorator implements the @when decorator for conditional execution based on patterns
type WhenDecorator struct{}

// Template for when execution code generation (unified contract: statement blocks)
const whenExecutionTemplate = `// Pattern matching for variable: {{.VariableName}}
// Get variable value from context or captured environment at runtime
var {{.VariableName}}Value string
if ctxValue, exists := variableContext[{{printf "%q" .VariableName}}]; exists {
	{{.VariableName}}Value = ctxValue
} else if envValue, exists := ctx.EnvContext[{{printf "%q" .VariableName}}]; exists {
	{{.VariableName}}Value = envValue
}

switch {{.VariableName}}Value {
{{range $pattern := .Patterns}}
{{if $pattern.IsDefault}}
default:
{{else}}
case {{printf "%q" $pattern.Name}}:
{{end}}
	// Execute commands for pattern: {{$pattern.Name}}
	{{range $i, $cmd := $pattern.Commands}}
	{{$cmd}}
	{{end}}
{{end}}
}`

// WhenPatternData holds data for a single pattern branch
type WhenPatternData struct {
	Name      string
	IsDefault bool
	Commands  []string // Generated shell code strings
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
func (w *WhenDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "variable",
			Type:        ast.StringType,
			Required:    true,
			Description: "Variable name to match against",
		},
	}
}

// decorators.PatternSchema defines what patterns @when accepts
func (w *WhenDecorator) PatternSchema() decorators.PatternSchema {
	return decorators.PatternSchema{
		AllowedPatterns:     []string{}, // No specific patterns - any identifier is allowed
		RequiredPatterns:    []string{}, // No required patterns
		AllowsWildcard:      true,       // "default" wildcard is allowed
		AllowsAnyIdentifier: true,       // Any identifier is allowed (production, staging, etc.)
		Description:         "Accepts any identifier patterns and 'default' wildcard",
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes pattern matching in interpreter mode
func (w *WhenDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	varName, err := w.extractVariableName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return w.executeInterpreterImpl(ctx, varName, patterns)
}

// ExecuteGenerator generates Go code for pattern matching with runtime variable resolution
func (w *WhenDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	varName, err := w.extractVariableName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	return w.executeGeneratorImpl(ctx, varName, patterns)
}

// ExecutePlan creates a plan element for dry-run mode showing the path taken for current environment
func (w *WhenDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	varName, err := w.extractVariableName(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return w.executePlanImpl(ctx, varName, patterns)
}

// extractVariableName extracts and validates the variable name parameter
func (w *WhenDecorator) extractVariableName(params []ast.NamedParameter) (string, error) {
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 1, "when"); err != nil {
		return "", err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, w.ParameterSchema(), "when"); err != nil {
		return "", err
	}

	// Parse parameters (validation passed, so these should be safe)
	varName := ast.GetStringParam(params, "variable", "")
	
	// Additional check for empty variable name (shouldn't happen after validation)
	if varName == "" {
		return "", fmt.Errorf("when decorator requires a valid 'variable' parameter")
	}

	return varName, nil
}

// executeInterpreterImpl executes pattern matching in interpreter mode
func (w *WhenDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Get the variable value (check context first, then captured environment)
	value := ""
	if ctxValue, exists := ctx.GetVariable(varName); exists {
		value = ctxValue
	} else if envValue, exists := ctx.GetEnv(varName); exists {
		value = envValue
	}

	// Find matching pattern branch
	for _, pattern := range patterns {
		if w.matchesPattern(value, pattern.Pattern) {
			// Execute the commands in the matching pattern
			if err := w.executeCommands(ctx, pattern.Commands); err != nil {
				return &execution.ExecutionResult{
					Data:  nil,
					Error: err,
				}
			}
			break
		}
	}

	// No pattern matched or execution succeeded
	return &execution.ExecutionResult{
		Data:  nil,
		Error: nil,
	}
}

// executeGeneratorImpl generates Go code for pattern matching with runtime variable resolution
func (w *WhenDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Track the variable for global environment capture in generated code
	// This ensures the variable is available in the generated binary's envContext
	ctx.TrackEnvironmentVariableReference(varName, "")

	// Convert patterns to template data with generated shell code
	var patternData []WhenPatternData
	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		isDefault := false
		if _, ok := pattern.Pattern.(*ast.WildcardPattern); ok {
			isDefault = true
		}

		// Generate shell code for each command in the pattern
		commandCodes := make([]string, len(pattern.Commands))
		for j, cmd := range pattern.Commands {
			switch c := cmd.(type) {
			case *ast.ShellContent:
				// Generate shell code using the execution context
				result := ctx.GenerateShellCode(c)
				if result.Error != nil {
					return &execution.ExecutionResult{
						Data:  "",
						Error: fmt.Errorf("failed to generate shell code for pattern %s: %w", patternStr, result.Error),
					}
				}
				commandCodes[j] = result.Data.(string)
			default:
				commandCodes[j] = fmt.Sprintf("// Unsupported command type: %T", cmd)
			}
		}

		patternData = append(patternData, WhenPatternData{
			Name:      patternStr,
			IsDefault: isDefault,
			Commands:  commandCodes,
		})
	}

	// Prepare template data for runtime variable resolution
	templateData := WhenTemplateData{
		VariableName: varName,
		Patterns:     patternData,
	}

	// Parse and execute template
	tmpl, err := template.New("when").Parse(whenExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse when template: %w", err),
		}
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute when template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
		Error: nil,
	}
}

// executePlanImpl creates a plan element for dry-run mode showing the path taken for current environment
func (w *WhenDecorator) executePlanImpl(ctx execution.PlanContext, varName string, patterns []ast.PatternBranch) *execution.ExecutionResult {
	// Get current value from context or captured environment to show which path would be taken
	currentValue := ""
	if value, exists := ctx.GetVariable(varName); exists {
		currentValue = value
	} else if envValue, exists := ctx.GetEnv(varName); exists {
		currentValue = envValue
	}

	// Find matching pattern
	selectedPattern := "default"
	var selectedCommands []ast.CommandContent

	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		if w.matchesPattern(currentValue, pattern.Pattern) {
			selectedPattern = patternStr
			selectedCommands = pattern.Commands
			break
		}
	}

	element := plan.Conditional(varName, currentValue, selectedPattern).
		WithReason(fmt.Sprintf("Variable %s matched %s", varName, selectedPattern))

	// Add all branch information for completeness
	for _, pattern := range patterns {
		patternStr := w.patternToString(pattern.Pattern)
		willExecute := patternStr == selectedPattern
		branchDesc := fmt.Sprintf("Execute %d commands", len(pattern.Commands))
		element = element.AddBranch(patternStr, branchDesc, willExecute)
	}

	// Build child plan elements for the selected commands only
	for _, cmd := range selectedCommands {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			// Create plan element for shell command
			result := ctx.GenerateShellPlan(c)
			if result.Error != nil {
				return &execution.ExecutionResult{
					Data:  nil,
					Error: fmt.Errorf("failed to create plan for shell content: %w", result.Error),
				}
			}

			// Add child plan element
			if childPlan, ok := result.Data.(*plan.ExecutionStep); ok {
				// Convert ExecutionStep to a Command element for the plan
				cmdElement := plan.Command(childPlan.Command).WithDescription(childPlan.Description)
				element = element.WithChildren(cmdElement)
			}
		case *ast.BlockDecorator:
			// For nested decorators, create a plan element
			childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription("Nested decorator")
			element = element.WithChildren(childElement)
		default:
			// Unknown command type
			childElement := plan.Command(fmt.Sprintf("Unknown command type: %T", cmd)).WithDescription("Unsupported command")
			element = element.WithChildren(childElement)
		}
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// executeCommands executes commands using the unified execution engine
func (w *WhenDecorator) executeCommands(ctx execution.InterpreterContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return fmt.Errorf("failed to execute shell command: %w", result.Error)
			}
		default:
			return fmt.Errorf("unsupported command type: %T", cmd)
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

// ImportRequirements returns the dependencies needed for code generation
func (w *WhenDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{}, // No imports needed - generates string literals
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the when decorator
func init() {
	decorators.RegisterPattern(&WhenDecorator{})
}
