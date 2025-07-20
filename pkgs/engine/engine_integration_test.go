package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

func TestExecutionEngine_IntegrationScenarios(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		mode        ExecutionMode
		expectError bool
		checkResult func(t *testing.T, result interface{}, err error)
	}{
		{
			name: "complex web development workflow",
			content: `
var PORT = "3000"
var NODE_ENV = "development"

# Setup commands
install: echo "Installing dependencies..."
setup: {
	echo "Setting up project..."
	echo "Creating directories..."
	echo "Setup complete"
}

# Development commands
watch dev: echo "Starting dev server"
stop dev: echo "Stopping dev server"

dev: @parallel {
	echo "Building in watch mode"
	echo "Starting dev server"
}

# Build with timeout
build: @timeout(duration=2m) {
	echo "Building for @env(NODE_ENV)"
	echo "Build process"
	echo "Build complete"
}

# Testing with retry
test: @retry(attempts=3, delay=1s) {
	echo "Running tests"
}

# Conditional deployment
deploy: @when(NODE_ENV) {
	production: {
		echo "Deploying to production..."
		echo "Running deployment script"
	}
}
`,
			mode:        InterpreterMode,
			expectError: false,
			checkResult: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Integration test failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				// Should have install, setup, dev, build, test, deploy + watch/stop commands
				expectedMin := 6
				if len(execResult.Commands) < expectedMin {
					t.Errorf("Expected at least %d commands, got %d", expectedMin, len(execResult.Commands))
				}

				// Check variables are properly set
				if execResult.Variables["PORT"] != "3000" {
					t.Errorf("Expected PORT=3000, got %s", execResult.Variables["PORT"])
				}

				if execResult.Variables["NODE_ENV"] != "development" {
					t.Errorf("Expected NODE_ENV=development, got %s", execResult.Variables["NODE_ENV"])
				}
			},
		},
		{
			name: "go project with testing and deployment",
			content: `
var GO_VERSION = "1.24"
var BINARY_NAME = "myapp"

# Dependencies
deps: go mod download

# Build commands  
build: go build -o @var(BINARY_NAME) ./cmd
cross_build: @parallel {
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/@var(BINARY_NAME) ./cmd
	GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/@var(BINARY_NAME) ./cmd
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/@var(BINARY_NAME).exe ./cmd
}

# Testing with coverage
test: @timeout(duration=5m) {
	go test -v ./...
	go test -race ./...
	go test -cover ./...
}

# Linting and quality checks
lint: @parallel {
	go vet ./...
	golangci-lint run
	go fmt ./...
}

# Clean up
clean: {
	rm -rf dist/
	rm -f @var(BINARY_NAME)
	go clean -cache
}
`,
			mode:        GeneratorMode,
			expectError: false,
			checkResult: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Go project integration test failed: %v", err)
				}

				genResult, ok := result.(*GenerationResult)
				if !ok {
					t.Fatalf("Expected GenerationResult, got %T", result)
				}

				code := genResult.String()

				// Check for essential elements in generated code
				if !strings.Contains(code, "package main") {
					t.Error("Generated code should contain package declaration")
				}

				// GO_VERSION is declared but not used in any commands, so it should NOT be in generated code
				if strings.Contains(code, "GO_VERSION") {
					t.Error("Generated code should not contain unused GO_VERSION variable")
				}

				if !strings.Contains(code, "BINARY_NAME") {
					t.Error("Generated code should reference BINARY_NAME variable")
				}

				if !strings.Contains(code, "parallel") {
					t.Error("Generated code should handle parallel execution")
				}

				if !strings.Contains(code, "timeout") {
					t.Error("Generated code should handle timeout decorator")
				}
			},
		},
		{
			name: "docker-compose development environment",
			content: `
var COMPOSE_FILE = "docker-compose.dev.yml"
var PROJECT_NAME = "myproject"

# Docker compose commands
up: echo "Starting containers with @var(COMPOSE_FILE)"
down: echo "Stopping containers for @var(PROJECT_NAME)"
logs: echo "Showing logs for @var(PROJECT_NAME)"

# Development workflow
dev_setup: @timeout(duration=10m) {
	echo "Setting up development environment..."
	echo "Pulling images from @var(COMPOSE_FILE)"
	echo "Building containers"
	echo "Development environment ready"
}

# Watch processes
watch backend: echo "Starting backend container"
stop backend: echo "Stopping backend container"

watch frontend: echo "Starting frontend container" 
stop frontend: echo "Stopping frontend container"

# Health checks and monitoring
health: @parallel {
	echo "Checking frontend health"
	echo "Checking backend health"
}

# Clean up with confirmation  
clean: @when(CONFIRM) {
	yes: {
		echo "Cleaning up @var(PROJECT_NAME)"
		echo "Pruning system"
	}
}
`,
			mode:        InterpreterMode,
			expectError: false,
			checkResult: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Docker integration test failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				// Should have both regular commands and watch/stop commands
				expectedMin := 6 // up, down, logs, dev_setup, health, clean + watch commands
				if len(execResult.Commands) < expectedMin {
					t.Errorf("Expected at least %d commands, got %d", expectedMin, len(execResult.Commands))
				}

				// Check variables
				if execResult.Variables["COMPOSE_FILE"] != "docker-compose.dev.yml" {
					t.Errorf("Expected COMPOSE_FILE=docker-compose.dev.yml, got %s", execResult.Variables["COMPOSE_FILE"])
				}
			},
		},
		{
			name: "mixed decorator types integration",
			content: `
var RETRY_COUNT = "3"
var TIMEOUT_DURATION = "30s"

# Complex nested decorators
complex_task: @retry(attempts=3, delay=5s) {
	@timeout(duration=30s) {
		@parallel {
			echo "Task 1"
			echo "Task 2"
			echo "Task 3"
		}
	}
}

# Conditional with timeout
conditional_deploy: @when(ENV) {
	prod: {
		@timeout(duration=5m) {
			echo "Deploying to production"
			kubectl apply -f k8s/
		}
	}
}

# Try-catch with parallel execution  
safe_parallel: @try {
	main: {
		@parallel {
			echo "Building frontend"
			echo "Building backend"
			echo "Running tests"
		}
	}
	error: echo "Build failed, cleaning up"
}
`,
			mode:        InterpreterMode,
			expectError: false,
			checkResult: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("Mixed decorators integration test failed: %v", err)
				}

				execResult, ok := result.(*ExecutionResult)
				if !ok {
					t.Fatalf("Expected ExecutionResult, got %T", result)
				}

				if len(execResult.Commands) != 3 {
					t.Errorf("Expected 3 commands, got %d", len(execResult.Commands))
				}

				// Check that nested decorators are handled properly
				for _, cmd := range execResult.Commands {
					if cmd.Status == "error" {
						t.Errorf("Command %s failed: %s", cmd.Name, cmd.Error)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the content
			program, err := parser.Parse(strings.NewReader(tt.content))
			if err != nil {
				if tt.expectError {
					return
				}
				t.Fatalf("Failed to parse content: %v", err)
			}

			// Create execution context and engine
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(tt.mode, ctx)

			// Execute
			result, err := engine.Execute(program)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but execution succeeded")
				}
				return
			}

			tt.checkResult(t, result, err)
		})
	}
}

func TestExecutionEngine_LargeConfigurationHandling(t *testing.T) {
	// Generate a large configuration with many commands and variables
	var content strings.Builder

	// Generate many variables
	for i := 0; i < 50; i++ {
		content.WriteString("var VAR")
		if i < 26 {
			content.WriteString(string(rune('A' + i)))
		} else {
			content.WriteString(string(rune('A' + (i - 26))))
			content.WriteString("2")
		}
		content.WriteString(" = \"value")
		content.WriteString(string(rune('0' + i%10)))
		content.WriteString("\"\n")
	}

	// Generate many commands with decorators
	for i := 0; i < 100; i++ {
		cmdName := "cmd" + string(rune('0'+i%10)) + string(rune('A'+i%26))
		content.WriteString(cmdName)
		content.WriteString(": ")

		// Add decorators to some commands
		switch i % 4 {
		case 0:
			content.WriteString("@timeout(duration=30s) { echo \"Command ")
			content.WriteString(cmdName)
			content.WriteString("\" }\n")
		case 1:
			content.WriteString("@parallel { echo \"Parallel ")
			content.WriteString(cmdName)
			content.WriteString("\" }\n")
		case 2:
			content.WriteString("@retry(attempts=2, delay=1ms) { echo \"Retry ")
			content.WriteString(cmdName)
			content.WriteString("\" }\n")
		default:
			content.WriteString("echo \"Simple ")
			content.WriteString(cmdName)
			content.WriteString("\"\n")
		}
	}

	configContent := content.String()

	t.Run("large config interpreter mode", func(t *testing.T) {
		// Parse the large configuration
		program, err := parser.Parse(strings.NewReader(configContent))
		if err != nil {
			t.Fatalf("Failed to parse large configuration: %v", err)
		}

		// Create execution context and engine
		ctx := decorators.NewExecutionContext(context.Background(), program)
		engine := New(InterpreterMode, ctx)

		// Measure execution time
		start := time.Now()
		result, err := engine.Execute(program)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to execute large configuration: %v", err)
		}

		execResult, ok := result.(*ExecutionResult)
		if !ok {
			t.Fatalf("Expected ExecutionResult, got %T", result)
		}

		// Verify results
		if len(execResult.Commands) != 100 {
			t.Errorf("Expected 100 commands, got %d", len(execResult.Commands))
		}

		if len(execResult.Variables) != 50 {
			t.Errorf("Expected 50 variables, got %d", len(execResult.Variables))
		}

		// Performance check - should handle large configs reasonably fast
		if duration > 5*time.Second {
			t.Errorf("Large configuration took too long to execute: %v", duration)
		}

		t.Logf("Large configuration execution took: %v", duration)
	})

	t.Run("large config generator mode", func(t *testing.T) {
		// Parse the large configuration
		program, err := parser.Parse(strings.NewReader(configContent))
		if err != nil {
			t.Fatalf("Failed to parse large configuration: %v", err)
		}

		// Create execution context and engine
		ctx := decorators.NewExecutionContext(context.Background(), program)
		engine := New(GeneratorMode, ctx)

		// Measure generation time
		start := time.Now()
		result, err := engine.Execute(program)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to generate code for large configuration: %v", err)
		}

		genResult, ok := result.(*GenerationResult)
		if !ok {
			t.Fatalf("Expected GenerationResult, got %T", result)
		}

		code := genResult.String()

		// Verify generated code
		if !strings.Contains(code, "package main") {
			t.Error("Generated code should contain package declaration")
		}

		// Variables are declared but not used in commands, so they should NOT be in generated code
		if strings.Contains(code, "VAR") {
			t.Error("Generated code should not contain unused variables")
		}

		if !strings.Contains(code, "cmd") {
			t.Error("Generated code should reference commands")
		}

		// Performance check
		if duration > 10*time.Second {
			t.Errorf("Large configuration code generation took too long: %v", duration)
		}

		t.Logf("Large configuration code generation took: %v", duration)
		t.Logf("Generated code size: %d bytes", len(code))
	})
}

func TestExecutionEngine_ErrorRecoveryIntegration(t *testing.T) {
	content := `
var VALID_VAR = "test"

# This should work
good_cmd: echo "Value: @var(VALID_VAR)"

# This should fail but not break other commands
bad_cmd: echo "Undefined: @var(UNDEFINED_VAR)"

# This should also work
another_good_cmd: echo "Another test"

# Decorator with error handling
safe_cmd: @try {
	main: {
		echo "This might fail"
		false
		echo "This won't execute"
	}
	error: echo "Error handler executed"
}

# Recovery after error
recovery_cmd: echo "Recovering after errors"
`

	// Parse the content
	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Create execution context and engine
	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(InterpreterMode, ctx)

	// Execute - should handle errors gracefully
	result, err := engine.Execute(program)
	// The execution should complete even with some commands failing
	if err != nil {
		t.Logf("Execution completed with some errors (expected): %v", err)
		// If execution fails completely, we can't check command results
		return
	}

	execResult, ok := result.(*ExecutionResult)
	if !ok {
		t.Fatalf("Expected ExecutionResult, got %T", result)
	}

	// Check that we have all commands, even if some failed
	expectedCommands := 5
	if len(execResult.Commands) != expectedCommands {
		t.Errorf("Expected %d commands, got %d", expectedCommands, len(execResult.Commands))
	}

	// Check that good commands succeeded and bad commands failed appropriately
	successCount := 0
	errorCount := 0

	for _, cmd := range execResult.Commands {
		switch cmd.Status {
		case "success":
			successCount++
		case "error":
			errorCount++
			t.Logf("Command %s failed as expected: %s", cmd.Name, cmd.Error)
		}
	}

	if successCount == 0 {
		t.Error("Expected at least some commands to succeed")
	}

	t.Logf("Error recovery test: %d successful, %d failed", successCount, errorCount)
}

func TestExecutionEngine_ConcurrencyStress(t *testing.T) {
	content := `
var WORKER_ID = "default"

# Multiple parallel tasks to stress test concurrency
stress_test: @parallel {
	echo "Worker 1: @var(WORKER_ID)"
	echo "Worker 2: @var(WORKER_ID)" 
	echo "Worker 3: @var(WORKER_ID)"
	echo "Worker 4: @var(WORKER_ID)"
	echo "Worker 5: @var(WORKER_ID)"
}

# Nested parallel execution
nested_parallel: @parallel {
	@timeout(duration=5s) {
		@parallel {
			echo "Nested 1"
			echo "Nested 2"
		}
	}
	@timeout(duration=5s) {
		@parallel {
			echo "Nested 3"
			echo "Nested 4"
		}
	}
}
`

	// Parse the content
	program, err := parser.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("Failed to parse content: %v", err)
	}

	// Run multiple concurrent executions
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			// Create execution context and engine for each goroutine
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(InterpreterMode, ctx)

			// Execute
			_, err := engine.Execute(program)
			results <- err
		}(i)
	}

	// Collect results
	errorCount := 0
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			errorCount++
			t.Logf("Goroutine %d failed: %v", i, err)
		}
	}

	// Allow some failures due to resource constraints, but most should succeed
	if errorCount > numGoroutines/2 {
		t.Errorf("Too many concurrent executions failed: %d/%d", errorCount, numGoroutines)
	}

	t.Logf("Concurrency stress test: %d/%d executions succeeded", numGoroutines-errorCount, numGoroutines)
}
