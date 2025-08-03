package execution

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
)

func TestShellCodeBuilder_GenerateShellExecutionTemplate(t *testing.T) {
	program := &ast.Program{}
	ctx := NewGeneratorContext(nil, program)
	builder := NewShellCodeBuilder(ctx)

	tests := []struct {
		name            string
		shell           *ast.ShellContent
		expectError     bool
		expectedContent []string
	}{
		{
			name: "Simple text only",
			shell: ast.Shell(
				ast.Text("npm run build"),
			),
			expectError: false,
			expectedContent: []string{
				"CmdStr := \"npm run build\"",
				"exec.CommandContext(ctx, \"sh\", \"-c\"",
				"Err := ",
			},
		},
		{
			name: "Text with spaces",
			shell: ast.Shell(
				ast.Text("echo hello world"),
			),
			expectError: false,
			expectedContent: []string{
				"CmdStr := \"echo hello world\"",
				"exec.CommandContext(ctx, \"sh\", \"-c\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.GenerateShellExecutionTemplate(tt.shell)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected generated template to contain %q\nGenerated template:\n%s", expected, result)
				}
			}
		})
	}
}

func TestShellCodeBuilder_MeaningfulVariableNaming(t *testing.T) {
	tests := []struct {
		name        string
		commandName string
		expectVar   string
	}{
		{
			name:        "Build command",
			commandName: "build",
			expectVar:   "build",
		},
		{
			name:        "Test command", 
			commandName: "test",
			expectVar:   "test",
		},
		{
			name:        "No command name (default)",
			commandName: "",
			expectVar:   "command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := &ast.Program{}
			ctx := NewGeneratorContext(nil, program)
			if tt.commandName != "" {
				ctx = ctx.WithCurrentCommand(tt.commandName)
			}
			builder := NewShellCodeBuilder(ctx)

			shell := ast.Shell(ast.Text("echo test"))
			result, err := builder.GenerateShellExecutionTemplate(shell)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !strings.Contains(result, tt.expectVar) {
				t.Errorf("Expected variable pattern %q in generated code\nGenerated:\n%s", tt.expectVar, result)
			}
		})
	}
}

func TestShellCodeBuilder_FormatParamsFunction(t *testing.T) {
	program := &ast.Program{}
	ctx := NewGeneratorContext(nil, program)
	builder := NewShellCodeBuilder(ctx)

	tests := []struct {
		name     string
		params   []ast.NamedParameter
		expected string
	}{
		{
			name:     "No parameters",
			params:   []ast.NamedParameter{},
			expected: "nil",
		},
		{
			name: "With parameters", 
			params: []ast.NamedParameter{
				{Name: "timeout", Value: ast.Str("30s")},
			},
			expected: "[]ast.NamedParameter{}", // Simplified for now
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.formatParams(tt.params)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestShellCodeBuilder_GetTemplateFunctions(t *testing.T) {
	program := &ast.Program{}
	ctx := NewGeneratorContext(nil, program)
	builder := NewShellCodeBuilder(ctx)

	funcs := builder.GetTemplateFunctions()

	// Test that all expected functions are present
	expectedFuncs := []string{
		"generateShellCode",
		"formatParams", 
		"title",
		"cmdFunctionName",
	}

	for _, funcName := range expectedFuncs {
		if _, exists := funcs[funcName]; !exists {
			t.Errorf("Expected template function %q to be registered", funcName)
		}
	}

	// Test formatParams function
	formatParams := funcs["formatParams"]
	if fn, ok := formatParams.(func([]ast.NamedParameter) string); ok {
		result := fn([]ast.NamedParameter{})
		if result != "nil" {
			t.Errorf("formatParams should return 'nil' for empty params, got %q", result)
		}
	} else {
		t.Errorf("formatParams function has wrong signature")
	}

	// Test title function (should be strings.Title)
	title := funcs["title"]
	if fn, ok := title.(func(string) string); ok {
		result := fn("hello")
		if result != "Hello" {
			t.Errorf("title function should capitalize, got %q", result)
		}
	} else {
		t.Errorf("title function has wrong signature")
	}
}

func TestShellCodeBuilder_GenerateShellCode_Basic(t *testing.T) {
	program := &ast.Program{}
	ctx := NewGeneratorContext(nil, program)
	builder := NewShellCodeBuilder(ctx)

	// Test basic shell content generation
	shell := ast.Shell(ast.Text("echo hello"))
	result, err := builder.GenerateShellCode(shell)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}

	if !strings.Contains(result, "echo hello") {
		t.Error("Expected result to contain shell command")
	}
}