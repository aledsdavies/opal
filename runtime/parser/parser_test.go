package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

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
