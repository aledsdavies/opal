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
		t.Errorf("Step 0: expected 1 command, got %d", len(plan.Steps[0].Tree))
	}
	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "First"` {
		t.Errorf("Step 0: wrong command: %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}

	// Verify second step
	if plan.Steps[1].Tree == nil {
		t.Errorf("Step 1: expected 1 command, got %d", len(plan.Steps[1].Tree))
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
		t.Fatalf("Expected 1 command, got %d", len(plan.Steps[0].Tree))
	}

	cmd := plan.Steps[0].Tree[0]
	if cmd.Decorator != "@shell" {
		t.Errorf("Expected @shell, got %q", cmd.Decorator)
	}

	if cmd.Args[0].Val.Str != `echo "Hello, World!"` {
		t.Errorf("Wrong command: %q", cmd.Args[0].Val.Str)
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
	cmd := plan.Steps[0].Tree[0]
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
	// Test that commands with operators are grouped into a single step
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

	// Should have ONE step with TWO commands
	if len(plan.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(plan.Steps))
	}

	if len(plan.Steps[0].Tree) != 2 {
		t.Errorf("Expected 2 commands in step, got %d", len(plan.Steps[0].Tree))
	}

	// First command should have && operator
	if plan.Steps[0].Tree[0].Operator != "&&" {
		t.Errorf("Expected first command to have && operator, got %q", plan.Steps[0].Tree[0].Operator)
	}

	// Second command should have empty operator (last in step)
	if plan.Steps[0].Tree[1].Operator != "" {
		t.Errorf("Expected second command to have empty operator, got %q", plan.Steps[0].Tree[1].Operator)
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

	// Should have TWO steps, each with ONE command
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	if plan.Steps[0].Tree == nil {
		t.Errorf("Expected first step to have 1 command, got %d", len(plan.Steps[0].Tree))
	}

	if plan.Steps[1].Tree == nil {
		t.Errorf("Expected second step to have 1 command, got %d", len(plan.Steps[1].Tree))
	}
}

func TestPlanFunctionWithOperatorsAndNewline(t *testing.T) {
	// Test: fun hello = echo "A" && echo "B" || echo "C"\necho "D"
	// In SCRIPT MODE (no target), should produce: 2 steps, 4 commands total
	// Step 1: 3 commands (A && B || C) - top-level operators in one step
	// Step 2: 1 command (D) - newline creates new step
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

	// Step 1: THREE commands with operators
	if len(plan.Steps[0].Tree) != 3 {
		t.Errorf("Expected step 0 to have 3 commands, got %d", len(plan.Steps[0].Tree))
	}

	// Verify step 1 commands and operators
	if getCommandArg(plan.Steps[0].Tree, "command") != `echo "A"` {
		t.Errorf("Step 0, cmd 0: expected 'echo \"A\"', got %q", getCommandArg(plan.Steps[0].Tree, "command"))
	}
	if plan.Steps[0].Tree[0].Operator != "&&" {
		t.Errorf("Step 0, cmd 0: expected && operator, got %q", plan.Steps[0].Tree[0].Operator)
	}

	if plan.Steps[0].Tree[1].Args[0].Val.Str != `echo "B"` {
		t.Errorf("Step 0, cmd 1: expected 'echo \"B\"', got %q", plan.Steps[0].Tree[1].Args[0].Val.Str)
	}
	if plan.Steps[0].Tree[1].Operator != "||" {
		t.Errorf("Step 0, cmd 1: expected || operator, got %q", plan.Steps[0].Tree[1].Operator)
	}

	if plan.Steps[0].Tree[2].Args[0].Val.Str != `echo "C"` {
		t.Errorf("Step 0, cmd 2: expected 'echo \"C\"', got %q", plan.Steps[0].Tree[2].Args[0].Val.Str)
	}
	if plan.Steps[0].Tree[2].Operator != "" {
		t.Errorf("Step 0, cmd 2: expected empty operator (last in step), got %q", plan.Steps[0].Tree[2].Operator)
	}

	// Step 2: ONE command (top-level echo "D")
	if plan.Steps[1].Tree == nil {
		t.Errorf("Expected step 1 to have 1 command, got %d", len(plan.Steps[1].Tree))
	}

	if getCommandArg(plan.Steps[1].Tree, "command") != `echo "D"` {
		t.Errorf("Step 1, cmd 0: expected 'echo \"D\"', got %q", getCommandArg(plan.Steps[1].Tree, "command"))
	}
	if plan.Steps[1].Tree[0].Operator != "" {
		t.Errorf("Step 1, cmd 0: expected empty operator, got %q", plan.Steps[1].Tree[0].Operator)
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
