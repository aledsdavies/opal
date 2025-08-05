package decorators

import (
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// WorkdirDecorator implements the @workdir decorator for changing working directory
type WorkdirDecorator struct{}

// generateUniqueVarName generates a unique variable name based on input content
// This helps avoid variable name conflicts in generated code
func generateUniqueVarName(prefix, content string) string {
	h := fnv.New32a()
	h.Write([]byte(content))
	return fmt.Sprintf("%s%d", prefix, h.Sum32())
}

// generateUniqueContextVar generates a unique context variable name for decorators
func generateUniqueContextVar(prefix, path, additionalContent string) string {
	return generateUniqueVarName(prefix+"Ctx", path+additionalContent)
}

// generateUniqueResultVar generates a unique result variable name for shell commands
func generateUniqueResultVar(prefix, command, context string) string {
	return generateUniqueVarName(prefix+"Result", command+context)
}

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
	return decorators.RequiresFileSystem() // Uses ResourceCleanupPattern + os operations
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
	// Use centralized validation
	if err := decorators.ValidateParameterCount(params, 1, 2, "workdir"); err != nil {
		return "", false, err
	}

	// Validate parameter schema compliance
	if err := decorators.ValidateSchemaCompliance(params, d.ParameterSchema(), "workdir"); err != nil {
		return "", false, err
	}

	// Enhanced security validation for path safety (no directory traversal, etc.)
	if err := decorators.ValidatePathSafety(params, "path", "workdir"); err != nil {
		return "", false, err
	}

	// Perform comprehensive security validation for all parameters
	_, err := decorators.PerformComprehensiveSecurityValidation(params, d.ParameterSchema(), "workdir")
	if err != nil {
		return "", false, err
	}

	// Parse parameters (validation passed, so these should be safe)
	path := ast.GetStringParam(params, "path", "")
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

// executeInterpreterImpl executes the workdir in interpreter mode using utilities
func (d *WorkdirDecorator) executeInterpreterImpl(ctx execution.InterpreterContext, path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Handle directory creation or verification
	if createIfNotExists {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(path, 0o755); err != nil {
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

	// Use CommandExecutor utility to handle command execution
	commandExecutor := decorators.NewCommandExecutor()
	defer commandExecutor.Cleanup()

	// Execute all commands in the workdir context
	err := commandExecutor.ExecuteCommandsWithInterpreter(workdirCtx, content)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  nil,
			Error: fmt.Errorf("execution failed in directory %s: %w", path, err),
		}
	}

	return &execution.ExecutionResult{
		Data:  nil,
		Error: nil,
	}
}

// executeGeneratorImpl generates Go code for the workdir decorator using new utilities
func (d *WorkdirDecorator) executeGeneratorImpl(ctx execution.GeneratorContext, path string, createIfNotExists bool, content []ast.CommandContent) *execution.ExecutionResult {
	// Generate inline code that integrates with sequential execution
	var generatedCode strings.Builder

	// Add directory verification/creation code
	if createIfNotExists {
		generatedCode.WriteString("// Create directory if it doesn't exist\n")
		generatedCode.WriteString(fmt.Sprintf("if err := os.MkdirAll(%q, 0755); err != nil {\n", path))
		generatedCode.WriteString(fmt.Sprintf("\treturn CommandResult{Stdout: \"\", Stderr: fmt.Sprintf(\"failed to create directory %s: %%v\", err), ExitCode: 1}\n", path))
		generatedCode.WriteString("}\n")
	} else {
		generatedCode.WriteString("// Verify target directory exists\n")
		generatedCode.WriteString(fmt.Sprintf("if _, err := os.Stat(%q); err != nil {\n", path))
		generatedCode.WriteString(fmt.Sprintf("\treturn CommandResult{Stdout: \"\", Stderr: fmt.Sprintf(\"failed to access directory %s: %%v\", err), ExitCode: 1}\n", path))
		generatedCode.WriteString("}\n")
	}

	// Generate unique context variable name using utility function
	contextVarName := generateUniqueContextVar("workdir", path, fmt.Sprintf("%p", &generatedCode))

	// Generate code to create ExecutionContext with updated working directory
	generatedCode.WriteString(fmt.Sprintf("// Create ExecutionContext with working directory: %s\n", path))
	generatedCode.WriteString(fmt.Sprintf("%s := execCtx.Child().WithWorkingDir(%q)\n", contextVarName, path))
	generatedCode.WriteString("\n")

	// Generate shell commands using template
	for _, cmdContent := range content {
		switch c := cmdContent.(type) {
		case *ast.ShellContent:
			// Generate shell code using unique workdir context template
			shellCode, err := d.generateShellCodeWithTemplate(c, contextVarName)
			if err != nil {
				return &execution.ExecutionResult{
					Data:  "",
					Error: fmt.Errorf("failed to generate shell code in workdir: %w", err),
				}
			}
			generatedCode.WriteString(shellCode)
		case *ast.BlockDecorator:
			// Handle nested decorators - this would need recursive processing
			// For now, return an error as this is complex
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("nested decorators in @workdir are not yet supported"),
			}
		default:
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("unsupported command content type in @workdir: %T", cmdContent),
			}
		}
	}

	return &execution.ExecutionResult{
		Data:  generatedCode.String(),
		Error: nil,
	}
}

// ShellTemplateData holds template data for workdir shell execution
type ShellTemplateData struct {
	Command string
}

// generateShellCodeWithTemplate generates shell execution code using unique workdir context
func (d *WorkdirDecorator) generateShellCodeWithTemplate(content *ast.ShellContent, contextVarName string) (string, error) {
	// Build the command string from shell content parts
	var cmdParts []string
	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			cmdParts = append(cmdParts, p.Text)
		case *ast.ValueDecorator:
			// For now, we'll include the decorator as-is
			// A full implementation would process @var decorators here
			cmdParts = append(cmdParts, fmt.Sprintf("@%s", p.Name))
		default:
			return "", fmt.Errorf("unsupported shell part type in workdir: %T", part)
		}
	}

	commandStr := strings.Join(cmdParts, "")

	// Generate unique variable name using utility function
	varName := generateUniqueResultVar("workdir", commandStr, contextVarName)

	// Define the template for shell execution with unique workdir context
	const workdirShellTemplate = `// Execute shell command in working directory
{{.VarName}} := executeShellCommand({{.ContextVar}}, {{printf "%q" .Command}})
if {{.VarName}}.Failed() {
	return {{.VarName}}
}
`

	// Parse and execute the template
	tmpl, err := template.New("workdirShell").Parse(workdirShellTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse workdir shell template: %w", err)
	}

	templateData := struct {
		Command    string
		VarName    string
		ContextVar string
	}{
		Command:    commandStr,
		VarName:    varName,
		ContextVar: contextVarName,
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return "", fmt.Errorf("failed to execute workdir shell template: %w", err)
	}

	return result.String(), nil
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
