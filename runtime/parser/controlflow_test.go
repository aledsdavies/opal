package parser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIfStatement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "simple if with boolean",
			input: "fun test { if true { echo \"yes\" } }",
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 10},  // If
				{EventToken, 3},  // if
				{EventToken, 4},  // true (condition)
				{EventOpen, 3},   // Block
				{EventToken, 5},  // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 6},  // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 7},  // "yes"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 8},  // }
				{EventClose, 3},  // Block
				{EventClose, 10}, // If
				{EventToken, 9},  // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

func TestIfElseStatement(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "if-else",
			input: "fun test { if true { echo \"yes\" } else { echo \"no\" } }",
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 10},  // If
				{EventToken, 3},  // if
				{EventToken, 4},  // true
				{EventOpen, 3},   // Block
				{EventToken, 5},  // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 6},  // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 7},  // "yes"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 8},  // }
				{EventClose, 3},  // Block
				{EventOpen, 11},  // Else
				{EventToken, 9},  // else
				{EventOpen, 3},   // Block
				{EventToken, 10}, // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 11}, // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 12}, // "no"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 13}, // }
				{EventClose, 3},  // Block
				{EventClose, 11}, // Else
				{EventClose, 10}, // If
				{EventToken, 14}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

func TestIfElseIfChain(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		events []Event
	}{
		{
			name:  "if-else-if-else",
			input: "fun test { if true { echo \"a\" } else if false { echo \"b\" } else { echo \"c\" } }",
			events: []Event{
				{EventOpen, 0},   // Source
				{EventOpen, 1},   // Function
				{EventToken, 0},  // fun
				{EventToken, 1},  // test
				{EventOpen, 3},   // Block
				{EventToken, 2},  // {
				{EventOpen, 10},  // If
				{EventToken, 3},  // if
				{EventToken, 4},  // true
				{EventOpen, 3},   // Block
				{EventToken, 5},  // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 6},  // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 7},  // "a"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 8},  // }
				{EventClose, 3},  // Block
				{EventOpen, 11},  // Else
				{EventToken, 9},  // else
				{EventOpen, 10},  // If (nested)
				{EventToken, 10}, // if
				{EventToken, 11}, // false
				{EventOpen, 3},   // Block
				{EventToken, 12}, // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 13}, // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 14}, // "b"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 15}, // }
				{EventClose, 3},  // Block
				{EventOpen, 11},  // Else
				{EventToken, 16}, // else
				{EventOpen, 3},   // Block
				{EventToken, 17}, // {
				{EventOpen, 8},   // ShellCommand
				{EventOpen, 9},   // ShellArg
				{EventToken, 18}, // echo
				{EventClose, 9},  // ShellArg
				{EventOpen, 9},   // ShellArg
				{EventToken, 19}, // "c"
				{EventClose, 9},  // ShellArg
				{EventClose, 8},  // ShellCommand
				{EventToken, 20}, // }
				{EventClose, 3},  // Block
				{EventClose, 11}, // Else
				{EventClose, 10}, // If (nested)
				{EventClose, 11}, // Else
				{EventClose, 10}, // If
				{EventToken, 21}, // }
				{EventClose, 3},  // Block
				{EventClose, 1},  // Function
				{EventClose, 0},  // Source
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

func TestIfAtTopLevel(t *testing.T) {
	// If statements ARE allowed at top level (script mode)
	input := "if true { echo \"hello\" }"

	tree := ParseString(input)

	if len(tree.Errors) != 0 {
		t.Errorf("Expected no errors for top-level if (script mode), got: %v", tree.Errors)
	}

	// Should have events for the if statement
	if len(tree.Events) == 0 {
		t.Error("Expected events for if statement, got none")
	}
}

func TestFunInsideControlFlow(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "fun inside if block",
			input: "fun test { if true { fun helper() { } } }",
		},
		{
			name:  "fun inside else block",
			input: "fun test { if true { } else { fun helper() { } } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Fatal("Expected error for fun inside control flow, got none")
			}

			err := tree.Errors[0]
			if err.Message != "function declarations must be at top level" {
				t.Errorf("Expected error about fun at top level, got: %s", err.Message)
			}
		})
	}
}

func TestElseWithoutIf(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "else at start of block",
			input: "fun test { else { echo \"hello\" } }",
		},
		{
			name:  "else after shell command",
			input: "fun test { echo \"hello\" \n else { echo \"world\" } }",
		},
		{
			name:  "else after var declaration",
			input: "fun test { var x = 5 \n else { echo \"world\" } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) == 0 {
				t.Fatal("Expected error for else without if, got none")
			}

			err := tree.Errors[0]
			if err.Message != "else without matching if" {
				t.Errorf("Expected error 'else without matching if', got: %s", err.Message)
			}
			if err.Context != "statement" {
				t.Errorf("Expected context 'statement', got: %s", err.Context)
			}
		})
	}
}

func TestIfStatementErrorRecovery(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		minErrorCount int
		containsError string
	}{
		{
			name:          "missing condition - block immediately after if",
			input:         "fun test { if { echo \"hello\" } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "missing block after condition",
			input:         "fun test { if true }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "missing block after else",
			input:         "fun test { if true { } else }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "nested if missing condition",
			input:         "fun test { if true { if { } } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "else if missing condition",
			input:         "fun test { if true { } else if { } }",
			minErrorCount: 1,
			containsError: "missing condition after 'if'",
		},
		{
			name:          "orphaned else with type error",
			input:         "fun test { else if 42 { } }",
			minErrorCount: 2,
			containsError: "else without matching if",
		},
		{
			name:          "type error and missing block",
			input:         "fun test { if \"string\" }",
			minErrorCount: 2,
			containsError: "if condition must be a boolean expression",
		},
		{
			name:          "multiple if statements with errors",
			input:         "fun test { if true { } if 42 { } }",
			minErrorCount: 1,
			containsError: "if condition must be a boolean expression",
		},
		{
			name:          "if with statement instead of block",
			input:         "fun test { if true echo \"hi\" }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
		{
			name:          "else with statement instead of block",
			input:         "fun test { if true { } else echo \"hi\" }",
			minErrorCount: 1,
			containsError: "missing '{'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if len(tree.Errors) < tt.minErrorCount {
				t.Errorf("Expected at least %d error(s), got %d: %v",
					tt.minErrorCount, len(tree.Errors), tree.Errors)
				return
			}

			// Check that at least one error contains the expected message
			found := false
			for _, err := range tree.Errors {
				if containsSubstring(err.Message, tt.containsError) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error containing '%s', got errors: %v",
					tt.containsError, tree.Errors)
			}

			// Verify parser didn't panic and produced some events
			if len(tree.Events) == 0 {
				t.Error("Parser produced no events (possible panic or early exit)")
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIfConditionTypeChecking(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "string literal condition",
			input:     `fun test { if "hello" { echo "yes" } }`,
			expectErr: true,
			errMsg:    "if condition must be a boolean expression",
		},
		{
			name:      "integer literal condition",
			input:     `fun test { if 42 { echo "yes" } }`,
			expectErr: true,
			errMsg:    "if condition must be a boolean expression",
		},
		{
			name:      "boolean true",
			input:     `fun test { if true { echo "yes" } }`,
			expectErr: false,
		},
		{
			name:      "boolean false",
			input:     `fun test { if false { echo "yes" } }`,
			expectErr: false,
		},
		{
			name:      "identifier (could be boolean)",
			input:     `fun test { if isReady { echo "yes" } }`,
			expectErr: false, // Identifiers are allowed (runtime check)
		},
		{
			name:      "decorator (could be boolean)",
			input:     `fun test { if @var.enabled { echo "yes" } }`,
			expectErr: false, // Decorators are allowed (runtime check)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := ParseString(tt.input)

			if tt.expectErr {
				if len(tree.Errors) == 0 {
					t.Fatalf("Expected error for non-boolean condition, got none")
				}
				err := tree.Errors[0]
				if err.Message != tt.errMsg {
					t.Errorf("Expected error '%s', got: %s", tt.errMsg, err.Message)
				}
			} else {
				if len(tree.Errors) != 0 {
					t.Errorf("Expected no errors, got: %v", tree.Errors)
				}
			}
		})
	}
}
