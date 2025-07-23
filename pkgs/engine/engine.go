package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/execution"
)

// ProcessGroup represents a group of watch/stop commands for the same identifier
type ProcessGroup struct {
	Identifier   string
	WatchCommand *ast.CommandDecl
	StopCommand  *ast.CommandDecl
}

// CommandGroups holds the analyzed command structure
type CommandGroups struct {
	RegularCommands []*ast.CommandDecl
	ProcessGroups   []ProcessGroup
}

// Engine provides a unified AST walker for both interpreter and generator modes
type Engine struct {
	ctx       *execution.ExecutionContext
	goVersion string // Go version for generated code (e.g., "1.24")
}

// New creates a new execution engine with the execution context
func New(program *ast.Program) *Engine {
	ctx := execution.NewExecutionContext(context.Background(), program)
	engine := &Engine{
		ctx:       ctx,
		goVersion: "1.24", // Default Go version
	}
	// Set up dependency injection for decorators
	engine.setupDependencyInjection()
	return engine
}

// NewWithGoVersion creates a new execution engine with specified Go version
func NewWithGoVersion(program *ast.Program, goVersion string) *Engine {
	ctx := execution.NewExecutionContext(context.Background(), program)
	engine := &Engine{
		ctx:       ctx,
		goVersion: goVersion,
	}
	// Set up dependency injection for decorators
	engine.setupDependencyInjection()
	return engine
}

// ExecuteCommand executes a single command in interpreter mode
func (e *Engine) ExecuteCommand(command *ast.CommandDecl) (*CommandResult, error) {
	// Set execution mode to interpreter
	ctx := e.ctx.WithMode(execution.InterpreterMode)

	// Initialize variables if not already done
	if err := ctx.InitializeVariables(); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	cmdResult := &CommandResult{
		Name:   command.Name,
		Status: "success",
		Output: []string{},
		Error:  "",
	}

	// Execute the command content directly
	for _, content := range command.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			// Execute shell content using the execution context
			result := ctx.ExecuteShell(c)
			if result.Error != nil {
				cmdResult.Status = "failed"
				cmdResult.Error = result.Error.Error()
				return cmdResult, result.Error
			}
		default:
			err := fmt.Errorf("unsupported command content type in interpreter mode: %T", content)
			cmdResult.Status = "failed"
			cmdResult.Error = err.Error()
			return cmdResult, err
		}
	}

	return cmdResult, nil
}

// GenerateCode generates Go code for the entire program
func (e *Engine) GenerateCode(program *ast.Program) (*GenerationResult, error) {
	// Set execution mode to generator
	ctx := e.ctx.WithMode(execution.GeneratorMode)

	// Initialize variables
	if err := ctx.InitializeVariables(); err != nil {
		return nil, fmt.Errorf("failed to initialize variables: %w", err)
	}

	return e.generateProgram(program)
}

// setupDependencyInjection sets up dependency injection for the execution context
func (e *Engine) setupDependencyInjection() {
	// Set up function decorator lookup
	e.ctx.SetFunctionDecoratorLookup(func(name string) (execution.FunctionDecorator, bool) {
		decorator, exists := decorators.GetFunctionDecorator(name)
		return decorator, exists
	})

	// Command content execution is handled directly by the execution context

	// Set up template functions for code generation
	e.setupTemplateFunctions()
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
	result.AddStandardImport("os")

	// Analyze and group commands first
	commandGroups := e.analyzeCommands(program.Commands)

	// Add shell execution imports if the program has commands
	if len(program.Commands) > 0 {
		result.AddStandardImport("context")
		result.AddStandardImport("os/exec")
		result.AddThirdPartyImport("github.com/spf13/cobra")
	}

	// Add process management imports if we have process groups
	if len(commandGroups.ProcessGroups) > 0 {
		result.AddStandardImport("io/ioutil")
		result.AddStandardImport("path/filepath")
		result.AddStandardImport("strconv")
		result.AddStandardImport("syscall")
		result.AddStandardImport("time")
	}

	// Collect imports from decorators
	if err := e.collectDecoratorImports(program, result); err != nil {
		return nil, fmt.Errorf("failed to collect decorator imports: %w", err)
	}

	// Generate package declaration and imports
	result.Code.WriteString("package main\n\n")
	result.Code.WriteString("import (\n")
	for imp := range result.StandardImports {
		result.Code.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	for imp := range result.ThirdPartyImports {
		result.Code.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
	}
	result.Code.WriteString(")\n\n")

	// Generate the main function header
	result.Code.WriteString("func main() {\n")

	// Add context for shell execution
	if len(program.Commands) > 0 {
		result.Code.WriteString("\tctx := context.Background()\n\n")
	}

	// Generate variables inside main function
	if err := e.generateVariables(program.Variables, program, result); err != nil {
		return nil, fmt.Errorf("failed to generate variables: %w", err)
	}

	// Generate variable groups inside main function
	if err := e.generateVarGroups(program.VarGroups, program, result); err != nil {
		return nil, fmt.Errorf("failed to generate variable groups: %w", err)
	}
	result.Code.WriteString("\trootCmd := &cobra.Command{\n")
	result.Code.WriteString("\t\tUse:   \"cli\",\n")
	result.Code.WriteString("\t\tShort: \"Generated CLI from devcmd\",\n")
	result.Code.WriteString("\t}\n\n")

	// Generate regular commands
	for _, cmd := range commandGroups.RegularCommands {
		if err := e.generateCobraCommand(cmd, result); err != nil {
			return nil, fmt.Errorf("failed to generate command %s: %w", cmd.Name, err)
		}
	}

	// Generate process management commands (watch/stop groups)
	for _, group := range commandGroups.ProcessGroups {
		if err := e.generateProcessCommand(group, result); err != nil {
			return nil, fmt.Errorf("failed to generate process command %s: %w", group.Identifier, err)
		}
	}

	// Execute the root command
	result.Code.WriteString("\tif err := rootCmd.Execute(); err != nil {\n")
	result.Code.WriteString("\t\tfmt.Println(err, os.Stderr)\n")
	result.Code.WriteString("\t\tos.Exit(1)\n")
	result.Code.WriteString("\t}\n")
	result.Code.WriteString("}\n")

	return result, nil
}

// generateCobraCommand generates a Cobra command from an AST command declaration
func (e *Engine) generateCobraCommand(cmd *ast.CommandDecl, result *GenerationResult) error {
	// Generate command variable name based on command identifier
	var cmdVarName string
	if strings.HasPrefix(cmd.Name, "watch ") {
		// For watch commands: "watch dev" -> "watchDev"
		ident := strings.TrimPrefix(cmd.Name, "watch ")
		cmdVarName = "watch" + capitalizeFirst(toCamelCase(ident))
	} else if strings.HasPrefix(cmd.Name, "stop ") {
		// For stop commands: "stop dev" -> "stopDev"
		ident := strings.TrimPrefix(cmd.Name, "stop ")
		cmdVarName = "stop" + capitalizeFirst(toCamelCase(ident))
	} else {
		// Regular commands: "build" -> "build", "test-all" -> "testAll"
		cmdVarName = toCamelCase(cmd.Name)
	}

	// Generate command function
	result.Code.WriteString(fmt.Sprintf("\t%s := func(cmd *cobra.Command, args []string) {\n", cmdVarName))

	// Generate command body using execution context
	generatorCtx := e.ctx.WithMode(execution.GeneratorMode)
	for _, content := range cmd.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			// Use execution context to generate proper shell code
			shellResult := generatorCtx.ExecuteShell(c)
			if shellResult.Error != nil {
				return fmt.Errorf("failed to generate shell code: %w", shellResult.Error)
			}
			if code, ok := shellResult.Data.(string); ok {
				result.Code.WriteString(code)
			}
		case *ast.BlockDecorator:
			// Add decorator marker comment
			result.Code.WriteString(fmt.Sprintf("\t\t// Block decorator: @%s\n", c.Name))

			// Generate block decorator code using decorator registry
			decorator, err := decorators.GetBlock(c.Name)
			if err != nil {
				return fmt.Errorf("block decorator @%s not found: %w", c.Name, err)
			}
			decoratorResult := decorator.Execute(generatorCtx, c.Args, c.Content)
			if decoratorResult.Error != nil {
				return fmt.Errorf("failed to generate block decorator code: %w", decoratorResult.Error)
			}
			if code, ok := decoratorResult.Data.(string); ok {
				result.Code.WriteString(code)
			}
		case *ast.PatternDecorator:
			// Add decorator marker comment
			result.Code.WriteString(fmt.Sprintf("\t\t// Pattern decorator: @%s\n", c.Name))

			// Generate pattern decorator code using decorator registry
			decorator, err := decorators.GetPattern(c.Name)
			if err != nil {
				return fmt.Errorf("pattern decorator @%s not found: %w", c.Name, err)
			}
			decoratorResult := decorator.Execute(generatorCtx, c.Args, c.Patterns)
			if decoratorResult.Error != nil {
				return fmt.Errorf("failed to generate pattern decorator code: %w", decoratorResult.Error)
			}
			if code, ok := decoratorResult.Data.(string); ok {
				result.Code.WriteString(code)
			}
		default:
			result.Code.WriteString("\t\t// Unknown command content\n")
		}
	}

	result.Code.WriteString("\t}\n\n")

	// Create the Cobra command variable
	cmdStructName := cmdVarName + "Cmd"
	result.Code.WriteString(fmt.Sprintf("\t%s := &cobra.Command{\n", cmdStructName))
	result.Code.WriteString(fmt.Sprintf("\t\tUse:   \"%s\",\n", cmd.Name))
	result.Code.WriteString(fmt.Sprintf("\t\tRun:   %s,\n", cmdVarName))
	result.Code.WriteString("\t}\n")
	result.Code.WriteString(fmt.Sprintf("\trootCmd.AddCommand(%s)\n\n", cmdStructName))

	return nil
}

// processVariablesIntoContext processes variables and variable groups into the execution context
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
				return fmt.Errorf("failed to resolve variable %s in group: %w", variable.Name, err)
			}
			e.ctx.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// resolveVariableValue resolves a variable value from an expression
func (e *Engine) resolveVariableValue(expr ast.Expression) (string, error) {
	switch v := expr.(type) {
	case *ast.StringLiteral:
		return v.Value, nil
	case *ast.NumberLiteral:
		return v.Value, nil
	case *ast.BooleanLiteral:
		return fmt.Sprintf("%t", v.Value), nil
	case *ast.Identifier:
		// Handle variable references
		if value, exists := e.ctx.GetVariable(v.Name); exists {
			return value, nil
		}
		return "", fmt.Errorf("undefined variable: %s", v.Name)
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// collectDecoratorImports collects import requirements from all decorators used in the program
func (e *Engine) collectDecoratorImports(program *ast.Program, result *GenerationResult) error {
	// Collect from commands
	for _, cmd := range program.Commands {
		if err := e.collectDecoratorImportsFromContent(cmd.Body.Content, result); err != nil {
			return err
		}
	}
	return nil
}

// collectDecoratorImportsFromContent recursively collects decorator imports from command content
func (e *Engine) collectDecoratorImportsFromContent(content []ast.CommandContent, result *GenerationResult) error {
	for _, item := range content {
		switch c := item.(type) {
		case *ast.ShellContent:
			// Collect from function decorators in shell parts
			for _, part := range c.Parts {
				if fn, ok := part.(*ast.FunctionDecorator); ok {
					if err := e.addDecoratorImports("function", fn.Name, result); err != nil {
						return err
					}
				}
			}
		case *ast.BlockDecorator:
			if err := e.addDecoratorImports("block", c.Name, result); err != nil {
				return err
			}
			// Recursively collect from block content
			if err := e.collectDecoratorImportsFromContent(c.Content, result); err != nil {
				return err
			}
		case *ast.PatternDecorator:
			if err := e.addDecoratorImports("pattern", c.Name, result); err != nil {
				return err
			}
			// Recursively collect from pattern branches
			for _, pattern := range c.Patterns {
				if err := e.collectDecoratorImportsFromContent(pattern.Commands, result); err != nil {
					return err
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

	// Get import requirements if the decorator supports it
	if importProvider, ok := decorator.(interface {
		ImportRequirements() decorators.ImportRequirement
	}); ok {
		requirements := importProvider.ImportRequirements()

		// Add standard library imports
		for _, pkg := range requirements.StandardLibrary {
			result.AddStandardImport(pkg)
		}

		// Add third-party imports
		for _, pkg := range requirements.ThirdParty {
			result.AddThirdPartyImport(pkg)
		}

		// Add Go modules
		for module, version := range requirements.GoModules {
			result.AddGoModule(module, version)
		}
	}

	return nil
}

// generateVariables generates Go code for variable declarations
func (e *Engine) generateVariables(variables []ast.VariableDecl, program *ast.Program, result *GenerationResult) error {
	if len(variables) == 0 {
		return nil
	}

	// Only generate variables that are actually used in commands
	usedVars := e.findUsedVariables(program)
	hasUsedVars := false

	for _, variable := range variables {
		if usedVars[variable.Name] {
			if !hasUsedVars {
				result.Code.WriteString("\t// Variables\n")
				hasUsedVars = true
			}
			value, err := e.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
			}
			result.Code.WriteString(fmt.Sprintf("\t%s := %q\n", variable.Name, value))
		}
	}

	if hasUsedVars {
		result.Code.WriteString("\n")
	}

	return nil
}

// generateVarGroups generates Go code for variable group declarations
func (e *Engine) generateVarGroups(groups []ast.VarGroup, program *ast.Program, result *GenerationResult) error {
	if len(groups) == 0 {
		return nil
	}

	// Only generate variables that are actually used in commands
	usedVars := e.findUsedVariables(program)

	for _, group := range groups {
		hasUsedVars := false
		for _, variable := range group.Variables {
			if usedVars[variable.Name] {
				if !hasUsedVars {
					result.Code.WriteString("\t// Variable group\n")
					hasUsedVars = true
				}
				value, err := e.resolveVariableValue(variable.Value)
				if err != nil {
					return fmt.Errorf("failed to resolve variable %s in group: %w", variable.Name, err)
				}
				result.Code.WriteString(fmt.Sprintf("\t%s := %q\n", variable.Name, value))
			}
		}
		if hasUsedVars {
			result.Code.WriteString("\n")
		}
	}

	return nil
}

// findUsedVariables scans the program to find which variables are actually used
func (e *Engine) findUsedVariables(program *ast.Program) map[string]bool {
	used := make(map[string]bool)

	// Scan commands for variable usage
	for _, cmd := range program.Commands {
		e.scanCommandContentForVariables(cmd.Body.Content, used)
	}

	// DEBUG: For now, mark all variables as used to avoid compilation errors
	// This is a temporary fix until proper shell command generation is implemented
	for _, variable := range program.Variables {
		used[variable.Name] = true
	}
	for _, group := range program.VarGroups {
		for _, variable := range group.Variables {
			used[variable.Name] = true
		}
	}

	return used
}

// scanCommandContentForVariables recursively scans command content for variable usage
func (e *Engine) scanCommandContentForVariables(content []ast.CommandContent, used map[string]bool) {
	for _, item := range content {
		switch c := item.(type) {
		case *ast.ShellContent:
			// Scan shell parts for function decorators (like @var)
			for _, part := range c.Parts {
				if fn, ok := part.(*ast.FunctionDecorator); ok && fn.Name == "var" {
					// Extract variable name from @var() decorator
					if len(fn.Args) > 0 {
						if param := fn.Args[0]; param.Name == "" || param.Name == "name" {
							if identifier, ok := param.Value.(*ast.Identifier); ok {
								used[identifier.Name] = true
							}
						}
					}
				}
			}
		case *ast.BlockDecorator:
			// Recursively scan block content
			e.scanCommandContentForVariables(c.Content, used)
		case *ast.PatternDecorator:
			// Recursively scan pattern branches
			for _, pattern := range c.Patterns {
				e.scanCommandContentForVariables(pattern.Commands, used)
			}
		}
	}
}

// toCamelCase converts a command name to camelCase for variable naming
// Examples: "build" -> "build", "test-all" -> "testAll", "dev_flow" -> "devFlow"
func toCamelCase(name string) string {
	// Handle different separators: hyphens, underscores, and spaces
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})

	if len(parts) == 0 {
		return name
	}

	// First part stays lowercase, subsequent parts get title case
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += capitalizeFirst(parts[i])
	}

	return result
}

// capitalizeFirst capitalizes the first letter of a string (replacement for deprecated strings.Title)
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// analyzeCommands groups watch/stop commands and separates regular commands
func (e *Engine) analyzeCommands(commands []ast.CommandDecl) CommandGroups {
	groups := CommandGroups{
		RegularCommands: []*ast.CommandDecl{},
		ProcessGroups:   []ProcessGroup{},
	}

	// Track watch/stop commands by identifier
	processMap := make(map[string]ProcessGroup)

	for i, cmd := range commands {
		switch cmd.Type {
		case ast.WatchCommand:
			// Watch command - use the name as identifier
			identifier := cmd.Name
			group := processMap[identifier]
			group.Identifier = identifier
			group.WatchCommand = &commands[i]
			processMap[identifier] = group
		case ast.StopCommand:
			// Stop command - use the name as identifier
			identifier := cmd.Name
			group := processMap[identifier]
			group.Identifier = identifier
			group.StopCommand = &commands[i]
			processMap[identifier] = group
		default:
			// Regular command
			groups.RegularCommands = append(groups.RegularCommands, &commands[i])
		}
	}

	// Convert map to slice
	for _, group := range processMap {
		groups.ProcessGroups = append(groups.ProcessGroups, group)
	}

	return groups
}

// generateProcessCommand generates a process management command with run/stop/status/logs subcommands
func (e *Engine) generateProcessCommand(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	// Generate the main run function first (will be used by main command and run subcommand)
	var runFunctionName string
	if group.WatchCommand != nil {
		runFunctionName = fmt.Sprintf("%sRun", identifier)
		if err := e.generateProcessRunFunction(group, result); err != nil {
			return fmt.Errorf("failed to generate run function: %w", err)
		}
	}

	// Generate the main process command with default run behavior
	result.Code.WriteString(fmt.Sprintf("\t// Process management for %s\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t%sCmd := &cobra.Command{\n", identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tUse:   \"%s\",\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tShort: \"Manage %s process\",\n", group.Identifier))
	if runFunctionName != "" {
		result.Code.WriteString(fmt.Sprintf("\t\tRun:   %s, // Default action is to run\n", runFunctionName))
	}
	result.Code.WriteString("\t}\n\n")

	// Generate run subcommand (from watch command)
	if group.WatchCommand != nil {
		if err := e.generateProcessRunCommand(group, result); err != nil {
			return fmt.Errorf("failed to generate run command: %w", err)
		}
	}

	// Generate stop subcommand (from stop command or default)
	if err := e.generateProcessStopCommand(group, result); err != nil {
		return fmt.Errorf("failed to generate stop command: %w", err)
	}

	// Generate status subcommand
	if err := e.generateProcessStatusCommand(group, result); err != nil {
		return fmt.Errorf("failed to generate status command: %w", err)
	}

	// Generate logs subcommand
	if err := e.generateProcessLogsCommand(group, result); err != nil {
		return fmt.Errorf("failed to generate logs command: %w", err)
	}

	// Add the main command to root
	result.Code.WriteString(fmt.Sprintf("\trootCmd.AddCommand(%sCmd)\n\n", identifier))

	return nil
}

// generateProcessRunFunction generates the run function that can be shared by main command and run subcommand
func (e *Engine) generateProcessRunFunction(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	result.Code.WriteString(fmt.Sprintf("\t// %s run function\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t%sRun := func(cmd *cobra.Command, args []string) {\n", identifier))

	// Generate the command content from watch command
	generatorCtx := e.ctx.WithMode(execution.GeneratorMode)
	for _, content := range group.WatchCommand.Body.Content {
		switch c := content.(type) {
		case *ast.ShellContent:
			shellResult := generatorCtx.ExecuteShell(c)
			if shellResult.Error != nil {
				return fmt.Errorf("failed to generate shell code: %w", shellResult.Error)
			}
			if code, ok := shellResult.Data.(string); ok {
				result.Code.WriteString(code)
			}
		}
	}

	result.Code.WriteString("\t}\n\n")
	return nil
}

// generateProcessRunCommand generates the 'run' subcommand for process management
func (e *Engine) generateProcessRunCommand(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	// Create the run subcommand (reuses the function generated earlier)
	result.Code.WriteString(fmt.Sprintf("\t%sRunCmd := &cobra.Command{\n", identifier))
	result.Code.WriteString("\t\tUse:   \"run\",\n")
	result.Code.WriteString(fmt.Sprintf("\t\tShort: \"Start %s process (explicit)\",\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tRun:   %sRun,\n", identifier))
	result.Code.WriteString("\t}\n")
	result.Code.WriteString(fmt.Sprintf("\t%sCmd.AddCommand(%sRunCmd)\n\n", identifier, identifier))

	return nil
}

// generateProcessStopCommand generates the 'stop' subcommand
func (e *Engine) generateProcessStopCommand(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	result.Code.WriteString(fmt.Sprintf("\t// %s stop command\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t%sStop := func(cmd *cobra.Command, args []string) {\n", identifier))

	if group.StopCommand != nil {
		// Use custom stop command if provided
		generatorCtx := e.ctx.WithMode(execution.GeneratorMode)
		for _, content := range group.StopCommand.Body.Content {
			switch c := content.(type) {
			case *ast.ShellContent:
				shellResult := generatorCtx.ExecuteShell(c)
				if shellResult.Error != nil {
					return fmt.Errorf("failed to generate shell code: %w", shellResult.Error)
				}
				if code, ok := shellResult.Data.(string); ok {
					result.Code.WriteString(code)
				}
			}
		}
	} else {
		// Default stop behavior - kill process by name
		result.Code.WriteString("\t\tfunc() {\n")
		result.Code.WriteString(fmt.Sprintf("\t\t\tcmdStr := \"pkill -f '%s'\"\n", group.Identifier))
		result.Code.WriteString("\t\t\texecCmd := exec.CommandContext(ctx, \"sh\", \"-c\", cmdStr)\n")
		result.Code.WriteString("\t\t\texecCmd.Stdout = os.Stdout\n")
		result.Code.WriteString("\t\t\texecCmd.Stderr = os.Stderr\n")
		result.Code.WriteString("\t\t\tif err := execCmd.Run(); err != nil {\n")
		result.Code.WriteString("\t\t\t\tfmt.Fprintf(os.Stderr, \"Stop command failed: %v\\n\", err)\n")
		result.Code.WriteString("\t\t\t}\n")
		result.Code.WriteString("\t\t}()\n")
	}

	result.Code.WriteString("\t}\n\n")

	// Create the stop subcommand
	result.Code.WriteString(fmt.Sprintf("\t%sStopCmd := &cobra.Command{\n", identifier))
	result.Code.WriteString("\t\tUse:   \"stop\",\n")
	result.Code.WriteString(fmt.Sprintf("\t\tShort: \"Stop %s process\",\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tRun:   %sStop,\n", identifier))
	result.Code.WriteString("\t}\n")
	result.Code.WriteString(fmt.Sprintf("\t%sCmd.AddCommand(%sStopCmd)\n\n", identifier, identifier))

	return nil
}

// generateProcessStatusCommand generates the 'status' subcommand
func (e *Engine) generateProcessStatusCommand(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	result.Code.WriteString(fmt.Sprintf("\t// %s status command\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t%sStatus := func(cmd *cobra.Command, args []string) {\n", identifier))
	result.Code.WriteString("\t\tfunc() {\n")
	result.Code.WriteString(fmt.Sprintf("\t\t\tcmdStr := \"pgrep -f '%s' || echo 'Process not running'\"\n", group.Identifier))
	result.Code.WriteString("\t\t\texecCmd := exec.CommandContext(ctx, \"sh\", \"-c\", cmdStr)\n")
	result.Code.WriteString("\t\t\texecCmd.Stdout = os.Stdout\n")
	result.Code.WriteString("\t\t\texecCmd.Stderr = os.Stderr\n")
	result.Code.WriteString("\t\t\tif err := execCmd.Run(); err != nil {\n")
	result.Code.WriteString("\t\t\t\tfmt.Fprintf(os.Stderr, \"Status command failed: %v\\n\", err)\n")
	result.Code.WriteString("\t\t\t}\n")
	result.Code.WriteString("\t\t}()\n")
	result.Code.WriteString("\t}\n\n")

	// Create the status subcommand
	result.Code.WriteString(fmt.Sprintf("\t%sStatusCmd := &cobra.Command{\n", identifier))
	result.Code.WriteString("\t\tUse:   \"status\",\n")
	result.Code.WriteString(fmt.Sprintf("\t\tShort: \"Show %s process status\",\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tRun:   %sStatus,\n", identifier))
	result.Code.WriteString("\t}\n")
	result.Code.WriteString(fmt.Sprintf("\t%sCmd.AddCommand(%sStatusCmd)\n\n", identifier, identifier))

	return nil
}

// generateProcessLogsCommand generates the 'logs' subcommand
func (e *Engine) generateProcessLogsCommand(group ProcessGroup, result *GenerationResult) error {
	identifier := toCamelCase(group.Identifier)

	result.Code.WriteString(fmt.Sprintf("\t// %s logs command\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t%sLogs := func(cmd *cobra.Command, args []string) {\n", identifier))
	result.Code.WriteString("\t\tfunc() {\n")
	result.Code.WriteString(fmt.Sprintf("\t\t\tcmdStr := \"echo 'Logs for %s process'\"\n", group.Identifier))
	result.Code.WriteString("\t\t\texecCmd := exec.CommandContext(ctx, \"sh\", \"-c\", cmdStr)\n")
	result.Code.WriteString("\t\t\texecCmd.Stdout = os.Stdout\n")
	result.Code.WriteString("\t\t\texecCmd.Stderr = os.Stderr\n")
	result.Code.WriteString("\t\t\tif err := execCmd.Run(); err != nil {\n")
	result.Code.WriteString("\t\t\t\tfmt.Fprintf(os.Stderr, \"Logs command failed: %v\\n\", err)\n")
	result.Code.WriteString("\t\t\t}\n")
	result.Code.WriteString("\t\t}()\n")
	result.Code.WriteString("\t}\n\n")

	// Create the logs subcommand
	result.Code.WriteString(fmt.Sprintf("\t%sLogsCmd := &cobra.Command{\n", identifier))
	result.Code.WriteString("\t\tUse:   \"logs\",\n")
	result.Code.WriteString(fmt.Sprintf("\t\tShort: \"Show %s process logs\",\n", group.Identifier))
	result.Code.WriteString(fmt.Sprintf("\t\tRun:   %sLogs,\n", identifier))
	result.Code.WriteString("\t}\n")
	result.Code.WriteString(fmt.Sprintf("\t%sCmd.AddCommand(%sLogsCmd)\n\n", identifier, identifier))

	return nil
}

// setupTemplateFunctions sets up template functions for the execution context
func (e *Engine) setupTemplateFunctions() {
	templateFuncs := map[string]interface{}{
		"executeCommand": func(content ast.CommandContent) string {
			// Generate code for executing a command content within template context
			generatorCtx := e.ctx.WithMode(execution.GeneratorMode)
			switch c := content.(type) {
			case *ast.ShellContent:
				result := generatorCtx.ExecuteShell(c)
				if result.Error != nil {
					return fmt.Sprintf("return fmt.Errorf(\"shell generation error: %v\")", result.Error)
				}
				if code, ok := result.Data.(string); ok {
					// Modify the shell code to return an error instead of calling os.Exit
					modifiedCode := strings.ReplaceAll(code, "os.Exit(1)", "return err")
					// Remove the function wrapper since we're inside a template function
					modifiedCode = strings.Replace(modifiedCode, "func() {", "", 1)
					modifiedCode = strings.Replace(modifiedCode, "}()", "return nil", 1)
					return modifiedCode
				}
				return "return fmt.Errorf(\"failed to generate shell code\")"
			default:
				return fmt.Sprintf("return fmt.Errorf(\"unsupported command content type: %T\")", content)
			}
		},
	}
	e.ctx.SetTemplateFunctions(templateFuncs)
}
