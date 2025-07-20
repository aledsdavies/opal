package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

func TestEngineInterpreterMode(t *testing.T) {
	// Parse a simple program
	input := `var PORT = 8080
build: echo "Building on port $PORT"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(InterpreterMode, ctx)

	// Execute the program
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	}

	// Verify result type
	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	// Check variables were processed
	if len(execResult.Variables) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(execResult.Variables))
	}

	if execResult.Variables["PORT"] != "8080" {
		t.Errorf("Expected PORT=8080, got %s", execResult.Variables["PORT"])
	}

	// Check commands were processed
	if len(execResult.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
	}

	if execResult.Commands[0].Name != "build" {
		t.Errorf("Expected command name 'build', got %s", execResult.Commands[0].Name)
	}
}

func TestEngineVariableResolutionWithDecorators(t *testing.T) {
	// This test reproduces the issue where @var() decorators can't find variables
	input := `var PORT = "3000"
serve: echo "Server starting on port @var(PORT)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(InterpreterMode, ctx)

	// Execute the program - this should not fail
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to execute program with variable resolution: %v", err)
	}

	// Verify result type
	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	// Check that the variable was properly resolved
	if len(execResult.Variables) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(execResult.Variables))
	}

	if execResult.Variables["PORT"] != "3000" {
		t.Errorf("Expected PORT=3000, got %s", execResult.Variables["PORT"])
	}

	// The command should have been executed successfully
	if len(execResult.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
	}
}

func TestEngineGeneratorMode(t *testing.T) {
	// Parse a simple program
	input := `var PORT = 8080
build: echo "Building on port $PORT"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	// Generate code for the program
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Verify result type
	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Check generated code contains expected elements
	code := genResult.String()

	if !strings.Contains(code, "package main") {
		t.Error("Generated code should contain package declaration")
	}

	if !strings.Contains(code, "func main()") {
		t.Error("Generated code should contain main function")
	}

	if !strings.Contains(code, "PORT := \"8080\"") {
		t.Error("Generated code should contain variable declaration")
	}

	if !strings.Contains(code, "// Command: build") {
		t.Error("Generated code should contain command comment")
	}
}

func TestEngineWithDecorators(t *testing.T) {
	// Parse a program with decorators
	input := `var USER = "admin"
deploy: {
  @timeout(30s) {
    echo "Deploying as @var(USER)"
  }
}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(InterpreterMode, ctx)

	// Execute the program
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	}

	// Verify result
	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	// Check variables were processed
	if execResult.Variables["USER"] != "admin" {
		t.Errorf("Expected USER=admin, got %s", execResult.Variables["USER"])
	}

	// Check commands were processed
	if len(execResult.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
	}

	if execResult.Commands[0].Name != "deploy" {
		t.Errorf("Expected command name 'deploy', got %s", execResult.Commands[0].Name)
	}
}

func TestExecutionResultSummary(t *testing.T) {
	result := &ExecutionResult{
		Variables: map[string]string{
			"PORT": "8080",
			"ENV":  "dev",
		},
		Commands: []CommandResult{
			{Name: "build", Status: "success"},
			{Name: "test", Status: "failed", Error: "test failed"},
		},
	}

	summary := result.Summary()

	if !strings.Contains(summary, "PORT = 8080") {
		t.Error("Summary should contain PORT variable")
	}

	if !strings.Contains(summary, "build: success") {
		t.Error("Summary should contain build command status")
	}

	if !strings.Contains(summary, "test: failed") {
		t.Error("Summary should contain test command status")
	}

	// Test error checking
	if !result.HasErrors() {
		t.Error("Result should have errors")
	}

	failedCommands := result.GetFailedCommands()
	if len(failedCommands) != 1 {
		t.Errorf("Expected 1 failed command, got %d", len(failedCommands))
	}
}
