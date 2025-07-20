package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
build: echo "Building on port @var(PORT)"`

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

// TestGeneratorModeSpecialCharacters tests code generation with special characters
// This reproduces the issue from the error handling test
func TestGeneratorModeSpecialCharacters(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:  "valid command without special chars",
			input: `valid: echo "This works"`,
			shouldContain: []string{
				"package main",
				"func main()",
				"// Command: valid",
				"// Shell: echo \"This works\"",
			},
		},
		{
			name:  "command with special characters",
			input: `special-chars: echo "Special: !#\$%^&*()"`,
			shouldContain: []string{
				"package main",
				"func main()",
				"// Command: special-chars",
				"\"echo \\\"Special: !#\\\\$%^&*()\\\"\"", // Properly escaped in Go string
			},
		},
		{
			name:  "unicode command",
			input: `unicode: echo "Hello 世界"`,
			shouldContain: []string{
				"package main",
				"func main()",
				"// Command: unicode",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the program
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			// Create execution context and engine in generator mode
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)

			// Generate code
			result, err := engine.Execute(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			// Verify result type
			genResult, ok := result.(*GenerationResult)
			if !ok {
				t.Fatalf("Expected GenerationResult, got %T", result)
			}

			// Get generated code
			code := genResult.String()
			t.Logf("Generated code for %s:\n%s", tt.name, code)

			// Check required content
			for _, content := range tt.shouldContain {
				if !strings.Contains(code, content) {
					t.Errorf("Generated code should contain %q", content)
				}
			}

			// Check forbidden content
			for _, content := range tt.shouldNotContain {
				if strings.Contains(code, content) {
					t.Errorf("Generated code should not contain %q", content)
				}
			}

			// Most importantly: try to compile the generated Go code by writing it to a temp file
			// This will catch the syntax errors that are causing the CI to fail
			tempFile := t.TempDir() + "/test.go"
			if err := writeAndValidateGoCode(tempFile, code); err != nil {
				t.Errorf("Generated Go code has syntax errors: %v", err)
				t.Logf("Problematic code:\n%s", code)
			}
		})
	}
}

// writeAndValidateGoCode writes Go code to a file and validates it compiles
func writeAndValidateGoCode(filename, code string) error {
	// Write the code to a temporary file
	if err := writeToFile(filename, code); err != nil {
		return err
	}

	// Try to compile it with go build -o /dev/null
	cmd := exec.Command("go", "build", "-o", "/dev/null", filename)
	cmd.Dir = filepath.Dir(filename)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build failed: %v\nOutput: %s", err, output)
	}

	return nil
}

func writeToFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = file.WriteString(content)
	return err
}

func TestParallelDecoratorInterpreter(t *testing.T) {
	// Test parallel decorator in interpreter mode
	input := `test: @parallel {
		echo "Task 1";
		echo "Task 2";
		echo "Task 3"
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

	// Verify result type
	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	// Check that we have one command
	if len(execResult.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(execResult.Commands))
	}

	// Check command executed successfully
	if execResult.Commands[0].Status != "success" {
		t.Errorf("Expected command to succeed, got status: %s", execResult.Commands[0].Status)
	}
}

func TestParallelDecoratorGenerator(t *testing.T) {
	// Test parallel decorator in generator mode
	input := `test: @parallel {
		echo "Task 1";
		echo "Task 2";
		echo "Task 3"
	}`

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
	
	// Print the generated code for inspection
	t.Logf("Generated code:\n%s", code)

	if !strings.Contains(code, "package main") {
		t.Error("Generated code should contain package declaration")
	}

	if !strings.Contains(code, "sync.WaitGroup") {
		t.Error("Generated code should contain WaitGroup for parallel execution")
	}

	if !strings.Contains(code, "go func()") {
		t.Error("Generated code should contain goroutines for parallel execution")
	}

	if !strings.Contains(code, "semaphore") {
		t.Error("Generated code should contain semaphore for concurrency control")
	}
}

func TestParallelDecoratorGeneratedCodeCompiles(t *testing.T) {
	// Test that generated code actually compiles and runs
	input := `test: @parallel {
		echo "Task 1";
		echo "Task 2"
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	// Generate code
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Write generated main.go
	mainFile := filepath.Join(tempDir, "main.go")
	if err := writeToFile(mainFile, genResult.Code.String()); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Write go.mod
	goModFile := filepath.Join(tempDir, "go.mod")
	if err := writeToFile(goModFile, genResult.GoMod.String()); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Test that it compiles
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Generated code failed to compile: %v\nOutput: %s", err, string(output))
		t.Logf("Generated code:\n%s", genResult.Code.String())
	}

	// Test that it runs without error
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Generated code failed to run: %v\nOutput: %s", err, string(output))
		t.Logf("Generated code:\n%s", genResult.Code.String())
	}

	// Check that output contains our expected tasks
	outputStr := string(output)
	if !strings.Contains(outputStr, "Task 1") || !strings.Contains(outputStr, "Task 2") {
		t.Errorf("Expected output to contain 'Task 1' and 'Task 2', got: %s", outputStr)
	}
}

func TestNestedDecoratorGeneratedCodeCompiles(t *testing.T) {
	// Test that nested decorator generated code actually compiles and runs
	input := `test: @parallel {
		echo "Task 1";
		@retry(attempts=2) { echo "Task 2" }
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	// Generate code
	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Create temporary directory for test
	tempDir := t.TempDir()
	
	// Write generated main.go
	mainFile := filepath.Join(tempDir, "main.go")
	if err := writeToFile(mainFile, genResult.Code.String()); err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Write go.mod
	goModFile := filepath.Join(tempDir, "go.mod")
	if err := writeToFile(goModFile, genResult.GoMod.String()); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Test that it compiles
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Nested decorator code failed to compile: %v\nOutput: %s", err, string(output))
		t.Logf("Generated code:\n%s", genResult.Code.String())
		return
	}

	// Test that it runs without error
	cmd = exec.Command("go", "run", ".")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Nested decorator code failed to run: %v\nOutput: %s", err, string(output))
		t.Logf("Generated code:\n%s", genResult.Code.String())
		return
	}

	// Check that output contains our expected tasks
	outputStr := string(output)
	t.Logf("Execution output:\n%s", outputStr)
	
	if !strings.Contains(outputStr, "Task 1") {
		t.Errorf("Expected output to contain 'Task 1', got: %s", outputStr)
	}
	
	if !strings.Contains(outputStr, "Task 2") {
		t.Errorf("Expected output to contain 'Task 2', got: %s", outputStr)
	}
}

func TestComplexNestedDecorators(t *testing.T) {
	// Test multiple levels of nesting with different decorator types
	input := `test: @parallel {
		echo "Task 1";
		@retry(attempts=2) { 
			@when(ENV) {
				production: echo "Conditional Task"
				default: echo "Default task"
			};
			echo "After condition"
		};
		@try {
			main: {
				echo "Try block";
				@timeout(duration=5s) { echo "Timeout task" }
			}
			error: echo "Catch block"
		};
		echo "Task 4"
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Test in generator mode
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Print the generated code to see complex nesting
	code := genResult.String()
	t.Logf("Complex nested decorators code:\n%s", code)

	// Basic structure checks
	if !strings.Contains(code, "sync.WaitGroup") {
		t.Error("Should contain parallel execution")
	}
	
	if !strings.Contains(code, "maxAttempts") {
		t.Error("Should contain retry logic")
	}
}

func TestPatternDecoratorInParallel(t *testing.T) {
	// Test pattern decorators inside parallel
	input := `test: @parallel {
		@when(ENV) {
			production: echo "When production"
			default: echo "When default"
		};
		@try {
			main: echo "Try success"
			error: echo "Try failed"
		}
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	code := genResult.String()
	t.Logf("Pattern decorators in parallel:\n%s", code)

	// Check for parallel structure
	if !strings.Contains(code, "go func()") {
		t.Error("Should contain goroutines for parallel execution")
	}
}

func TestRetryWithPatternDecorator(t *testing.T) {
	// Test retry containing pattern decorators
	input := `test: @retry(attempts=3) {
		@when(ENV) {
			development: echo "Dev environment"
			default: echo "Other environment"
		};
		echo "Always execute";
		@try {
			main: echo "Try operation"
			error: echo "Fallback"
		}
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	code := genResult.String()
	t.Logf("Retry with pattern decorators:\n%s", code)

	// Check for retry structure
	if !strings.Contains(code, "for attempt") {
		t.Error("Should contain retry loop")
	}
}

func TestParallelWithNestedDecorators(t *testing.T) {
	// Test parallel decorator with nested decorators
	input := `test: @parallel {
		echo "Task 1";
		@retry(attempts=2) { echo "Task 2" };
		echo "Task 3"
	}`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	// Test in generator mode to see what code is generated
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Print the generated code to see how nested decorators are handled
	code := genResult.String()
	t.Logf("Generated code with nested decorators:\n%s", code)

	// Check that it contains parallel structure
	if !strings.Contains(code, "sync.WaitGroup") {
		t.Error("Should contain WaitGroup for parallel execution")
	}

	// Check how nested retry decorator is handled
	if !strings.Contains(code, "Task 1") {
		t.Error("Should contain Task 1")
	}
	
	if !strings.Contains(code, "Task 2") {
		t.Error("Should contain Task 2") 
	}
	
	if !strings.Contains(code, "Task 3") {
		t.Error("Should contain Task 3")
	}
}
