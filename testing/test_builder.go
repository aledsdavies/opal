package testing

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestBuilder provides a fluent interface for building decorator tests
type TestBuilder struct {
	harness    *DecoratorHarness
	testName   string
	decorator  interface{}
	testType   string
	params     map[string]interface{}
	commands   []string
	patterns   map[string][]string
	validators []ResultValidator
}

// ResultValidator is a function that validates test results
type ResultValidator func(TestResult) error

// NewTestBuilder creates a new test builder
func NewTestBuilder(testName string) *TestBuilder {
	return &TestBuilder{
		harness:    NewDecoratorHarness(),
		testName:   testName,
		params:     make(map[string]interface{}),
		patterns:   make(map[string][]string),
		validators: []ResultValidator{},
	}
}

// WithDecorator sets the decorator to test
func (b *TestBuilder) WithDecorator(decorator interface{}) *TestBuilder {
	b.decorator = decorator
	return b
}

// AsFunctionDecorator marks this as a function decorator test
func (b *TestBuilder) AsFunctionDecorator() *TestBuilder {
	b.testType = "function"
	return b
}

// AsBlockDecorator marks this as a block decorator test
func (b *TestBuilder) AsBlockDecorator() *TestBuilder {
	b.testType = "block"
	return b
}

// AsPatternDecorator marks this as a pattern decorator test
func (b *TestBuilder) AsPatternDecorator() *TestBuilder {
	b.testType = "pattern"
	return b
}

// WithVariable adds a variable to the test environment
func (b *TestBuilder) WithVariable(name, value string) *TestBuilder {
	b.harness.SetVariable(name, value)
	return b
}

// WithEnv adds an environment variable to the test environment
func (b *TestBuilder) WithEnv(name, value string) *TestBuilder {
	b.harness.SetEnv(name, value)
	return b
}

// WithParam adds a parameter for the decorator
func (b *TestBuilder) WithParam(name string, value interface{}) *TestBuilder {
	b.params[name] = value
	return b
}

// WithCommand adds a command for block decorators
func (b *TestBuilder) WithCommand(command string) *TestBuilder {
	b.commands = append(b.commands, command)
	return b
}

// WithCommands adds multiple commands for block decorators
func (b *TestBuilder) WithCommands(commands ...string) *TestBuilder {
	b.commands = append(b.commands, commands...)
	return b
}

// WithPattern adds a pattern branch for pattern decorators
func (b *TestBuilder) WithPattern(pattern string, commands ...string) *TestBuilder {
	b.patterns[pattern] = commands
	return b
}

// WithWorkingDir sets the working directory for testing
func (b *TestBuilder) WithWorkingDir(dir string) *TestBuilder {
	b.harness.SetWorkingDir(dir)
	return b
}

// ExpectSuccess adds a validator that expects successful execution
func (b *TestBuilder) ExpectSuccess() *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		if !result.Success {
			return fmt.Errorf("expected success but got error: %v", result.Error)
		}
		return nil
	})
	return b
}

// ExpectFailure adds a validator that expects execution to fail
func (b *TestBuilder) ExpectFailure(expectedErrorContains string) *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		if result.Success {
			return fmt.Errorf("expected failure but execution succeeded")
		}
		if expectedErrorContains != "" && !strings.Contains(result.Error.Error(), expectedErrorContains) {
			return fmt.Errorf("expected error containing %q, got: %v", expectedErrorContains, result.Error)
		}
		return nil
	})
	return b
}

// ExpectData adds a validator that checks the result data
func (b *TestBuilder) ExpectData(expected interface{}) *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		if !reflect.DeepEqual(result.Data, expected) {
			return fmt.Errorf("expected data %v, got %v", expected, result.Data)
		}
		return nil
	})
	return b
}

// ExpectDataContains adds a validator that checks if string data contains expected text
func (b *TestBuilder) ExpectDataContains(expectedContains string) *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		if result.Data == nil {
			return fmt.Errorf("expected data to contain %q, but data is nil", expectedContains)
		}
		if str, ok := result.Data.(string); ok {
			if !strings.Contains(str, expectedContains) {
				return fmt.Errorf("expected data to contain %q, got: %v", expectedContains, str)
			}
		} else {
			return fmt.Errorf("expected string data, got %T", result.Data)
		}
		return nil
	})
	return b
}

// ExpectExecutionTime adds a validator that checks execution time
func (b *TestBuilder) ExpectExecutionTime(maxDuration time.Duration) *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		if result.Duration > maxDuration {
			return fmt.Errorf("execution took %v, expected under %v", result.Duration, maxDuration)
		}
		return nil
	})
	return b
}

// ExpectCommandsExecuted adds a validator that checks the number of executed commands
func (b *TestBuilder) ExpectCommandsExecuted(count int) *TestBuilder {
	b.validators = append(b.validators, func(result TestResult) error {
		actualCount := len(b.harness.GetExecutionHistory())
		if actualCount != count {
			return fmt.Errorf("expected %d commands executed, got %d", count, actualCount)
		}
		return nil
	})
	return b
}

// WithCustomValidator adds a custom result validator
func (b *TestBuilder) WithCustomValidator(validator ResultValidator) *TestBuilder {
	b.validators = append(b.validators, validator)
	return b
}

// RunTest executes the test with the built configuration
func (b *TestBuilder) RunTest(t *testing.T) {
	t.Run(b.testName, func(t *testing.T) {
		// Validate configuration
		if b.decorator == nil {
			t.Fatal("No decorator specified")
		}
		if b.testType == "" {
			t.Fatal("No test type specified (use AsFunctionDecorator, AsBlockDecorator, or AsPatternDecorator)")
		}

		// Clear execution history before test
		b.harness.ClearHistory()

		// Run the test based on type
		var results map[ExecutionMode]TestResult

		switch b.testType {
		case "function":
			results = b.harness.TestFunctionDecorator(b.decorator, b.params)
		case "block":
			results = b.harness.TestBlockDecorator(b.decorator, b.params, b.commands)
		case "pattern":
			results = b.harness.TestPatternDecorator(b.decorator, b.params, b.patterns)
		default:
			t.Fatalf("Unknown test type: %s", b.testType)
		}

		// Validate results for each mode
		for mode, result := range results {
			t.Run(string(mode), func(t *testing.T) {
				for i, validator := range b.validators {
					if err := validator(result); err != nil {
						t.Errorf("Validator %d failed: %v", i+1, err)
					}
				}
			})
		}
	})
}

// QuickTest runs a simple test without detailed validation
func (b *TestBuilder) QuickTest(t *testing.T) TestResult {
	if b.decorator == nil {
		t.Fatal("No decorator specified")
	}

	var input interface{}
	switch b.testType {
	case "function":
		input = b.params
	case "block":
		input = map[string]interface{}{
			"params":   b.params,
			"commands": b.commands,
		}
	case "pattern":
		input = map[string]interface{}{
			"params":   b.params,
			"patterns": b.patterns,
		}
	default:
		t.Fatal("No test type specified")
	}

	return b.harness.QuickTest(b.decorator, b.testType, input)
}

// TestSuite provides utilities for running multiple related tests
type TestSuite struct {
	name     string
	tests    []*TestBuilder
	setup    func(*DecoratorHarness)
	teardown func(*DecoratorHarness)
}

// NewTestSuite creates a new test suite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		name:  name,
		tests: []*TestBuilder{},
	}
}

// WithSetup adds a setup function that runs before each test
func (s *TestSuite) WithSetup(setup func(*DecoratorHarness)) *TestSuite {
	s.setup = setup
	return s
}

// WithTeardown adds a teardown function that runs after each test
func (s *TestSuite) WithTeardown(teardown func(*DecoratorHarness)) *TestSuite {
	s.teardown = teardown
	return s
}

// AddTest adds a test to the suite
func (s *TestSuite) AddTest(test *TestBuilder) *TestSuite {
	s.tests = append(s.tests, test)
	return s
}

// RunSuite executes all tests in the suite
func (s *TestSuite) RunSuite(t *testing.T) {
	t.Run(s.name, func(t *testing.T) {
		for _, test := range s.tests {
			// Apply setup if configured
			if s.setup != nil {
				s.setup(test.harness)
			}

			// Run the test
			test.RunTest(t)

			// Apply teardown if configured
			if s.teardown != nil {
				s.teardown(test.harness)
			}
		}
	})
}

// BenchmarkBuilder provides utilities for benchmarking decorators
type BenchmarkBuilder struct {
	harness   *DecoratorHarness
	decorator interface{}
	testType  string
	params    map[string]interface{}
	commands  []string
	patterns  map[string][]string
}

// NewBenchmarkBuilder creates a new benchmark builder
func NewBenchmarkBuilder() *BenchmarkBuilder {
	return &BenchmarkBuilder{
		harness:  NewDecoratorHarness(),
		params:   make(map[string]interface{}),
		patterns: make(map[string][]string),
	}
}

// WithDecorator sets the decorator to benchmark
func (b *BenchmarkBuilder) WithDecorator(decorator interface{}) *BenchmarkBuilder {
	b.decorator = decorator
	return b
}

// AsFunctionDecorator marks this as a function decorator benchmark
func (b *BenchmarkBuilder) AsFunctionDecorator() *BenchmarkBuilder {
	b.testType = "function"
	return b
}

// AsBlockDecorator marks this as a block decorator benchmark
func (b *BenchmarkBuilder) AsBlockDecorator() *BenchmarkBuilder {
	b.testType = "block"
	return b
}

// AsPatternDecorator marks this as a pattern decorator benchmark
func (b *BenchmarkBuilder) AsPatternDecorator() *BenchmarkBuilder {
	b.testType = "pattern"
	return b
}

// WithParam adds a parameter for the decorator
func (b *BenchmarkBuilder) WithParam(name string, value interface{}) *BenchmarkBuilder {
	b.params[name] = value
	return b
}

// WithCommands adds commands for block decorators
func (b *BenchmarkBuilder) WithCommands(commands ...string) *BenchmarkBuilder {
	b.commands = commands
	return b
}

// WithPattern adds a pattern for pattern decorators
func (b *BenchmarkBuilder) WithPattern(pattern string, commands ...string) *BenchmarkBuilder {
	b.patterns[pattern] = commands
	return b
}

// RunBenchmark executes the benchmark
func (b *BenchmarkBuilder) RunBenchmark(bench *testing.B) {
	if b.decorator == nil {
		bench.Fatal("No decorator specified")
	}

	bench.ResetTimer()

	for i := 0; i < bench.N; i++ {
		switch b.testType {
		case "function":
			b.harness.TestFunctionDecorator(b.decorator, b.params)
		case "block":
			b.harness.TestBlockDecorator(b.decorator, b.params, b.commands)
		case "pattern":
			b.harness.TestPatternDecorator(b.decorator, b.params, b.patterns)
		}

		// Clear history to avoid memory buildup during benchmarking
		b.harness.ClearHistory()
	}
}