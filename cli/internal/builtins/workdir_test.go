package decorators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// TestWorkdirDecorator_ErrorPropagation tests that workdir properly propagates errors
func TestWorkdirDecorator_ErrorPropagation(t *testing.T) {
	// Create a simple program for context
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	// Mock content executor that always fails
	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		return fmt.Errorf("exit status 1")
	})
	
	decorator := &WorkdirDecorator{}
	
	// Create parameters for current directory
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: "."},
		},
	}
	
	// Create content that would fail
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "exit 1"},
			},
		},
	}
	
	// Execute the decorator - should fail
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error
	if result.Error == nil {
		t.Error("Expected workdir decorator to propagate error, but got nil")
	}
	
	// Error should mention the command failure
	if !strings.Contains(result.Error.Error(), "command 1 failed") {
		t.Errorf("Expected error to mention command failure, got: %v", result.Error)
	}
	
	// Should have the correct mode
	if result.Mode != execution.InterpreterMode {
		t.Errorf("Expected mode InterpreterMode, got %v", result.Mode)
	}
}

// TestWorkdirDecorator_DirectoryRestore tests that workdir restores directory on failure
func TestWorkdirDecorator_DirectoryRestore(t *testing.T) {
	// Get original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	// Create a simple program for context
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	// Mock content executor that always fails
	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		return fmt.Errorf("exit status 1")
	})
	
	decorator := &WorkdirDecorator{}
	
	// Create parameters for /tmp directory
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: "/tmp"},
		},
	}
	
	// Create content that would fail
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "exit 1"},
			},
		},
	}
	
	// Execute the decorator - should fail but restore directory
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error
	if result.Error == nil {
		t.Error("Expected workdir decorator to propagate error, but got nil")
	}
	
	// Check that we're back in the original directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory after test: %v", err)
	}
	
	if currentDir != originalDir {
		t.Errorf("Directory not properly restored. Expected: %s, Got: %s", originalDir, currentDir)
	}
	
	t.Logf("Directory properly restored to: %s after workdir failure", currentDir)
}

// TestWorkdirDecorator_Success tests that workdir works correctly on success
func TestWorkdirDecorator_Success(t *testing.T) {
	// Get original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	// Create a simple program for context
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	// Mock content executor that succeeds
	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		return nil // Success
	})
	
	decorator := &WorkdirDecorator{}
	
	// Create parameters for /tmp directory
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: "/tmp"},
		},
	}
	
	// Create content that would succeed
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "echo success"},
			},
		},
	}
	
	// Execute the decorator - should succeed
	result := decorator.Execute(ctx, params, content)
	
	// Should not return an error
	if result.Error != nil {
		t.Errorf("Expected workdir decorator to succeed, but got error: %v", result.Error)
	}
	
	// Check that we're back in the original directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory after test: %v", err)
	}
	
	if currentDir != originalDir {
		t.Errorf("Directory not properly restored. Expected: %s, Got: %s", originalDir, currentDir)
	}
	
	t.Logf("Directory properly restored to: %s after workdir success", currentDir)
}

// TestWorkdirDecorator_IsolatedExecution tests that workdir works correctly 
// when running in an isolated environment (like within @parallel sandboxes)
func TestWorkdirDecorator_IsolatedExecution(t *testing.T) {
	// Get original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	// This test simulates what should happen when workdir runs in an isolated environment
	// Each goroutine represents a separate parallel task with its own working directory state
	
	numIsolatedTasks := 5
	results := make(chan error, numIsolatedTasks)
	
	for i := 0; i < numIsolatedTasks; i++ {
		go func(taskID int) {
			// Each task operates independently - no shared global state
			program := &ast.Program{
				Variables: []ast.VariableDecl{},
				Commands:  []ast.CommandDecl{},
			}
			
			ctx := execution.NewExecutionContext(nil, program)
			ctx = ctx.WithMode(execution.InterpreterMode)
			
			// Mock content executor that succeeds
			ctx.SetContentExecutor(func(content ast.CommandContent) error {
				return nil // Success
			})
			
			decorator := &WorkdirDecorator{}
			
			// Each task tries to use a different directory
			targetDir := "/tmp"
			params := []ast.NamedParameter{
				{
					Name:  "path", 
					Value: &ast.StringLiteral{Value: targetDir},
				},
			}
			
			content := []ast.CommandContent{
				&ast.ShellContent{
					Parts: []ast.ShellPart{
						&ast.TextPart{Text: fmt.Sprintf("echo isolated-task-%d", taskID)},
					},
				},
			}
			
			// Execute - should work without any mutex contention
			result := decorator.Execute(ctx, params, content)
			results <- result.Error
		}(i)
	}
	
	// Collect all results
	var errors []error
	for i := 0; i < numIsolatedTasks; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}
	
	// All tasks should succeed without interference
	if len(errors) > 0 {
		t.Errorf("Errors occurred in isolated execution: %v", errors)
	}
	
	// Verify we're still in the original directory (no interference)
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory after test: %v", err)
	}
	
	if currentDir != originalDir {
		t.Errorf("Directory changed unexpectedly. Expected: %s, Got: %s", originalDir, currentDir)
	}
	
	t.Logf("Successfully ran %d isolated workdir tasks", numIsolatedTasks)
}

// TestWorkdirDecorator_ErrorPropagationInSandbox tests that workdir properly propagates errors in sandboxed environments
func TestWorkdirDecorator_ErrorPropagationInSandbox(t *testing.T) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	decorator := &WorkdirDecorator{}
	
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: "/tmp"},
		},
	}
	
	// Create content with multiple commands, where the second will fail
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "echo success"},
			},
		},
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "exit 1"},
			},
		},
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "echo should not execute"},
			},
		},
	}
	
	// Execute - should fail on second command
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error
	if result.Error == nil {
		t.Error("Expected workdir to propagate error from failed command, but got nil")
	}
	
	// Error should mention the command failure
	if !strings.Contains(result.Error.Error(), "command 2 failed") {
		t.Errorf("Expected error to mention command 2 failure, got: %v", result.Error)
	}
	
	t.Logf("Successfully verified workdir error propagation in sandbox: %v", result.Error)
}

// TestWorkdirDecorator_DirectoryRestoreOnError tests directory restoration in sandboxed environments
func TestWorkdirDecorator_DirectoryRestoreOnError(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	// Mock content executor that always fails
	ctx.SetContentExecutor(func(content ast.CommandContent) error {
		return fmt.Errorf("simulated failure")
	})
	
	decorator := &WorkdirDecorator{}
	
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: "/tmp"},
		},
	}
	
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "failing command"},
			},
		},
	}
	
	// Execute - should fail but restore directory
	result := decorator.Execute(ctx, params, content)
	
	// Should return an error
	if result.Error == nil {
		t.Error("Expected workdir to propagate error, but got nil")
	}
	
	// Directory should be restored even on failure
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory after test: %v", err)
	}
	
	if currentDir != originalDir {
		t.Errorf("Directory not properly restored after error. Expected: %s, Got: %s", originalDir, currentDir)
	}
	
	t.Logf("Successfully verified directory restoration on error: %v", result.Error)
}

// TestWorkdirDecorator_ContextBasedExecution tests that workdir executes shell commands in the correct directory
func TestWorkdirDecorator_ContextBasedExecution(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "workdir-context-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test file in the temp directory
	testFile := "workdir-test.txt"
	testFilePath := filepath.Join(tempDir, testFile)
	if err := os.WriteFile(testFilePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	decorator := &WorkdirDecorator{}
	
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: tempDir},
		},
	}
	
	// Use a command that will only succeed if executed in the correct directory
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "test -f " + testFile},
			},
		},
	}
	
	// Execute the workdir decorator
	result := decorator.Execute(ctx, params, content)
	
	// Should succeed only if executed in the correct directory
	if result.Error != nil {
		t.Errorf("Expected workdir to succeed (indicating correct directory execution), but got error: %v", result.Error)
	}
	
	t.Logf("✅ Workdir decorator successfully executed shell command in target directory: %s", tempDir)
}

// TestWorkdirDecorator_WorkingDirPropagation tests that shell commands execute in the specified directory
func TestWorkdirDecorator_WorkingDirPropagation(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "workdir-propagation-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a unique file in the temp directory
	uniqueFile := "unique-workdir-file.txt"
	uniqueFilePath := filepath.Join(tempDir, uniqueFile)
	if err := os.WriteFile(uniqueFilePath, []byte("unique content"), 0644); err != nil {
		t.Fatalf("Failed to create unique file: %v", err)
	}
	
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.InterpreterMode)
	
	decorator := &WorkdirDecorator{}
	
	params := []ast.NamedParameter{
		{
			Name:  "path", 
			Value: &ast.StringLiteral{Value: tempDir},
		},
	}
	
	// Command that will only pass if executed in the temp directory
	content := []ast.CommandContent{
		&ast.ShellContent{
			Parts: []ast.ShellPart{
				&ast.TextPart{Text: "test -f " + uniqueFile + " && echo 'Found unique file'"},
			},
		},
	}
	
	// Execute the workdir decorator
	result := decorator.Execute(ctx, params, content)
	
	// Should succeed only if executed in the correct directory
	if result.Error != nil {
		t.Errorf("Expected workdir to succeed (command should find unique file), but got error: %v", result.Error)
	}
	
	t.Logf("✅ Workdir decorator successfully executed command in target directory: %s", tempDir)
}

// TestWorkdirDecorator_ParallelExecution tests that multiple workdir decorators
// can run in parallel without race conditions
func TestWorkdirDecorator_ParallelExecution(t *testing.T) {
	// Get original working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	
	// Create a simple program for context
	program := &ast.Program{
		Variables: []ast.VariableDecl{},
		Commands:  []ast.CommandDecl{},
	}
	
	// Number of parallel workdir operations to test
	numWorkers := 10
	
	// Channel to collect results
	results := make(chan error, numWorkers)
	
	// Launch multiple workdir decorators in parallel
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			ctx := execution.NewExecutionContext(nil, program)
			ctx = ctx.WithMode(execution.InterpreterMode)
			
			// Mock content executor that succeeds
			ctx.SetContentExecutor(func(content ast.CommandContent) error {
				return nil // Success
			})
			
			decorator := &WorkdirDecorator{}
			
			// Each worker uses a different target directory (but /tmp should exist on most systems)
			targetDir := "/tmp"
			params := []ast.NamedParameter{
				{
					Name:  "path", 
					Value: &ast.StringLiteral{Value: targetDir},
				},
			}
			
			// Create content
			content := []ast.CommandContent{
				&ast.ShellContent{
					Parts: []ast.ShellPart{
						&ast.TextPart{Text: fmt.Sprintf("echo worker-%d", workerID)},
					},
				},
			}
			
			// Execute the decorator
			result := decorator.Execute(ctx, params, content)
			results <- result.Error
		}(i)
	}
	
	// Collect all results
	var errors []error
	for i := 0; i < numWorkers; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}
	
	// Check that we're back in the original directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory after parallel test: %v", err)
	}
	
	if currentDir != originalDir {
		t.Errorf("Directory not properly restored after parallel execution. Expected: %s, Got: %s", originalDir, currentDir)
	}
	
	// Check that no errors occurred
	if len(errors) > 0 {
		t.Errorf("Errors occurred during parallel execution: %v", errors)
	}
	
	t.Logf("Successfully ran %d parallel workdir operations", numWorkers)
	t.Logf("Directory properly restored to: %s after parallel execution", currentDir)
}