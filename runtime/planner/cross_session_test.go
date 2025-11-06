package planner

import (
	"errors"
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

// TestValueRegistrySessionIsolation verifies that the ValueRegistry
// enforces session boundaries correctly.
func TestValueRegistrySessionIsolation(t *testing.T) {
	registry := NewValueRegistry()

	// Store a session-agnostic value (literal)
	registry.Store("COUNT", "literal", 42, "")

	// Store a session-sensitive value (from @env in local session)
	registry.Store("LOCAL_HOME", "@env.HOME", "/home/alice", "local")

	// Store another session-sensitive value (from @env in SSH session)
	registry.Store("REMOTE_HOME", "@env.HOME", "/home/bob", "ssh:server1")

	t.Run("session-agnostic values work in any session", func(t *testing.T) {
		// Literal can be accessed from local session
		val, err := registry.Get("COUNT", "local")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}

		// Literal can be accessed from SSH session
		val, err = registry.Get("COUNT", "ssh:server1")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})

	t.Run("session-sensitive values only work in same session", func(t *testing.T) {
		// LOCAL_HOME can be accessed from local session
		val, err := registry.Get("LOCAL_HOME", "local")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if val != "/home/alice" {
			t.Errorf("Expected /home/alice, got %v", val)
		}

		// REMOTE_HOME can be accessed from SSH session
		val, err = registry.Get("REMOTE_HOME", "ssh:server1")
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if val != "/home/bob" {
			t.Errorf("Expected /home/bob, got %v", val)
		}
	})

	t.Run("cross-session access returns error", func(t *testing.T) {
		// Try to access LOCAL_HOME from SSH session (SHOULD FAIL)
		_, err := registry.Get("LOCAL_HOME", "ssh:server1")
		if err == nil {
			t.Fatal("Expected CrossSessionLeakageError, got nil")
		}

		var leakageErr *CrossSessionLeakageError
		if !errors.As(err, &leakageErr) {
			t.Fatalf("Expected CrossSessionLeakageError, got %T: %v", err, err)
		}

		// Verify error details
		if leakageErr.VarName != "LOCAL_HOME" {
			t.Errorf("Expected VarName=LOCAL_HOME, got %s", leakageErr.VarName)
		}
		if leakageErr.SourceSession != "local" {
			t.Errorf("Expected SourceSession=local, got %s", leakageErr.SourceSession)
		}
		if leakageErr.TargetSession != "ssh:server1" {
			t.Errorf("Expected TargetSession=ssh:server1, got %s", leakageErr.TargetSession)
		}
	})

	t.Run("reverse cross-session access also fails", func(t *testing.T) {
		// Try to access REMOTE_HOME from local session (SHOULD FAIL)
		_, err := registry.Get("REMOTE_HOME", "local")
		if err == nil {
			t.Fatal("Expected CrossSessionLeakageError, got nil")
		}

		var leakageErr *CrossSessionLeakageError
		if !errors.As(err, &leakageErr) {
			t.Fatalf("Expected CrossSessionLeakageError, got %T: %v", err, err)
		}

		// Verify error details
		if leakageErr.VarName != "REMOTE_HOME" {
			t.Errorf("Expected VarName=REMOTE_HOME, got %s", leakageErr.VarName)
		}
		if leakageErr.SourceSession != "ssh:server1" {
			t.Errorf("Expected SourceSession=ssh:server1, got %s", leakageErr.SourceSession)
		}
		if leakageErr.TargetSession != "local" {
			t.Errorf("Expected TargetSession=local, got %s", leakageErr.TargetSession)
		}
	})
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
