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
{{if .CreateIfNotExists}}
// Create directory if it doesn't exist
if err := os.MkdirAll({{.Path | printf "%q"}}, 0755); err != nil {
	return CommandResult{Stdout: "", Stderr: fmt.Sprintf("failed to create directory %s: %v", {{.Path | printf "%q"}}, err), ExitCode: 1}
}
{{else}}
// Verify target directory exists
if _, err := os.Stat({{.Path | printf "%q"}}); err != nil {
	return CommandResult{Stdout: "", Stderr: fmt.Sprintf("failed to access directory %s: %v", {{.Path | printf "%q"}}, err), ExitCode: 1}
}
{{end}}

{{range $i, $cmd := .Commands}}
// Execute workdir command {{add $i 1}} in directory {{$.Path}}
{{.GeneratedCode}}
{{end}}`

// WorkdirTemplateData holds data for the workdir execution template
type WorkdirTemplateData struct {
	Path             string
	CreateIfNotExists bool
	Commands         []WorkdirCommandData
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
		{
			Name:        "createIfNotExists",
			Type:        ast.BooleanType,
			Required:    false,
			Description: "Create directory if it doesn't exist (default: false)",
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

// ExecuteInterpreter executes workdir in interpreter mode
func (d *WorkdirDecorator) ExecuteInterpreter(ctx execution.InterpreterContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	return d.executeInterpreterImpl(ctx, pathParam, createIfNotExists, content)
}

// ExecuteGenerator generates Go code for workdir logic
func (d *WorkdirDecorator) ExecuteGenerator(ctx execution.GeneratorContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	return d.executeGeneratorImpl(ctx, pathParam, createIfNotExists, content)
}

// ExecutePlan creates a plan element for dry-run mode
func (d *WorkdirDecorator) ExecutePlan(ctx execution.PlanContext, params []ast.NamedParameter, content []ast.CommandContent) *execution.ExecutionResult {
	pathParam, createIfNotExists, err := d.extractWorkdirParams(params)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("workdir parameter error: %w", err),
		}
	}

	return d.executePlanImpl(pathParam, createIfNotExists, content)
}

// extractWorkdirParams extracts and validates workdir parameters
func (d *WorkdirDecorator) extractWorkdirParams(params []ast.NamedParameter) (string, bool, error) {
	if len(params) == 0 {
		return "", false, fmt.Errorf("workdir requires a path parameter")
	}

	pathParam := ast.FindParameter(params, "path")
	if pathParam == nil && len(params) > 0 {
		pathParam = &params[0]
	}

	if pathParam == nil {
		return "", false, fmt.Errorf("workdir requires a path parameter")
	}

	var path string
	if str, ok := pathParam.Value.(*ast.StringLiteral); ok {
		path = str.Value
	} else {
		return "", false, fmt.Errorf("workdir path must be a string literal, got %T", pathParam.Value)
	}

	// Get createIfNotExists parameter (default: false)
	createIfNotExists := ast.GetBoolParam(params, "createIfNotExists", false)

	return path, createIfNotExists, nil
}

// getPathParameter extracts and validates the path parameter (deprecated - use extractWorkdirParams)
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

// executePlanImpl creates a plan element for dry-run display
func (d *WorkdirDecorator) executePlanImpl(path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	description := fmt.Sprintf("@workdir(\"%s\")", path)
	if createIfNotExists {
		description += " (create if needed)"
	}
	
	element := plan.Decorator("workdir").
		WithType("block").
		WithParameter("path", path).
		WithDescription(description)

	if createIfNotExists {
		element = element.WithParameter("createIfNotExists", "true")
	}

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
		Data:  element,
		Error: nil,
	}
}

// executeInterpreterImpl executes the workdir in interpreter mode
func (d *WorkdirDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Handle directory creation or verification
	if createIfNotExists {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(path, 0755); err != nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("failed to create directory %s: %w", path, err),
			}
		}
	} else {
		// Verify the target directory exists before proceeding
		if _, err := os.Stat(path); err != nil {
			return &execution.ExecutionResult{
				Data:  nil,
				Error: fmt.Errorf("failed to access directory %s: %w", path, err),
			}
		}
	}

	// Create a new context with the updated working directory
	// This ensures isolated execution without affecting global process directory
	workdirCtx := ctx.WithWorkingDir(path)

	// Execute the content in the new directory context
	var executionError error
	for i, cmdContent := range content {
		// Execute all content types using the unified executor with workdir context
		err := workdirCtx.ExecuteCommandContent(cmdContent)
		
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
			Data:  nil,
			Error: executionError,
		}
	}

	return &execution.ExecutionResult{
		Data:  "",
		Error: nil,
	}
}

// executeGeneratorImpl generates Go code for the workdir decorator using templates
func (d *WorkdirDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
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
		Path:             path,
		CreateIfNotExists: createIfNotExists,
		Commands:         commandData,
	}

	// Parse and execute template with basic functions
	tmpl, err := template.New("workdir").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(workdirExecutionTemplate)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to parse workdir template: %w", err),
		}
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to execute workdir template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  result.String(),
		Error: nil,
	}
}

// generateShellExpression generates a Go string expression for a shell command
// This method is now DEPRECATED in favor of the unified shell template system
func (d *WorkdirDecorator) generateShellExpression(ctx execution.GeneratorContext, content *ast.ShellContent) (string, error) {
	// This method is deprecated - use the unified shell generation system
	return "", fmt.Errorf("generateShellExpression is deprecated - use GenerateShellCode directly")
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
