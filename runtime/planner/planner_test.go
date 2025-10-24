package planner_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/runtime/lexer"
	"github.com/aledsdavies/opal/runtime/parser"
	"github.com/aledsdavies/opal/runtime/planner"
)

// Helper: Extract command argument from tree (assumes tree is a CommandNode)
func getCommandArg(tree planfmt.ExecutionNode, key string) string {
	if tree == nil {
		return ""
	}
	cmd, ok := tree.(*planfmt.CommandNode)
	if !ok {
		return ""
	}
	for _, arg := range cmd.Args {
		if arg.Key == key {
			return arg.Val.Str
		}
	}
	return ""
}

// Helper: Get decorator name from tree (assumes tree is a CommandNode)
func getDecorator(tree planfmt.ExecutionNode) string {
	if tree == nil {
		return ""
	}
	cmd, ok := tree.(*planfmt.CommandNode)
	if !ok {
		return ""
	}
	return cmd.Decorator
}

// TestSimpleShellCommand tests converting a simple shell command to @shell decorator
func TestSimpleShellCommand(t *testing.T) {
	source := []byte(`echo "Hello, World!"`)

	// Parse
	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Plan (script mode - no target)
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify plan structure
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	step := plan.Steps[0]
	if step.Tree == nil {
		t.Fatal("Expected tree, got nil")
	}

	// Tree should be a CommandNode with @shell decorator
	if getDecorator(step.Tree) != "@shell" {
		t.Errorf("Expected @shell decorator, got %q", getDecorator(step.Tree))
	}

	// Check command argument
	expectedCmd := `echo "Hello, World!"`
	if getCommandArg(step.Tree, "command") != expectedCmd {
		t.Errorf("Expected command %q, got %q", expectedCmd, getCommandArg(step.Tree, "command"))
	}
}

// TestShellCommandWithDashes tests that shell commands with dashes are reconstructed correctly
func TestShellCommandWithDashes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wc with -l flag",
			input:    `wc -l`,
			expected: `wc -l`,
		},
		{
			name:     "echo with -n flag",
			input:    `echo -n "hello"`,
			expected: `echo -n "hello"`,
		},
		{
			name:     "ls with -la flags",
			input:    `ls -la`,
			expected: `ls -la`,
		},
		{
			name:     "kubectl with -f flag",
			input:    `kubectl apply -f deployment.yaml`,
			expected: `kubectl apply -f deployment.yaml`,
		},
		{
			name:     "grep with -v flag",
			input:    `grep -v "pattern"`,
			expected: `grep -v "pattern"`,
		},
		{
			name:     "double dash --",
			input:    `echo -- "end of options"`,
			expected: `echo -- "end of options"`,
		},
		{
			name:     "long flag --file",
			input:    `command --file config.yaml`,
			expected: `command --file config.yaml`,
		},
		{
			name:     "double dash with space",
			input:    `git commit -- file.txt`,
			expected: `git commit -- file.txt`,
		},
		{
			name:     "mixed single and double dash",
			input:    `curl -X POST --data "test"`,
			expected: `curl -X POST --data "test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			tree := parser.Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			// Plan (script mode)
			plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "",
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			// Verify plan structure
			if len(plan.Steps) != 1 {
				t.Errorf("Expected 1 step, got %d", len(plan.Steps))
			}

			step := plan.Steps[0]
			if step.Tree == nil {
				t.Fatal("Expected tree, got nil")
			}

			// Tree should be a CommandNode with @shell decorator
			if getDecorator(step.Tree) != "@shell" {
				t.Errorf("Expected @shell decorator, got %q", getDecorator(step.Tree))
			}

			// Check command argument - this is the critical test
			actualCmd := getCommandArg(step.Tree, "command")
			if actualCmd != tt.expected {
				t.Errorf("Command mismatch:\n  want: %q\n  got:  %q", tt.expected, actualCmd)
			}
		})
	}
}

// TestMultipleShellCommands tests multiple newline-separated commands
func TestMultipleShellCommands(t *testing.T) {
	source := []byte(`echo "First"
echo "Second"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 steps (newline-separated)
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify first step
	if plan.Steps[0].Tree == nil {
		t.Fatal("Step 0: tree is nil")
	}
	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "First"` {
		t.Errorf("Step 0: wrong command: %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}

	// Verify second step
	if plan.Steps[1].Tree == nil {
		t.Fatal("Step 1: tree is nil")
	}
	if getCommandArg(plan.Steps[1].Tree, "command") != `echo "Second"` {
		t.Errorf("Step 1: wrong command: %q", getCommandArg(plan.Steps[1].Tree, "command"))
	}
}

// TestFunctionDefinition tests parsing a function definition
func TestFunctionDefinition(t *testing.T) {
	source := []byte(`fun hello = echo "Hello, World!"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Command mode - target "hello"
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (the function body)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's the shell command from the function body
	if plan.Steps[0].Tree == nil {
		t.Fatal("Expected tree, got nil")
	}

	if getDecorator(plan.Steps[0].Tree) != "@shell" {
		t.Errorf("Expected @shell, got %q", getDecorator(plan.Steps[0].Tree))
	}

	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "Hello, World!"` {
		t.Errorf("Wrong command: %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}
}

// TestCommandModeTargetSelection tests finding a specific function
func TestCommandModeTargetSelection(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"
fun goodbye = echo "Goodbye"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Target "goodbye" - should only plan that function
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "goodbye",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 1 step (only goodbye function body)
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Verify it's the goodbye command
	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "Goodbye"` {
		t.Errorf("Wrong command: %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}
}

// TestScriptModeFullExecution tests executing entire file (no target)
func TestScriptModeFullExecution(t *testing.T) {
	source := []byte(`echo "Top level"
fun hello = echo "In function"
echo "Another top level"`)

	tree := parser.Parse(source)

	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	// Script mode - no target, execute all top-level commands
	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have 2 steps (two top-level echo commands, function is skipped)
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Verify first command
	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "Top level"` {
		t.Errorf("Step 0: wrong command: %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}

	// Verify second command
	if getCommandArg(plan.Steps[1].Tree, "command") != `echo "Another top level"` {
		t.Errorf("Step 1: wrong command: %q", getCommandArg(plan.Steps[1].Tree, "command"))
	}
}

// TestEmptyPlan tests empty input
func TestEmptyPlan(t *testing.T) {
	source := []byte(``)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Empty plan is valid
	if len(plan.Steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(plan.Steps))
	}
}

// TestStepIDUniqueness tests that all step IDs are unique
func TestStepIDUniqueness(t *testing.T) {
	source := []byte(`echo "First"
echo "Second"
echo "Third"`)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Collect all step IDs
	seen := make(map[uint64]bool)
	for _, step := range plan.Steps {
		if seen[step.ID] {
			t.Errorf("Duplicate step ID: %d", step.ID)
		}
		seen[step.ID] = true
	}

	// All IDs should be unique
	if len(seen) != len(plan.Steps) {
		t.Errorf("Expected %d unique IDs, got %d", len(plan.Steps), len(seen))
	}
}

// TestArgSorting tests that args are sorted by key
func TestArgSorting(t *testing.T) {
	source := []byte(`echo "Hello"`)

	tree := parser.Parse(source)

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify args are sorted (for determinism)
	cmd, ok := plan.Steps[0].Tree.(*planfmt.CommandNode)
	if !ok {
		t.Fatalf("Expected CommandNode, got %T", plan.Steps[0].Tree)
	}
	for i := 1; i < len(cmd.Args); i++ {
		if cmd.Args[i-1].Key >= cmd.Args[i].Key {
			t.Errorf("Args not sorted: %q >= %q", cmd.Args[i-1].Key, cmd.Args[i].Key)
		}
	}
}

// TestTargetNotFound tests error when target function doesn't exist
func TestTargetNotFound(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"`)

	tree := parser.Parse(source)

	// Target a function that doesn't exist
	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "nonexistent",
	})

	if err == nil {
		t.Fatal("Expected error for nonexistent target, got nil")
	}

	// Error message should contain key information
	errMsg := err.Error()
	if !strings.Contains(errMsg, "command not found: nonexistent") {
		t.Errorf("Expected 'command not found: nonexistent', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Define the function") {
		t.Errorf("Expected suggestion in error message, got: %s", errMsg)
	}
}

// TestTargetNotFoundWithSuggestion tests "did you mean" suggestions
func TestTargetNotFoundWithSuggestion(t *testing.T) {
	source := []byte(`fun hello = echo "Hello"
fun deploy = echo "Deploying"`)

	tree := parser.Parse(source)

	// Target a function with a typo
	_, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "helo", // Missing 'l'
	})

	if err == nil {
		t.Fatal("Expected error for nonexistent target, got nil")
	}

	// Should suggest "hello"
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Did you mean 'hello'?") {
		t.Errorf("Expected 'Did you mean' suggestion, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "Available commands:") {
		t.Errorf("Expected available commands list, got: %s", errMsg)
	}
}

func TestPlanShellCommandWithOperators(t *testing.T) {
	// Test that commands with operators are grouped into a single step with tree structure
	source := []byte(`fun test = echo "A" && echo "B"`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "test",
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have ONE step with an AndNode tree
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	// Tree should be an AndNode
	andNode, ok := plan.Steps[0].Tree.(*planfmt.AndNode)
	if !ok {
		t.Fatalf("Expected AndNode, got %T", plan.Steps[0].Tree)
	}

	// Left side should be echo "A"
	if getCommandArg(andNode.Left, "command") != `echo "A"` {
		t.Errorf("Left command wrong: %q", getCommandArg(andNode.Left, "command"))
	}

	// Right side should be echo "B"
	if getCommandArg(andNode.Right, "command") != `echo "B"` {
		t.Errorf("Right command wrong: %q", getCommandArg(andNode.Right, "command"))
	}
}

func TestPlanMultipleSteps(t *testing.T) {
	// Test that newline-separated commands in SCRIPT MODE create separate steps
	source := []byte("echo \"A\"\necho \"B\"")

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have TWO steps, each with a tree
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].Tree == nil {
		t.Fatal("Expected first step to have tree, got nil")
	}

	if plan.Steps[1].Tree == nil {
		t.Fatal("Expected second step to have tree, got nil")
	}
}

func TestPlanFunctionWithOperatorsAndNewline(t *testing.T) {
	// Test: echo "A" && echo "B" || echo "C"\necho "D"
	// In SCRIPT MODE (no target), should produce: 2 steps
	// Step 1: OrNode with AndNode on left (operator precedence: && > ||)
	// Step 2: Simple command (newline creates new step)
	source := []byte(`echo "A" && echo "B" || echo "C"
echo "D"`)

	tree := parser.Parse(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse errors: %v", tree.Errors)
	}

	plan, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
		Target: "", // Script mode
	})
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Should have TWO steps
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	// Step 1: Should be OrNode with structure: (A && B) || C
	orNode, ok := plan.Steps[0].Tree.(*planfmt.OrNode)
	if !ok {
		t.Fatalf("Step 0: Expected OrNode, got %T", plan.Steps[0].Tree)
	}

	// Left side of OR should be AndNode: A && B
	andNode, ok := orNode.Left.(*planfmt.AndNode)
	if !ok {
		t.Fatalf("Step 0 left: Expected AndNode, got %T", orNode.Left)
	}

	// Verify A && B
	if getCommandArg(andNode.Left, "command") != `echo "A"` {
		t.Errorf("Step 0 AND left: expected 'echo \"A\"', got %q", getCommandArg(andNode.Left, "command"))
	}
	if getCommandArg(andNode.Right, "command") != `echo "B"` {
		t.Errorf("Step 0 AND right: expected 'echo \"B\"', got %q", getCommandArg(andNode.Right, "command"))
	}

	// Right side of OR should be C
	if getCommandArg(orNode.Right, "command") != `echo "C"` {
		t.Errorf("Step 0 OR right: expected 'echo \"C\"', got %q", getCommandArg(orNode.Right, "command"))
	}

	// Step 2: Simple command echo "D"
	if plan.Steps[1].Tree == nil {
		t.Fatal("Step 1: tree is nil")
	}

	if getCommandArg(plan.Steps[1].Tree, "command") != `echo "D"` {
		t.Errorf("Step 1: expected 'echo \"D\"', got %q", getCommandArg(plan.Steps[1].Tree, "command"))
	}
}

// TestContractStability verifies that changing an unrelated function
// doesn't invalidate the contract for the target function
func TestContractStability(t *testing.T) {
	// Original source with two functions
	source1 := []byte(`fun hello = echo "Hello"
fun log = echo "Log"`)

	// Modified source (only log changed)
	source2 := []byte(`fun hello = echo "Hello"
fun log = echo "Different log"`)

	// Plan hello from source1
	tree1 := parser.Parse(source1)
	lex1 := lexer.NewLexer()
	lex1.Init(source1)
	plan1, err := planner.Plan(tree1.Events, lex1.GetTokens(), planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan1 failed: %v", err)
	}

	// Compute hash for plan1
	var buf1 bytes.Buffer
	hash1, err := planfmt.Write(&buf1, plan1)
	if err != nil {
		t.Fatalf("Write plan1 failed: %v", err)
	}

	// Plan hello from source2
	tree2 := parser.Parse(source2)
	lex2 := lexer.NewLexer()
	lex2.Init(source2)
	plan2, err := planner.Plan(tree2.Events, lex2.GetTokens(), planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan2 failed: %v", err)
	}

	// Compute hash for plan2
	var buf2 bytes.Buffer
	hash2, err := planfmt.Write(&buf2, plan2)
	if err != nil {
		t.Fatalf("Write plan2 failed: %v", err)
	}

	// Hashes should be IDENTICAL (hello didn't change)
	if hash1 != hash2 {
		t.Errorf("Contract instability detected!\nChanging 'log' function invalidated 'hello' contract\nhash1: %x\nhash2: %x", hash1, hash2)
	}
}

// TestContractStabilityWithNewFunction verifies that adding a new function
// doesn't invalidate existing contracts
func TestContractStabilityWithNewFunction(t *testing.T) {
	// Original source
	source1 := []byte(`fun hello = echo "Hello"`)

	// Modified source (new function added)
	source2 := []byte(`fun hello = echo "Hello"
fun log = echo "Log"`)

	// Plan hello from source1
	tree1 := parser.Parse(source1)
	lex1 := lexer.NewLexer()
	lex1.Init(source1)
	plan1, err := planner.Plan(tree1.Events, lex1.GetTokens(), planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan1 failed: %v", err)
	}

	// Compute hash for plan1
	var buf1 bytes.Buffer
	hash1, err := planfmt.Write(&buf1, plan1)
	if err != nil {
		t.Fatalf("Write plan1 failed: %v", err)
	}

	// Plan hello from source2
	tree2 := parser.Parse(source2)
	lex2 := lexer.NewLexer()
	lex2.Init(source2)
	plan2, err := planner.Plan(tree2.Events, lex2.GetTokens(), planner.Config{
		Target: "hello",
	})
	if err != nil {
		t.Fatalf("Plan2 failed: %v", err)
	}

	// Compute hash for plan2
	var buf2 bytes.Buffer
	hash2, err := planfmt.Write(&buf2, plan2)
	if err != nil {
		t.Fatalf("Write plan2 failed: %v", err)
	}

	// Hashes should be IDENTICAL (hello didn't change)
	if hash1 != hash2 {
		t.Errorf("Contract instability detected!\nAdding 'log' function invalidated 'hello' contract\nhash1: %x\nhash2: %x", hash1, hash2)
	}
}

// TestRedirectOperators tests output redirection (> and >>)
func TestRedirectOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode planfmt.RedirectMode
	}{
		{
			name:     "simple redirect overwrite",
			input:    `echo "hello" > output.txt`,
			wantMode: planfmt.RedirectOverwrite,
		},
		{
			name:     "simple redirect append",
			input:    `echo "world" >> output.txt`,
			wantMode: planfmt.RedirectAppend,
		},
		{
			name:     "redirect with variable",
			input:    `echo "data" > @var.OUTPUT_FILE`,
			wantMode: planfmt.RedirectOverwrite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			tree := parser.Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			// Plan (script mode)
			result, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "",
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			// Verify plan structure
			if len(result.Steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(result.Steps))
			}

			step := result.Steps[0]
			if step.Tree == nil {
				t.Fatal("Expected tree, got nil")
			}

			// Tree should be a RedirectNode
			redirectNode, ok := step.Tree.(*planfmt.RedirectNode)
			if !ok {
				t.Fatalf("Expected RedirectNode, got %T", step.Tree)
			}

			// Check redirect mode
			if redirectNode.Mode != tt.wantMode {
				t.Errorf("Expected mode %v, got %v", tt.wantMode, redirectNode.Mode)
			}

			// Source should be a CommandNode with @shell
			sourceCmd, ok := redirectNode.Source.(*planfmt.CommandNode)
			if !ok {
				t.Fatalf("Expected source to be CommandNode, got %T", redirectNode.Source)
			}
			if sourceCmd.Decorator != "@shell" {
				t.Errorf("Expected source decorator @shell, got %q", sourceCmd.Decorator)
			}

			// Target should be a CommandNode with @shell
			if redirectNode.Target.Decorator != "@shell" {
				t.Errorf("Expected target decorator @shell, got %q", redirectNode.Target.Decorator)
			}
		})
	}
}

// TestRedirectWithChaining tests redirect followed by chaining operators (&&, ||)
// Bug: Previously, redirect consumed the operator slot, so && was never captured
func TestRedirectWithChaining(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantMode     planfmt.RedirectMode
		wantOperator string // Expected operator AFTER redirect
	}{
		{
			name:         "redirect then AND",
			input:        `echo "a" > out.txt && echo "b"`,
			wantMode:     planfmt.RedirectOverwrite,
			wantOperator: "&&",
		},
		{
			name:         "redirect then OR",
			input:        `echo "a" > out.txt || echo "b"`,
			wantMode:     planfmt.RedirectOverwrite,
			wantOperator: "||",
		},
		{
			name:         "append then AND",
			input:        `echo "a" >> out.txt && echo "b"`,
			wantMode:     planfmt.RedirectAppend,
			wantOperator: "&&",
		},
		{
			name:         "redirect then PIPE",
			input:        `echo "a" > out.txt | cat`,
			wantMode:     planfmt.RedirectOverwrite,
			wantOperator: "|",
		},
		{
			name:         "redirect then SEMICOLON",
			input:        `echo "a" > out.txt; echo "b"`,
			wantMode:     planfmt.RedirectOverwrite,
			wantOperator: ";",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			tree := parser.Parse([]byte(tt.input))

			if len(tree.Errors) > 0 {
				t.Fatalf("Parse errors: %v", tree.Errors)
			}

			// Plan (script mode)
			result, err := planner.Plan(tree.Events, tree.Tokens, planner.Config{
				Target: "",
			})
			if err != nil {
				t.Fatalf("Plan failed: %v", err)
			}

			// Verify plan structure
			if len(result.Steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(result.Steps))
			}

			step := result.Steps[0]
			if step.Tree == nil {
				t.Fatal("Expected tree, got nil")
			}

			// Tree should be an operator node (&&, ||, |, ;)
			// with left side being a RedirectNode
			var leftNode, rightNode planfmt.ExecutionNode
			switch tt.wantOperator {
			case "&&":
				andNode, ok := step.Tree.(*planfmt.AndNode)
				if !ok {
					t.Fatalf("Expected AndNode for &&, got %T", step.Tree)
				}
				leftNode = andNode.Left
				rightNode = andNode.Right
			case "||":
				orNode, ok := step.Tree.(*planfmt.OrNode)
				if !ok {
					t.Fatalf("Expected OrNode for ||, got %T", step.Tree)
				}
				leftNode = orNode.Left
				rightNode = orNode.Right
			case "|":
				pipeNode, ok := step.Tree.(*planfmt.PipelineNode)
				if !ok {
					t.Fatalf("Expected PipelineNode for |, got %T", step.Tree)
				}
				if len(pipeNode.Commands) != 2 {
					t.Fatalf("Expected 2 pipeline commands, got %d", len(pipeNode.Commands))
				}
				// For pipe, we need to check the first command is a redirect
				// This is a special case - pipe doesn't have Left/Right like And/Or
				t.Skip("Pipe with redirect needs special handling - skipping for now")
				return
			case ";":
				seqNode, ok := step.Tree.(*planfmt.SequenceNode)
				if !ok {
					t.Fatalf("Expected SequenceNode for ;, got %T", step.Tree)
				}
				if len(seqNode.Nodes) != 2 {
					t.Fatalf("Expected 2 sequence nodes, got %d", len(seqNode.Nodes))
				}
				leftNode = seqNode.Nodes[0]
				rightNode = seqNode.Nodes[1]
			default:
				t.Fatalf("Unknown operator: %q", tt.wantOperator)
			}

			// Left side should be a RedirectNode
			redirectNode, ok := leftNode.(*planfmt.RedirectNode)
			if !ok {
				t.Fatalf("Expected left side to be RedirectNode, got %T", leftNode)
			}

			// Check redirect mode
			if redirectNode.Mode != tt.wantMode {
				t.Errorf("Expected redirect mode %v, got %v", tt.wantMode, redirectNode.Mode)
			}

			// Right side should be a CommandNode
			rightCmd, ok := rightNode.(*planfmt.CommandNode)
			if !ok {
				t.Fatalf("Expected right side to be CommandNode, got %T", rightNode)
			}

			// Right command should be @shell
			if rightCmd.Decorator != "@shell" {
				t.Errorf("Expected right decorator @shell, got %q", rightCmd.Decorator)
			}
		})
	}
}
