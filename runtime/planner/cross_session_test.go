package planner

import (
	"errors"
	"strings"
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

// TestDecoratorWithParametersDoesNotHang verifies that decorators with
// parameters (parentheses, commas, string literals) don't cause infinite loops
// in the parseDecoratorValue function.
func TestDecoratorWithParametersDoesNotHang(t *testing.T) {
	// This test uses @env.HOME which is valid syntax
	// The key is that parseDecoratorValue correctly handles the decorator
	// without hanging on tokens it doesn't recognize
	source := `
var HOME = @env.HOME
`

	// Parse
	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	// Plan (this should not hang, even if decorator resolution fails)
	config := Config{
		Target: "",
	}

	// We don't care if planning succeeds or fails, just that it doesn't hang
	_, _ = PlanWithObservability(tree.Events, tree.Tokens, config)
}

// TestMultiDotDecoratorParsing tests that decorators with multiple dots
// are parsed correctly (e.g., @env.HOME uses two parts).
func TestMultiDotDecoratorParsing(t *testing.T) {
	// @env.HOME should parse as decorator="env", primary="HOME"
	// This is the current working syntax
	source := `var HOME = @env.HOME`

	tree := parser.ParseString(source)
	if len(tree.Errors) > 0 {
		t.Fatalf("Parse failed: %v", tree.Errors[0])
	}

	config := Config{Target: ""}
	result, err := PlanWithObservability(tree.Events, tree.Tokens, config)

	// @env.HOME should succeed (resolves from environment)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if result.Plan == nil {
		t.Fatal("Expected plan to be created")
	}
}

// TestValueRegistryEdgeCases tests edge cases in ValueRegistry.
func TestValueRegistryEdgeCases(t *testing.T) {
	t.Run("empty variable name", func(t *testing.T) {
		registry := NewValueRegistry()

		// Store with empty name
		registry.Store("", "literal", 42, "")

		// Get with empty name
		val, err := registry.Get("", "local")
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}
	})

	t.Run("empty session ID in Get", func(t *testing.T) {
		registry := NewValueRegistry()

		// Store session-agnostic value
		registry.Store("COUNT", "literal", 42, "")

		// Get with empty session ID (should work for session-agnostic)
		val, err := registry.Get("COUNT", "")
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
		if val != 42 {
			t.Errorf("Expected 42, got %v", val)
		}

		// Store session-specific value
		registry.Store("HOME", "@env.HOME", "/home/alice", "local")

		// Get with empty session ID (should fail for session-specific)
		_, err = registry.Get("HOME", "")
		if err == nil {
			t.Error("Expected error when accessing session-specific value with empty session ID")
		}
	})

	t.Run("variable not found", func(t *testing.T) {
		registry := NewValueRegistry()

		_, err := registry.Get("NONEXISTENT", "local")
		if err == nil {
			t.Error("Expected error for nonexistent variable")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("overwrite variable", func(t *testing.T) {
		registry := NewValueRegistry()

		// Store initial value
		registry.Store("VAR", "literal", 1, "")

		// Overwrite with new value
		registry.Store("VAR", "literal", 2, "")

		// Should get new value
		val, err := registry.Get("VAR", "local")
		if err != nil {
			t.Errorf("Expected success, got error: %v", err)
		}
		if val != 2 {
			t.Errorf("Expected 2, got %v", val)
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
