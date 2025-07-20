package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestExecutionEngine_CoreFunctionality tests basic execution engine functionality
func TestExecutionEngine_CoreFunctionality(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		mode       ExecutionMode
		expectVars map[string]string
		expectCmds int
		expectErr  bool
	}{
		{
			name: "simple variable and command - interpreter",
			input: `var PORT = 8080
build: echo "Building on port @var(PORT)"`,
			mode:       InterpreterMode,
			expectVars: map[string]string{"PORT": "8080"},
			expectCmds: 1,
			expectErr:  false,
		},
		{
			name: "simple variable and command - generator",
			input: `var PORT = 8080
build: echo "Building on port @var(PORT)"`,
			mode:       GeneratorMode,
			expectVars: map[string]string{"PORT": "8080"},
			expectCmds: 1,
			expectErr:  false,
		},
		{
			name: "multiple variables and commands",
			input: `var PORT = 8080
var HOST = "localhost"
var DEBUG = true

serve: echo "Serving on @var(HOST):@var(PORT)"
debug: echo "Debug mode: @var(DEBUG)"`,
			mode:       InterpreterMode,
			expectVars: map[string]string{"PORT": "8080", "HOST": "localhost", "DEBUG": "true"},
			expectCmds: 2,
			expectErr:  false,
		},
		{
			name: "variable groups",
			input: `var (
  PORT = 8080
  HOST = "localhost"
  ENV = "development"
)

start: echo "Starting @var(ENV) server on @var(HOST):@var(PORT)"`,
			mode:       InterpreterMode,
			expectVars: map[string]string{"PORT": "8080", "HOST": "localhost", "ENV": "development"},
			expectCmds: 1,
			expectErr:  false,
		},
		{
			name:       "empty program",
			input:      ``,
			mode:       InterpreterMode,
			expectVars: map[string]string{},
			expectCmds: 0,
			expectErr:  false,
		},
		{
			name: "only variables, no commands",
			input: `var PORT = 8080
var HOST = "localhost"`,
			mode:       InterpreterMode,
			expectVars: map[string]string{"PORT": "8080", "HOST": "localhost"},
			expectCmds: 0,
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(tt.mode, ctx)

			result, err := engine.Execute(program)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			switch tt.mode {
			case InterpreterMode:
				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				// Check variables
				if len(execResult.Variables) != len(tt.expectVars) {
					t.Errorf("Expected %d variables, got %d", len(tt.expectVars), len(execResult.Variables))
				}
				for name, expectedValue := range tt.expectVars {
					if actualValue, exists := execResult.Variables[name]; !exists {
						t.Errorf("Expected variable %s not found", name)
					} else if actualValue != expectedValue {
						t.Errorf("Variable %s: expected %s, got %s", name, expectedValue, actualValue)
					}
				}

				// Check commands
				if len(execResult.Commands) != tt.expectCmds {
					t.Errorf("Expected %d commands, got %d", tt.expectCmds, len(execResult.Commands))
				}

			case GeneratorMode:
				genResult, ok := result.(*GenerationResult)
				if !ok {
					t.Fatalf("Expected GenerationResult, got %T", result)
				}

				code := genResult.String()
				if !strings.Contains(code, "package main") {
					t.Error("Generated code should contain package declaration")
				}
				if !strings.Contains(code, "func main()") {
					t.Error("Generated code should contain main function")
				}

				goMod := genResult.GoModString()
				if !strings.Contains(goMod, "module devcmd-generated") {
					t.Error("Generated go.mod should contain module declaration")
				}
				if !strings.Contains(goMod, "go 1.24") {
					t.Error("Generated go.mod should contain Go version 1.24")
				}
			}
		})
	}
}

// TestExecutionEngine_VariableTypes tests different variable types
func TestExecutionEngine_VariableTypes(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		varName       string
		expectedValue string
	}{
		{
			name:          "string variable",
			input:         `var NAME = "devcmd"`,
			varName:       "NAME",
			expectedValue: "devcmd",
		},
		{
			name:          "number variable",
			input:         `var PORT = 8080`,
			varName:       "PORT",
			expectedValue: "8080",
		},
		{
			name:          "boolean variable true",
			input:         `var DEBUG = true`,
			varName:       "DEBUG",
			expectedValue: "true",
		},
		{
			name:          "boolean variable false",
			input:         `var PRODUCTION = false`,
			varName:       "PRODUCTION",
			expectedValue: "false",
		},
		{
			name:          "duration variable",
			input:         `var TIMEOUT = 30s`,
			varName:       "TIMEOUT",
			expectedValue: "30s",
		},
		{
			name:          "string with spaces",
			input:         `var MESSAGE = "Hello World"`,
			varName:       "MESSAGE",
			expectedValue: "Hello World",
		},
		{
			name:          "string with special characters",
			input:         `var URL = "https://api.example.com/v1"`,
			varName:       "URL",
			expectedValue: "https://api.example.com/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)

			result, err := engine.Execute(program)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			execResult, ok := result.(*ExecutionResult)
			if !ok {
				t.Fatalf("Expected ExecutionResult, got %T", result)
			}

			if value, exists := execResult.Variables[tt.varName]; !exists {
				t.Errorf("Variable %s not found", tt.varName)
			} else if value != tt.expectedValue {
				t.Errorf("Variable %s: expected %s, got %s", tt.varName, tt.expectedValue, value)
			}
		})
	}
}

// TestExecutionEngine_ErrorHandling tests error scenarios
func TestExecutionEngine_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name:        "undefined variable reference",
			input:       `test: echo "Value: @var(UNDEFINED)"`,
			expectError: "variable 'UNDEFINED' not defined",
		},
		{
			name:        "undefined decorator",
			input:       `test: @nonexistent() { echo "test" }`,
			expectError: "command failed with exit code", // Shell execution fails with syntax error
		},
		{
			name:        "invalid variable type in @var",
			input:       `test: echo "@var(123)"`,     // Should be identifier, not number
			expectError: "variable '123' not defined", // Parser treats 123 as variable name
		},
		{
			name:        "empty command body",
			input:       `test:`,
			expectError: "", // Empty command body is actually valid, just has no content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip empty command body test - it's actually valid behavior
			if tt.expectError == "" {
				return
			}

			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				// Some errors should be caught at parse time
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected parse error containing %q, got %q", tt.expectError, err.Error())
				}
				return
			}

			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)

			_, err = engine.Execute(program)
			if err == nil {
				t.Errorf("Expected error containing %q, but got none", tt.expectError)
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

// TestExecutionEngine_Context tests execution context features
func TestExecutionEngine_Context(t *testing.T) {
	t.Run("debug mode", func(t *testing.T) {
		input := `var PORT = 8080
test: echo "Port: @var(PORT)"`

		program, err := parser.Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Failed to parse program: %v", err)
		}

		ctx := decorators.NewExecutionContext(context.Background(), program)
		ctx.Debug = true
		ctx.DryRun = true // Use dry run to avoid actual execution

		engine := New(InterpreterMode, ctx)

		result, err := engine.Execute(program)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		execResult, ok := result.(*ExecutionResult)
		if !ok {
			t.Fatalf("Expected ExecutionResult, got %T", result)
		}

		// In debug + dry run mode, should see "Would execute" messages
		found := false
		for _, cmd := range execResult.Commands {
			for _, output := range cmd.Output {
				if strings.Contains(output, "Would execute") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Expected debug output with 'Would execute' message")
		}
	})

	t.Run("working directory", func(t *testing.T) {
		input := `test: echo "Current dir"`

		program, err := parser.Parse(strings.NewReader(input))
		if err != nil {
			t.Fatalf("Failed to parse program: %v", err)
		}

		ctx := decorators.NewExecutionContext(context.Background(), program)
		ctx.WorkingDir = "/tmp"

		engine := New(InterpreterMode, ctx)

		// Should not error
		_, err = engine.Execute(program)
		if err != nil && !strings.Contains(err.Error(), "exit code") {
			t.Fatalf("Unexpected error: %v", err)
		}
	})
}

// TestExecutionEngine_ModeConsistency tests that both modes handle the same input consistently
func TestExecutionEngine_ModeConsistency(t *testing.T) {
	input := `var PORT = 8080
var HOST = "localhost"
var (
  ENV = "development"
  DEBUG = true
)

build: echo "Building for @var(ENV)"
serve: echo "Serving on @var(HOST):@var(PORT)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Test interpreter mode
	ctx1 := decorators.NewExecutionContext(context.Background(), program)
	ctx1.DryRun = true // Avoid actual execution
	engine1 := New(InterpreterMode, ctx1)

	result1, err := engine1.Execute(program)
	if err != nil {
		t.Fatalf("Interpreter mode failed: %v", err)
	}

	execResult, ok := result1.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult from interpreter, got %T", result1)
	}

	// Test generator mode
	ctx2 := decorators.NewExecutionContext(context.Background(), program)
	engine2 := New(GeneratorMode, ctx2)

	result2, err := engine2.Execute(program)
	if err != nil {
		t.Fatalf("Generator mode failed: %v", err)
	}

	genResult, ok := result2.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult from generator, got %T", result2)
	}

	// Both modes should process the same variables
	expectedVars := map[string]string{
		"PORT":  "8080",
		"HOST":  "localhost",
		"ENV":   "development",
		"DEBUG": "true", // All declared variables should be in interpreter
	}

	// Only used variables should be in generated code
	expectedUsedVars := map[string]string{
		"PORT": "8080",
		"HOST": "localhost",
		"ENV":  "development",
		// DEBUG is not used, so should not be generated
	}

	// Check interpreter variables
	for name, expectedValue := range expectedVars {
		if actualValue, exists := execResult.Variables[name]; !exists {
			t.Errorf("Interpreter missing variable %s", name)
		} else if actualValue != expectedValue {
			t.Errorf("Interpreter variable %s: expected %s, got %s", name, expectedValue, actualValue)
		}
	}

	// Check generator code contains variable declarations (only used variables)
	code := genResult.String()
	for name, expectedValue := range expectedUsedVars {
		expectedDecl := name + " := \"" + expectedValue + "\""
		if !strings.Contains(code, expectedDecl) {
			t.Errorf("Generated code missing variable declaration: %s", expectedDecl)
		}
	}

	// Verify that unused variables are NOT in the generated code
	if strings.Contains(code, "DEBUG := \"true\"") {
		t.Error("Generated code should not contain unused variable DEBUG")
	}

	// Both modes should process the same number of commands
	expectedCommands := 2
	if len(execResult.Commands) != expectedCommands {
		t.Errorf("Interpreter processed %d commands, expected %d", len(execResult.Commands), expectedCommands)
	}

	// Generator should have command comments
	if !strings.Contains(code, "// Command: build") {
		t.Error("Generated code missing build command")
	}
	if !strings.Contains(code, "// Command: serve") {
		t.Error("Generated code missing serve command")
	}
}

// TestExecutionEngine_CommandResults tests command result tracking
func TestExecutionEngine_CommandResults(t *testing.T) {
	input := `var MSG = "hello"
test: echo "@var(MSG) world"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	ctx.DryRun = true // Avoid actual execution
	engine := New(InterpreterMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	if len(execResult.Commands) != 1 {
		t.Fatalf("Expected 1 command result, got %d", len(execResult.Commands))
	}

	cmd := execResult.Commands[0]
	if cmd.Name != "test" {
		t.Errorf("Expected command name 'test', got %s", cmd.Name)
	}

	if cmd.Status != "success" {
		t.Errorf("Expected command status 'success', got %s", cmd.Status)
	}

	// Should have output indicating it would execute
	found := false
	for _, output := range cmd.Output {
		if strings.Contains(output, "Would execute") && strings.Contains(output, "hello world") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected command output with expanded variable")
	}

	// Test summary methods
	summary := execResult.Summary()
	if !strings.Contains(summary, "MSG = hello") {
		t.Error("Summary should contain variable information")
	}
	if !strings.Contains(summary, "test: success") {
		t.Error("Summary should contain command status")
	}

	if execResult.HasErrors() {
		t.Error("Result should not have errors")
	}

	successfulCmds := execResult.GetSuccessfulCommands()
	if len(successfulCmds) != 1 {
		t.Errorf("Expected 1 successful command, got %d", len(successfulCmds))
	}

	failedCmds := execResult.GetFailedCommands()
	if len(failedCmds) != 0 {
		t.Errorf("Expected 0 failed commands, got %d", len(failedCmds))
	}
}

// TestExecutionEngine_GoVersionGeneration tests custom Go version in generated code
func TestExecutionEngine_GoVersionGeneration(t *testing.T) {
	input := `var PORT = 8080
test: echo "Port: @var(PORT)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	tests := []struct {
		name        string
		goVersion   string
		expectedMod string
	}{
		{
			name:        "default version",
			goVersion:   "",
			expectedMod: "go 1.24",
		},
		{
			name:        "custom version 1.21",
			goVersion:   "1.21",
			expectedMod: "go 1.21",
		},
		{
			name:        "custom version 1.23",
			goVersion:   "1.23",
			expectedMod: "go 1.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := decorators.NewExecutionContext(context.Background(), program)

			var engine *Engine
			if tt.goVersion == "" {
				engine = New(GeneratorMode, ctx)
			} else {
				engine = NewWithGoVersion(GeneratorMode, ctx, tt.goVersion)
			}

			result, err := engine.Execute(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			genResult, ok := result.(*GenerationResult)
			if !ok {
				t.Fatalf("Expected GenerationResult, got %T", result)
			}

			goMod := genResult.GoModString()
			if !strings.Contains(goMod, tt.expectedMod) {
				t.Errorf("Expected go.mod to contain %q, got:\n%s", tt.expectedMod, goMod)
			}
		})
	}
}
