package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

func TestExecutionEngine_BasicDecoratorIntegration(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    ExecutionMode
		check   func(t *testing.T, result interface{}, err error)
	}{
		{
			name: "function decorator in interpreter mode",
			content: `
var USER = "admin"
build: echo "Building for @var(USER)"
`,
			mode: InterpreterMode,
			check: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Execution failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				if len(execResult.Commands) != 1 {
					t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
				}

				if execResult.Commands[0].Name != "build" {
					t.Errorf("Expected command name 'build', got %s", execResult.Commands[0].Name)
				}

				if execResult.Variables["USER"] != "admin" {
					t.Errorf("Expected USER=admin, got %s", execResult.Variables["USER"])
				}
			},
		},
		{
			name: "function decorator in generator mode",
			content: `
var USER = "admin"
build: echo "Building for @var(USER)"
`,
			mode: GeneratorMode,
			check: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Code generation failed: %v", err)
				}

				genResult, ok := result.(*GenerationResult)
				if !ok {
					t.Fatalf("Expected GenerationResult, got %T", result)
				}

				code := genResult.String()

				if !strings.Contains(code, "package main") {
					t.Error("Generated code should contain package declaration")
				}

				if !strings.Contains(code, "USER") {
					t.Error("Generated code should reference USER variable")
				}

				if !strings.Contains(code, "build") {
					t.Error("Generated code should contain build command")
				}
			},
		},
		{
			name: "timeout decorator",
			content: `
timeout_cmd: @timeout(duration=30s) {
    echo "test command"
}
`,
			mode: InterpreterMode,
			check: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Execution failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				if len(execResult.Commands) != 1 {
					t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
				}

				if execResult.Commands[0].Name != "timeout_cmd" {
					t.Errorf("Expected command name 'timeout_cmd', got %s", execResult.Commands[0].Name)
				}
			},
		},
		{
			name: "environment variable decorator",
			content: `
env_test: echo "Home is @env(HOME)"
`,
			mode: InterpreterMode,
			check: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Execution failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				if len(execResult.Commands) != 1 {
					t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the content using Parse
			program, err := parser.Parse(strings.NewReader(tt.content))
			if err != nil {
				t.Fatalf("Failed to parse content: %v", err)
			}

			// Create execution context and engine
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(tt.mode, ctx)

			// Execute or generate
			result, err := engine.Execute(program)

			tt.check(t, result, err)
		})
	}
}

func TestExecutionEngine_DecoratorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "undefined variable in var decorator",
			content: `
test: echo "Value: @var(UNDEFINED_VAR)"
`,
			expectError: true,
			errorMsg:    "not defined",
		},
		{
			name: "missing duration in timeout decorator",
			content: `
test: @timeout echo "test"
`,
			expectError: true,
		},
		{
			name: "valid environment variable decorator with fallback",
			content: `
test: echo "Value: @env(name="NONEXISTENT_VAR", default="fallback")"
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the content
			program, err := parser.Parse(strings.NewReader(tt.content))
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to parse content: %v", err)
			}
			if err != nil && tt.expectError {
				// Expected parse error
				return
			}

			// Create execution context and engine
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)

			// Execute
			_, err = engine.Execute(program)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but execution succeeded")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestExecutionEngine_DecoratorParameterTypes(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    ExecutionMode
	}{
		{
			name: "string parameters",
			content: `
var USER = "admin"
test: echo "User: @var(USER)"
`,
			mode: InterpreterMode,
		},
		{
			name: "duration parameters",
			content: `
test: @timeout(duration=5s) {
    echo "Quick task"
}
`,
			mode: InterpreterMode,
		},
		{
			name: "boolean and number parameters",
			content: `
test: @parallel(concurrency=2, failOnFirstError=true) {
    echo "parallel task"
}
`,
			mode: InterpreterMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the content
			program, err := parser.Parse(strings.NewReader(tt.content))
			if err != nil {
				t.Fatalf("Failed to parse content: %v", err)
			}

			// Create execution context and engine
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(tt.mode, ctx)

			// Execute
			_, err = engine.Execute(program)
			if err != nil {
				t.Errorf("Execution failed: %v", err)
			}
		})
	}
}

func TestExecutionEngine_GeneratorModeWithDecorators(t *testing.T) {
	content := `
var USER = "admin"
var PORT = "8080"

build: echo "Building for @var(USER) on port @var(PORT)"
deploy: @timeout(duration=1m) {
    echo "Deploying application"
}
`

	// Parse the content
	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	// Generate code
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Generator mode execution failed: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	generatedCode := genResult.String()

	if generatedCode == "" {
		t.Error("Expected generated code but got empty string")
	}

	// Verify the generated code contains expected elements
	if !strings.Contains(generatedCode, "USER") {
		t.Error("Generated code should contain USER variable")
	}

	if !strings.Contains(generatedCode, "PORT") {
		t.Error("Generated code should contain PORT variable")
	}

	if !strings.Contains(generatedCode, "timeout") {
		t.Error("Generated code should contain timeout logic")
	}

	if !strings.Contains(generatedCode, "package main") {
		t.Error("Generated code should contain package declaration")
	}
}

func TestExecutionEngine_EnvironmentVariableDecorators(t *testing.T) {
	// Set up test environment variables
	t.Setenv("TEST_HOME", "/test/home")
	t.Setenv("TEST_USER", "testuser")

	content := `
test1: echo "Home: @env(TEST_HOME)"
test2: echo "User: @env(TEST_USER)"
test3: echo "Fallback: @env(name="NONEXISTENT", default="fallback_value")"
`

	// Parse the content
	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(InterpreterMode, ctx)

	// Execute
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	if len(execResult.Commands) != 3 {
		t.Errorf("Expected 3 commands, got %d", len(execResult.Commands))
	}

	// Check that all commands executed successfully
	for i, cmd := range execResult.Commands {
		if cmd.Status != "success" {
			t.Errorf("Command %d (%s) failed: %s", i, cmd.Name, cmd.Error)
		}
	}
}
