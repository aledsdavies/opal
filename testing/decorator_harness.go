// Package testing provides testing utilities for devcmd decorators
// This package offers a clean API for testing decorators without requiring
// knowledge of internal parser, engine, or execution details.
package testing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// DecoratorHarness provides a high-level testing interface for decorators
// It abstracts away the complexity of execution contexts, AST manipulation,
// and engine simulation to provide a simple testing API.
type DecoratorHarness struct {
	// Test environment
	variables map[string]string
	env       map[string]string

	// Execution tracking
	executedCommands []ExecutionRecord

	// Configuration
	workingDir string
	debug      bool
}

// ExecutionRecord captures what happened during decorator execution
type ExecutionRecord struct {
	Type      string    // "shell", "decorator", etc.
	Command   string    // The command that was executed
	Timestamp time.Time // When it was executed
	Success   bool      // Whether it succeeded
	Output    string    // Any output produced
	Error     error     // Any error that occurred
}

// ExecutionMode represents different ways decorators can be executed
type ExecutionMode string

const (
	// InterpreterMode runs decorators directly (like devcmd run)
	InterpreterMode ExecutionMode = "interpreter"

	// GeneratorMode generates Go code (like devcmd build)
	GeneratorMode ExecutionMode = "generator"

	// PlanMode creates execution plans (like devcmd plan)
	PlanMode ExecutionMode = "plan"
)

// TestResult contains the outcome of decorator testing
type TestResult struct {
	Mode     ExecutionMode
	Success  bool
	Data     interface{} // Mode-specific result data
	Error    error
	Duration time.Duration
}

// NewDecoratorHarness creates a new testing harness
func NewDecoratorHarness() *DecoratorHarness {
	return &DecoratorHarness{
		variables:        make(map[string]string),
		env:              make(map[string]string),
		executedCommands: []ExecutionRecord{},
		workingDir:       "/tmp",
		debug:            false,
	}
}

// SetVariable sets a variable in the test environment
func (h *DecoratorHarness) SetVariable(name, value string) *DecoratorHarness {
	h.variables[name] = value
	return h
}

// SetEnv sets an environment variable in the test environment
func (h *DecoratorHarness) SetEnv(name, value string) *DecoratorHarness {
	h.env[name] = value
	return h
}

// SetWorkingDir sets the working directory for command execution
func (h *DecoratorHarness) SetWorkingDir(dir string) *DecoratorHarness {
	h.workingDir = dir
	return h
}

// EnableDebug enables debug output for testing
func (h *DecoratorHarness) EnableDebug() *DecoratorHarness {
	h.debug = true
	return h
}

// GetExecutionHistory returns all executed commands during testing
func (h *DecoratorHarness) GetExecutionHistory() []ExecutionRecord {
	return h.executedCommands
}

// ClearHistory clears the execution history
func (h *DecoratorHarness) ClearHistory() {
	h.executedCommands = []ExecutionRecord{}
}

// TestFunctionDecorator tests a function decorator (like @var, @env)
// Function decorators return values that get substituted into commands
func (h *DecoratorHarness) TestFunctionDecorator(decorator interface{}, params map[string]interface{}) map[ExecutionMode]TestResult {
	results := make(map[ExecutionMode]TestResult)

	// Test in all modes
	modes := []ExecutionMode{InterpreterMode, GeneratorMode, PlanMode}

	for _, mode := range modes {
		start := time.Now()
		result := h.testFunctionDecoratorInMode(decorator, params, mode)
		result.Duration = time.Since(start)
		results[mode] = result
	}

	return results
}

// TestBlockDecorator tests a block decorator (like @timeout, @parallel)
// Block decorators wrap and modify the execution of commands
func (h *DecoratorHarness) TestBlockDecorator(decorator interface{}, params map[string]interface{}, commands []string) map[ExecutionMode]TestResult {
	results := make(map[ExecutionMode]TestResult)

	// Test in all modes
	modes := []ExecutionMode{InterpreterMode, GeneratorMode, PlanMode}

	for _, mode := range modes {
		start := time.Now()
		result := h.testBlockDecoratorInMode(decorator, params, commands, mode)
		result.Duration = time.Since(start)
		results[mode] = result
	}

	return results
}

// TestPatternDecorator tests a pattern decorator (like @when, @try)
// Pattern decorators execute different commands based on conditions
func (h *DecoratorHarness) TestPatternDecorator(decorator interface{}, params map[string]interface{}, patterns map[string][]string) map[ExecutionMode]TestResult {
	results := make(map[ExecutionMode]TestResult)

	// Test in all modes
	modes := []ExecutionMode{InterpreterMode, GeneratorMode, PlanMode}

	for _, mode := range modes {
		start := time.Now()
		result := h.testPatternDecoratorInMode(decorator, params, patterns, mode)
		result.Duration = time.Since(start)
		results[mode] = result
	}

	return results
}

// QuickTest provides a simple test interface for basic decorator validation
func (h *DecoratorHarness) QuickTest(decorator interface{}, testType string, input interface{}) TestResult {
	start := time.Now()

	switch testType {
	case "function":
		if params, ok := input.(map[string]interface{}); ok {
			results := h.TestFunctionDecorator(decorator, params)
			// Return interpreter mode result for quick testing
			result := results[InterpreterMode]
			result.Duration = time.Since(start)
			return result
		}
	case "block":
		if testData, ok := input.(map[string]interface{}); ok {
			if params, hasParams := testData["params"].(map[string]interface{}); hasParams {
				if commands, hasCommands := testData["commands"].([]string); hasCommands {
					results := h.TestBlockDecorator(decorator, params, commands)
					result := results[InterpreterMode]
					result.Duration = time.Since(start)
					return result
				}
			}
		}
	case "pattern":
		if testData, ok := input.(map[string]interface{}); ok {
			if params, hasParams := testData["params"].(map[string]interface{}); hasParams {
				if patterns, hasPatterns := testData["patterns"].(map[string][]string); hasPatterns {
					results := h.TestPatternDecorator(decorator, params, patterns)
					result := results[InterpreterMode]
					result.Duration = time.Since(start)
					return result
				}
			}
		}
	}

	return TestResult{
		Mode:     InterpreterMode,
		Success:  false,
		Error:    fmt.Errorf("invalid test type or input format"),
		Duration: time.Since(start),
	}
}

// testFunctionDecoratorInMode tests a function decorator in a specific mode
func (h *DecoratorHarness) testFunctionDecoratorInMode(decorator interface{}, params map[string]interface{}, mode ExecutionMode) TestResult {
	// Convert to internal types and test
	ctx := h.createExecutionContext(mode)

	// Use type assertion to call the appropriate decorator interface
	if funcDec, ok := decorator.(decorators.ValueDecorator); ok {
		// Get parameter schema from the decorator to convert params correctly
		astParams := h.convertParamsUsingSchema(params, funcDec)
		result := funcDec.Expand(ctx, astParams)
		return h.convertExecutionResult(result, mode)
	}

	return TestResult{
		Mode:    mode,
		Success: false,
		Error:   fmt.Errorf("decorator does not implement FunctionDecorator interface"),
	}
}

// testBlockDecoratorInMode tests a block decorator in a specific mode
func (h *DecoratorHarness) testBlockDecoratorInMode(decorator interface{}, params map[string]interface{}, commands []string, mode ExecutionMode) TestResult {
	ctx := h.createExecutionContext(mode)
	astCommands := h.convertCommands(commands)

	if blockDec, ok := decorator.(decorators.BlockDecorator); ok {
		// Get parameter schema from the decorator to convert params correctly
		astParams := h.convertParamsUsingSchema(params, blockDec)
		result := blockDec.Execute(ctx, astParams, astCommands)
		return h.convertExecutionResult(result, mode)
	}

	return TestResult{
		Mode:    mode,
		Success: false,
		Error:   fmt.Errorf("decorator does not implement BlockDecorator interface"),
	}
}

// testPatternDecoratorInMode tests a pattern decorator in a specific mode
func (h *DecoratorHarness) testPatternDecoratorInMode(decorator interface{}, params map[string]interface{}, patterns map[string][]string, mode ExecutionMode) TestResult {
	ctx := h.createExecutionContext(mode)
	astPatterns := h.convertPatterns(patterns)

	if patternDec, ok := decorator.(decorators.PatternDecorator); ok {
		// Get parameter schema from the decorator to convert params correctly
		astParams := h.convertParamsUsingSchema(params, patternDec)
		result := patternDec.Execute(ctx, astParams, astPatterns)
		return h.convertExecutionResult(result, mode)
	}

	return TestResult{
		Mode:    mode,
		Success: false,
		Error:   fmt.Errorf("decorator does not implement PatternDecorator interface"),
	}
}

// createExecutionContext creates a mock execution context for testing
func (h *DecoratorHarness) createExecutionContext(mode ExecutionMode) *execution.ExecutionContext {
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
		VarGroups: []ast.VarGroup{},
	}

	// Set environment variables before creating context (they're captured immutably)
	originalEnv := make(map[string]string)
	for name, value := range h.env {
		originalEnv[name] = os.Getenv(name)
		os.Setenv(name, value)
	}

	ctx := execution.NewExecutionContext(context.Background(), program)

	// Restore original environment
	for name, originalValue := range originalEnv {
		if originalValue == "" {
			os.Unsetenv(name)
		} else {
			os.Setenv(name, originalValue)
		}
	}

	// Set up variables
	for name, value := range h.variables {
		ctx.SetVariable(name, value)
	}

	// Configure context
	ctx.WorkingDir = h.workingDir
	ctx.Debug = h.debug

	// Set execution mode
	var execMode execution.ExecutionMode
	switch mode {
	case InterpreterMode:
		execMode = execution.InterpreterMode
	case GeneratorMode:
		execMode = execution.GeneratorMode
	case PlanMode:
		execMode = execution.PlanMode
	}
	ctx = ctx.WithMode(execMode)

	// Set up mock executors
	h.setupMockExecutors(ctx)

	return ctx
}

// setupMockExecutors sets up mock command executors for testing
func (h *DecoratorHarness) setupMockExecutors(ctx *execution.ExecutionContext) {
	// Mock content executor
	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		record := ExecutionRecord{
			Type:      "content",
			Command:   h.contentToString(content),
			Timestamp: time.Now(),
			Success:   true,
		}

		// Simulate command execution
		if strings.Contains(record.Command, "fail") {
			record.Success = false
			record.Error = fmt.Errorf("mock command failed")
			h.executedCommands = append(h.executedCommands, record)
			return record.Error
		}

		h.executedCommands = append(h.executedCommands, record)
		return nil
	})

	// Mock template functions for code generation
	templateFuncs := template.FuncMap{
		"generateShellCode": func(cmd ast.CommandContent) string {
			return fmt.Sprintf("// Generated: %s", h.contentToString(cmd))
		},
	}
	ctx.SetTemplateFunctions(templateFuncs)
}

// convertParamsUsingSchema converts parameters using the decorator's parameter schema
func (h *DecoratorHarness) convertParamsUsingSchema(params map[string]interface{}, decorator decorators.Decorator) []ast.NamedParameter {
	var result []ast.NamedParameter
	schema := decorator.ParameterSchema()

	for name, value := range params {
		param := ast.NamedParameter{Name: name}

		// Find the expected type from the schema
		var expectedType ast.ExpressionType = ast.StringType // default
		for _, schemaParam := range schema {
			if schemaParam.Name == name {
				expectedType = schemaParam.Type
				break
			}
		}

		// Convert based on expected type
		switch v := value.(type) {
		case string:
			switch expectedType {
			case ast.IdentifierType:
				param.Value = &ast.Identifier{Name: v}
			case ast.StringType:
				param.Value = &ast.StringLiteral{Value: v}
			default:
				param.Value = &ast.StringLiteral{Value: v}
			}
		case int:
			param.Value = &ast.NumberLiteral{Value: fmt.Sprintf("%d", v)}
		case bool:
			param.Value = &ast.BooleanLiteral{Value: v}
		case time.Duration:
			param.Value = &ast.DurationLiteral{Value: v.String()}
		default:
			param.Value = &ast.StringLiteral{Value: fmt.Sprintf("%v", v)}
		}

		result = append(result, param)
	}
	return result
}

// Helper functions for converting between external and internal types (fallback)
func (h *DecoratorHarness) convertParams(params map[string]interface{}) []ast.NamedParameter {
	var result []ast.NamedParameter
	for name, value := range params {
		param := ast.NamedParameter{Name: name}

		switch v := value.(type) {
		case string:
			param.Value = &ast.StringLiteral{Value: v}
		case int:
			param.Value = &ast.NumberLiteral{Value: fmt.Sprintf("%d", v)}
		case bool:
			param.Value = &ast.BooleanLiteral{Value: v}
		case time.Duration:
			param.Value = &ast.DurationLiteral{Value: v.String()}
		default:
			param.Value = &ast.StringLiteral{Value: fmt.Sprintf("%v", v)}
		}

		result = append(result, param)
	}
	return result
}

func (h *DecoratorHarness) convertCommands(commands []string) []ast.CommandContent {
	var result []ast.CommandContent
	for _, cmd := range commands {
		result = append(result, &ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: cmd},
			},
		})
	}
	return result
}

func (h *DecoratorHarness) convertPatterns(patterns map[string][]string) []ast.PatternBranch {
	var result []ast.PatternBranch
	for pattern, commands := range patterns {
		var astPattern ast.Pattern
		if pattern == "default" {
			astPattern = &ast.WildcardPattern{}
		} else {
			astPattern = &ast.IdentifierPattern{Name: pattern}
		}

		branch := ast.PatternBranch{
			Pattern:  astPattern,
			Commands: h.convertCommands(commands),
		}
		result = append(result, branch)
	}
	return result
}

func (h *DecoratorHarness) convertExecutionResult(result *execution.ExecutionResult, mode ExecutionMode) TestResult {
	return TestResult{
		Mode:    mode,
		Success: result.Error == nil,
		Data:    result.Data,
		Error:   result.Error,
	}
}

func (h *DecoratorHarness) contentToString(content ast.CommandContent) string {
	switch c := content.(type) {
	case *ast.ShellContent:
		var parts []string
		for _, part := range c.Parts {
			switch p := part.(type) {
			case *ast.TextPart:
				parts = append(parts, p.Text)
			case *ast.ValueDecorator:
				parts = append(parts, fmt.Sprintf("@%s(...)", p.Name))
			}
		}
		return strings.Join(parts, "")
	default:
		return fmt.Sprintf("<%T>", content)
	}
}