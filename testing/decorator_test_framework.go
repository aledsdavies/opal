package testing

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/decorators"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// DecoratorTestSuite provides a comprehensive testing framework for decorators
// that is completely independent of the engine and focuses on decorator validation
type DecoratorTestSuite struct {
	t         *testing.T
	decorator decorators.Decorator
	program   *ast.Program
	variables map[string]string
	env       map[string]string
}

// NewDecoratorTest creates a new independent decorator test suite
func NewDecoratorTest(t *testing.T, decorator decorators.Decorator) *DecoratorTestSuite {
	return &DecoratorTestSuite{
		t:         t,
		decorator: decorator,
		program:   ast.NewProgram(),
		variables: make(map[string]string),
		env:       make(map[string]string),
	}
}

// WithVariable adds a variable to the test environment
func (d *DecoratorTestSuite) WithVariable(name, value string) *DecoratorTestSuite {
	d.variables[name] = value
	return d
}

// WithEnv adds an environment variable to the test environment  
func (d *DecoratorTestSuite) WithEnv(name, value string) *DecoratorTestSuite {
	d.env[name] = value
	return d
}

// WithCommand adds a command definition to the test program
func (d *DecoratorTestSuite) WithCommand(name string, content ...string) *DecoratorTestSuite {
	// Create shell content for each line
	var commandContent []ast.CommandContent
	for _, line := range content {
		commandContent = append(commandContent, &ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{
					Text: line,
				},
			},
		})
	}
	
	// Add command to program
	d.program.Commands = append(d.program.Commands, ast.CommandDecl{
		Name: name,
		Body: ast.CommandBody{
			Content: commandContent,
		},
	})
	
	return d
}

// TestResult contains the results from testing a decorator in a specific mode
type TestResult struct {
	Mode     string
	Success  bool
	Data     interface{}
	Error    error
	Duration time.Duration
}

// ValidationResult contains comprehensive validation results across all modes
type ValidationResult struct {
	InterpreterResult TestResult
	GeneratorResult   TestResult
	PlanResult        TestResult
	StructuralValid   bool
	ValidationErrors  []string
}

// === VALUE DECORATOR TESTING ===

// TestValueDecorator tests a ValueDecorator across all modes with comprehensive validation
func (d *DecoratorTestSuite) TestValueDecorator(params []ast.NamedParameter) ValidationResult {
	valueDecorator, ok := d.decorator.(decorators.ValueDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a ValueDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := valueDecorator.ExpandInterpreter(ctx, params)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		execResult := valueDecorator.ExpandGenerator(ctx, params)
		duration := time.Since(start)

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateGeneratorResult(execResult, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := valueDecorator.ExpandPlan(ctx, params)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode structural validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === ACTION DECORATOR TESTING ===

// TestActionDecorator tests an ActionDecorator across all modes
func (d *DecoratorTestSuite) TestActionDecorator(params []ast.NamedParameter) ValidationResult {
	actionDecorator, ok := d.decorator.(decorators.ActionDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not an ActionDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := actionDecorator.ExpandInterpreter(ctx, params)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		execResult := actionDecorator.ExpandGenerator(ctx, params)
		duration := time.Since(start)

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateGeneratorResult(execResult, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := actionDecorator.ExpandPlan(ctx, params)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === BLOCK DECORATOR TESTING ===

// TestBlockDecorator tests a BlockDecorator across all modes
func (d *DecoratorTestSuite) TestBlockDecorator(params []ast.NamedParameter, content []ast.CommandContent) ValidationResult {
	blockDecorator, ok := d.decorator.(decorators.BlockDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a BlockDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := blockDecorator.ExecuteInterpreter(ctx, params, content)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		execResult := blockDecorator.ExecuteGenerator(ctx, params, content)
		duration := time.Since(start)

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateGeneratorResult(execResult, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := blockDecorator.ExecutePlan(ctx, params, content)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === PATTERN DECORATOR TESTING ===

// TestPatternDecorator tests a PatternDecorator across all modes
func (d *DecoratorTestSuite) TestPatternDecorator(params []ast.NamedParameter, patterns []ast.PatternBranch) ValidationResult {
	patternDecorator, ok := d.decorator.(decorators.PatternDecorator)
	if !ok {
		d.t.Fatalf("Decorator %s is not a PatternDecorator", d.decorator.Name())
	}

	result := ValidationResult{
		ValidationErrors: []string{},
	}

	// Test Interpreter Mode
	d.t.Run("InterpreterMode", func(t *testing.T) {
		ctx := d.createInterpreterContext()
		start := time.Now()
		execResult := patternDecorator.ExecuteInterpreter(ctx, params, patterns)
		duration := time.Since(start)

		result.InterpreterResult = TestResult{
			Mode:     "interpreter",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateInterpreterResult(execResult, &result)
	})

	// Test Generator Mode
	d.t.Run("GeneratorMode", func(t *testing.T) {
		ctx := d.createGeneratorContext()
		start := time.Now()
		execResult := patternDecorator.ExecuteGenerator(ctx, params, patterns)
		duration := time.Since(start)

		result.GeneratorResult = TestResult{
			Mode:     "generator",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validateGeneratorResult(execResult, &result)
	})

	// Test Plan Mode
	d.t.Run("PlanMode", func(t *testing.T) {
		ctx := d.createPlanContext()
		start := time.Now()
		execResult := patternDecorator.ExecutePlan(ctx, params, patterns)
		duration := time.Since(start)

		result.PlanResult = TestResult{
			Mode:     "plan",
			Success:  execResult.Error == nil,
			Data:     execResult.Data,
			Error:    execResult.Error,
			Duration: duration,
		}

		d.validatePlanResult(execResult, &result)
	})

	// Cross-mode validation
	d.validateCrossModeConsistency(&result)

	return result
}

// === CONTEXT CREATION ===

func (d *DecoratorTestSuite) createInterpreterContext() execution.InterpreterContext {
	ctx := execution.NewInterpreterContext(context.Background(), d.program)
	
	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}
	
	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize interpreter context: %v", err)
	}
	
	return ctx
}

func (d *DecoratorTestSuite) createGeneratorContext() execution.GeneratorContext {
	ctx := execution.NewGeneratorContext(context.Background(), d.program)
	
	// CRITICAL: Set up decorator lookup functions FIRST, before any other operations
	// This ensures they're available when template functions are created
	d.setupDecoratorLookups(ctx)
	
	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}
	
	// Set up environment variables for tracking
	for name, value := range d.env {
		// In a real scenario, we'd set the actual env var too
		// but for testing we just track it
		// TODO: Add proper env var tracking when method is available
		_ = name
		_ = value
	}
	
	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize generator context: %v", err)
	}
	
	return ctx
}

func (d *DecoratorTestSuite) createPlanContext() execution.PlanContext {
	ctx := execution.NewPlanContext(context.Background(), d.program)
	
	// Set up variables
	for name, value := range d.variables {
		ctx.SetVariable(name, value)
	}
	
	if err := ctx.InitializeVariables(); err != nil {
		d.t.Fatalf("Failed to initialize plan context: %v", err)
	}
	
	return ctx
}

// === MODE-SPECIFIC VALIDATION ===

func (d *DecoratorTestSuite) validateInterpreterResult(execResult *execution.ExecutionResult, result *ValidationResult) {
	// Interpreter mode should either succeed or fail gracefully
	// Data can be anything or nil
	// Error should be descriptive if present
	
	if execResult.Error != nil && strings.TrimSpace(execResult.Error.Error()) == "" {
		result.ValidationErrors = append(result.ValidationErrors, 
			"Interpreter mode returned empty error message")
	}
}

func (d *DecoratorTestSuite) validateGeneratorResult(execResult *execution.ExecutionResult, result *ValidationResult) {
	// Generator mode should return valid Go code as string
	if execResult.Error == nil {
		if execResult.Data == nil {
			result.ValidationErrors = append(result.ValidationErrors,
				"Generator mode returned nil data - expected Go code string")
		} else if code, ok := execResult.Data.(string); ok {
			if strings.TrimSpace(code) == "" {
				result.ValidationErrors = append(result.ValidationErrors,
					"Generator mode returned empty code string")
			}
			// TODO: Add Go syntax validation here
		} else {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("Generator mode returned %T, expected string", execResult.Data))
		}
	}
}

func (d *DecoratorTestSuite) validatePlanResult(execResult *execution.ExecutionResult, result *ValidationResult) {
	// Plan mode should return plan data structure
	if execResult.Error == nil {
		if execResult.Data == nil {
			result.ValidationErrors = append(result.ValidationErrors,
				"Plan mode returned nil data - expected plan element")
		}
		// TODO: Add plan structure validation here
	}
}

func (d *DecoratorTestSuite) validateCrossModeConsistency(result *ValidationResult) {
	// Check that modes are consistent with each other
	// For example, if interpreter fails, generator might still work
	// but they should fail for similar reasons
	
	result.StructuralValid = len(result.ValidationErrors) == 0
	
	// Add more cross-mode validation logic here
	// - Parameter handling consistency
	// - Error condition consistency  
	// - Data type consistency where applicable
}

// setupDecoratorLookups configures decorator registry access for testing nested decorators
func (d *DecoratorTestSuite) setupDecoratorLookups(ctx execution.GeneratorContext) {
	// Cast to the concrete type to access the setup methods
	if generatorCtx, ok := ctx.(*execution.GeneratorExecutionContext); ok {
		// Set up block decorator lookup function using the decorator registry
		generatorCtx.SetBlockDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetBlock(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})
		
		// Set up pattern decorator lookup function using the decorator registry
		generatorCtx.SetPatternDecoratorLookup(func(name string) (interface{}, bool) {
			decorator, err := decorators.GetPattern(name)
			if err != nil {
				return nil, false
			}
			return decorator, true
		})
	}
}