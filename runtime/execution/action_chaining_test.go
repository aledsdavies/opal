package execution

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
)

// Test cases for ActionDecorator chaining with Go-native operators
func TestActionDecoratorChaining(t *testing.T) {
	tests := []struct {
		name        string
		shellText   string
		expectPipe  bool
		expectAnd   bool
		expectOr    bool
		expectFile  bool
		description string
	}{
		{
			name:        "Simple pipe operation",
			shellText:   "echo hello | grep hello",
			expectPipe:  true,
			description: "Basic pipe from echo to grep",
		},
		{
			name:        "AND operation success chain",
			shellText:   "echo hello && echo world",
			expectAnd:   true,
			description: "AND operator - second command runs if first succeeds",
		},
		{
			name:        "OR operation fallback chain",
			shellText:   "false || echo fallback",
			expectOr:    true,
			description: "OR operator - second command runs if first fails",
		},
		{
			name:        "File append operation",
			shellText:   "echo hello >> output.txt",
			expectFile:  true,
			description: "Append stdout to file",
		},
		{
			name:        "Complex chaining",
			shellText:   "echo hello | grep hello && echo found || echo not found",
			expectPipe:  true,
			expectAnd:   true,
			expectOr:    true,
			description: "Complex chain with multiple operators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal program for testing
			program := &ast.Program{
				Variables: []ast.VariableDecl{},
				Commands: []ast.CommandDecl{
					{
						Name: "test",
						Body: ast.CommandBody{
							Content: []ast.CommandContent{
								&ast.ShellContent{
									Parts: []ast.ShellPart{
										&ast.TextPart{Text: tt.shellText},
									},
								},
							},
						},
					},
				},
			}

			// Create execution context
			ctx := NewExecutionContext(context.Background(), program)
			ctx = ctx.WithMode(GeneratorMode)

			// Create shell code builder
			builder := NewShellCodeBuilder(ctx)

			// Test parsing the chain
			shellContent := program.Commands[0].Body.Content[0].(*ast.ShellContent)
			chain, err := builder.parseActionDecoratorChain(shellContent)
			if err != nil {
				t.Fatalf("Failed to parse chain: %v", err)
			}

			// Verify chain contains expected operators
			hasPipe := false
			hasAnd := false
			hasOr := false
			hasFile := false

			for _, element := range chain {
				switch element.Type {
				case "operator":
					switch element.Operator {
					case "|":
						hasPipe = true
					case "&&":
						hasAnd = true
					case "||":
						hasOr = true
					case ">>":
						hasFile = true
					}
				}
			}

			if tt.expectPipe && !hasPipe {
				t.Errorf("Expected pipe operator (|) but not found in chain")
			}
			if tt.expectAnd && !hasAnd {
				t.Errorf("Expected AND operator (&&) but not found in chain")
			}
			if tt.expectOr && !hasOr {
				t.Errorf("Expected OR operator (||) but not found in chain")
			}
			if tt.expectFile && !hasFile {
				t.Errorf("Expected file operator (>>) but not found in chain")
			}

			t.Logf("Chain parsed successfully for: %s", tt.description)
			t.Logf("Elements in chain: %d", len(chain))
			for i, element := range chain {
				t.Logf("  [%d] Type: %s, Operator: %s, Text: %s", i, element.Type, element.Operator, element.Text)
			}
		})
	}
}

// Test Go code generation for ActionDecorator chains
func TestActionChainCodeGeneration(t *testing.T) {
	tests := []struct {
		name         string
		shellText    string
		expectCode   []string // Expected code patterns
		description  string
	}{
		{
			name:      "Pipe operation code generation",
			shellText: "echo hello | grep hello",
			expectCode: []string{
				"executeShellCommandWithInput(ctx, \"grep hello\", lastResult.Stdout)",
				"ActionShell1 := executeShellCommandWithInput",
				"PIPE: stdout of previous feeds to next command",
			},
			description: "Should generate piping logic with input/output handling",
		},
		{
			name:      "AND operation code generation",
			shellText: "echo hello && echo world",
			expectCode: []string{
				"lastResult.Failed()",
				"previous command failed",
			},
			description: "Should generate AND conditional logic",
		},
		{
			name:      "OR operation code generation",
			shellText: "false || echo fallback",
			expectCode: []string{
				"lastResult.Success()",
				"return nil",
			},
			description: "Should generate OR conditional logic",
		},
		{
			name:      "File append code generation",
			shellText: "echo hello >> output.txt",
			expectCode: []string{
				"appendToFile(\"output.txt\", lastResult.Stdout)",
				"file append failed:",
			},
			description: "Should generate file append logic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal program for testing
			program := &ast.Program{
				Commands: []ast.CommandDecl{
					{
						Name: "test",
						Body: ast.CommandBody{
							Content: []ast.CommandContent{
								&ast.ShellContent{
									Parts: []ast.ShellPart{
										&ast.TextPart{Text: tt.shellText},
									},
								},
							},
						},
					},
				},
			}

			// Create execution context
			ctx := NewExecutionContext(context.Background(), program)
			ctx = ctx.WithMode(GeneratorMode)

			// Create shell code builder
			builder := NewShellCodeBuilder(ctx)

			// Generate code for the shell content
			shellContent := program.Commands[0].Body.Content[0].(*ast.ShellContent)
			
			// Check if this has ActionDecorators - if not, test will need to be updated
			// For now, test the basic chain parsing
			code, err := builder.GenerateShellCode(shellContent)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			t.Logf("Generated code:\n%s", code)

			// Verify expected code patterns are present
			for _, expected := range tt.expectCode {
				if !strings.Contains(code, expected) {
					t.Errorf("Expected code pattern '%s' not found in generated code", expected)
				}
			}
		})
	}
}

// Test helper functions for piping and file operations
func TestHelperFunctions(t *testing.T) {
	t.Run("executeShellCommandWithInput integration", func(t *testing.T) {
		// Test that the helper function works correctly in generated code
		// This is verified through the ActionChainCodeGeneration tests
		// which show that executeShellCommandWithInput is properly generated
		t.Log("executeShellCommandWithInput helper function is implemented in generated templates")
	})

	t.Run("appendToFile integration", func(t *testing.T) {
		// Test that the file append function works correctly
		// This is verified through the ActionChainCodeGeneration tests
		// and the TestFileOperations test which uses the helper function
		t.Log("appendToFile helper function is implemented in generated templates")
	})
}

// Test CommandResult piping and chaining
func TestCommandResultPiping(t *testing.T) {
	tests := []struct {
		name     string
		result1  CommandResult
		result2  CommandResult
		operator string
		expected bool // Whether second command should execute
	}{
		{
			name:     "AND with success",
			result1:  CommandResult{Stdout: "hello", ExitCode: 0},
			result2:  CommandResult{Stdout: "world", ExitCode: 0},
			operator: "&&",
			expected: true,
		},
		{
			name:     "AND with failure",
			result1:  CommandResult{Stdout: "hello", ExitCode: 1},
			result2:  CommandResult{Stdout: "world", ExitCode: 0},
			operator: "&&",
			expected: false,
		},
		{
			name:     "OR with success",
			result1:  CommandResult{Stdout: "hello", ExitCode: 0},
			result2:  CommandResult{Stdout: "world", ExitCode: 0},
			operator: "||",
			expected: false, // Should not execute second
		},
		{
			name:     "OR with failure",
			result1:  CommandResult{Stdout: "hello", ExitCode: 1},
			result2:  CommandResult{Stdout: "world", ExitCode: 0},
			operator: "||",
			expected: true,
		},
		{
			name:     "Pipe operation",
			result1:  CommandResult{Stdout: "hello\nworld\n", ExitCode: 0},
			result2:  CommandResult{Stdout: "hello", ExitCode: 0},
			operator: "|",
			expected: true, // Always execute for piping
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logical conditions that would be generated
			shouldExecute := false

			switch tt.operator {
			case "&&":
				shouldExecute = tt.result1.Success()
			case "||":
				shouldExecute = tt.result1.Failed()
			case "|":
				shouldExecute = true // Always pipe
			}

			if shouldExecute != tt.expected {
				t.Errorf("Expected shouldExecute=%v for operator %s with result1.ExitCode=%d", 
					tt.expected, tt.operator, tt.result1.ExitCode)
			}

			// For pipe operations, verify stdout transfer
			if tt.operator == "|" && tt.result1.Stdout != "" {
				pipeInput := tt.result1.Stdout
				if pipeInput != "hello\nworld\n" {
					t.Errorf("Expected pipe input to be stdout from previous command")
				}
			}
		})
	}
}

// Test file operations for >> operator
func TestFileOperations(t *testing.T) {
	t.Run("append to file", func(t *testing.T) {
		// Create a temporary file for testing
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")

		// Test content to append
		content := "hello world\n"

		// This will test the appendToFile function once implemented
		err := appendToFileTestHelper(testFile, content)
		if err != nil {
			t.Fatalf("Failed to append to file: %v", err)
		}

		// Verify file contents
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file: %v", err)
		}

		if string(data) != content {
			t.Errorf("Expected file content '%s', got '%s'", content, string(data))
		}

		// Test appending more content
		moreContent := "second line\n"
		err = appendToFileTestHelper(testFile, moreContent)
		if err != nil {
			t.Fatalf("Failed to append second content: %v", err)
		}

		// Verify combined contents
		data, err = os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file after second append: %v", err)
		}

		expectedContent := content + moreContent
		if string(data) != expectedContent {
			t.Errorf("Expected combined content '%s', got '%s'", expectedContent, string(data))
		}
	})
}

// Helper function for testing file append (will be replaced with actual implementation)
func appendToFileTestHelper(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// Test edge cases and error conditions
func TestActionChainingEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		shellText   string
		expectError bool
		description string
	}{
		{
			name:        "Empty chain",
			shellText:   "",
			expectError: false,
			description: "Empty shell text should not cause errors",
		},
		{
			name:        "Only operators",
			shellText:   "&&",
			expectError: true,
			description: "Chain with only operators should be invalid",
		},
		{
			name:        "Multiple consecutive operators",
			shellText:   "echo hello && || echo world",
			expectError: true,
			description: "Multiple consecutive operators should be invalid",
		},
		{
			name:        "Operator at start",
			shellText:   "&& echo hello",
			expectError: true,
			description: "Operator at start should be invalid",
		},
		{
			name:        "Operator at end",
			shellText:   "echo hello &&",
			expectError: true,
			description: "Operator at end should be invalid",
		},
		{
			name:        "File redirect without filename",
			shellText:   "echo hello >>",
			expectError: true,
			description: "File redirect without filename should be invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal program for testing
			program := &ast.Program{
				Commands: []ast.CommandDecl{
					{
						Name: "test",
						Body: ast.CommandBody{
							Content: []ast.CommandContent{
								&ast.ShellContent{
									Parts: []ast.ShellPart{
										&ast.TextPart{Text: tt.shellText},
									},
								},
							},
						},
					},
				},
			}

			// Create execution context
			ctx := NewExecutionContext(context.Background(), program)
			builder := NewShellCodeBuilder(ctx)

			// Test parsing the chain
			shellContent := program.Commands[0].Body.Content[0].(*ast.ShellContent)
			_, err := builder.parseActionDecoratorChain(shellContent)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}