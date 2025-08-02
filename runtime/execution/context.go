package execution

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
)

// FunctionDecoratorType represents the behavior type of function decorators
type FunctionDecoratorType int

const (
	// SubstitutionDecorator decorators replace themselves with values in shell commands
	// Examples: @var, @env - return string values that get substituted
	SubstitutionDecorator FunctionDecoratorType = iota
	
	// ExecutionDecorator decorators execute actions when encountered
	// Examples: @cmd - execute commands or other actions
	ExecutionDecorator
)

// ChainElement represents an element in an ActionDecorator command chain
type ChainElement struct {
	Type           string // "action", "operator", "text"
	ActionName     string // For ActionDecorator
	ActionArgs     []ast.NamedParameter // For ActionDecorator
	Operator       string // "&&", "||", "|", ">>"
	Text           string // For text parts
	VariableName   string // Generated variable name
	IsPipeTarget   bool   // True if this element receives piped input
	IsFileTarget   bool   // True if this element is a file for >> operation
}

// ChainOperator represents the type of chaining operator
type ChainOperator string

const (
	AndOperator  ChainOperator = "&&" // Execute next if current succeeds
	OrOperator   ChainOperator = "||" // Execute next if current fails  
	PipeOperator ChainOperator = "|"  // Pipe stdout to next command
	AppendOperator ChainOperator = ">>" // Append stdout to file
)

// ExecutionContext provides execution context for decorators and implements context.Context
type ExecutionContext struct {
	context.Context

	// Core data
	Program   *ast.Program
	Variables map[string]string // Resolved variable values
	env       map[string]string // Immutable environment variables captured at command start

	// Execution state
	WorkingDir string
	Debug      bool
	DryRun     bool

	// Execution mode for the unified pattern
	mode ExecutionMode

	// Current command name for generating meaningful variable names
	currentCommand string

	// Shell execution counter for unique variable naming
	shellCounter int

	// Template functions for code generation (populated by engine)
	templateFunctions template.FuncMap

	// Command content executor for nested command execution (populated by engine)
	contentExecutor func(ast.CommandContent) error

	// Value decorator lookup (populated by engine to avoid circular imports)
	valueDecoratorLookup func(name string) (interface{}, bool)
	
	// Action decorator lookup (populated by engine to avoid circular imports)
	actionDecoratorLookup func(name string) (interface{}, bool)

	// Command executor for executing full commands (populated by engine)
	commandExecutor func(*ast.CommandDecl) error

	// Command plan generator for generating command plans (populated by engine)
	commandPlanGenerator func(*ast.CommandDecl) (*ExecutionResult, error)
}

// NewExecutionContext creates a new execution context and captures the environment immutably
func NewExecutionContext(parent context.Context, program *ast.Program) *ExecutionContext {
	if parent == nil {
		parent = context.Background()
	}

	// Initialize working directory to current directory
	workingDir, err := os.Getwd()
	if err != nil {
		// Fallback to empty string if we can't get current directory
		workingDir = ""
	}

	// Capture environment variables immutably at command start for security
	// This prevents manipulation during execution and ensures consistent environment state
	capturedEnv := make(map[string]string)
	for _, envVar := range os.Environ() {
		if idx := strings.Index(envVar, "="); idx > 0 {
			key := envVar[:idx]
			value := envVar[idx+1:]
			capturedEnv[key] = value
		}
	}

	return &ExecutionContext{
		Context:           parent,
		Program:           program,
		Variables:         make(map[string]string),
		env:               capturedEnv, // Immutable captured environment
		WorkingDir:        workingDir,
		Debug:             false,
		DryRun:            false,
		mode:              InterpreterMode, // Default mode
		templateFunctions: make(template.FuncMap),
	}
}

// WithTimeout creates a new context with timeout
func (c *ExecutionContext) WithTimeout(timeout time.Duration) (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Context, timeout)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// WithCancel creates a new context with cancellation
func (c *ExecutionContext) WithCancel() (*ExecutionContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.Context)
	newCtx := *c
	newCtx.Context = ctx
	return &newCtx, cancel
}

// WithMode creates a new context with the specified execution mode
func (c *ExecutionContext) WithMode(mode ExecutionMode) *ExecutionContext {
	newCtx := *c
	newCtx.mode = mode
	return &newCtx
}

// Mode returns the current execution mode
func (c *ExecutionContext) Mode() ExecutionMode {
	return c.mode
}

// WithCurrentCommand creates a new context with the specified current command name
func (c *ExecutionContext) WithCurrentCommand(commandName string) *ExecutionContext {
	newCtx := *c
	newCtx.currentCommand = commandName
	return &newCtx
}

// Child creates a child context that inherits from the parent but can be modified independently
// This is used by block and pattern decorators to create isolated execution environments
func (c *ExecutionContext) Child() *ExecutionContext {
	childCtx := &ExecutionContext{
		Context:   c.Context,
		Program:   c.Program,
		Variables: make(map[string]string),
		env:       c.env, // Share the same immutable environment reference
		
		// Copy execution state
		WorkingDir:    c.WorkingDir,
		Debug:         c.Debug,
		DryRun:        c.DryRun,
		mode:          c.mode,
		currentCommand: c.currentCommand,
		
		// Copy function references
		templateFunctions:         c.templateFunctions,
		contentExecutor:           c.contentExecutor,
		valueDecoratorLookup:      c.valueDecoratorLookup,
		actionDecoratorLookup:     c.actionDecoratorLookup,
		commandExecutor:           c.commandExecutor,
		commandPlanGenerator:      c.commandPlanGenerator,
	}
	
	// Copy variables (child gets its own copy)
	for name, value := range c.Variables {
		childCtx.Variables[name] = value
	}
	
	return childCtx
}

// WithWorkingDir creates a new context with the specified working directory
func (c *ExecutionContext) WithWorkingDir(workingDir string) *ExecutionContext {
	newCtx := *c
	newCtx.WorkingDir = workingDir
	return &newCtx
}

// ExecuteShell executes shell content in the current mode
func (c *ExecutionContext) ExecuteShell(content *ast.ShellContent) *ExecutionResult {
	switch c.mode {
	case InterpreterMode:
		return c.executeShellInterpreter(content)
	case GeneratorMode:
		return c.executeShellGenerator(content)
	case PlanMode:
		return c.executeShellPlan(content)
	default:
		return &ExecutionResult{
			Mode:  c.mode,
			Data:  nil,
			Error: fmt.Errorf("unsupported execution mode: %v", c.mode),
		}
	}
}

// executeShellInterpreter executes shell content directly
func (c *ExecutionContext) executeShellInterpreter(content *ast.ShellContent) *ExecutionResult {
	// Compose the command string from parts
	cmdStr, err := c.composeShellCommand(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  InterpreterMode,
			Data:  nil,
			Error: fmt.Errorf("failed to compose shell command: %w", err),
		}
	}

	// Execute the command
	cmd := exec.CommandContext(c.Context, "sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.WorkingDir != "" {
		cmd.Dir = c.WorkingDir
	}

	err = cmd.Run()
	return &ExecutionResult{
		Mode:  InterpreterMode,
		Data:  nil,
		Error: err,
	}
}

// executeShellGenerator generates Go code for shell execution using unified decorator model
func (c *ExecutionContext) executeShellGenerator(content *ast.ShellContent) *ExecutionResult {
	// Use the unified shell code builder
	shellBuilder := NewShellCodeBuilder(c)
	code, err := shellBuilder.GenerateShellCode(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to generate shell execution code: %w", err),
		}
	}

	return &ExecutionResult{
		Mode:  GeneratorMode,
		Data:  code,
		Error: nil,
	}
}

// generateDirectActionCode generates Go code for ActionDecorator direct execution with CommandResult chaining
func (c *ExecutionContext) generateDirectActionCode(content *ast.ShellContent) *ExecutionResult {
	// Use the unified shell code builder
	shellBuilder := NewShellCodeBuilder(c)
	code, err := shellBuilder.GenerateDirectActionTemplate(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  GeneratorMode,
			Data:  "",
			Error: fmt.Errorf("failed to generate action chain code: %w", err),
		}
	}

	return &ExecutionResult{
		Mode:  GeneratorMode,
		Data:  code,
		Error: nil,
	}
}

// parseActionDecoratorChain parses shell content into a chain of commands and operators
func (c *ExecutionContext) parseActionDecoratorChain(content *ast.ShellContent) ([]ChainElement, error) {
	var chain []ChainElement
	var currentIndex int

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.ActionDecorator:
			element := ChainElement{
				Type:         "action",
				ActionName:   p.Name,
				ActionArgs:   p.Args,
				VariableName: fmt.Sprintf("%sResult%d", c.getBaseName(), currentIndex),
			}
			chain = append(chain, element)
			currentIndex++

		case *ast.TextPart:
			text := strings.TrimSpace(p.Text)
			if text == "" {
				continue
			}

			// Check if this is a chain operator
			switch text {
			case "&&", "||", "|", ">>":
				if len(chain) == 0 {
					return nil, fmt.Errorf("operator %s cannot be at the beginning of chain", text)
				}
				element := ChainElement{
					Type:     "operator",
					Operator: text,
				}
				chain = append(chain, element)
			default:
				// Regular text - treat as shell command
				element := ChainElement{
					Type:         "text",
					Text:         text,
					VariableName: fmt.Sprintf("%sShell%d", c.getBaseName(), currentIndex),
				}
				chain = append(chain, element)
				currentIndex++
			}

		case *ast.ValueDecorator:
			// ValueDecorators in ActionDecorator context should be resolved to values
			if value, exists := c.GetVariable(p.Name); exists {
				element := ChainElement{
					Type: "text",
					Text: value,
				}
				chain = append(chain, element)
			} else {
				return nil, fmt.Errorf("undefined variable in ActionDecorator chain: %s", p.Name)
			}
		}
	}

	return chain, nil
}

// getBaseName returns the base name for variable generation
func (c *ExecutionContext) getBaseName() string {
	if c.currentCommand != "" {
		return strings.Title(c.currentCommand)
	}
	return "Action"
}

// Helper function to format parameters for Go code generation
func formatParams(params []ast.NamedParameter) string {
	if len(params) == 0 {
		return "nil"
	}
	// For now, return simple representation - this needs to be expanded
	return "[]ast.NamedParameter{}"
}

// executeShellPlan creates a plan element for shell execution
func (c *ExecutionContext) executeShellPlan(content *ast.ShellContent) *ExecutionResult {
	// CRITICAL FIX: Don't use InterpreterMode for plan generation as it executes ActionDecorators
	// Instead, create a plan-safe command string that doesn't execute anything
	cmdStr, err := c.composeShellCommandForPlan(content)
	if err != nil {
		return &ExecutionResult{
			Mode:  PlanMode,
			Data:  nil,
			Error: fmt.Errorf("failed to compose shell command for plan: %w", err),
		}
	}

	// For now, return a simple plan representation
	// TODO: Replace with proper plan.PlanElement when we move plan package
	planData := map[string]interface{}{
		"type":        "shell",
		"command":     cmdStr,
		"description": "Execute shell command: " + cmdStr,
	}

	return &ExecutionResult{
		Mode:  PlanMode,
		Data:  planData,
		Error: nil,
	}
}

// composeShellCommand composes the shell command string from AST parts
func (c *ExecutionContext) composeShellCommand(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		result, err := c.processShellPart(part)
		if err != nil {
			return "", err
		}

		if value, ok := result.(string); ok {
			parts = append(parts, value)
		} else {
			return "", fmt.Errorf("shell part returned non-string result: %T", result)
		}
	}

	return strings.Join(parts, ""), nil
}

// composeShellCommandForPlan composes shell command for plan display without executing ActionDecorators
func (c *ExecutionContext) composeShellCommandForPlan(content *ast.ShellContent) (string, error) {
	var parts []string

	for _, part := range content.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			parts = append(parts, p.Text)
		case *ast.ValueDecorator:
			// For plan mode, resolve variables but don't execute anything
			// Use the same logic as the value decorator but without execution
			valueDecorator, exists := c.valueDecoratorLookup(p.Name)
			if exists {
				// Create a temporary interpreter context to resolve the value for display
				tempCtx := c.Child().WithMode(InterpreterMode)
				result := valueDecorator.(interface{ Expand(*ExecutionContext, []ast.NamedParameter) *ExecutionResult }).Expand(tempCtx, p.Args)
				if result.Error == nil {
					if value, ok := result.Data.(string); ok {
						parts = append(parts, value)
					} else {
						parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
					}
				} else {
					parts = append(parts, fmt.Sprintf("@%s(error)", p.Name))
				}
			} else {
				parts = append(parts, fmt.Sprintf("@%s(undefined)", p.Name))
			}
		case *ast.ActionDecorator:
			// For plan mode, just show the decorator syntax without executing
			parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
		default:
			return "", fmt.Errorf("unsupported shell part type for plan: %T", part)
		}
	}

	return strings.Join(parts, ""), nil
}

// processShellPart processes any shell part (text, value decorator, action decorator) 
// based on the current execution mode, returning the appropriate result
func (c *ExecutionContext) processShellPart(part ast.ShellPart) (interface{}, error) {
	switch p := part.(type) {
	case *ast.TextPart:
		switch c.mode {
		case InterpreterMode:
			return p.Text, nil
		case GeneratorMode:
			return strconv.Quote(p.Text), nil
		case PlanMode:
			return p.Text, nil
		default:
			return nil, fmt.Errorf("unsupported mode: %v", c.mode)
		}

	case *ast.ValueDecorator:
		return c.processValueDecoratorUnified(p)

	case *ast.ActionDecorator:
		return c.processActionDecoratorUnified(p)

	default:
		return nil, fmt.Errorf("unsupported shell part type: %T", part)
	}
}

// processValueDecoratorUnified handles value decorators across all execution modes
func (c *ExecutionContext) processValueDecoratorUnified(decorator *ast.ValueDecorator) (interface{}, error) {
	if c.valueDecoratorLookup == nil {
		return nil, fmt.Errorf("value decorator lookup not available (engine not properly initialized)")
	}

	valueDecorator, exists := c.valueDecoratorLookup(decorator.Name)
	if !exists {
		return nil, fmt.Errorf("value decorator @%s not found in registry", decorator.Name)
	}

	result := valueDecorator.(interface{ Expand(*ExecutionContext, []ast.NamedParameter) *ExecutionResult }).Expand(c, decorator.Args)
	if result.Error != nil {
		return nil, fmt.Errorf("@%s decorator execution failed: %w", decorator.Name, result.Error)
	}

	// Return appropriate data type based on mode
	switch c.mode {
	case InterpreterMode:
		if value, ok := result.Data.(string); ok {
			return value, nil
		}
		return nil, fmt.Errorf("@%s decorator returned non-string result: %T", decorator.Name, result.Data)
	case GeneratorMode:
		if code, ok := result.Data.(string); ok {
			return code, nil
		}
		return nil, fmt.Errorf("@%s decorator returned non-string code: %T", decorator.Name, result.Data)
	case PlanMode:
		return fmt.Sprintf("@%s(...)", decorator.Name), nil
	default:
		return nil, fmt.Errorf("unsupported mode: %v", c.mode)
	}
}

// processActionDecoratorUnified handles action decorators across all execution modes  
func (c *ExecutionContext) processActionDecoratorUnified(decorator *ast.ActionDecorator) (interface{}, error) {
	if c.actionDecoratorLookup == nil {
		return nil, fmt.Errorf("action decorator lookup not available (engine not properly initialized)")
	}

	actionDecorator, exists := c.actionDecoratorLookup(decorator.Name)
	if !exists {
		return nil, fmt.Errorf("action decorator @%s not found in registry", decorator.Name)
	}

	// CRITICAL FIX: Ensure ActionDecorators are called with the correct context mode
	// If we're in GeneratorMode, make sure we pass a GeneratorMode context
	contextForExpansion := c
	if c.mode == GeneratorMode {
		// Use Child() to create a proper independent context in GeneratorMode
		contextForExpansion = c.Child().WithMode(GeneratorMode)
	}

	result := actionDecorator.(interface{ Expand(*ExecutionContext, []ast.NamedParameter) *ExecutionResult }).Expand(contextForExpansion, decorator.Args)
	if result.Error != nil {
		return nil, fmt.Errorf("@%s decorator execution failed: %w", decorator.Name, result.Error)
	}

	// Handle result based on mode and type
	switch c.mode {
	case InterpreterMode:
		if commandResult, ok := result.Data.(CommandResult); ok {
			return commandResult.Stdout, nil
		} else if value, ok := result.Data.(string); ok {
			return value, nil
		}
		return nil, fmt.Errorf("@%s action decorator returned unsupported result type: %T", decorator.Name, result.Data)
	case GeneratorMode:
		if code, ok := result.Data.(string); ok {
			return code, nil
		}
		return nil, fmt.Errorf("@%s action decorator returned non-string code: %T", decorator.Name, result.Data)
	case PlanMode:
		return fmt.Sprintf("@%s(...)", decorator.Name), nil
	default:
		return nil, fmt.Errorf("unsupported mode: %v", c.mode)
	}
}


// GetVariable retrieves a variable value
func (c *ExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := c.Variables[name]
	return value, exists
}

// SetVariable sets a variable value
func (c *ExecutionContext) SetVariable(name, value string) {
	c.Variables[name] = value
}

// GetEnv retrieves an environment variable from the immutable captured environment
func (c *ExecutionContext) GetEnv(name string) (string, bool) {
	value, exists := c.env[name]
	return value, exists
}

// InitializeVariables processes and sets all variables from the program
func (c *ExecutionContext) InitializeVariables() error {
	if c.Program == nil {
		return nil
	}

	// Process individual variables
	for _, variable := range c.Program.Variables {
		value, err := c.resolveVariableValue(variable.Value)
		if err != nil {
			return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
		}
		c.SetVariable(variable.Name, value)
	}

	// Process variable groups
	for _, group := range c.Program.VarGroups {
		for _, variable := range group.Variables {
			value, err := c.resolveVariableValue(variable.Value)
			if err != nil {
				return fmt.Errorf("failed to resolve variable %s: %w", variable.Name, err)
			}
			c.SetVariable(variable.Name, value)
		}
	}

	return nil
}

// GetTemplateFunctions returns the template function map for code generation
func (c *ExecutionContext) GetTemplateFunctions() template.FuncMap {
	// Create a unified shell code builder and get its template functions
	shellBuilder := NewShellCodeBuilder(c)
	unifiedFunctions := shellBuilder.GetTemplateFunctions()
	
	// Merge with any existing custom template functions
	for name, fn := range c.templateFunctions {
		unifiedFunctions[name] = fn
	}
	
	return unifiedFunctions
}

// SetTemplateFunctions sets the template function map (used by engine)
func (c *ExecutionContext) SetTemplateFunctions(funcs template.FuncMap) {
	c.templateFunctions = funcs
}

// ExecuteCommandContent executes command content using the engine's executor (used by decorators)
func (c *ExecutionContext) ExecuteCommandContent(content ast.CommandContent) error {
	if c.contentExecutor == nil {
		return fmt.Errorf("command content executor not available (engine not properly initialized)")
	}
	return c.contentExecutor(content)
}

// SetContentExecutor sets the command content executor (used by engine)
func (c *ExecutionContext) SetContentExecutor(executor func(ast.CommandContent) error) {
	c.contentExecutor = executor
}

// SetValueDecoratorLookup sets the value decorator lookup (used by engine)
func (c *ExecutionContext) SetValueDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.valueDecoratorLookup = lookup
}

// SetActionDecoratorLookup sets the action decorator lookup (used by engine)
func (c *ExecutionContext) SetActionDecoratorLookup(lookup func(name string) (interface{}, bool)) {
	c.actionDecoratorLookup = lookup
}

// ExecuteCommand executes a full command by name (used by decorators like @cmd)
func (c *ExecutionContext) ExecuteCommand(commandName string) error {
	if c.commandExecutor == nil {
		return fmt.Errorf("command executor not available (engine not properly initialized)")
	}

	// Find the command in the program
	for _, cmd := range c.Program.Commands {
		if cmd.Name == commandName {
			return c.commandExecutor(&cmd)
		}
	}

	return fmt.Errorf("command '%s' not found", commandName)
}

// GenerateCommandPlan generates a plan for a command by name (used by decorators like @cmd)
func (c *ExecutionContext) GenerateCommandPlan(commandName string) (*ExecutionResult, error) {
	if c.commandPlanGenerator == nil {
		return nil, fmt.Errorf("command plan generator not available (engine not properly initialized)")
	}

	// Find the command in the program
	for _, cmd := range c.Program.Commands {
		if cmd.Name == commandName {
			return c.commandPlanGenerator(&cmd)
		}
	}

	return &ExecutionResult{
		Mode:  c.Mode(),
		Data:  nil,
		Error: fmt.Errorf("command '%s' not found", commandName),
	}, nil
}

// SetCommandExecutor sets the command executor (used by engine)
func (c *ExecutionContext) SetCommandExecutor(executor func(*ast.CommandDecl) error) {
	c.commandExecutor = executor
}

// SetCommandPlanGenerator sets the command plan generator (used by engine)
func (c *ExecutionContext) SetCommandPlanGenerator(generator func(*ast.CommandDecl) (*ExecutionResult, error)) {
	c.commandPlanGenerator = generator
}

// Legacy template constants removed - now using unified template system in templates.go

// GenerateShellCodeForTemplate is deprecated - use unified template system in templates.go
// This function is kept for backward compatibility but redirects to the new system
func (c *ExecutionContext) GenerateShellCodeForTemplate(content *ast.ShellContent) (string, error) {
	shellBuilder := NewShellCodeBuilder(c)
	return shellBuilder.GenerateShellCode(content)
}

// resolveVariableValue converts an AST expression to its string value
func (c *ExecutionContext) resolveVariableValue(expr ast.Expression) (string, error) {
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
