package decorators

import (
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// workdirExecutionTemplate generates Go code for workdir execution (unified contract: statement blocks)
const workdirExecutionTemplate = `// Workdir execution setup
// Verify target directory exists
if _, err := os.Stat({{.Path | printf "%q"}}); err != nil {
	return CommandResult{Stdout: "", Stderr: fmt.Sprintf("failed to access directory %s: %v", {{.Path | printf "%q"}}, err), ExitCode: 1}
}

{{range $i, $cmd := .Commands}}
// Execute workdir command {{add $i 1}} in directory {{$.Path}}
{{.GeneratedCode}}
{{end}}`

// WorkdirTemplateData holds data for the workdir execution template
type WorkdirTemplateData struct {
	Path     string
	Commands []WorkdirCommandData
}

// WorkdirCommandData holds generated code for a single command within workdir
type WorkdirCommandData struct {
	GeneratedCode string
}

// WorkdirDecorator implements the @workdir decorator for changing working directory
type WorkdirDecorator struct{}

// Name returns the decorator name
func (d *WorkdirDecorator) Name() string {
	return "workdir"
}

// Description returns a human-readable description
func (d *WorkdirDecorator) Description() string {
	return "Changes working directory for the duration of the block, then restores original directory"
}

// ParameterSchema returns the expected parameters
func (d *WorkdirDecorator) ParameterSchema() []decorators.ParameterSchema {
	return []decorators.ParameterSchema{
		{
			Name:        "path",
			Type:        ast.StringType,
			Required:    true,
			Description: "Directory path to change to",
		},
	}
}

// ImportRequirements returns the dependencies needed for code generation
func (d *WorkdirDecorator) ImportRequirements() decorators.ImportRequirement {
	return decorators.ImportRequirement{
		StandardLibrary: []string{"os", "fmt"},
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// Execute provides unified execution for all modes
func (d *WorkdirDecorator) Execute(ctx *execution.ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	// Get the path parameter
	pathParam, err := d.getPathParameter(params)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	switch ctx.Mode() {
	case execution.PlanMode:
		return d.executePlan(pathParam, content)
	case execution.InterpreterMode:
		return d.executeInterpreter(ctx, pathParam, content)
	case execution.GeneratorMode:
		return d.executeGenerator(ctx, pathParam, content)
	default:
		return &execution.ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", ctx.Mode()),
		}
	}
}

// getPathParameter extracts and validates the path parameter
func (d *WorkdirDecorator) getPathParameter(params []ast.NamedParameter) (string, error) {
	if len(params) == 0 {
		return "", fmt.Errorf("workdir requires a path parameter")
	}

	pathParam := ast.FindParameter(params, "path")
	if pathParam == nil && len(params) > 0 {
		pathParam = &params[0]
	}

	if pathParam == nil {
		return "", fmt.Errorf("workdir requires a path parameter")
	}

	if str, ok := pathParam.Value.(*ast.StringLiteral); ok {
		return str.Value, nil
	}

	return "", fmt.Errorf("workdir path must be a string literal, got %T", pathParam.Value)
}

// executePlan creates a plan element for dry-run display
func (d *WorkdirDecorator) executePlan(path string, content []ast.CommandContent) *execution.ExecutionResult {
	element := plan.Decorator("workdir").
		WithType("block").
		WithParameter("path", path).
		WithDescription(fmt.Sprintf("@workdir(\"%s\")", path))

	// Add children for each content item to show nested structure
	for _, cmdContent := range content {
		switch c := cmdContent.(type) {
		case *ast.ShellContent:
			// Convert shell content to command element
			if len(c.Parts) > 0 {
				if text, ok := c.Parts[0].(*ast.TextPart); ok {
					cmd := strings.TrimSpace(text.Text)
					element.AddChild(plan.Command(cmd).WithDescription(cmd))
				}
			}
		case *ast.BlockDecorator:
			// For nested decorators, create a placeholder (the actual decorator will be processed separately)
			element.AddChild(plan.Command(fmt.Sprintf("@%s", c.Name)).WithDescription(fmt.Sprintf("@%s decorator", c.Name)))
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.PlanMode,
		Data:  element,
		Error: nil,
	}
}

// executeInterpreter executes the workdir in interpreter mode
func (d *WorkdirDecorator) executeInterpreter(ctx *execution.ExecutionContext, path string, content []ast.CommandContent) *execution.ExecutionResult {
	// Verify the target directory exists before proceeding
	if _, err := os.Stat(path); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to access directory %s: %w", path, err),
		}
	}

	// Create a new context with the updated working directory
	// This ensures isolated execution without affecting global process directory
	workdirCtx := *ctx  // Shallow copy the struct
	workdirCtx.WorkingDir = path  // Update the working directory
	
	// Note: Shallow copy preserves all function pointers and maps from original context

	// Execute the content in the new directory context
	var executionError error
	for i, cmdContent := range content {
		// Handle different content types with the workdir context
		var err error
		switch cmd := cmdContent.(type) {
		case *ast.ShellContent:
			// Execute shell content directly using our workdir context
			result := workdirCtx.ExecuteShell(cmd)
			err = result.Error
		default:
			// For other content types, fall back to the content executor
			err = workdirCtx.ExecuteCommandContent(cmdContent)
		}
		
		if err != nil {
			executionError = fmt.Errorf("command %d failed in directory %s: %w", i+1, path, err)
			break
		}
	}

	// No need to restore directory - we never changed the global process directory
	// The original context remains unchanged

	// Return execution error if any
	if executionError != nil {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: executionError,
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  "",
		Error: nil,
	}
}

// executeGenerator generates Go code for the workdir decorator using templates
func (d *WorkdirDecorator) executeGenerator(ctx *execution.ExecutionContext, path string, content []ast.CommandContent) *execution.ExecutionResult {
	// Create a child context with the working directory set
	// This ensures all nested commands get the correct working directory
	workdirCtx := ctx.Child().WithWorkingDir(path)
	
	// Pre-generate code for each command using the unified shell code builder
	// This supports all command content types: ShellContent, BlockDecorator, PatternDecorator
	var commandData []WorkdirCommandData
	for _, cmdContent := range content {
		// Use the unified shell code builder to handle all command content types
		shellBuilder := execution.NewShellCodeBuilder(workdirCtx)
		generatedCode, err := shellBuilder.GenerateShellCode(cmdContent)
		if err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.GeneratorMode,
				Data:  "",
				Error: fmt.Errorf("failed to generate code for workdir command: %w", err),
			}
		}
		
		commandData = append(commandData, WorkdirCommandData{
			GeneratedCode: generatedCode,
		})
	}
	
	// Prepare template data with pre-generated code
	templateData := WorkdirTemplateData{
		Path:     path,
		Commands: commandData,
	}

	// Parse and execute template with basic functions
	tmpl, err := template.New("workdir").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(workdirExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to parse workdir template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to execute workdir template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  result.String(),
		Error: nil,
	}
}

// generateShellExpression generates a Go string expression for a shell command
// This method is now DEPRECATED in favor of the unified shell template system
func (d *WorkdirDecorator) generateShellExpression(ctx *execution.ExecutionContext, content *ast.ShellContent) (string, error) {
	// Use the unified shell code generation system from ExecutionContext
	// This properly handles @var expansion and all other decorators
	return ctx.GenerateShellCodeForTemplate(content)
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
