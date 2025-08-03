package decorators

import (
	"fmt"
	"os"
	"strings"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)


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
	// Create a child context with the working directory set
	workdirCtx := ctx.Child().WithWorkingDir(path)
	
	// Convert commands to operations using the workdir context
	executor := decorators.NewCommandResultExecutor(workdirCtx)
	operations, err := executor.ConvertCommandsToCommandResultOperations(content)
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to convert commands to operations: %w", err),
		}
	}

	// Combine all operations into a single sequential operation
	var combinedCode string
	if len(operations) == 0 {
		combinedCode = "// No operations to execute"
	} else if len(operations) == 1 {
		combinedCode = operations[0].Code
	} else {
		// Use TemplateBuilder to create sequential execution
		sequentialBuilder := decorators.NewTemplateBuilder()
		sequentialBuilder.WithSequentialExecution(operations, true) // Stop on error
		
		sequentialCode, err := sequentialBuilder.BuildTemplate()
		if err != nil {
			return &execution.ExecutionResult{
				Data:  "",
				Error: fmt.Errorf("failed to build sequential template: %w", err),
			}
		}
		combinedCode = sequentialCode
	}

	// Create setup code for directory creation/verification
	var setupCode string
	if createIfNotExists {
		setupCode = fmt.Sprintf(`// Create directory if it doesn't exist
if err := os.MkdirAll(%q, 0755); err != nil {
	return CommandResult{Stdout: "", Stderr: fmt.Sprintf("failed to create directory %s: %%v", %q, err), ExitCode: 1}
}`, path, path, path)
	} else {
		setupCode = fmt.Sprintf(`// Verify target directory exists
if _, err := os.Stat(%q); err != nil {
	return CommandResult{Stdout: "", Stderr: fmt.Sprintf("failed to access directory %s: %%v", %q, err), ExitCode: 1}
}`, path, path, path)
	}

	// Create operation from combined code
	operation := decorators.Operation{Code: combinedCode}
	
	// Use TemplateBuilder to create resource cleanup pattern
	builder := decorators.NewTemplateBuilder()
	builder.WithResourceCleanup(setupCode, operation, "// No cleanup needed - working directory changes are isolated")

	// Build the template
	generatedCode, err := builder.BuildTemplate()
	if err != nil {
		return &execution.ExecutionResult{
			Data:  "",
			Error: fmt.Errorf("failed to build workdir template: %w", err),
		}
	}

	return &execution.ExecutionResult{
		Data:  generatedCode,
		Error: nil,
	}
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
