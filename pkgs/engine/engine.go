package engine

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/decorators"
)

// ExecutionMode determines how the engine operates
type ExecutionMode int

const (
	InterpreterMode ExecutionMode = iota // Run commands directly
	GeneratorMode                        // Generate Go code
)

// Engine provides a unified AST walker for both interpreter and generator modes
type Engine struct {
	mode      ExecutionMode
	ctx       *decorators.ExecutionContext
	goVersion string // Go version for generated code (e.g., "1.24")
}

// New creates a new execution engine
func New(mode ExecutionMode, ctx *decorators.ExecutionContext) *Engine {
	return &Engine{
		mode:      mode,
		ctx:       ctx,
		goVersion: "1.24", // Default Go version
	}
}

// NewWithGoVersion creates a new execution engine with specified Go version
func NewWithGoVersion(mode ExecutionMode, ctx *decorators.ExecutionContext, goVersion string) *Engine {
	return &Engine{
		mode:      mode,
		ctx:       ctx,
		goVersion: goVersion,
	}
}

// Execute processes the entire program
func (e *Engine) Execute(program *ast.Program) (interface{}, error) {
	switch e.mode {
	case InterpreterMode:
		return e.executeProgram(program)
	case GeneratorMode:
		return e.generateProgram(program)
	default:
		return nil, fmt.Errorf("unsupported execution mode: %v", e.mode)
	}
}

// executeProgram runs the program in interpreter mode
func (e *Engine) executeProgram(program *ast.Program) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Variables: make(map[string]string),
		Commands:  make([]CommandResult, 0),
	}

	// Process variables first
	if err := e.processVariables(program.Variables, result); err != nil {
		return nil, fmt.Errorf("failed to process variables: %w", err)
	}

	// Process variable groups
	if err := e.processVarGroups(program.VarGroups, result); err != nil {
		return nil, fmt.Errorf("failed to process variable groups: %w", err)
	}

	// Process commands
	for _, cmd := range program.Commands {
		cmdResult, err := e.executeCommand(&cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to execute command %s: %w", cmd.Name, err)
		}
		result.Commands = append(result.Commands, *cmdResult)
	}

	return result, nil
}

// generateProgram generates Go code for the program
func (e *Engine) generateProgram(program *ast.Program) (*GenerationResult, error) {
	result := &GenerationResult{
		Code:              strings.Builder{},
		GoMod:             strings.Builder{},
		StandardImports:   make(map[string]bool),
		ThirdPartyImports: make(map[string]bool),
		GoModules:         make(map[string]string),
	}

	// Process variables into context first (needed for decorator generation)
	if err := e.processVariablesIntoContext(program); err != nil {
		return nil, fmt.Errorf("failed to process variables: %w", err)
	}

	// Add base imports that are always needed
	result.AddStandardImport("context")
	result.AddStandardImport("fmt")
	result.AddStandardImport("os")

	// Collect import requirements from all decorators used in commands
	if err := e.collectDecoratorImports(program, result); err != nil {
		return nil, fmt.Errorf("failed to collect decorator imports: %w", err)
	}

	// Generate go.mod file with collected dependencies
	result.GoMod.WriteString("module devcmd-generated\n\n")
	result.GoMod.WriteString(fmt.Sprintf("go %s\n", e.goVersion))
	if len(result.GoModules) > 0 {
		result.GoMod.WriteString("\n")
		for module, version := range result.GoModules {
			result.GoMod.WriteString(fmt.Sprintf("require %s %s\n", module, version))
		}
	}

	// Generate package declaration and imports
	result.Code.WriteString("package main\n\n")
	result.Code.WriteString("import (\n")

	// Standard library imports
	for pkg := range result.StandardImports {
		result.Code.WriteString(fmt.Sprintf("\t\"%s\"\n", pkg))
	}

	// Third-party imports (if any)
	if len(result.ThirdPartyImports) > 0 {
		result.Code.WriteString("\n")
		for pkg := range result.ThirdPartyImports {
			result.Code.WriteString(fmt.Sprintf("\t\"%s\"\n", pkg))
		}
	}

	result.Code.WriteString(")\n\n")

	// Generate main function
	result.Code.WriteString("func main() {\n")
	result.Code.WriteString("\tctx := context.Background()\n\n")

	// Generate variable declarations
	if err := e.generateVariables(program.Variables, result); err != nil {
		return nil, fmt.Errorf("failed to generate variables: %w", err)
	}

	// Generate variable groups
	if err := e.generateVarGroups(program.VarGroups, result); err != nil {
		return nil, fmt.Errorf("failed to generate variable groups: %w", err)
	}

	// Generate commands
	for _, cmd := range program.Commands {
		if err := e.generateCommand(&cmd, result); err != nil {
			return nil, fmt.Errorf("failed to generate command %s: %w", cmd.Name, err)
		}
	}

	result.Code.WriteString("}\n")
	return result, nil
}

// executeCommand executes a single command in interpreter mode
func (e *Engine) executeCommand(cmd *ast.CommandDecl) (*CommandResult, error) {
	result := &CommandResult{
		Name:   cmd.Name,
		Status: "success",
	}

	// Process command content
	for _, content := range cmd.Body.Content {
		if err := e.executeCommandContent(content, result); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			return result, err
		}
	}

	return result, nil
}

// generateCommand generates Go code for a single command
func (e *Engine) generateCommand(cmd *ast.CommandDecl, result *GenerationResult) error {
	result.Code.WriteString(fmt.Sprintf("\t// Command: %s\n", cmd.Name))
	result.Code.WriteString("\tfunc() {\n")

	// Generate command content
	for _, content := range cmd.Body.Content {
		if err := e.generateCommandContent(content, result); err != nil {
			return fmt.Errorf("failed to generate content for command %s: %w", cmd.Name, err)
		}
	}

	result.Code.WriteString("\t}()\n\n")
	return nil
}

// executeCommandContent executes command content based on its type
func (e *Engine) executeCommandContent(content ast.CommandContent, result *CommandResult) error {
	switch c := content.(type) {
	case *ast.ShellContent:
		return e.executeShellContent(c, result)
	case *ast.BlockDecorator:
		return e.executeBlockDecorator(c, result)
	case *ast.PatternDecorator:
		return e.executePatternDecorator(c, result)
	case *ast.FunctionDecorator:
		return e.executeFunctionDecorator(c, result)
	default:
		return fmt.Errorf("unsupported command content type: %T", content)
	}
}

// generateCommandContent generates Go code for command content
func (e *Engine) generateCommandContent(content ast.CommandContent, result *GenerationResult) error {
	switch c := content.(type) {
	case *ast.ShellContent:
		return e.generateShellContent(c, result)
	case *ast.BlockDecorator:
		return e.generateBlockDecorator(c, result)
	case *ast.PatternDecorator:
		return e.generatePatternDecorator(c, result)
	case *ast.FunctionDecorator:
		return e.generateFunctionDecorator(c, result)
	default:
		return fmt.Errorf("unsupported command content type: %T", content)
	}
}

// executeShellContent executes shell commands by processing each part
func (e *Engine) executeShellContent(shell *ast.ShellContent, result *CommandResult) error {
	// Build the final command by processing each part
	var cmdBuilder strings.Builder

	for _, part := range shell.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - use as-is
			cmdBuilder.WriteString(p.Text)

		case *ast.FunctionDecorator:
			// Function decorator - execute using registry
			decorator, err := decorators.GetFunction(p.Name)
			if err != nil {
				return fmt.Errorf("function decorator %s not found: %w", p.Name, err)
			}

			value, err := decorator.Run(e.ctx, p.Args)
			if err != nil {
				return fmt.Errorf("failed to execute function decorator %s: %w", p.Name, err)
			}

			cmdBuilder.WriteString(value)

		default:
			return fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	expandedCmd := cmdBuilder.String()

	if e.ctx.Debug {
		result.Output = append(result.Output, fmt.Sprintf("Executing: %s", expandedCmd))
	}

	if e.ctx.DryRun {
		result.Output = append(result.Output, fmt.Sprintf("Would execute: %s", expandedCmd))
		return nil
	}

	// Execute the shell command
	execCmd := exec.Command("sh", "-c", expandedCmd)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	if err := execCmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// Don't exit immediately, let the caller handle the error
			return fmt.Errorf("command failed with exit code %d", exitError.ExitCode())
		}
		return fmt.Errorf("command failed: %w", err)
	}

	result.Output = append(result.Output, "Command completed successfully")
	return nil
}

// generateShellContent generates Go code for shell execution
func (e *Engine) generateShellContent(shell *ast.ShellContent, result *GenerationResult) error {
	// Build the final command by processing each part
	var cmdBuilder strings.Builder

	for _, part := range shell.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - use as-is
			cmdBuilder.WriteString(p.Text)

		case *ast.FunctionDecorator:
			// Function decorator - generate code using registry
			decorator, err := decorators.GetFunction(p.Name)
			if err != nil {
				return fmt.Errorf("function decorator %s not found: %w", p.Name, err)
			}

			code, err := decorator.Generate(e.ctx, p.Args)
			if err != nil {
				return fmt.Errorf("failed to generate code for function decorator %s: %w", p.Name, err)
			}

			cmdBuilder.WriteString(code)

		default:
			return fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	expandedCmd := cmdBuilder.String()
	result.Code.WriteString(fmt.Sprintf("\t\t// Shell: %s\n", expandedCmd))
	result.Code.WriteString(fmt.Sprintf("\t\tfmt.Printf(\"Executing: %s\\n\")\n", expandedCmd))
	result.Code.WriteString("\t\t// TODO: Add actual command execution\n")
	return nil
}

// executeBlockDecorator executes block decorators using the registry
func (e *Engine) executeBlockDecorator(block *ast.BlockDecorator, result *CommandResult) error {
	decorator, err := decorators.GetBlock(block.Name)
	if err != nil {
		return fmt.Errorf("block decorator %s not found: %w", block.Name, err)
	}

	return decorator.Run(e.ctx, block.Args, block.Content)
}

// generateBlockDecorator generates Go code for block decorators
func (e *Engine) generateBlockDecorator(block *ast.BlockDecorator, result *GenerationResult) error {
	decorator, err := decorators.GetBlock(block.Name)
	if err != nil {
		return fmt.Errorf("block decorator %s not found: %w", block.Name, err)
	}

	code, err := decorator.Generate(e.ctx, block.Args, block.Content)
	if err != nil {
		return fmt.Errorf("failed to generate code for block decorator %s: %w", block.Name, err)
	}

	result.Code.WriteString(fmt.Sprintf("\t\t// Block decorator: @%s\n", block.Name))
	result.Code.WriteString(code)
	result.Code.WriteString("\n")
	return nil
}

// executePatternDecorator executes pattern decorators using the registry
func (e *Engine) executePatternDecorator(pattern *ast.PatternDecorator, result *CommandResult) error {
	decorator, err := decorators.GetPattern(pattern.Name)
	if err != nil {
		return fmt.Errorf("pattern decorator %s not found: %w", pattern.Name, err)
	}

	return decorator.Run(e.ctx, pattern.Args, pattern.Patterns)
}

// generatePatternDecorator generates Go code for pattern decorators
func (e *Engine) generatePatternDecorator(pattern *ast.PatternDecorator, result *GenerationResult) error {
	decorator, err := decorators.GetPattern(pattern.Name)
	if err != nil {
		return fmt.Errorf("pattern decorator %s not found: %w", pattern.Name, err)
	}

	code, err := decorator.Generate(e.ctx, pattern.Args, pattern.Patterns)
	if err != nil {
		return fmt.Errorf("failed to generate code for pattern decorator %s: %w", pattern.Name, err)
	}

	result.Code.WriteString(fmt.Sprintf("\t\t// Pattern decorator: @%s\n", pattern.Name))
	result.Code.WriteString(code)
	result.Code.WriteString("\n")
	return nil
}

// executeFunctionDecorator executes function decorators using the registry
func (e *Engine) executeFunctionDecorator(fn *ast.FunctionDecorator, result *CommandResult) error {
	decorator, err := decorators.GetFunction(fn.Name)
	if err != nil {
		return fmt.Errorf("function decorator %s not found: %w", fn.Name, err)
	}

	value, err := decorator.Run(e.ctx, fn.Args)
	if err != nil {
		return fmt.Errorf("failed to execute function decorator %s: %w", fn.Name, err)
	}

	result.Output = append(result.Output, fmt.Sprintf("Function %s returned: %s", fn.Name, value))
	return nil
}

// generateFunctionDecorator generates Go code for function decorators
func (e *Engine) generateFunctionDecorator(fn *ast.FunctionDecorator, result *GenerationResult) error {
	decorator, err := decorators.GetFunction(fn.Name)
	if err != nil {
		return fmt.Errorf("function decorator %s not found: %w", fn.Name, err)
	}

	code, err := decorator.Generate(e.ctx, fn.Args)
	if err != nil {
		return fmt.Errorf("failed to generate code for function decorator %s: %w", fn.Name, err)
	}

	result.Code.WriteString(fmt.Sprintf("\t\t// Function decorator: @%s\n", fn.Name))
	result.Code.WriteString(code)
	result.Code.WriteString("\n")
	return nil
}

// processVariables processes variable declarations in interpreter mode
func (e *Engine) processVariables(variables []ast.VariableDecl, result *ExecutionResult) error {
	for _, variable := range variables {
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		result.Variables[variable.Name] = value
		e.ctx.SetVariable(variable.Name, value)
	}
	return nil
}

// processVarGroups processes variable groups in interpreter mode
func (e *Engine) processVarGroups(groups []ast.VarGroup, result *ExecutionResult) error {
	for _, group := range groups {
		if err := e.processVariables(group.Variables, result); err != nil {
			return err
		}
	}
	return nil
}

// generateVariables generates Go code for variable declarations
func (e *Engine) generateVariables(variables []ast.VariableDecl, result *GenerationResult) error {
	for _, variable := range variables {
		result.Code.WriteString(fmt.Sprintf("\t// Variable: %s\n", variable.Name))

		// Resolve the variable value to get the proper string representation
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}

		result.Code.WriteString(fmt.Sprintf("\t%s := %q\n", variable.Name, value))
	}
	result.Code.WriteString("\n")
	return nil
}

// generateVarGroups generates Go code for variable groups
func (e *Engine) generateVarGroups(groups []ast.VarGroup, result *GenerationResult) error {
	for _, group := range groups {
		result.Code.WriteString("\t// Variable group\n")
		if err := e.generateVariables(group.Variables, result); err != nil {
			return err
		}
	}
	return nil
}

// resolveVariableValue resolves a variable value expression
func (e *Engine) resolveVariableValue(expr ast.Expression) (string, error) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value, nil
	case *ast.NumberLiteral:
		return v.Value, nil
	case *ast.BooleanLiteral:
		if v.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.DurationLiteral:
		return v.Value, nil
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// ExecuteCommand executes a single command in interpreter mode (public method for external use)
func (e *Engine) ExecuteCommand(cmd *ast.CommandDecl) (*CommandResult, error) {
	return e.executeCommand(cmd)
}

// processVariablesIntoContext processes variables and variable groups into the execution context
// This is needed for generator mode to have access to variable values during code generation
func (e *Engine) processVariablesIntoContext(program *ast.Program) error {
	// Process individual variables
	for _, variable := range program.Variables {
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		e.ctx.SetVariable(variable.Name, value)
	}

	// Process variable groups
	for _, group := range program.VarGroups {
		for _, variable := range group.Variables {
			value, err := e.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
			}
			e.ctx.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// collectDecoratorImports scans the program and collects import requirements from all used decorators
func (e *Engine) collectDecoratorImports(program *ast.Program, result *GenerationResult) error {
	// Scan all commands for decorators
	for _, cmd := range program.Commands {
		if err := e.collectDecoratorImportsFromContent(cmd.Body.Content, result); err != nil {
			return fmt.Errorf("failed to collect imports from command %s: %w", cmd.Name, err)
		}
	}
	return nil
}

// collectDecoratorImportsFromContent recursively collects imports from command content
func (e *Engine) collectDecoratorImportsFromContent(content []ast.CommandContent, result *GenerationResult) error {
	for _, item := range content {
		switch c := item.(type) {
		case *ast.BlockDecorator:
			if err := e.addDecoratorImports("block", c.Name, result); err != nil {
				return err
			}
			// Recursively process nested content
			if err := e.collectDecoratorImportsFromContent(c.Content, result); err != nil {
				return err
			}
		case *ast.PatternDecorator:
			if err := e.addDecoratorImports("pattern", c.Name, result); err != nil {
				return err
			}
			// Process pattern branches
			for _, pattern := range c.Patterns {
				if err := e.collectDecoratorImportsFromContent(pattern.Commands, result); err != nil {
					return err
				}
			}
		case *ast.FunctionDecorator:
			if err := e.addDecoratorImports("function", c.Name, result); err != nil {
				return err
			}
		case *ast.ShellContent:
			// Check shell content parts for function decorators
			for _, part := range c.Parts {
				if fnDec, ok := part.(*ast.FunctionDecorator); ok {
					if err := e.addDecoratorImports("function", fnDec.Name, result); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// addDecoratorImports adds import requirements for a specific decorator
func (e *Engine) addDecoratorImports(decoratorType, name string, result *GenerationResult) error {
	var decorator decorators.Decorator
	var err error

	switch decoratorType {
	case "function":
		decorator, err = decorators.GetFunction(name)
	case "block":
		decorator, err = decorators.GetBlock(name)
	case "pattern":
		decorator, err = decorators.GetPattern(name)
	default:
		return fmt.Errorf("unknown decorator type: %s", decoratorType)
	}

	if err != nil {
		return fmt.Errorf("decorator %s not found: %w", name, err)
	}

	// Get import requirements from the decorator
	imports := decorator.ImportRequirements()

	// Add standard library imports
	for _, pkg := range imports.StandardLibrary {
		result.AddStandardImport(pkg)
	}

	// Add third-party imports
	for _, pkg := range imports.ThirdParty {
		result.AddThirdPartyImport(pkg)
	}

	// Add go.mod dependencies
	for module, version := range imports.GoModules {
		result.AddGoModule(module, version)
	}

	return nil
}
