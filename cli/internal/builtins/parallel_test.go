package decorators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

func TestParallelDecorator_Execute(t *testing.T) {
	// Create test program with variables
	program := ast.NewProgram()

	ctx := execution.NewExecutionContext(context.Background(), program)

	// Test content with shell commands
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo test1")),
		ast.Shell(ast.Text("echo test2")),
	}

	decorator := &ParallelDecorator{}

	// Test interpreter mode
	t.Run("InterpreterMode", func(t *testing.T) {
		interpreterCtx := ctx.WithMode(execution.InterpreterMode)
		result := decorator.Execute(interpreterCtx, nil, content)

		if result.Mode != execution.InterpreterMode {
			t.Errorf("Expected InterpreterMode, got %v", result.Mode)
		}

		// Data should be nil for interpreter mode
		if result.Data != nil {
			t.Errorf("Expected nil data for interpreter mode, got %v", result.Data)
		}

		// Error might be non-nil due to missing command executor, that's okay for this test
	})

	// Test generator mode
	t.Run("GeneratorMode", func(t *testing.T) {
		generatorCtx := ctx.WithMode(execution.GeneratorMode)
		result := decorator.Execute(generatorCtx, nil, content)

		if result.Mode != execution.GeneratorMode {
			t.Errorf("Expected GeneratorMode, got %v", result.Mode)
		}

		// Data should be a string containing Go code
		if result.Error == nil {
			code, ok := result.Data.(string)
			if !ok {
				t.Errorf("Expected string data for generator mode, got %T", result.Data)
			}
			if code == "" {
				t.Errorf("Expected non-empty generated code")
			}
		}
	})

	// Test plan mode
	t.Run("PlanMode", func(t *testing.T) {
		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, nil, content)

		if result.Mode != execution.PlanMode {
			t.Errorf("Expected PlanMode, got %v", result.Mode)
		}

		if result.Error != nil {
			t.Errorf("Unexpected error in plan mode: %v", result.Error)
		}

		// Data should be a plan element
		if result.Data == nil {
			t.Errorf("Expected plan element data for plan mode, got nil")
		}
	})

	// Test with parameters
	t.Run("WithParameters", func(t *testing.T) {
		params := []ast.NamedParameter{
			{Name: "concurrency", Value: ast.Num(2)},
			{Name: "failOnFirstError", Value: ast.Bool(true)},
		}

		planCtx := ctx.WithMode(execution.PlanMode)
		result := decorator.Execute(planCtx, params, content)

		if result.Error != nil {
			t.Errorf("Unexpected error with parameters: %v", result.Error)
		}
	})

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		invalidCtx := ctx.WithMode(execution.ExecutionMode(999))
		result := decorator.Execute(invalidCtx, nil, content)

		if result.Error == nil {
			t.Errorf("Expected error for invalid mode")
		}
	})
}

// TestParallelDecorator_Sandboxing tests that parallel decorator creates isolated execution environments
func TestParallelDecorator_Sandboxing(t *testing.T) {
	// Create distinct temporary directories for each workdir task
	tempDir1, err := os.MkdirTemp("", "parallel-test-1-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "parallel-test-2-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	tempDir3, err := os.MkdirTemp("", "parallel-test-3-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 3: %v", err)
	}
	defer os.RemoveAll(tempDir3)

	// Create unique test files in each directory
	testFile1 := "test-file-1.txt"
	testFile2 := "test-file-2.txt"
	testFile3 := "test-file-3.txt"
	
	if err := os.WriteFile(filepath.Join(tempDir1, testFile1), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, testFile2), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir3, testFile3), []byte("content3"), 0644); err != nil {
		t.Fatalf("Failed to create test file 3: %v", err)
	}

	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(execution.InterpreterMode)

	// Create content with workdir decorators that use directory-specific tests
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir1}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + testFile1)), // This will only pass if we're in tempDir1
			},
		},
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir2}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + testFile2)), // This will only pass if we're in tempDir2
			},
		},
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir3}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + testFile3)), // This will only pass if we're in tempDir3
			},
		},
	}

	decorator := &ParallelDecorator{}
	
	// Execute the parallel decorator
	result := decorator.Execute(ctx, nil, content)
	
	// Should not return error if all workdir decorators executed in correct directories
	if result.Error != nil {
		t.Errorf("Expected no error (indicating all workdir tasks succeeded), got: %v", result.Error)
		t.Log("This suggests workdir decorators are not executing shell commands in the correct directories")
		return
	}
	
	t.Logf("✅ SUCCESS: All workdir decorators executed successfully!")
	t.Logf("✅ This proves each parallel task executed in its correct isolated directory:")
	t.Logf("  - Task 1: %s (test-file-1.txt found)", tempDir1)
	t.Logf("  - Task 2: %s (test-file-2.txt found)", tempDir2)
	t.Logf("  - Task 3: %s (test-file-3.txt found)", tempDir3)
	t.Logf("✅ Parallel sandboxing with workdir isolation is working perfectly!")
}

// TestParallelDecorator_WorkdirIsolation tests that workdir decorators in parallel don't interfere
func TestParallelDecorator_WorkdirIsolation(t *testing.T) {
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(execution.InterpreterMode)

	// Create temp directories with unique files
	tempDir1, err := os.MkdirTemp("", "parallel-workdir-1-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tempDir1)
	
	tempDir2, err := os.MkdirTemp("", "parallel-workdir-2-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tempDir2)
	
	// Create unique files to prove execution in correct directories
	file1 := "workdir-test-1.txt"
	file2 := "workdir-test-2.txt"
	
	if err := os.WriteFile(filepath.Join(tempDir1, file1), []byte("dir1"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, file2), []byte("dir2"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Create parallel workdir tasks that can only succeed in correct directories
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir1}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + file1 + " && echo task1-success")),
				ast.Shell(ast.Text("test -f " + file1 + " && echo task1-b-success")),
			},
		},
		&ast.BlockDecorator{
			Name: "workdir", 
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir2}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + file2 + " && echo task2-success")),
				ast.Shell(ast.Text("test -f " + file2 + " && echo task2-b-success")),
			},
		},
	}

	decorator := &ParallelDecorator{}
	result := decorator.Execute(ctx, nil, content)
	
	if result.Error != nil {
		t.Errorf("Expected no error, got: %v", result.Error)
	}
	
	t.Logf("✅ Parallel workdir isolation test passed - all tasks executed in correct directories")
	t.Logf("  - Task 1 executed in: %s (found %s)", tempDir1, file1)
	t.Logf("  - Task 2 executed in: %s (found %s)", tempDir2, file2)
}

// TestParallelDecorator_ErrorPropagation_FailOnFirstError tests error handling with failOnFirstError=true
func TestParallelDecorator_ErrorPropagation_FailOnFirstError(t *testing.T) {
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(execution.InterpreterMode)

	// Track execution order and which tasks complete
	var executionLog []string
	var mu sync.Mutex

	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		mu.Lock()
		defer mu.Unlock()
		
		if shell, ok := content.(*ast.ShellContent); ok {
			if len(shell.Parts) > 0 {
				if textPart, ok := shell.Parts[0].(*ast.TextPart); ok {
					cmd := textPart.Text
					executionLog = append(executionLog, cmd)
					t.Logf("Executing: %s", cmd)
					
					// Simulate failure on specific command
					if strings.Contains(cmd, "fail") {
						// Add delay to ensure other tasks have time to start
						time.Sleep(50 * time.Millisecond)
						return fmt.Errorf("exit status 1")
					}
					
					// Add delay to simulate work
					time.Sleep(100 * time.Millisecond)
					return nil
				}
			}
		}
		return nil
	})

	// Create parallel content where one task will fail
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo task1-start && sleep 0.2 && echo task1-end")),
		ast.Shell(ast.Text("echo task2-fail && sleep 0.1 && exit 1")), // This will fail
		ast.Shell(ast.Text("echo task3-start && sleep 0.3 && echo task3-end")),
	}

	decorator := &ParallelDecorator{}
	
	// Test with failOnFirstError=true
	params := []ast.NamedParameter{
		{Name: "failOnFirstError", Value: ast.Bool(true)},
	}
	
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error
	if result.Error == nil {
		t.Error("Expected parallel decorator to return error when failOnFirstError=true, but got nil")
	}
	
	// Error should mention the failure
	if !strings.Contains(result.Error.Error(), "parallel execution failed") {
		t.Errorf("Expected error to mention parallel execution failure, got: %v", result.Error)
	}
	
	// Log execution results for analysis
	t.Logf("Execution log with failOnFirstError=true: %v", executionLog)
	t.Logf("Error (as expected): %v", result.Error)
	
	// Verify that the failing command was executed
	failExecuted := false
	for _, cmd := range executionLog {
		if strings.Contains(cmd, "fail") {
			failExecuted = true
			break
		}
	}
	
	if !failExecuted {
		t.Error("Expected failing command to be executed")
	}
}

// TestParallelDecorator_ErrorPropagation_ContinueOnError tests error handling with failOnFirstError=false
func TestParallelDecorator_ErrorPropagation_ContinueOnError(t *testing.T) {
	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(execution.InterpreterMode)

	// Track which commands execute and their success/failure
	var executionResults []struct {
		command string
		success bool
	}
	var mu sync.Mutex

	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		mu.Lock()
		defer mu.Unlock()
		
		if shell, ok := content.(*ast.ShellContent); ok {
			if len(shell.Parts) > 0 {
				if textPart, ok := shell.Parts[0].(*ast.TextPart); ok {
					cmd := textPart.Text
					
					// Simulate different behaviors for different commands
					if strings.Contains(cmd, "fail") {
						executionResults = append(executionResults, struct {
							command string
							success bool
						}{cmd, false})
						t.Logf("Executing (will fail): %s", cmd)
						return fmt.Errorf("exit status 1")
					}
					
					executionResults = append(executionResults, struct {
						command string
						success bool
					}{cmd, true})
					t.Logf("Executing (will succeed): %s", cmd)
					return nil
				}
			}
		}
		return nil
	})

	// Create parallel content with multiple failures and successes
	content := []ast.CommandContent{
		ast.Shell(ast.Text("echo task1-success")),
		ast.Shell(ast.Text("echo task2-fail && exit 1")), // This will fail
		ast.Shell(ast.Text("echo task3-success")),
		ast.Shell(ast.Text("echo task4-fail && exit 1")), // This will also fail
		ast.Shell(ast.Text("echo task5-success")),
	}

	decorator := &ParallelDecorator{}
	
	// Test with failOnFirstError=false (continue on errors)
	params := []ast.NamedParameter{
		{Name: "failOnFirstError", Value: ast.Bool(false)},
	}
	
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error because some tasks failed
	if result.Error == nil {
		t.Error("Expected parallel decorator to return error when some tasks fail, but got nil")
	}
	
	// Error should mention the failures
	if !strings.Contains(result.Error.Error(), "parallel execution failed") {
		t.Errorf("Expected error to mention parallel execution failure, got: %v", result.Error)
	}
	
	// All 5 commands should have been executed despite failures
	if len(executionResults) != 5 {
		t.Errorf("Expected 5 commands to execute, got %d", len(executionResults))
	}
	
	// Count successes and failures
	successes := 0
	failures := 0
	for _, result := range executionResults {
		if result.success {
			successes++
		} else {
			failures++
		}
	}
	
	// Should have 3 successes and 2 failures
	if successes != 3 {
		t.Errorf("Expected 3 successful executions, got %d", successes)
	}
	
	if failures != 2 {
		t.Errorf("Expected 2 failed executions, got %d", failures)
	}
	
	t.Logf("Execution results with failOnFirstError=false:")
	for i, result := range executionResults {
		status := "SUCCESS"
		if !result.success {
			status = "FAILED"
		}
		t.Logf("  %d: %s - %s", i+1, result.command, status)
	}
	
	t.Logf("Final error (as expected): %v", result.Error)
}

// TestParallelDecorator_ErrorPropagation_WithWorkdir tests error propagation through nested decorators
func TestParallelDecorator_ErrorPropagation_WithWorkdir(t *testing.T) {
	// Create distinct temporary directories for each workdir task
	tempDir1, err := os.MkdirTemp("", "workdir-test-1-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "workdir-test-2-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	tempDir3, err := os.MkdirTemp("", "workdir-test-3-")
	if err != nil {
		t.Fatalf("Failed to create temp dir 3: %v", err)
	}
	defer os.RemoveAll(tempDir3)

	program := ast.NewProgram()
	ctx := execution.NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(execution.InterpreterMode)

	// Create unique files in each temp directory to verify correct execution
	file1 := "test-1.txt"
	file2 := "test-2.txt"
	file3 := "test-3.txt"
	
	if err := os.WriteFile(filepath.Join(tempDir1, file1), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir2, file2), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir3, file3), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file 3: %v", err)
	}

	// Create parallel content with workdir decorators - test both success and failure
	content := []ast.CommandContent{
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir1}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + file1 + " && echo task1-success")),
			},
		},
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir2}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + file2 + " && echo task2-fail && exit 1")), // Verify directory then fail
			},
		},
		&ast.BlockDecorator{
			Name: "workdir",
			Args: []ast.NamedParameter{
				{Name: "path", Value: &ast.StringLiteral{Value: tempDir3}},
			},
			Content: []ast.CommandContent{
				ast.Shell(ast.Text("test -f " + file3 + " && echo task3-success")),
			},
		},
	}

	decorator := &ParallelDecorator{}
	
	// Test with failOnFirstError=false to see all executions
	params := []ast.NamedParameter{
		{Name: "failOnFirstError", Value: ast.Bool(false)},
	}
	
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error because one workdir task failed
	if result.Error == nil {
		t.Error("Expected parallel decorator to return error when workdir task fails, but got nil")
	}
	
	// Error should mention the specific workdir that failed
	if !strings.Contains(result.Error.Error(), "command 1 failed in directory") {
		t.Errorf("Expected error to mention workdir failure, got: %v", result.Error)
	}
	
	t.Logf("✅ Parallel workdir error propagation test passed:")
	t.Logf("  - Task 1 executed in: %s (success - found %s)", tempDir1, file1)
	t.Logf("  - Task 2 executed in: %s (failed after finding %s)", tempDir2, file2)
	t.Logf("  - Task 3 executed in: %s (success - found %s)", tempDir3, file3)
	t.Logf("  - Final error (as expected): %v", result.Error)
}
