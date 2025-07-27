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
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to get current directory: %w", err),
		}
	}

	// Change to target directory
	if err := os.Chdir(path); err != nil {
		return &execution.ExecutionResult{
			Mode:  execution.InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to change to directory %s: %w", path, err),
		}
	}

	// Ensure we restore the original directory
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore directory to %s: %v\n", originalDir, err)
		}
	}()

	// Execute the content in the new directory
	for _, cmdContent := range content {
		if err := ctx.ExecuteCommandContent(cmdContent); err != nil {
			return &execution.ExecutionResult{
				Mode:  execution.InterpreterMode,
				Data:  nil,
				Error: fmt.Errorf("failed to execute command in directory %s: %w", path, err),
			}
		}
	}

	return &execution.ExecutionResult{
		Mode:  execution.InterpreterMode,
		Data:  "",
		Error: nil,
	}
}

// executeGenerator generates Go code for the workdir decorator
func (d *WorkdirDecorator) executeGenerator(ctx *execution.ExecutionContext, path string, content []ast.CommandContent) *execution.ExecutionResult {
	// For now, return a simple code generation result
	// This would need more sophisticated code generation based on the content
	code := fmt.Sprintf(`func() error {
		originalDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %%w", err)
		}
		
		if err := os.Chdir(%q); err != nil {
			return fmt.Errorf("failed to change to directory %%s: %%w", %q, err)
		}
		
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to restore directory to %%s: %%v\n", originalDir, err)
			}
		}()
		
		// TODO: Execute content commands here
		return nil
	}()`, path, path)

	return &execution.ExecutionResult{
		Mode:  execution.GeneratorMode,
		Data:  code,
		Error: nil,
	}
}

// init registers the workdir decorator
func init() {
	decorators.RegisterBlock(&WorkdirDecorator{})
}
