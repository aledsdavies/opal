package testing

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestExampleVarDecorator demonstrates testing a function decorator
func TestExampleVarDecorator(t *testing.T) {
	// Test using the fluent TestBuilder API
	NewTestBuilder("var_decorator_basic").
		WithDecorator(&TestVarDecorator{}).
		AsFunctionDecorator().
		WithVariable("TEST_VAR", "hello").
		WithParam("name", "TEST_VAR").
		ExpectSuccess().
		ExpectExecutionTime(100 * time.Millisecond).
		WithCustomValidator(func(result TestResult) error {
			// Validate mode-specific expectations
			switch result.Mode {
			case InterpreterMode:
				if result.Data != "hello" {
					return fmt.Errorf("interpreter mode: expected 'hello', got %v", result.Data)
				}
			case GeneratorMode:
				if result.Data != "TEST_VAR" {
					return fmt.Errorf("generator mode: expected 'TEST_VAR', got %v", result.Data)
				}
			case PlanMode:
				if result.Data == nil {
					return fmt.Errorf("plan mode: expected plan data, got nil")
				}
			}
			return nil
		}).
		RunTest(t)
}

// TestExampleVarDecoratorUndefined demonstrates testing error conditions
func TestExampleVarDecoratorUndefined(t *testing.T) {
	NewTestBuilder("var_decorator_undefined").
		WithDecorator(&TestVarDecorator{}).
		AsFunctionDecorator().
		WithParam("name", "UNDEFINED_VAR").
		WithCustomValidator(func(result TestResult) error {
			// Plan mode might not fail the same way as other modes
			if result.Mode == PlanMode {
				return nil // Plan mode can show undefined variables for visualization
			}
			if result.Success {
				return fmt.Errorf("expected failure but execution succeeded")
			}
			if !strings.Contains(result.Error.Error(), "not defined") {
				return fmt.Errorf("expected error containing 'not defined', got: %v", result.Error)
			}
			return nil
		}).
		RunTest(t)
}

// TestExampleTimeoutDecorator demonstrates testing a block decorator
func TestExampleTimeoutDecorator(t *testing.T) {
	NewTestBuilder("timeout_decorator_basic").
		WithDecorator(&TestTimeoutDecorator{}).
		AsBlockDecorator().
		WithParam("duration", 5*time.Second).
		WithCommands("echo hello", "sleep 1", "echo world").
		ExpectSuccess().
		ExpectCommandsExecuted(3).
		ExpectExecutionTime(2 * time.Second). // Should complete well under timeout
		RunTest(t)
}

// TestExampleWhenDecorator demonstrates testing a pattern decorator
func TestExampleWhenDecorator(t *testing.T) {
	NewTestBuilder("when_decorator_basic").
		WithDecorator(&TestWhenDecorator{}).
		AsPatternDecorator().
		WithEnv("NODE_ENV", "production").
		WithParam("variable", "NODE_ENV").
		WithPattern("production", "echo 'Running in production'", "npm run build").
		WithPattern("development", "echo 'Running in development'", "npm run dev").
		WithPattern("default", "echo 'Unknown environment'").
		ExpectSuccess().
		WithCustomValidator(func(result TestResult) error {
			// Pattern decorators execute as a single unit, not individual commands
			// The when decorator handles pattern matching internally
			return nil
		}).
		RunTest(t)
}

// TestExampleQuickTest demonstrates the quick test API
func TestExampleQuickTest(t *testing.T) {
	harness := NewDecoratorHarness().
		SetVariable("PORT", "8080").
		SetEnv("NODE_ENV", "test")

	// Quick test for function decorator
	result := harness.QuickTest(
		&TestVarDecorator{},
		"function",
		map[string]interface{}{
			"name": "PORT",
		},
	)

	if !result.Success {
		t.Errorf("Quick test failed: %v", result.Error)
	}

	if result.Data != "8080" {
		t.Errorf("Expected '8080', got %v", result.Data)
	}
}

// TestExampleTestSuite demonstrates using test suites
func TestExampleTestSuite(t *testing.T) {
	suite := NewTestSuite("var_decorator_comprehensive").
		WithSetup(func(harness *DecoratorHarness) {
			harness.SetVariable("GLOBAL_VAR", "global_value")
			harness.SetEnv("ENV_VAR", "env_value")
		}).
		AddTest(
			NewTestBuilder("test_global_var").
				WithDecorator(&TestVarDecorator{}).
				AsFunctionDecorator().
				WithParam("name", "GLOBAL_VAR").
				ExpectSuccess().
				ExpectData("global_value"),
		).
		AddTest(
			NewTestBuilder("test_undefined_var").
				WithDecorator(&TestVarDecorator{}).
				AsFunctionDecorator().
				WithParam("name", "UNDEFINED").
				ExpectFailure("not defined"),
		)

	suite.RunSuite(t)
}

// TestExampleCustomValidation demonstrates custom result validation
func TestExampleCustomValidation(t *testing.T) {
	NewTestBuilder("custom_validation").
		WithDecorator(&TestVarDecorator{}).
		AsFunctionDecorator().
		WithVariable("VERSION", "v1.2.3").
		WithParam("name", "VERSION").
		WithCustomValidator(func(result TestResult) error {
			if result.Mode == GeneratorMode {
				// In generator mode, should return variable name for Go code
				if result.Data != "VERSION" {
					return fmt.Errorf("Generator mode should return variable name, got: %v", result.Data)
				}
			}
			return nil
		}).
		ExpectSuccess().
		RunTest(t)
}

// TestExampleModeSpecificTesting demonstrates testing specific execution modes
func TestExampleModeSpecificTesting(t *testing.T) {
	harness := NewDecoratorHarness().
		SetVariable("APP_NAME", "myapp")

	// Test all modes
	results := harness.TestFunctionDecorator(
		&TestVarDecorator{},
		map[string]interface{}{
			"name": "APP_NAME",
		},
	)

	// Validate interpreter mode
	interpreterResult := results[InterpreterMode]
	if !interpreterResult.Success {
		t.Errorf("Interpreter mode failed: %v", interpreterResult.Error)
	}
	if interpreterResult.Data != "myapp" {
		t.Errorf("Interpreter mode: expected 'myapp', got %v", interpreterResult.Data)
	}

	// Validate generator mode
	generatorResult := results[GeneratorMode]
	if !generatorResult.Success {
		t.Errorf("Generator mode failed: %v", generatorResult.Error)
	}
	// In generator mode, should return the variable name for Go code generation
	if generatorResult.Data != "APP_NAME" {
		t.Errorf("Generator mode: expected 'APP_NAME', got %v", generatorResult.Data)
	}

	// Validate plan mode
	planResult := results[PlanMode]
	if !planResult.Success {
		t.Errorf("Plan mode failed: %v", planResult.Error)
	}
	// Plan mode should return a plan element
	if planResult.Data == nil {
		t.Error("Plan mode: expected plan data, got nil")
	}
}

// BenchmarkExampleVarDecorator demonstrates benchmarking decorators
func BenchmarkExampleVarDecorator(b *testing.B) {
	NewBenchmarkBuilder().
		WithDecorator(&TestVarDecorator{}).
		AsFunctionDecorator().
		WithParam("name", "BENCH_VAR").
		RunBenchmark(b)
}

// TestExampleErrorHandling demonstrates testing error conditions
func TestExampleErrorHandling(t *testing.T) {
	// Test missing parameter
	NewTestBuilder("missing_parameter").
		WithDecorator(&TestTimeoutDecorator{}).
		AsBlockDecorator().
		WithCommands("echo test").
		ExpectFailure("requires a duration parameter").
		RunTest(t)

	// Test invalid parameter type
	NewTestBuilder("invalid_parameter").
		WithDecorator(&TestTimeoutDecorator{}).
		AsBlockDecorator().
		WithParam("duration", "invalid"). // Should be duration, not string
		WithCommands("echo test").
		ExpectFailure("invalid duration").
		RunTest(t)
}

// TestExampleComplexScenario demonstrates testing a complex decorator scenario
func TestExampleComplexScenario(t *testing.T) {
	// Test parallel execution with timeout
	NewTestBuilder("complex_scenario").
		WithDecorator(&TestTimeoutDecorator{}).
		AsBlockDecorator().
		WithCommands(
			"npm run build",
			"npm run test",
			"npm run lint",
		).
		WithVariable("NODE_ENV", "ci").
		WithEnv("CI", "true").
		ExpectSuccess().
		ExpectCommandsExecuted(3).
		ExpectExecutionTime(5 * time.Second).
		WithCustomValidator(func(result TestResult) error {
			// Custom validation for parallel execution
			if result.Mode == GeneratorMode {
				if code, ok := result.Data.(string); ok {
					if !strings.Contains(code, "go func()") {
						return fmt.Errorf("Generated code should contain goroutines for parallel execution")
					}
				}
			}
			return nil
		}).
		RunTest(t)
}

// TestExampleExecutionHistory demonstrates inspecting execution history
func TestExampleExecutionHistory(t *testing.T) {
	harness := NewDecoratorHarness().
		SetVariable("ENV", "test")

	// Run a block decorator test
	results := harness.TestBlockDecorator(
		&TestTimeoutDecorator{},
		map[string]interface{}{
			"duration": 30 * time.Second,
		},
		[]string{
			"echo 'Starting test'",
			"npm test",
			"echo 'Test completed'",
		},
	)

	// Check that interpreter mode succeeded
	if !results[InterpreterMode].Success {
		t.Fatalf("Test failed: %v", results[InterpreterMode].Error)
	}

	// Inspect execution history
	history := harness.GetExecutionHistory()
	if len(history) != 3 {
		t.Errorf("Expected 3 commands in history, got %d", len(history))
	}

	// Validate specific commands were executed
	expectedCommands := []string{
		"echo 'Starting test'",
		"npm test",
		"echo 'Test completed'",
	}

	for i, expected := range expectedCommands {
		if i < len(history) {
			if history[i].Command != expected {
				t.Errorf("Command %d: expected %q, got %q", i, expected, history[i].Command)
			}
			if !history[i].Success {
				t.Errorf("Command %d should have succeeded: %v", i, history[i].Error)
			}
		}
	}
}
