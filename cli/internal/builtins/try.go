package decorators

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// TryDecorator implements the @try decorator for error handling with pattern matching
type TryDecorator struct{}

// Template for try execution code generation (unified contract: statement blocks)
const tryExecutionTemplate = `// Try-catch-finally execution with proper error handling
var tryMainErr error
var tryCatchErr error
var tryFinallyErr error

// Execute main block
tryMainErr = func() error {
	{{range $i, $cmd := .MainCommands}}
	{{$cmd}}
	{{end}}
	return nil
}()

{{if .HasCatchBranch}}
// Execute catch block if main failed
if tryMainErr != nil {
	tryCatchErr = func() error {
		{{range $i, $cmd := .CatchCommands}}
		{{$cmd}}
		{{end}}
		return nil
	}()
	if tryCatchErr != nil {
		fmt.Fprintf(os.Stderr, "Catch block failed: %v\n", tryCatchErr)
	}
}
{{end}}

{{if .HasFinallyBranch}}
// Always execute finally block regardless of main/catch success
tryFinallyErr = func() error {
	{{range $i, $cmd := .FinallyCommands}}
	{{$cmd}}
	{{end}}
	return nil
}()
if tryFinallyErr != nil {
	fmt.Fprintf(os.Stderr, "Finally block failed: %v\n", tryFinallyErr)
}
{{end}}

// Return the most significant error: main error takes precedence over catch/finally errors
if tryMainErr != nil {
	return fmt.Errorf("main block failed: %w", tryMainErr)
}
if tryCatchErr != nil {
	return fmt.Errorf("catch block failed: %w", tryCatchErr)
}
if tryFinallyErr != nil {
	return fmt.Errorf("finally block failed: %w", tryFinallyErr)
}
return nil`

// TryTemplateData holds data for template execution
type TryTemplateData struct {
	MainCommands     []string // Generated shell code strings
	CatchCommands    []string // Generated shell code strings
	FinallyCommands  []string // Generated shell code strings
	HasCatchBranch   bool
	HasFinallyBranch bool
}

// Name returns the decorator name
func (t *TryDecorator) Name() string {
	return "try"
}

// Description returns a human-readable description
func (t *TryDecorator) Description() string {
	return "Execute commands with try-catch-finally semantics (main required, catch/finally optional but at least one required)"
}

// ParameterSchema returns the expected parameters for this decorator
func (t *TryDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{} // @try takes no parameters
}

// decorators.PatternSchema defines what patterns @try accepts
func (t *TryDecorator) PatternSchema() decorators.PatternSchema {
	return decorators.PatternSchema{
		AllowedPatterns:     []string{"main", "catch", "finally"},
		RequiredPatterns:    []string{"main"},
		AllowsWildcard:      false, // No "default" wildcard for @try
		AllowsAnyIdentifier: false, // Only specific patterns allowed
		Description:         "Requires 'main', optionally accepts 'catch' and 'finally'",
	}
}

// Validate checks if the decorator usage is correct during parsing

// ExecuteInterpreter executes try-catch-finally in interpreter mode
func (t *TryDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	mainBranch, catchBranch, finallyBranch, err := t.validateAndExtractPatterns(params, patterns)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return t.executeInterpreterImpl(ctx, mainBranch, catchBranch, finallyBranch)
}

// ExecuteGenerator generates Go code for try-catch-finally logic
func (t *TryDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	mainBranch, catchBranch, finallyBranch, err := t.validateAndExtractPatterns(params, patterns)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: err,
		}
	}

	return t.executeGeneratorImpl(ctx, mainBranch, catchBranch, finallyBranch)
}

// ExecutePlan creates a plan element for dry-run mode
func (t *TryDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, patterns []ast.PatternBranch) *execution.ExecutionResult {
	mainBranch, catchBranch, finallyBranch, err := t.validateAndExtractPatterns(params, patterns)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: err,
		}
	}

	return t.executePlanImpl(ctx, mainBranch, catchBranch, finallyBranch)
}

// validateAndExtractPatterns validates parameters and extracts pattern branches
func (t *TryDecorator) validateAndExtractPatterns(params []ast.NamedParameter, patterns []ast.PatternBranch) (*ast.PatternBranch, *ast.PatternBranch, *ast.PatternBranch, error) {
	// Validate parameters first
	if len(params) > 0 {
		return nil, nil, nil, fmt.Errorf("try decorator takes no parameters, got %d", len(params))
	}

	// Find pattern branches
	var mainBranch, catchBranch, finallyBranch *ast.PatternBranch

	for i := range patterns {
		pattern := &patterns[i]
		patternStr := t.patternToString(pattern.Pattern)

		switch patternStr {
		case "main":
			mainBranch = pattern
		case "catch":
			catchBranch = pattern
		case "finally":
			finallyBranch = pattern
		default:
			return nil, nil, nil, fmt.Errorf("@try only supports 'main', 'catch', and 'finally' patterns, got '%s'", patternStr)
		}
	}

	// Validate required patterns
	if mainBranch == nil {
		return nil, nil, nil, fmt.Errorf("@try requires a 'main' pattern")
	}
	if catchBranch == nil && finallyBranch == nil {
		return nil, nil, nil, fmt.Errorf("@try requires at least one of 'catch' or 'finally' patterns")
	}

	return mainBranch, catchBranch, finallyBranch, nil
}

// executeInterpreterImpl executes try-catch-finally in interpreter mode with proper isolation
func (t *TryDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, mainBranch, catchBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	// Execute main block
	mainErr := t.executeCommands(ctx, mainBranch.Commands)

	// Execute catch block if main failed and catch pattern exists
	var catchErr error
	if mainErr != nil && catchBranch != nil {
		// Catch block executes in isolated context
		catchErr = t.executeCommands(ctx, catchBranch.Commands)
	}

	// Always execute finally block if it exists, regardless of main/catch success
	var finallyErr error
	if finallyBranch != nil {
		// Finally block executes in isolated context
		finallyErr = t.executeCommands(ctx, finallyBranch.Commands)
	}

	// Return the most significant error: main error takes precedence
	if mainErr != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("main block failed: %w", mainErr),
		}
	}
	if catchErr != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("catch block failed: %w", catchErr),
		}
	}
	if finallyErr != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("finally block failed: %w", finallyErr),
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: nil,
	}
}

// executeGeneratorImpl generates Go code for try-catch-finally logic with proper error handling
func (t *TryDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, mainBranch, catchBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	// Generate shell code for main commands
	mainCommands, err := t.generateCommandsCode(ctx, mainBranch.Commands)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to generate main commands: %w", err),
		}
	}

	// Generate shell code for catch commands if they exist
	var catchCommands []string
	if catchBranch != nil {
		catchCommands, err = t.generateCommandsCode(ctx, catchBranch.Commands)
		if err != nil {
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("failed to generate catch commands: %w", err),
			}
		}
	}

	// Generate shell code for finally commands if they exist
	var finallyCommands []string
	if finallyBranch != nil {
		finallyCommands, err = t.generateCommandsCode(ctx, finallyBranch.Commands)
		if err != nil {
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("failed to generate finally commands: %w", err),
			}
		}
	}

	// Prepare template data
	templateData := TryTemplateData{
		MainCommands:     mainCommands,
		CatchCommands:    catchCommands,
		FinallyCommands:  finallyCommands,
		HasCatchBranch:   catchBranch != nil,
		HasFinallyBranch: finallyBranch != nil,
	}

	// Parse and execute template
	tmpl, err := template.New("try").Parse(tryExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse try template: %w", err),
		}
	}

	var result bytes.Buffer
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute try template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
		Error: nil,
	}
}

// executePlanImpl creates a plan element for dry-run mode showing try-catch-finally structure
func (t *TryDecorator) executePlanImpl(ctx execution.PlanContext, mainBranch, catchBranch, finallyBranch *ast.PatternBranch) *execution.ExecutionResult {
	// Build description
	description := "Error handling with "
	var parts []string
	if mainBranch != nil {
		parts = append(parts, fmt.Sprintf("main (%d commands)", len(mainBranch.Commands)))
	}
	if catchBranch != nil {
		parts = append(parts, fmt.Sprintf("catch (%d commands)", len(catchBranch.Commands)))
	}
	if finallyBranch != nil {
		parts = append(parts, fmt.Sprintf("finally (%d commands)", len(finallyBranch.Commands)))
	}
	description += strings.Join(parts, ", ")

	// Create the main decorator element
	element := plan.Decorator("try").
		WithType("pattern").
		WithDescription(description)

	// Add main commands directly as children (always executed first)
	if mainBranch != nil {
		for _, cmd := range mainBranch.Commands {
			switch c := cmd.(type) {
			case *ast.ShellContent:
				result := ctx.GenerateShellPlan(c)
				if result.Error != nil {
					return &execution.ExecutionResult{
						Data:  nil,
						Error: fmt.Errorf("failed to create plan for main command: %w", result.Error),
					}
				}
				if childPlan, ok := result.Data.(*plan.ExecutionStep); ok {
					// Convert ExecutionStep to a Command element for the plan
					cmdElement := plan.Command(childPlan.Command).WithDescription(childPlan.Description)
					element = element.AddChild(cmdElement)
				}
			case *ast.BlockDecorator:
				// For nested decorators, create a plan element
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription("Nested decorator")
				element = element.AddChild(childElement)
			default:
				// Unknown command type
				childElement := plan.Command(fmt.Sprintf("Unknown command: %T", cmd)).WithDescription("Unsupported command")
				element = element.AddChild(childElement)
			}
		}
	}

	// Add catch block as a conditional child (executed only on error)
	if catchBranch != nil {
		// Create a conditional element for the catch block
		catchElement := plan.Decorator("[on error]").WithType("conditional").WithDescription("Executed only if main block fails")
		
		// Add catch commands as children of the conditional element
		for _, cmd := range catchBranch.Commands {
			switch c := cmd.(type) {
			case *ast.ShellContent:
				result := ctx.GenerateShellPlan(c)
				if result.Error != nil {
					return &execution.ExecutionResult{
						Data:  nil,
						Error: fmt.Errorf("failed to create plan for catch command: %w", result.Error),
					}
				}
				if childPlan, ok := result.Data.(*plan.ExecutionStep); ok {
					// Convert ExecutionStep to a Command element for the plan
					cmdElement := plan.Command(childPlan.Command).WithDescription(childPlan.Description)
					catchElement = catchElement.AddChild(cmdElement)
				}
			case *ast.BlockDecorator:
				// For nested decorators in catch
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription("Nested decorator in catch")
				catchElement = catchElement.AddChild(childElement)
			default:
				// Unknown command type in catch
				childElement := plan.Command(fmt.Sprintf("Unknown command: %T", cmd)).WithDescription("Unsupported command in catch")
				catchElement = catchElement.AddChild(childElement)
			}
		}
		
		// Add the catch element to the main try element
		element = element.AddChild(catchElement)
	}

	// Add finally block as an always-executed child
	if finallyBranch != nil {
		// Create an element for the finally block
		finallyElement := plan.Decorator("[always]").WithType("block").WithDescription("Always executed regardless of success/failure")
		
		// Add finally commands as children of the finally element
		for _, cmd := range finallyBranch.Commands {
			switch c := cmd.(type) {
			case *ast.ShellContent:
				result := ctx.GenerateShellPlan(c)
				if result.Error != nil {
					return &execution.ExecutionResult{
						Data:  nil,
						Error: fmt.Errorf("failed to create plan for finally command: %w", result.Error),
					}
				}
				if childPlan, ok := result.Data.(*plan.ExecutionStep); ok {
					// Convert ExecutionStep to a Command element for the plan
					cmdElement := plan.Command(childPlan.Command).WithDescription(childPlan.Description)
					finallyElement = finallyElement.AddChild(cmdElement)
				}
			case *ast.BlockDecorator:
				// For nested decorators in finally
				childElement := plan.Command(fmt.Sprintf("@%s{...}", c.Name)).WithDescription("Nested decorator in finally")
				finallyElement = finallyElement.AddChild(childElement)
			default:
				// Unknown command type in finally
				childElement := plan.Command(fmt.Sprintf("Unknown command: %T", cmd)).WithDescription("Unsupported command in finally")
				finallyElement = finallyElement.AddChild(childElement)
			}
		}
		
		// Add the finally element to the main try element
		element = element.AddChild(finallyElement)
	}

	return &execution.ExecutionResult{
		Data:  element,
		Error: nil,
	}
}

// executeCommands executes commands using the unified execution engine
func (t *TryDecorator) executeCommands(ctx execution.InterpreterContext, commands []ast.CommandContent) error {
	for _, cmd := range commands {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				return fmt.Errorf("shell command failed: %w", result.Error)
			}
		default:
			return fmt.Errorf("unsupported command type: %T", cmd)
		}
	}
	return nil
}

// generateCommandsCode generates Go code for a list of commands
func (t *TryDecorator) generateCommandsCode(ctx execution.GeneratorContext, commands []ast.CommandContent) ([]string, error) {
	result := make([]string, len(commands))
	for i, cmd := range commands {
		switch c := cmd.(type) {
		case *ast.ShellContent:
			shellResult := ctx.GenerateShellCode(c)
			if shellResult.Error != nil {
				return nil, fmt.Errorf("failed to generate shell code: %w", shellResult.Error)
			}
			result[i] = shellResult.Data.(string)
		default:
			result[i] = fmt.Sprintf("// Unsupported command type: %T", cmd)
		}
	}
	return result, nil
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

// ImportRequirements returns the dependencies needed for code generation
func (t *TryDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"fmt", "os"}, // Try decorator needs fmt and os for error handling
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// init registers the try decorator
func init() {
	decorators.RegisterPattern(&TryDecorator{})
}
