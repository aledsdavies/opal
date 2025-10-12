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
				{Kind: EventToken, Data: 0},                  // VAR
				{Kind: EventToken, Data: 1},                  // ready
				{Kind: EventToken, Data: 2},                  // =
				{Kind: EventOpen, Data: uint32(NodeLiteral)}, // Now correctly recognized as literal
				{Kind: EventToken, Data: 3},                  // true (lexed as BOOLEAN)
				{Kind: EventClose, Data: uint32(NodeLiteral)},
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

// TestAssignmentOperators tests assignment operators (+=, -=, *=, /=, %=)
func TestAssignmentOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "plus assign with integer",
			input: "fun test { total += 5 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // total
				{EventToken, 4}, // +=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 5
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 6}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "minus assign with decorator",
			input: "fun test { remaining -= @var.cost }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // remaining
				{EventToken, 4}, // -=
				{EventOpen, uint32(NodeDecorator)},
				{EventToken, 5}, // @
				{EventToken, 6}, // var
				{EventToken, 7}, // .
				{EventToken, 8}, // cost
				{EventClose, uint32(NodeDecorator)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 9}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "multiply assign",
			input: "fun test { replicas *= 3 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // replicas
				{EventToken, 4}, // *=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 3
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 6}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "divide assign",
			input: "fun test { batch_size /= 2 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // batch_size
				{EventToken, 4}, // /=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 2
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 6}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "modulo assign",
			input: "fun test { index %= 10 }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // index
				{EventToken, 4}, // %=
				{EventOpen, uint32(NodeLiteral)},
				{EventToken, 5}, // 10
				{EventClose, uint32(NodeLiteral)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 6}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},

		{
			name:  "assignment with expression",
			input: "fun test { total += x + y }",
			events: []Event{
				{EventOpen, uint32(NodeSource)},
				{EventOpen, uint32(NodeFunction)},
				{EventToken, 0}, // fun
				{EventToken, 1}, // test
				{EventOpen, uint32(NodeBlock)},
				{EventToken, 2}, // {
				{EventOpen, uint32(NodeAssignment)},
				{EventToken, 3}, // total
				{EventToken, 4}, // +=
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 5}, // x
				{EventClose, uint32(NodeIdentifier)},
				{EventOpen, uint32(NodeBinaryExpr)},
				{EventToken, 6}, // +
				{EventOpen, uint32(NodeIdentifier)},
				{EventToken, 7}, // y
				{EventClose, uint32(NodeIdentifier)},
				{EventClose, uint32(NodeBinaryExpr)},
				{EventClose, uint32(NodeAssignment)},
				{EventToken, 8}, // }
				{EventClose, uint32(NodeBlock)},
				{EventClose, uint32(NodeFunction)},
				{EventClose, uint32(NodeSource)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) != 0 {
				t.Errorf("Expected no errors, got: %v", tree.Errors)
			}

			if diff := cmp.Diff(tt.events, tree.Events); diff != "" {
				t.Errorf("Events mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
