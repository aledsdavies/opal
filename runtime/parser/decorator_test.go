package parser

import (
	"testing"

	_ "github.com/aledsdavies/opal/runtime/decorators" // Register built-in decorators
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
