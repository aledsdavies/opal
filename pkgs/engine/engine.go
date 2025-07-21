package engine

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

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
	engine := &Engine{
		mode:      mode,
		ctx:       ctx,
		goVersion: "1.24", // Default Go version
	}
	// Set up template functions for code generation
	engine.setupTemplateFunctions()
	return engine
}

// NewWithGoVersion creates a new execution engine with specified Go version
func NewWithGoVersion(mode ExecutionMode, ctx *decorators.ExecutionContext, goVersion string) *Engine {
	engine := &Engine{
		mode:      mode,
		ctx:       ctx,
		goVersion: goVersion,
	}
	// Set up template functions for code generation
	engine.setupTemplateFunctions()
	return engine
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
	result.AddStandardImport("fmt")

	// Add context and signal handling imports if the program has commands
	if len(program.Commands) > 0 {
		result.AddStandardImport("context")
		result.AddStandardImport("os")
		result.AddStandardImport("os/exec")
		result.AddStandardImport("os/signal")
		result.AddStandardImport("syscall")
	}

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

	// Generate package declaration
	result.Code.WriteString(packageTemplate)

	// Generate imports
	standardImports := make([]string, 0, len(result.StandardImports))
	for pkg := range result.StandardImports {
		standardImports = append(standardImports, pkg)
	}
	thirdPartyImports := make([]string, 0, len(result.ThirdPartyImports))
	for pkg := range result.ThirdPartyImports {
		thirdPartyImports = append(thirdPartyImports, pkg)
	}

	importsCode, err := renderImports(len(program.Commands) > 0, standardImports, thirdPartyImports)
	if err != nil {
		return nil, fmt.Errorf("failed to render imports: %w", err)
	}
	result.Code.WriteString(importsCode)

	// Generate main function start
	result.Code.WriteString(mainStartTemplate)

	// Add signal handling if there are commands
	if len(program.Commands) > 0 {
		result.Code.WriteString(signalHandlingTemplate)
	}

	// Generate variable declarations
	if err := e.generateVariables(program.Variables, program, result); err != nil {
		return nil, fmt.Errorf("failed to generate variables: %w", err)
	}

	// Generate variable groups
	if err := e.generateVarGroups(program.VarGroups, program, result); err != nil {
		return nil, fmt.Errorf("failed to generate variable groups: %w", err)
	}

	// Generate commands
	for _, cmd := range program.Commands {
		if err := e.generateCommand(&cmd, result); err != nil {
			return nil, fmt.Errorf("failed to generate command %s: %w", cmd.Name, err)
		}
	}

	result.Code.WriteString(mainEndTemplate)
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
	// Generate command content with command name context
	for _, content := range cmd.Body.Content {
		if err := e.generateCommandContentWithName(content, cmd.Name, result); err != nil {
			return fmt.Errorf("failed to generate content for command %s: %w", cmd.Name, err)
		}
	}

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

// generateCommandContentWithName generates Go code for command content with command name context
func (e *Engine) generateCommandContentWithName(content ast.CommandContent, commandName string, result *GenerationResult) error {
	switch c := content.(type) {
	case *ast.ShellContent:
		return e.generateShellContentWithName(c, commandName, result)
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

// generateShellContentWithName generates Go code for shell execution with command name
func (e *Engine) generateShellContentWithName(shell *ast.ShellContent, commandName string, result *GenerationResult) error {
	// Build a Go expression for the command string
	var goExprParts []string
	var displayParts []string

	for _, part := range shell.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			// Plain text - add as quoted string
			goExprParts = append(goExprParts, strconv.Quote(p.Text))
			displayParts = append(displayParts, p.Text)

		case *ast.FunctionDecorator:
			if p.Name == "var" {
				// For @var() decorators, reference the variable directly
				decorator, err := decorators.GetFunction(p.Name)
				if err != nil {
					return fmt.Errorf("function decorator %s not found: %w", p.Name, err)
				}

				varName, err := decorator.Generate(e.ctx, p.Args)
				if err != nil {
					return fmt.Errorf("failed to generate code for function decorator %s: %w", p.Name, err)
				}

				goExprParts = append(goExprParts, varName)
				displayParts = append(displayParts, `" + `+varName+` + "`)
			} else {
				// Other function decorators - expand at runtime
				decorator, err := decorators.GetFunction(p.Name)
				if err != nil {
					return fmt.Errorf("function decorator %s not found: %w", p.Name, err)
				}

				code, err := decorator.Generate(e.ctx, p.Args)
				if err != nil {
					return fmt.Errorf("failed to generate code for function decorator %s: %w", p.Name, err)
				}

				goExprParts = append(goExprParts, strconv.Quote(code))
				displayParts = append(displayParts, code)
			}

		default:
			return fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	// Build the Go expression for the command
	var cmdExpr string
	if len(goExprParts) == 1 {
		cmdExpr = goExprParts[0]
	} else {
		cmdExpr = strings.Join(goExprParts, " + ")
	}

	// Build display string for the shell comment
	displayCmd := strings.Join(displayParts, "")

	// Generate the command execution code
	result.Code.WriteString(fmt.Sprintf(`	// Command: %s
	func() {
		// Shell: %s
		cmdStr := %s
		fmt.Printf("Executing: %%s\n", cmdStr)
		// Execute command with context for cancellation
		cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Command failed: %%v\n", err)
		}
	}()

`, commandName, displayCmd, cmdExpr))

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

// collectUsedVariables scans the program and returns a set of variable names that are actually used
func (e *Engine) collectUsedVariables(program *ast.Program) map[string]bool {
	usedVars := make(map[string]bool)

	// Scan all commands for variable references
	for _, cmd := range program.Commands {
		e.collectUsedVariablesFromContent(cmd.Body.Content, usedVars)
	}

	return usedVars
}

// collectUsedVariablesFromContent recursively scans command content for variable references
func (e *Engine) collectUsedVariablesFromContent(content []ast.CommandContent, usedVars map[string]bool) {
	for _, item := range content {
		switch c := item.(type) {
		case *ast.ShellContent:
			// Check shell parts for @var() decorators
			for _, part := range c.Parts {
				if fnDec, ok := part.(*ast.FunctionDecorator); ok && fnDec.Name == "var" {
					// Extract variable name from @var(NAME) decorator
					if len(fnDec.Args) > 0 {
						if nameParam := ast.FindParameter(fnDec.Args, "name"); nameParam != nil {
							if ident, ok := nameParam.Value.(*ast.Identifier); ok {
								usedVars[ident.Name] = true
							}
						} else if len(fnDec.Args) > 0 {
							// Fallback to first parameter if no named parameter
							if ident, ok := fnDec.Args[0].Value.(*ast.Identifier); ok {
								usedVars[ident.Name] = true
							}
						}
					}
				}
			}
		case *ast.BlockDecorator:
			// Recursively scan block content
			e.collectUsedVariablesFromContent(c.Content, usedVars)
		case *ast.PatternDecorator:
			// Scan pattern branches
			for _, pattern := range c.Patterns {
				e.collectUsedVariablesFromContent(pattern.Commands, usedVars)
			}
		case *ast.FunctionDecorator:
			// Check if this is a @var decorator
			if c.Name == "var" && len(c.Args) > 0 {
				if nameParam := ast.FindParameter(c.Args, "name"); nameParam != nil {
					if ident, ok := nameParam.Value.(*ast.Identifier); ok {
						usedVars[ident.Name] = true
					}
				} else if len(c.Args) > 0 {
					// Fallback to first parameter if no named parameter
					if ident, ok := c.Args[0].Value.(*ast.Identifier); ok {
						usedVars[ident.Name] = true
					}
				}
			}
		}
	}
}

// generateVariables generates Go code for variable declarations, only for used variables
func (e *Engine) generateVariables(variables []ast.VariableDecl, program *ast.Program, result *GenerationResult) error {
	// Collect which variables are actually used
	usedVars := e.collectUsedVariables(program)

	generatedCount := 0
	for _, variable := range variables {
		if !usedVars[variable.Name] {
			// Warn about unused variable but don't generate it
			fmt.Fprintf(os.Stderr, "Warning: variable '%s' is declared but never used\n", variable.Name)
			continue
		}

		// Resolve the variable value to get the proper string representation
		value, err := e.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}

		// Use template to render variable declaration
		variableCode, err := renderVariable(variable.Name, value)
		if err != nil {
			return fmt.Errorf("failed to render variable %s: %w", variable.Name, err)
		}
		result.Code.WriteString(variableCode)
		generatedCount++
	}
	if generatedCount > 0 {
		result.Code.WriteString("\n")
	}
	return nil
}

// generateVarGroups generates Go code for variable groups
func (e *Engine) generateVarGroups(groups []ast.VarGroup, program *ast.Program, result *GenerationResult) error {
	for _, group := range groups {
		result.Code.WriteString("\t// Variable group\n")
		if err := e.generateVariables(group.Variables, program, result); err != nil {
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

// setupTemplateFunctions creates and sets template functions for code generation
func (e *Engine) setupTemplateFunctions() {
	funcMap := template.FuncMap{
		"executeCommand": e.generateCommandCode,
		"shellContext":   e.generateShellContextCode,
	}

	e.ctx.SetTemplateFunctions(funcMap)

	// Set up content executor for nested command execution in decorators
	e.ctx.SetContentExecutor(e.createContentExecutor())
}

// createContentExecutor creates a content executor function for decorators
func (e *Engine) createContentExecutor() func(ast.CommandContent) error {
	return func(content ast.CommandContent) error {
		// Create a dummy command result for the execution
		// In interpreter mode, we don't really use this result in decorators
		result := &CommandResult{
			Name:   "nested_content",
			Status: "success",
		}
		return e.executeCommandContent(content, result)
	}
}

// generateCommandCode generates Go code for any command content type
func (e *Engine) generateCommandCode(content ast.CommandContent) string {
	switch c := content.(type) {
	case *ast.ShellContent:
		return e.generateShellContextCode(c)
	case *ast.BlockDecorator:
		return e.generateNestedBlockDecorator(c)
	case *ast.PatternDecorator:
		return e.generateNestedPatternDecorator(c)
	case *ast.FunctionDecorator:
		return e.generateNestedFunctionDecorator(c)
	default:
		return fmt.Sprintf("// Unsupported command type: %T", content)
	}
}

// generateShellContextCode generates Go code for shell command execution
func (e *Engine) generateShellContextCode(shell *ast.ShellContent) string {
	return `// Execute shell command
var cmdBuilder strings.Builder
` + e.generateShellParts(shell.Parts) + `
cmdStr := strings.TrimSpace(cmdBuilder.String())
if cmdStr != "" {
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s (output: %s)", err, string(output))
	}
	if len(output) > 0 {
		fmt.Print(string(output))
	}
}
return nil`
}

// generateShellParts generates code for shell content parts
func (e *Engine) generateShellParts(parts []ast.ShellPart) string {
	var result strings.Builder

	for _, part := range parts {
		switch p := part.(type) {
		case *ast.TextPart:
			result.WriteString(fmt.Sprintf("cmdBuilder.WriteString(%q)\n", p.Text))
		case *ast.FunctionDecorator:
			// Call the function decorator's Generate method
			if decorator, err := decorators.GetFunction(p.Name); err == nil {
				if code, err := decorator.Generate(e.ctx, p.Args); err == nil {
					result.WriteString("cmdBuilder.WriteString(" + code + ")\n")
				}
			}
		}
	}

	return result.String()
}

// generateNestedBlockDecorator generates Go code for nested block decorators
func (e *Engine) generateNestedBlockDecorator(block *ast.BlockDecorator) string {
	// Look up the decorator in the registry
	decorator, err := decorators.GetBlock(block.Name)
	if err != nil {
		return fmt.Sprintf("// Block decorator @%s not found: %s", block.Name, err)
	}

	// Call the decorator's Generate method recursively
	code, err := decorator.Generate(e.ctx, block.Args, block.Content)
	if err != nil {
		return fmt.Sprintf("// Failed to generate nested block decorator @%s: %s", block.Name, err)
	}

	return code
}

// generateNestedPatternDecorator generates Go code for nested pattern decorators
func (e *Engine) generateNestedPatternDecorator(pattern *ast.PatternDecorator) string {
	// Look up the decorator in the registry
	decorator, err := decorators.GetPattern(pattern.Name)
	if err != nil {
		return fmt.Sprintf("// Pattern decorator @%s not found: %s", pattern.Name, err)
	}

	// Call the decorator's Generate method recursively
	code, err := decorator.Generate(e.ctx, pattern.Args, pattern.Patterns)
	if err != nil {
		return fmt.Sprintf("// Failed to generate nested pattern decorator @%s: %s", pattern.Name, err)
	}

	return code
}

// generateNestedFunctionDecorator generates Go code for nested function decorators
func (e *Engine) generateNestedFunctionDecorator(fn *ast.FunctionDecorator) string {
	// Look up the decorator in the registry
	decorator, err := decorators.GetFunction(fn.Name)
	if err != nil {
		return fmt.Sprintf("// Function decorator @%s not found: %s", fn.Name, err)
	}

	// Call the decorator's Generate method recursively
	code, err := decorator.Generate(e.ctx, fn.Args)
	if err != nil {
		return fmt.Sprintf("// Failed to generate nested function decorator @%s: %s", fn.Name, err)
	}

	return code
}
