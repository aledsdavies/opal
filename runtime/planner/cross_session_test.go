package planner

import (
	"testing"

	"github.com/aledsdavies/opal/runtime/parser"
)

// TestLiteralVariablesAreSessionAgnostic verifies that literal values
// are marked as session-agnostic and can be used in any session.
func TestLiteralVariablesAreSessionAgnostic(t *testing.T) {
	source := `
var COUNT = 3
var NAME = "test"
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	// Plan
	config := Config{
		Target: "",
	}

	result, err := PlanWithObservability(tree.Events, tree.Tokens, config)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	// Verify planner state (we need to access internal state for testing)
	// For now, just verify the plan was created successfully
	if result.Plan == nil {
		t.Fatal("Expected plan to be created")
	}

	// TODO: Once we expose planner state or add a way to query variable metadata,
	// verify that COUNT and NAME are marked as SessionSensitive: false
}

// TestCrossSessionLeakagePrevention is the MUST-PASS test for security.
// This test will be implemented once we add decorator resolution in Week 2.
// For now, it's a placeholder to document the requirement.
func TestCrossSessionLeakagePrevention(t *testing.T) {
	t.Skip("Decorator resolution not implemented yet (Week 2)")

	// This test will verify:
	// 1. var LOCAL_HOME = @env.HOME (resolved in local session)
	// 2. @ssh(host="server1") { echo @var.LOCAL_HOME } (used in SSH session)
	// 3. Planner should error with CrossSessionLeakageError

	// Example test structure:
	// source := `
	// var LOCAL_HOME = @env.HOME
	//
	// @ssh(host="server1") {
	//     echo @var.LOCAL_HOME
	// }
	// `
	//
	// _, err := Plan(events, tokens, config)
	// if err == nil {
	//     t.Fatal("Expected cross-session leakage error")
	// }
	//
	// var leakageErr *CrossSessionLeakageError
	// if !errors.As(err, &leakageErr) {
	//     t.Fatalf("Expected CrossSessionLeakageError, got %T", err)
	// }
}
