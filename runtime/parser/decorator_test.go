package parser

import (
	"testing"

	_ "github.com/aledsdavies/opal/runtime/decorators" // Register built-in decorators
	"github.com/google/go-cmp/cmp"
)

// TestDecoratorDetection tests that parser recognizes registered decorators
func TestDecoratorDetection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isDecorator bool
		reason      string
	}{
		{
			name:        "var decorator at top level",
			input:       "@var.name",
			isDecorator: true,
			reason:      "var is a registered decorator",
		},
		{
			name:        "env decorator at top level",
			input:       "@env.HOME",
			isDecorator: true,
			reason:      "env is a registered decorator",
		},
		{
			name:        "var decorator in assignment",
			input:       "var x = @var.name",
			isDecorator: true,
			reason:      "var is a registered decorator",
		},
		{
			name:        "env decorator in assignment",
			input:       "var home = @env.HOME",
			isDecorator: true,
			reason:      "env is a registered decorator",
		},
		{
			name:        "unknown decorator not recognized",
			input:       "@unknown.field",
			isDecorator: false,
			reason:      "unknown is not registered",
		},
		{
			name:        "email address not recognized as decorator",
			input:       "user@example.com",
			isDecorator: false,
			reason:      "example is not a registered decorator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			// Count decorator nodes
			decoratorCount := 0
			for _, evt := range tree.Events {
				if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
					decoratorCount++
				}
			}

			if tt.isDecorator && decoratorCount == 0 {
				t.Errorf("expected decorator node for %q (%s)", tt.input, tt.reason)
			}

			if !tt.isDecorator && decoratorCount > 0 {
				t.Errorf("expected no decorator node for %q (%s)", tt.input, tt.reason)
			}
		})
	}
}

// TestDecoratorInShellCommand tests decorator interpolation in shell commands
func TestDecoratorInShellCommand(t *testing.T) {
	input := `echo "Hello @var.name"`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected parse errors: %v", tree.Errors)
	}

	// Should have at least one decorator node
	hasDecorator := false
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
			hasDecorator = true
			break
		}
	}

	if !hasDecorator {
		t.Error("expected decorator node in shell command with @var.name")
	}
}

// TestLiteralAtSymbol tests that @ without registered decorator stays literal
func TestLiteralAtSymbol(t *testing.T) {
	input := `echo "Email: user@example.com"`

	tree := Parse([]byte(input))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected parse errors: %v", tree.Errors)
	}

	// Should NOT have decorator nodes (example is not registered)
	decoratorCount := 0
	for _, evt := range tree.Events {
		if evt.Kind == EventOpen && NodeKind(evt.Data) == NodeDecorator {
			decoratorCount++
		}
	}

	if decoratorCount > 0 {
		t.Errorf("expected no decorator nodes for literal @ in email address, got %d", decoratorCount)
	}
}

// TestDecoratorParameters tests parsing decorator parameters with exact event sequences
func TestDecoratorParameters(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "primary only - var",
			input: "@var.username",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // var
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // username
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "primary with single param",
			input: `@env.HOME(default="")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // HOME
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 4}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 5}, // default
				{Kind: EventToken, Data: 6}, // =
				{Kind: EventToken, Data: 7}, // ""
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 8}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "multiple params",
			input: `@env.HOME(default="/home/user")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventToken, Data: 2}, // .
				{Kind: EventToken, Data: 3}, // HOME
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 4}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 5}, // default
				{Kind: EventToken, Data: 6}, // =
				{Kind: EventToken, Data: 7}, // "/home/user"
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 8}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "all named params (unsugared)",
			input: `@env(property="HOME")`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeDecorator)},
				{Kind: EventToken, Data: 0}, // @
				{Kind: EventToken, Data: 1}, // env
				{Kind: EventOpen, Data: uint32(NodeParamList)},
				{Kind: EventToken, Data: 2}, // (
				{Kind: EventOpen, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 3}, // property
				{Kind: EventToken, Data: 4}, // =
				{Kind: EventToken, Data: 5}, // "HOME"
				{Kind: EventClose, Data: uint32(NodeParam)},
				{Kind: EventToken, Data: 6}, // )
				{Kind: EventClose, Data: uint32(NodeParamList)},
				{Kind: EventClose, Data: uint32(NodeDecorator)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			// Should have no errors
			if len(tree.Errors) > 0 {
				t.Errorf("unexpected errors: %v", tree.Errors)
			}

			// Compare events using cmp.Diff for exact match
			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestDecoratorParameterTypeValidation tests type checking for decorator parameters
func TestDecoratorParameterTypeValidation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		// Positive cases - correct types
		{
			name:      "string param with string value",
			input:     `@env.HOME(default="")`,
			wantError: false,
		},
		{
			name:      "string param with string value - non-empty",
			input:     `@env.HOME(default="/home/user")`,
			wantError: false,
		},
		{
			name:      "multiple params with correct types",
			input:     `@env.HOME(default="/home")`,
			wantError: false,
		},

		// Negative cases - type mismatches
		{
			name:           "string param with integer value",
			input:          `@env.HOME(default=42)`,
			wantError:      true,
			wantMessage:    "parameter 'default' expects string, got integer",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a string value like \"value\"",
		},
		{
			name:           "string param with boolean value",
			input:          `@env.HOME(default=true)`,
			wantError:      true,
			wantMessage:    "parameter 'default' expects string, got boolean",
			wantContext:    "decorator parameter",
			wantSuggestion: "Use a string value like \"value\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorRequiredParameters tests validation of required parameters
func TestDecoratorRequiredParameters(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		// Positive cases - required params provided
		{
			name:      "primary param provided via dot syntax",
			input:     `@env.HOME`,
			wantError: false,
		},
		{
			name:      "primary param provided via named param",
			input:     `@env(property="HOME")`,
			wantError: false,
		},

		// Negative cases - missing required params
		{
			name:           "missing primary param - no dot, no named param",
			input:          `@env`,
			wantError:      true,
			wantMessage:    "missing required parameter 'property'",
			wantContext:    "decorator parameters",
			wantSuggestion: "Use dot syntax like @env.HOME or provide property=\"HOME\"",
		},
		{
			name:           "missing primary param - empty parens",
			input:          `@env()`,
			wantError:      true,
			wantMessage:    "missing required parameter 'property'",
			wantContext:    "decorator parameters",
			wantSuggestion: "Use dot syntax like @env.HOME or provide property=\"HOME\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}

// TestDecoratorUnknownParameter tests validation of unknown parameters
func TestDecoratorUnknownParameter(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantError      bool
		wantMessage    string
		wantContext    string
		wantSuggestion string
	}{
		{
			name:      "known parameter",
			input:     `@env.HOME(default="")`,
			wantError: false,
		},
		{
			name:           "unknown parameter",
			input:          `@env.HOME(unknown="value")`,
			wantError:      true,
			wantMessage:    "unknown parameter 'unknown' for @env",
			wantContext:    "decorator parameter",
			wantSuggestion: "Valid parameters: default, property",
		},
		{
			name:           "mix of known and unknown parameters",
			input:          `@env.HOME(default="", invalid=true)`,
			wantError:      true,
			wantMessage:    "unknown parameter 'invalid' for @env",
			wantContext:    "decorator parameter",
			wantSuggestion: "Valid parameters: default, property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.input))

			if tt.wantError {
				if len(tree.Errors) == 0 {
					t.Fatal("Expected error but got none")
				}

				err := tree.Errors[0]

				if err.Message != tt.wantMessage {
					t.Errorf("Message mismatch:\ngot:  %q\nwant: %q", err.Message, tt.wantMessage)
				}

				if err.Context != tt.wantContext {
					t.Errorf("Context mismatch:\ngot:  %q\nwant: %q", err.Context, tt.wantContext)
				}

				if err.Suggestion != tt.wantSuggestion {
					t.Errorf("Suggestion mismatch:\ngot:  %q\nwant: %q", err.Suggestion, tt.wantSuggestion)
				}
			} else {
				if len(tree.Errors) > 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}
