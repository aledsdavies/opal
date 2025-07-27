package execution

import (
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
)

func TestShellCodeBuilder_GenerateShellExecutionTemplate(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	// Add the HOME environment variable for the test (modify captured environment)
	ctx.env["HOME"] = "/home/testuser"
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
				"ActionCmdStr := \"npm run build\"",
				"ActionExecCmd := exec.CommandContext(ctx, \"sh\", \"-c\", ActionCmdStr)",
				"ActionExecCmd.Stdout = os.Stdout",
				"ActionExecCmd.Stderr = os.Stderr", 
				"ActionExecCmd.Stdin = os.Stdin",
				"ActionExecCmd.Run()",
				"command failed:",
			},
		},
		{
			name: "Text with single variable",
			shell: ast.Shell(
				ast.Text("echo "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
			),
			expectError: false,
			expectedContent: []string{
				"ActionCmdStr := fmt.Sprintf(\"echo %s\", PROJECT)",
				"ActionExecCmd := exec.CommandContext(ctx, \"sh\", \"-c\", ActionCmdStr)",
				"ActionExecCmd.Run()",
			},
		},
		{
			name: "Complex variable expansion",
			shell: ast.Shell(
				ast.Text("docker build -t "),
				ast.At("var", ast.UnnamedParam(ast.Id("PROJECT"))),
				ast.Text(":"),
				ast.At("var", ast.UnnamedParam(ast.Id("VERSION"))),
				ast.Text(" ."),
			),
			expectError: false,
			expectedContent: []string{
				"ActionCmdStr := fmt.Sprintf(\"docker build -t %s:%s .\", PROJECT, VERSION)",
				"exec.CommandContext(ctx, \"sh\", \"-c\", ActionCmdStr)",
			},
		},
		{
			name: "Environment variable expansion", 
			shell: ast.Shell(
				ast.Text("echo $"),
				ast.At("env", ast.UnnamedParam(ast.Id("HOME"))),
			),
			expectError: false,
			expectedContent: []string{
				"ActionCmdStr := fmt.Sprintf(\"echo $%s\", os.Getenv(\"HOME\"))",
				"exec.CommandContext(ctx, \"sh\", \"-c\", ActionCmdStr)",
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
		expectedVar string
		expectedExec string
	}{
		{
			name:         "Build command",
			commandName:  "build",
			expectedVar:  "BuildCmdStr",
			expectedExec: "BuildExecCmd",
		},
		{
			name:         "Test command",
			commandName:  "test",
			expectedVar:  "TestCmdStr", 
			expectedExec: "TestExecCmd",
		},
		{
			name:         "Deploy command",
			commandName:  "deploy",
			expectedVar:  "DeployCmdStr",
			expectedExec: "DeployExecCmd",
		},
		{
			name:         "No command name (default)",
			commandName:  "",
			expectedVar:  "ActionCmdStr",
			expectedExec: "ActionExecCmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := setupTestContext(GeneratorMode)
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

			if !strings.Contains(result, tt.expectedVar) {
				t.Errorf("Expected variable name %q in generated code\nGenerated:\n%s", tt.expectedVar, result)
			}

			if !strings.Contains(result, tt.expectedExec) {
				t.Errorf("Expected variable name %q in generated code\nGenerated:\n%s", tt.expectedExec, result)
			}
		})
	}
}

func TestShellCodeBuilder_FormatParamsFunction(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
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
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	funcs := builder.GetTemplateFunctions()

	// Test that all expected functions are present
	expectedFuncs := []string{
		"generateShellCode",
		"formatParams", 
		"title",
	}

	for _, funcName := range expectedFuncs {
		if _, exists := funcs[funcName]; !exists {
			t.Errorf("Expected template function %q to be registered", funcName)
		}
	}

	// Test generateShellCode function
	generateShellCode := funcs["generateShellCode"]
	if fn, ok := generateShellCode.(func(ast.CommandContent) (string, error)); ok {
		shell := ast.Shell(ast.Text("echo hello"))
		result, err := fn(shell)
		if err != nil {
			t.Errorf("generateShellCode function failed: %v", err)
		}
		if !strings.Contains(result, "echo hello") {
			t.Errorf("generateShellCode result should contain shell content")
		}
	} else {
		t.Errorf("generateShellCode function has wrong signature")
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

func TestShellCodeBuilder_GenerateShellCode_SwitchOnCommandContent(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	tests := []struct {
		name        string
		content     ast.CommandContent
		expectError bool
		description string
	}{
		{
			name: "ShellContent",
			content: ast.Shell(
				ast.Text("echo hello"),
			),
			expectError: false,
			description: "Should handle ShellContent correctly",
		},
		{
			name: "BlockDecorator",
			content: &ast.BlockDecorator{
				Name:    "timeout",
				Args:    []ast.NamedParameter{{Name: "duration", Value: ast.Str("30s")}},
				Content: []ast.CommandContent{},
			},
			expectError: false,
			description: "Should handle BlockDecorator correctly",
		},
		{
			name: "PatternDecorator",
			content: &ast.PatternDecorator{
				Name:     "when",
				Args:     []ast.NamedParameter{{Name: "condition", Value: ast.Str("ENV")}},
				Patterns: []ast.PatternBranch{},
			},
			expectError: false,
			description: "Should handle PatternDecorator correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.GenerateShellCode(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result for %s", tt.description)
				}
			}
		})
	}
}

func TestShellCodeBuilder_BlockDecoratorTemplate(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	blockDecorator := &ast.BlockDecorator{
		Name: "parallel",
		Args: []ast.NamedParameter{
			{Name: "concurrency", Value: ast.Num("4")},
		},
		Content: []ast.CommandContent{},
	}

	result, err := builder.generateBlockDecoratorTemplate(blockDecorator)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	expectedContent := []string{
		"executeParallelDecorator(ctx,",
		"@parallel decorator failed:",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected block decorator template to contain %q\nGenerated:\n%s", expected, result)
		}
	}
}

func TestShellCodeBuilder_PatternDecoratorTemplate(t *testing.T) {
	ctx := setupTestContext(GeneratorMode)
	builder := NewShellCodeBuilder(ctx)

	patternDecorator := &ast.PatternDecorator{
		Name: "when",
		Args: []ast.NamedParameter{
			{Name: "condition", Value: ast.Id("ENV")},
		},
		Patterns: []ast.PatternBranch{},
	}

	result, err := builder.generatePatternDecoratorTemplate(patternDecorator)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	expectedContent := []string{
		"executeWhenDecorator(ctx,",
		"@when decorator failed:",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected pattern decorator template to contain %q\nGenerated:\n%s", expected, result)
		}
	}
}