package parser

import (
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/types"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/google/go-cmp/cmp"
)

func init() {
	// Register test decorators for pipe validation tests
	// @testvalue - value decorator without I/O support (for testing pipe validation)
	testValueSchema := types.NewSchema("testvalue", types.KindValue).
		Description("Test value decorator without I/O").
		Param("arg", types.TypeString).
		Description("Test argument").
		Required().
		Done().
		Returns(types.TypeString, "Test value").
		Build()

	// Register without I/O capabilities - this decorator doesn't support piping
	if err := types.Global().RegisterValueWithSchema(testValueSchema, nil); err != nil {
		panic(err)
	}

	// Register namespaced decorator for testing dot-separated names
	// @file.read - value decorator for testing namespaced decorator parsing
	fileReadSchema := types.NewSchema("file.read", types.KindValue).
		Description("Read file").
		Param("path", types.TypeString).
		Required().
		Done().
		Returns(types.TypeString, "Contents").
		Build()

	if err := types.Global().RegisterValueWithSchema(fileReadSchema, nil); err != nil {
		panic(err)
	}

	// Register @file.temp for redirect validation tests
	// Supports overwrite only (no append)
	fileTempSchema := types.NewSchema("file.temp", types.KindExecution).
		Description("Create temporary file").
		WithRedirect(types.RedirectOverwriteOnly).
		Build()

	if err := types.Global().RegisterSDKHandlerWithSchema(fileTempSchema, nil); err != nil {
		panic(err)
	}

	// Register @retry as an execution decorator in the NEW registry for role validation tests
	retryDec := &mockExecDecorator{path: "retry"}
	if err := decorator.Register("retry", retryDec); err != nil {
		panic(err)
	}
}

// Mock execution decorator for testing
type mockExecDecorator struct {
	path string
}

func (m *mockExecDecorator) Descriptor() decorator.Descriptor {
	return decorator.Descriptor{
		Path:  m.path,
		Roles: []decorator.Role{decorator.RoleWrapper},
		Capabilities: decorator.Capabilities{
			Block: decorator.BlockOptional, // Execution decorators can optionally have blocks
		},
	}
}

func (m *mockExecDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return nil // Stub for testing
}

// TestParseEventStructure uses table-driven tests to verify parse tree events
func TestParseEventStructure(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "empty file",
			input: "",
			events: []Event{
				{EventOpen, 0},  // Source
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with no parameters",
			input: "fun greet() {}",
			events: []Event{
				{EventOpen, 0},  // Source
				{EventOpen, 1},  // Function
				{EventToken, 0}, // fun
				{EventToken, 1}, // greet
				{EventOpen, 2},  // ParamList
				{EventToken, 2}, // (
				{EventToken, 3}, // )
				{EventClose, 2}, // ParamList
				{EventOpen, 3},  // Block
				{EventToken, 4}, // {
				{EventToken, 5}, // }
				{EventClose, 3}, // Block
				{EventClose, 1}, // Function
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with single parameter",
			input: "fun greet(name) {}",
			events: []Event{
				{EventOpen, 0},  // Source
				{EventOpen, 1},  // Function
				{EventToken, 0}, // fun
				{EventToken, 1}, // greet
				{EventOpen, 2},  // ParamList
				{EventToken, 2}, // (
				{EventOpen, 4},  // Param
				{EventToken, 3}, // name
				{EventClose, 4}, // Param
				{EventToken, 4}, // )
				{EventClose, 2}, // ParamList
				{EventOpen, 3},  // Block
				{EventToken, 5}, // {
				{EventToken, 6}, // }
				{EventClose, 3}, // Block
				{EventClose, 1}, // Function
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with typed parameter",
			input: "fun greet(name: String) {}",
			events: []Event{
				{EventOpen, 0},  // Source
				{EventOpen, 1},  // Function
				{EventToken, 0}, // fun
				{EventToken, 1}, // greet
				{EventOpen, 2},  // ParamList
				{EventToken, 2}, // (
				{EventOpen, 4},  // Param
				{EventToken, 3}, // name
				{EventOpen, 5},  // TypeAnnotation
				{EventToken, 4}, // :
				{EventToken, 5}, // String
				{EventClose, 5}, // TypeAnnotation
				{EventClose, 4}, // Param
				{EventToken, 6}, // )
				{EventClose, 2}, // ParamList
				{EventOpen, 3},  // Block
				{EventToken, 7}, // {
				{EventToken, 8}, // }
				{EventClose, 3}, // Block
				{EventClose, 1}, // Function
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with default parameter",
			input: `fun greet(name = "World") {}`,
			events: []Event{
				{EventOpen, 0},  // Source
				{EventOpen, 1},  // Function
				{EventToken, 0}, // fun
				{EventToken, 1}, // greet
				{EventOpen, 2},  // ParamList
				{EventToken, 2}, // (
				{EventOpen, 4},  // Param
				{EventToken, 3}, // name
				{EventOpen, 6},  // DefaultValue (new node kind)
				{EventToken, 4}, // =
				{EventToken, 5}, // "World"
				{EventClose, 6}, // DefaultValue
				{EventClose, 4}, // Param
				{EventToken, 6}, // )
				{EventClose, 2}, // ParamList
				{EventOpen, 3},  // Block
				{EventToken, 7}, // {
				{EventToken, 8}, // }
				{EventClose, 3}, // Block
				{EventClose, 1}, // Function
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with typed parameter and default value",
			input: `fun greet(name: String = "World") {}`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // greet
				{EventOpen, 2},   // ParamList
				{EventToken, 2},  // (
				{EventOpen, 4},   // Param
				{EventToken, 3},  // name
				{EventOpen, 5},   // TypeAnnotation
				{EventToken, 4},  // :
				{EventToken, 5},  // String
				{EventClose, 5},  // TypeAnnotation
				{EventOpen, 6},   // DefaultValue
				{EventToken, 6},  // =
				{EventToken, 7},  // "World"
				{EventClose, 6},  // DefaultValue
				{EventClose, 4},  // Param
				{EventToken, 8},  // )
				{EventClose, 2},  // ParamList
				{EventOpen, 3},   // Block
				{EventToken, 9},  // {
				{EventToken, 10}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
			},
		},
		{
			name:  "function with two untyped parameters",
			input: `fun greet(first, last) {}`,
			events: []Event{
				{EventOpen, 0},  // Source
				{EventOpen, 1},  // Function
				{EventToken, 0}, // fun
				{EventToken, 1}, // greet
				{EventOpen, 2},  // ParamList
				{EventToken, 2}, // (
				{EventOpen, 4},  // Param
				{EventToken, 3}, // first
				{EventClose, 4}, // Param
				{EventToken, 4}, // ,
				{EventOpen, 4},  // Param
				{EventToken, 5}, // last
				{EventClose, 4}, // Param
				{EventToken, 6}, // )
				{EventClose, 2}, // ParamList
				{EventOpen, 3},  // Block
				{EventToken, 7}, // {
				{EventToken, 8}, // }
				{EventClose, 3}, // Block
				{EventClose, 1}, // Function
				{EventClose, 0}, // Source
			},
		},
		{
			name:  "function with mixed typed and untyped parameters",
			input: `fun deploy(env: String, replicas = 3) {}`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // deploy
				{EventOpen, 2},   // ParamList
				{EventToken, 2},  // (
				{EventOpen, 4},   // Param
				{EventToken, 3},  // env
				{EventOpen, 5},   // TypeAnnotation
				{EventToken, 4},  // :
				{EventToken, 5},  // String
				{EventClose, 5},  // TypeAnnotation
				{EventClose, 4},  // Param
				{EventToken, 6},  // ,
				{EventOpen, 4},   // Param
				{EventToken, 7},  // replicas
				{EventOpen, 6},   // DefaultValue
				{EventToken, 8},  // =
				{EventToken, 9},  // 3
				{EventClose, 6},  // DefaultValue
				{EventClose, 4},  // Param
				{EventToken, 10}, // )
				{EventClose, 2},  // ParamList
				{EventOpen, 3},   // Block
				{EventToken, 11}, // {
				{EventToken, 12}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
			},
		},
		{
			name:  "function with all parameter variations",
			input: `fun deploy(env: String, replicas: Int = 3, verbose) {}`,
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // deploy
				{EventOpen, 2},   // ParamList
				{EventToken, 2},  // (
				{EventOpen, 4},   // Param
				{EventToken, 3},  // env
				{EventOpen, 5},   // TypeAnnotation
				{EventToken, 4},  // :
				{EventToken, 5},  // String
				{EventClose, 5},  // TypeAnnotation
				{EventClose, 4},  // Param
				{EventToken, 6},  // ,
				{EventOpen, 4},   // Param
				{EventToken, 7},  // replicas
				{EventOpen, 5},   // TypeAnnotation
				{EventToken, 8},  // :
				{EventToken, 9},  // Int
				{EventClose, 5},  // TypeAnnotation
				{EventOpen, 6},   // DefaultValue
				{EventToken, 10}, // =
				{EventToken, 11}, // 3
				{EventClose, 6},  // DefaultValue
				{EventClose, 4},  // Param
				{EventToken, 12}, // ,
				{EventOpen, 4},   // Param
				{EventToken, 13}, // verbose
				{EventClose, 4},  // Param
				{EventToken, 14}, // )
				{EventClose, 2},  // ParamList
				{EventOpen, 3},   // Block
				{EventToken, 15}, // {
				{EventToken, 16}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			// Should have no errors
			if len(tree.Errors) != 0 {
				t.Errorf("Expected no errors, got: %v", tree.Errors)
			}

			// Compare events using cmp.Diff for clear output
			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("Events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseBasics verifies basic parsing functionality
func TestParseBasics(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantNil    bool
		wantTokens bool
		wantEvents bool
	}{
		{
			name:       "empty file returns non-nil tree",
			input:      "",
			wantNil:    false,
			wantTokens: true, // Lexer always produces EOF token
			wantEvents: true,
		},
		{
			name:       "function declaration has tokens and events",
			input:      "fun greet() {}",
			wantNil:    false,
			wantTokens: true,
			wantEvents: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if (tree == nil) != tt.wantNil {
				t.Errorf("ParseString() nil = %v, want %v", tree == nil, tt.wantNil)
			}

			if tree != nil {
				hasTokens := len(tree.Tokens) > 0
				if hasTokens != tt.wantTokens {
					t.Errorf("Has tokens = %v, want %v", hasTokens, tt.wantTokens)
				}

				hasEvents := len(tree.Events) > 0
				if hasEvents != tt.wantEvents {
					t.Errorf("Has events = %v, want %v", hasEvents, tt.wantEvents)
				}
			}
		})
	}
}

// TestTelemetry verifies telemetry collection
func TestTelemetry(t *testing.T) {
	input := "fun greet(name: String) {}"

	t.Run("telemetry off by default", func(t *testing.T) {
		tree := ParseString(input)
		if tree.Telemetry != nil {
			t.Error("Expected nil telemetry by default")
		}
	})

	t.Run("telemetry timing enabled", func(t *testing.T) {
		tree := ParseString(input, WithTelemetryTiming())

		if tree.Telemetry == nil {
			t.Fatal("Expected telemetry to be non-nil")
		}

		if tree.Telemetry.TokenCount == 0 {
			t.Error("Expected non-zero token count")
		}

		if tree.Telemetry.EventCount == 0 {
			t.Error("Expected non-zero event count")
		}

		if tree.Telemetry.TotalTime == 0 {
			t.Error("Expected non-zero total time")
		}
	})

	t.Run("telemetry basic enabled", func(t *testing.T) {
		tree := ParseString(input, WithTelemetryBasic())

		if tree.Telemetry == nil {
			t.Fatal("Expected telemetry to be non-nil")
		}

		if tree.Telemetry.TokenCount == 0 {
			t.Error("Expected non-zero token count")
		}
	})
}

// TestDebugTracing verifies debug event collection
func TestDebugTracing(t *testing.T) {
	input := "fun greet(name: String) {}"

	t.Run("debug off by default", func(t *testing.T) {
		tree := ParseString(input)
		if len(tree.DebugEvents) != 0 {
			t.Error("Expected no debug events by default")
		}
	})

	t.Run("debug paths enabled", func(t *testing.T) {
		tree := ParseString(input, WithDebugPaths())

		if len(tree.DebugEvents) == 0 {
			t.Fatal("Expected debug events")
		}

		// Should have enter/exit events for source, function, paramList, etc.
		hasEnterSource := false
		hasExitSource := false
		for _, evt := range tree.DebugEvents {
			if evt.Event == "enter_source" {
				hasEnterSource = true
			}
			if evt.Event == "exit_source" {
				hasExitSource = true
			}
		}

		if !hasEnterSource {
			t.Error("Expected enter_source debug event")
		}
		if !hasExitSource {
			t.Error("Expected exit_source debug event")
		}
	})
}

// TestPipeOperatorValidation tests that pipe operator validates I/O capabilities
func TestPipeOperatorValidation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError *ParseError
	}{
		{
			name:  "pipe from decorator without stdout support",
			input: `@timeout(5s) { echo "test" } | grep "pattern"`,
			expectedError: &ParseError{
				Position:   lexer.Position{Line: 1, Column: 30, Offset: 29},
				Message:    "@timeout does not produce stdout",
				Context:    "pipe operator",
				Got:        lexer.PIPE,
				Suggestion: "Only shell commands and decorators with stdout support can be piped from",
				Example:    "echo \"test\" | grep \"pattern\"",
				Note:       "Only decorators that produce stdout can be piped from",
			},
		},
		{
			name:          "pipe from interpolated decorator is valid",
			input:         `echo @file.read("test.txt") | grep "pattern"`,
			expectedError: nil, // This is valid - @file.read is interpolated into echo, then echo is piped
		},
		{
			name:          "valid pipe between shell commands",
			input:         `echo "test" | grep "test"`,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))
			tree.ValidateSemantics() // Post-parse validation

			if tt.expectedError != nil {
				if len(tree.Errors) == 0 {
					t.Errorf("expected parse error but got none")
					return
				}

				if diff := cmp.Diff(*tt.expectedError, tree.Errors[0]); diff != "" {
					t.Errorf("Error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("unexpected parse errors: %v", tree.Errors)
				}
			}
		})
	}
}

// TestRedirectOperatorValidation tests that redirect operator validates redirect capabilities
func TestRedirectOperatorValidation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError *ParseError
	}{
		{
			name:  "redirect to decorator without redirect support",
			input: `echo "test" > @timeout(5s)`,
			expectedError: &ParseError{
				Position:   lexer.Position{Line: 1, Column: 13, Offset: 12},
				Message:    "@timeout does not support redirection",
				Context:    "redirect operator",
				Got:        lexer.GT,
				Suggestion: "Only decorators with redirect support can be used as redirect targets",
				Example:    "echo \"test\" > output.txt",
				Note:       "Use @shell(\"output.txt\") or decorators that support redirect",
			},
		},
		{
			name:  "append to decorator that only supports overwrite",
			input: `echo "test" >> @file.temp()`,
			expectedError: &ParseError{
				Position:   lexer.Position{Line: 1, Column: 13, Offset: 12},
				Message:    "@file.temp does not support append (>>)",
				Context:    "redirect operator",
				Got:        lexer.APPEND,
				Suggestion: "Use a different redirect mode or a decorator that supports append",
				Example:    "echo \"test\" >> output.txt",
				Note:       "@file.temp only supports overwrite-only",
			},
		},
		{
			name:          "valid redirect to shell (file path)",
			input:         `echo "test" > output.txt`,
			expectedError: nil,
		},
		{
			name:          "valid append to shell (file path)",
			input:         `echo "test" >> output.txt`,
			expectedError: nil,
		},
		{
			name:          "valid redirect with pipe",
			input:         `echo "test" | grep "test" > output.txt`,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))
			tree.ValidateSemantics() // Post-parse validation

			if tt.expectedError != nil {
				if len(tree.Errors) == 0 {
					t.Errorf("expected parse error but got none")
					return
				}

				if diff := cmp.Diff(*tt.expectedError, tree.Errors[0]); diff != "" {
					t.Errorf("Error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("unexpected parse errors: %v", tree.Errors)
				}
			}
		})
	}
}

// TestNamespacedDecoratorParsing tests that decorators with dots in their names are recognized
// Note: "file.read" is registered in init(), but "file" is NOT registered
func TestNamespacedDecoratorParsing(t *testing.T) {
	// Test that @file.read is recognized (full path extraction works)
	t.Run("namespaced decorator recognized", func(t *testing.T) {
		input := `var x = @file.read("test.txt")`
		tree := Parse([]byte(input))

		// Count decorator nodes in parse tree
		decoratorCount := 0
		for _, event := range tree.Events {
			if event.Kind == EventOpen && NodeKind(event.Data) == NodeDecorator {
				decoratorCount++
			}
		}

		if decoratorCount == 0 {
			t.Fatal("@file.read was not recognized as a decorator - parser needs to extract full namespaced path")
		}

		if len(tree.Errors) > 0 {
			t.Errorf("Parser recognized @file.read but had errors:")
			for _, err := range tree.Errors {
				t.Logf("  %s", err.Message)
			}
		}
	})

	// Test that @file alone is NOT recognized (only file.read is registered)
	t.Run("base name alone not recognized", func(t *testing.T) {
		input := `var x = @file("test.txt")`
		tree := Parse([]byte(input))

		// Since "file" is not registered, @ should be treated as literal
		// This would likely cause a parse error or treat it as shell syntax
		// We just verify it doesn't crash - the exact behavior depends on context
		t.Logf("Parsed @file (not registered) - errors: %d", len(tree.Errors))
	})
}

// TestEnumParameterValidation tests that enum parameters are validated
func TestEnumParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid enum value: none",
			input:       `@shell("echo test", scrub="none")`,
			expectError: false,
		},
		{
			name:        "valid enum value: stdin",
			input:       `@shell("echo test", scrub="stdin")`,
			expectError: false,
		},
		{
			name:        "valid enum value: stdout",
			input:       `@shell("echo test", scrub="stdout")`,
			expectError: false,
		},
		{
			name:        "valid enum value: both",
			input:       `@shell("echo test", scrub="both")`,
			expectError: false,
		},
		{
			name:        "invalid enum value",
			input:       `@shell("echo test", scrub="invalid")`,
			expectError: true,
			errorMsg:    "invalid value",
		},
		{
			name:        "wrong type for enum",
			input:       `@shell("echo test", scrub=true)`,
			expectError: true,
			errorMsg:    "expects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.expectError {
				if len(tree.Errors) == 0 {
					t.Errorf("expected parse error but got none")
				} else {
					found := false
					for _, err := range tree.Errors {
						if strings.Contains(err.Message, tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error containing %q, got: %v", tt.errorMsg, tree.Errors)
					}
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("unexpected parse errors: %v", tree.Errors)
				}
			}
		})
	}
}


// TestValueDecoratorRejectsBlock verifies value decorators cannot take blocks
func TestValueDecoratorRejectsBlock(t *testing.T) {
	// @var is a value decorator - should NOT take a block
	input := `@var.name { echo "test" }`
	tree := ParseString(input)

	if len(tree.Errors) == 0 {
		t.Fatal("Expected error for value decorator with block")
	}

	err := tree.Errors[0]
	
	// Verify error message follows established format
	if !strings.Contains(err.Message, "@var") {
		t.Errorf("Error should mention decorator name, got: %q", err.Message)
	}
	
	if !strings.Contains(err.Message, "cannot have a block") {
		t.Errorf("Error should mention block restriction, got: %q", err.Message)
	}
	
	if err.Context != "decorator block" {
		t.Errorf("Context: got %q, want %q", err.Context, "decorator block")
	}
}

// TestExecDecoratorAllowsBlock verifies execution decorators can take blocks
func TestExecDecoratorAllowsBlock(t *testing.T) {
	// @retry is an execution decorator - should work with blocks
	input := `@retry(times=3) { echo "test" }`
	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("@retry should work with blocks, got errors: %v", tree.Errors)
	}
}

// TestEnvDecoratorRejectsBlock verifies @env cannot take blocks
func TestEnvDecoratorRejectsBlock(t *testing.T) {
	// @env is a value decorator - should NOT take a block
	input := `@env.HOME { echo "test" }`
	tree := ParseString(input)

	if len(tree.Errors) == 0 {
		t.Fatal("Expected error for @env with block")
	}

	err := tree.Errors[0]
	if !strings.Contains(err.Message, "@env") {
		t.Errorf("Error should mention @env, got: %q", err.Message)
	}
}
