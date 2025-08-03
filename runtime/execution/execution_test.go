package execution

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
)

// Mock ValueDecorator for testing
type mockVarDecorator struct{}

func (m *mockVarDecorator) Expand(ctx GeneratorContext, params []ast.NamedParameter) *ExecutionResult {
	if len(params) == 0 {
		return NewFormattedErrorResult("@var decorator requires variable name")
	}

	var varName string
	if ident, ok := params[0].Value.(*ast.Identifier); ok {
		varName = ident.Name
	} else {
		return NewFormattedErrorResult("@var decorator requires identifier argument")
	}

	switch ctx.Mode() {
	case InterpreterMode:
		if value, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{Mode: InterpreterMode, Data: value, Error: nil}
		}
		return &ExecutionResult{Mode: InterpreterMode, Data: nil, Error: fmt.Errorf("variable '%s' not defined", varName)}
	case GeneratorMode:
		if _, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{Mode: GeneratorMode, Data: varName, Error: nil}
		}
		return &ExecutionResult{Mode: GeneratorMode, Data: nil, Error: fmt.Errorf("variable '%s' not defined", varName)}
	case PlanMode:
		if value, exists := ctx.GetVariable(varName); exists {
			return &ExecutionResult{Mode: PlanMode, Data: value, Error: nil}
		}
		return &ExecutionResult{Mode: PlanMode, Data: nil, Error: fmt.Errorf("variable '%s' not defined", varName)}
	default:
		return &ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("unsupported mode: %v", ctx.Mode())}
	}
}

type mockEnvDecorator struct{}

func (m *mockEnvDecorator) Expand(ctx *ExecutionContext, params []ast.NamedParameter) *ExecutionResult {
	if len(params) == 0 {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@env decorator requires environment variable name"),
		}
	}

	var envName string
	if ident, ok := params[0].Value.(*ast.Identifier); ok {
		envName = ident.Name
	} else {
		return &ExecutionResult{
			Mode:  ctx.Mode(),
			Data:  nil,
			Error: fmt.Errorf("@env decorator requires identifier argument"),
		}
	}

	switch ctx.Mode() {
	case InterpreterMode:
		if value, exists := ctx.GetEnv(envName); exists {
			return &ExecutionResult{Mode: InterpreterMode, Data: value, Error: nil}
		}
		return &ExecutionResult{Mode: InterpreterMode, Data: os.Getenv(envName), Error: nil}
	case GeneratorMode:
		return &ExecutionResult{Mode: GeneratorMode, Data: fmt.Sprintf("os.Getenv(%q)", envName), Error: nil}
	case PlanMode:
		if value, exists := ctx.GetEnv(envName); exists {
			return &ExecutionResult{Mode: PlanMode, Data: value, Error: nil}
		}
		return &ExecutionResult{Mode: PlanMode, Data: os.Getenv(envName), Error: nil}
	default:
		return &ExecutionResult{Mode: ctx.Mode(), Data: nil, Error: fmt.Errorf("unsupported mode: %v", ctx.Mode())}
	}
}

// Mock decorator lookup functions
func mockValueDecoratorLookup(name string) (interface{}, bool) {
	switch name {
	case "var":
		return &mockVarDecorator{}, true
	case "env":
		return &mockEnvDecorator{}, true
	default:
		return nil, false
	}
}

func mockActionDecoratorLookup(name string) (interface{}, bool) {
	// No action decorators in these basic tests
	return nil, false
}

func setupTestContext(mode ExecutionMode) *ExecutionContext {
	program := ast.NewProgram(
		ast.Var("PROJECT", ast.Str("testproject")),
		ast.Var("VERSION", ast.Str("1.0.0")),
	)

	ctx := NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(mode)
	ctx.SetValueDecoratorLookup(mockValueDecoratorLookup)
	ctx.SetActionDecoratorLookup(mockActionDecoratorLookup)
	
	err := ctx.InitializeVariables()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize variables: %v", err))
	}

	// Set up test environment variables in the context (simulating captured environment)
	// We'll directly modify the private env field for testing purposes
	// This simulates what would happen if these were set when the context was created
	ctx.env["USER"] = "testuser"
	ctx.env["HOME"] = "/home/testuser"

	return ctx
}

func TestExecutionContext_InterpreterMode(t *testing.T) {
	ctx := setupTestContext(InterpreterMode)

	tests := []struct {
		name        string
		shell       *ast.ShellContent
		expectError bool
	}{
		{
			name: "Simple text command",
			shell: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError: false,
		},
		{
			name: "Command with variable expansion",
			shell: ast.Shell(
				ast.Text("echo \"Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text(" v"),
				ast.At("var", ast.UnnamedParam(ast.Id("VERSION"))),
				ast.Text("\""),
			),
			expectError: false,
		},
		{
			name: "Command with undefined variable",
			shell: ast.Shell(
				ast.Text("echo "),
				ast.At("var", ast.UnnamedParam(ast.Id("UNDEFINED"))),
			),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shell)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
			}

			if result.Mode != InterpreterMode {
				t.Errorf("Expected mode %v, got %v", InterpreterMode, result.Mode)
			}

			// In interpreter mode, Data should be nil (execution happens via side effects)
			if result.Data != nil {
				t.Errorf("Expected nil data in interpreter mode, got %v", result.Data)
			}
		})
	}
}

func TestExecutionContext_GeneratorMode(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)

	tests := []struct {
		name            string
		shell           *ast.ShellContent
		expectError     bool
		expectedContent []string // Parts that should be in the generated code
	}{
		{
			name: "Simple text command",
			shell: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError: false,
			expectedContent: []string{
				"CmdStr := \"echo hello\"",  // Direct string assignment
				"ExecCmd := exec.CommandContext(ctx, \"sh\", \"-c\"", // Command creation
				"ExecCmd.Stdout = os.Stdout", // Output redirection
				"ExecCmd.Run()",              // Command execution
			},
		},
		{
			name: "Command with variable expansion",
			shell: ast.Shell(
				ast.Text("Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text(" v"),
				ast.At("var", ast.UnnamedParam(ast.Id("VERSION"))),
			),
			expectError: false,
			expectedContent: []string{
				"fmt.Sprintf(",               // Variable expansion uses fmt.Sprintf
				"Building %s v%s",            // Format string
				"PROJECT, VERSION",           // Variable references
				"exec.CommandContext(ctx, \"sh\", \"-c\"", // Command creation
			},
		},
		{
			name: "Command with environment variable",
			shell: ast.Shell(
				ast.Text("echo $"),
				ast.At("env", ast.UnnamedParam(ast.Id("USER"))),
			),
			expectError: false,
			expectedContent: []string{
				"fmt.Sprintf(",               // Env var expansion uses fmt.Sprintf
				"echo $%s",                   // Format string
				"os.Getenv(\"USER\")",        // Environment variable lookup
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shell)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}

			if result.Mode != GeneratorMode {
				t.Errorf("Expected mode %v, got %v", GeneratorMode, result.Mode)
			}

			code, ok := result.Data.(string)
			if !ok {
				t.Errorf("Expected string data, got %T", result.Data)
				return
			}

			// Verify all expected content is present
			for _, expectedPart := range tt.expectedContent {
				if !strings.Contains(code, expectedPart) {
					t.Errorf("Expected generated code to contain %q\nGenerated code:\n%s", expectedPart, code)
				}
			}

			t.Logf("Generated code for %s:\n%s", tt.name, code)
		})
	}
}

func TestExecutionContext_PlanMode(t *testing.T) {
	ctx := setupTestContext(PlanMode)

	tests := []struct {
		name            string
		shell           *ast.ShellContent
		expectError     bool
		expectedCommand string
	}{
		{
			name: "Simple text command",
			shell: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError:     false,
			expectedCommand: "echo hello",
		},
		{
			name: "Command with variable expansion",
			shell: ast.Shell(
				ast.Text("echo \"Building "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text(" v"),
				ast.At("var", ast.UnnamedParam(ast.Id("VERSION"))),
				ast.Text("\""),
			),
			expectError:     false,
			expectedCommand: "echo \"Building testproject v1.0.0\"", // Variables resolved to actual values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.ExecuteShell(tt.shell)

			if tt.expectError {
				if result.Error == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if result.Error != nil {
				t.Errorf("Unexpected error: %v", result.Error)
				return
			}

			if result.Mode != PlanMode {
				t.Errorf("Expected mode %v, got %v", PlanMode, result.Mode)
			}

			planData, ok := result.Data.(map[string]interface{})
			if !ok {
				t.Errorf("Expected map[string]interface{} data, got %T", result.Data)
				return
			}

			if planData["type"] != "shell" {
				t.Errorf("Expected plan type 'shell', got %v", planData["type"])
			}

			command, ok := planData["command"].(string)
			if !ok {
				t.Errorf("Expected command to be string, got %T", planData["command"])
				return
			}

			if command != tt.expectedCommand {
				t.Errorf("Expected command %q, got %q", tt.expectedCommand, command)
			}
		})
	}
}

func TestShellCodeBuilder_UnifiedTemplateSystem(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	// Test the unified template system directly
	shell := ast.Shell(
		ast.Text("echo "),
		ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
	)

	code, err := builder.GenerateShellCode(shell)
	if err != nil {
		t.Fatalf("Failed to generate shell code: %v", err)
	}

	// Verify meaningful variable naming
	if !strings.Contains(code, "CmdStr") {
		t.Errorf("Expected meaningful variable name 'CmdStr', got:\n%s", code)
	}

	if !strings.Contains(code, "ExecCmd") {
		t.Errorf("Expected meaningful variable name 'ExecCmd', got:\n%s", code)
	}

	// Verify proper template structure
	expectedStructure := []string{
		"fmt.Sprintf(",
		"echo %s",
		"PROJECT",
		"exec.CommandContext(ctx, \"sh\", \"-c\"",
		"ExecCmd.Stdout = os.Stdout",
		"ExecCmd.Stderr = os.Stderr",
		"ExecCmd.Stdin = os.Stdin",
		"ExecCmd.Run()",
		"command failed:",
	}

	for _, part := range expectedStructure {
		if !strings.Contains(code, part) {
			t.Errorf("Expected code structure to contain %q\nGenerated code:\n%s", part, code)
		}
	}
}

func TestShellCodeBuilder_TemplateFunctions(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	// Test that template functions are available
	funcs := builder.GetTemplateFunctions()

	expectedFunctions := []string{
		"generateShellCode",
		"formatParams",
		"title",
	}

	for _, funcName := range expectedFunctions {
		if _, exists := funcs[funcName]; !exists {
			t.Errorf("Expected template function %q to be available", funcName)
		}
	}

	// Test generateShellCode function specifically
	generateShellCode, exists := funcs["generateShellCode"]
	if !exists {
		t.Fatal("generateShellCode function not found")
	}

	// Test calling the function
	shell := ast.Shell(ast.Text("echo test"))
	if fn, ok := generateShellCode.(func(ast.CommandContent) (string, error)); ok {
		result, err := fn(shell)
		if err != nil {
			t.Errorf("generateShellCode function failed: %v", err)
		}
		if !strings.Contains(result, "echo test") {
			t.Errorf("generateShellCode result doesn't contain expected content: %s", result)
		}
	} else {
		t.Errorf("generateShellCode has wrong type: %T", generateShellCode)
	}
}

func TestExecutionContext_MeaningfulVariableNames(t *testing.T) {
	program := ast.NewProgram(ast.Var("TEST", ast.Str("value")))
	ctx := NewExecutionContext(context.Background(), program)
	ctx = ctx.WithMode(GeneratorMode).WithCurrentCommand("build")
	ctx.SetValueDecoratorLookup(mockValueDecoratorLookup)
	ctx.SetActionDecoratorLookup(mockActionDecoratorLookup)
	ctx.InitializeVariables()

	builder := NewShellCodeBuilder(ctx)
	shell := ast.Shell(ast.Text("echo test"))

	code, err := builder.GenerateShellCode(shell)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should use command name in variable names
	if !strings.Contains(code, "BuildCmdStr") {
		t.Errorf("Expected meaningful variable name with command prefix 'BuildCmdStr', got:\n%s", code)
	}

	if !strings.Contains(code, "BuildExecCmd") {
		t.Errorf("Expected meaningful variable name with command prefix 'BuildExecCmd', got:\n%s", code)
	}
}

func TestExecutionContext_UnsupportedMode(t *testing.T) {
	ctx := NewExecutionContext(context.Background(), ast.NewProgram())
	ctx.SetValueDecoratorLookup(mockValueDecoratorLookup)
	ctx.SetActionDecoratorLookup(mockActionDecoratorLookup)
	ctx.mode = ExecutionMode(999) // Invalid mode

	shell := ast.Shell(ast.Text("echo test"))
	result := ctx.ExecuteShell(shell)

	if result.Error == nil {
		t.Error("Expected error for unsupported mode")
	}

	if !strings.Contains(result.Error.Error(), "unsupported execution mode") {
		t.Errorf("Expected error about unsupported mode, got: %v", result.Error)
	}
}