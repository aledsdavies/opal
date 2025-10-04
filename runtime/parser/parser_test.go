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

// TestParseErrors verifies error recovery for invalid syntax
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrors  bool
		description string
	}{
		{
			name:        "missing closing parenthesis",
			input:       "fun greet(name {}",
			wantErrors:  true,
			description: "should report missing )",
		},
		{
			name:        "missing function name",
			input:       "fun () {}",
			wantErrors:  false, // Parser might accept this, semantic analysis rejects
			description: "parser accepts, semantic analysis should reject",
		},
		{
			name:        "missing parameter name before colon",
			input:       "fun greet(: String) {}",
			wantErrors:  false, // Parser might accept this, semantic analysis rejects
			description: "parser accepts, semantic analysis should reject",
		},
		{
			name:        "trailing comma in parameters",
			input:       "fun greet(name,) {}",
			wantErrors:  false, // Parser might accept this, semantic analysis rejects
			description: "parser accepts, semantic analysis should reject",
		},
		{
			name:        "missing opening brace",
			input:       "fun greet() }",
			wantErrors:  true,
			description: "should report missing {",
		},
		{
			name:        "missing closing brace",
			input:       "fun greet() {",
			wantErrors:  true,
			description: "should report missing }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			hasErrors := len(tree.Errors) > 0

			if hasErrors != tt.wantErrors {
				t.Errorf("Error expectation mismatch: got errors=%v, want errors=%v\nErrors: %v\nDescription: %s",
					hasErrors, tt.wantErrors, tree.Errors, tt.description)
			}

			// Parser should still produce events even with errors (error recovery)
			if len(tree.Events) == 0 {
				t.Error("Parser should produce events even with errors (for error recovery)")
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
