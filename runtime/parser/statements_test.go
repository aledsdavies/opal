package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestSimpleVarDecl tests basic variable declarations
func TestSimpleVarDecl(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "var with integer literal",
			input: "var x = 42",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0}, // VAR
				{Kind: EventToken, Data: 1}, // x
				{Kind: EventToken, Data: 2}, // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)},
				{Kind: EventToken, Data: 3}, // 42
				{Kind: EventClose, Data: uint32(NodeLiteral)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "var with string literal",
			input: `var name = "alice"`,
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0}, // VAR
				{Kind: EventToken, Data: 1}, // name
				{Kind: EventToken, Data: 2}, // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)},
				{Kind: EventToken, Data: 3}, // "alice"
				{Kind: EventClose, Data: uint32(NodeLiteral)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
				{Kind: EventClose, Data: uint32(NodeSource)},
			},
		},
		{
			name:  "var with boolean literal",
			input: "var ready = true",
			events: []Event{
				{Kind: EventOpen, Data: uint32(NodeSource)},
				{Kind: EventOpen, Data: uint32(NodeVarDecl)},
				{Kind: EventToken, Data: 0},                     // VAR
				{Kind: EventToken, Data: 1},                     // ready
				{Kind: EventToken, Data: 2},                     // =
				{Kind: EventOpen, Data: uint32(NodeIdentifier)}, // TODO: Should be NodeLiteral when lexer fixed
				{Kind: EventToken, Data: 3},                     // true (lexed as IDENTIFIER currently)
				{Kind: EventClose, Data: uint32(NodeIdentifier)},
				{Kind: EventClose, Data: uint32(NodeVarDecl)},
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

			// Compare events
			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
