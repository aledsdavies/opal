package decorators

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// TestCmdDecorator_ModeIsolation ensures @cmd decorator never executes in wrong modes
func TestCmdDecorator_ModeIsolation(t *testing.T) {
	tests := []struct {
		name           string
		mode           execution.ExecutionMode
		shouldExecute  bool
		expectedResult string
	}{
		{
			name:           "interpreter_mode_executes",
			mode:           execution.InterpreterMode,
			shouldExecute:  true,
			expectedResult: "", // Will fail/succeed based on command existence
		},
		{
			name:           "generator_mode_generates_code",
			mode:           execution.GeneratorMode,
			shouldExecute:  false,
			expectedResult: "execute", // Should contain execute function call
		},
		{
			name:           "plan_mode_creates_plan",
			mode:           execution.PlanMode,
			shouldExecute:  false,
			expectedResult: "", // Should return plan data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple program with helper and main commands
			program := &ast.Program{
				Commands: []ast.CommandDecl{
					{
						Name: "helper",
						Body: ast.CommandBody{
							Content: []ast.CommandContent{
								&ast.ShellContent{
									Parts: []ast.ShellPart{
										&ast.TextPart{Text: "echo 'EXECUTION_DETECTED'"},
									},
								},
							},
						},
					},
					{
						Name: "main",
						Body: ast.CommandBody{
							Content: []ast.CommandContent{
								&ast.ActionDecorator{
									Name: "cmd",
									Args: []ast.NamedParameter{
										{Value: &ast.Identifier{Name: "helper"}},
									},
								},
							},
						},
					},
				},
			}

			// Create execution context
			ctx := execution.NewExecutionContext(nil, program)
			ctx = ctx.WithMode(tt.mode)

			// Initialize variables 
			if err := ctx.InitializeVariables(); err != nil {
				t.Fatalf("Failed to initialize variables: %v", err)
			}

			// Create the @cmd decorator
			decorator := &CmdDecorator{}

			// Test the decorator in the specified mode
			params := []ast.NamedParameter{
				{Value: &ast.Identifier{Name: "helper"}},
			}

			result := decorator.Expand(ctx, params)

			// Verify behavior based on mode
			switch tt.mode {
			case execution.GeneratorMode:
				if result.Error != nil {
					t.Errorf("Generator mode should not error: %v", result.Error)
				}
				if result.Data == nil {
					t.Error("Generator mode should return generated code")
				}
				if code, ok := result.Data.(string); ok {
					if !strings.Contains(code, tt.expectedResult) {
						t.Errorf("Generated code should contain %q, got: %s", tt.expectedResult, code)
					}
					// Most importantly - should not execute the command
					if strings.Contains(code, "EXECUTION_DETECTED") {
						t.Error("CRITICAL: Generator mode executed the command!")
					}
				}

			case execution.PlanMode:
				if result.Error != nil {
					t.Errorf("Plan mode should not error: %v", result.Error)
				}
				// Plan mode should return plan data, not execute
				if result.Data == nil {
					t.Error("Plan mode should return plan data")
				}

			case execution.InterpreterMode:
				// Interpreter mode may error if command doesn't exist - that's ok
				// The key is that it should attempt execution
				if result.Error != nil && !strings.Contains(result.Error.Error(), "not found") &&
				   !strings.Contains(result.Error.Error(), "failed to execute") {
					t.Errorf("Unexpected error in interpreter mode: %v", result.Error)
				}
			}
		})
	}
}

// TestCmdDecorator_NoExecutionDuringGeneration is a safety test that must always pass
func TestCmdDecorator_NoExecutionDuringGeneration(t *testing.T) {
	// This test ensures @cmd decorators never execute commands during code generation
	// Create a program that would have obvious side effects if executed
	program := &ast.Program{
		Commands: []ast.CommandDecl{
			{
				Name: "dangerous",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{
						&ast.ShellContent{
							Parts: []ast.ShellPart{
								&ast.TextPart{Text: "echo 'DANGER: Command was executed!' && touch EXECUTION_DETECTED"},
							},
						},
					},
				},
			},
			{
				Name: "caller",
				Body: ast.CommandBody{
					Content: []ast.CommandContent{
						&ast.ActionDecorator{
							Name: "cmd",
							Args: []ast.NamedParameter{
								{Value: &ast.Identifier{Name: "dangerous"}},
							},
						},
					},
				},
			},
		},
	}

	// Test in generator mode (used during code generation)
	ctx := execution.NewExecutionContext(nil, program)
	ctx = ctx.WithMode(execution.GeneratorMode)

	if err := ctx.InitializeVariables(); err != nil {
		t.Fatalf("Failed to initialize variables: %v", err)
	}

	decorator := &CmdDecorator{}
	params := []ast.NamedParameter{
		{Value: &ast.Identifier{Name: "dangerous"}},
	}

	// This should generate code, NOT execute the dangerous command
	result := decorator.Expand(ctx, params)

	if result.Error != nil {
		t.Fatalf("Generator mode failed: %v", result.Error)
	}

	if result.Data == nil {
		t.Fatal("Generator mode should return generated code")
	}

	// Verify it generated code (not executed)
	code, ok := result.Data.(string)
	if !ok {
		t.Fatalf("Expected string code, got %T", result.Data)
	}

	// Should generate function call code
	if !strings.Contains(code, "execute") {
		t.Errorf("Generated code should contain function call, got: %s", code)
	}

	// CRITICAL: Should never contain evidence of execution
	if strings.Contains(code, "DANGER: Command was executed!") ||
	   strings.Contains(code, "EXECUTION_DETECTED") {
		t.Fatal("CRITICAL: @cmd decorator executed command during generation!")
	}

	t.Logf("✅ @cmd decorator correctly generated code without execution: %s", code)
}