package planner

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/runtime/parser"
)

// Test helpers

func mustPlan(t *testing.T, input string, target string) *planfmt.Plan {
	t.Helper()
	tree := parser.Parse([]byte(input))
	if len(tree.Errors) > 0 {
		t.Fatalf("parse errors: %v", tree.Errors)
	}

	plan, err := Plan(tree.Events, tree.Tokens, Config{
		Target:    target,
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	})
	if err != nil {
		t.Fatalf("planning failed: %v", err)
	}

	return plan
}

func assertValidPlan(t *testing.T, plan *planfmt.Plan) {
	t.Helper()
	if err := plan.Validate(); err != nil {
		t.Errorf("plan validation failed: %v", err)
	}
}

// Basic planning tests

func TestSimpleShellCommand(t *testing.T) {
	plan := mustPlan(t, `echo "Hello, World!"`, "")

	if plan.Root == nil {
		t.Fatal("expected non-nil root")
	}

	if plan.Root.Op != "shell" {
		t.Errorf("expected op=shell, got %q", plan.Root.Op)
	}

	assertValidPlan(t, plan)
}

func TestMultipleShellCommands(t *testing.T) {
	plan := mustPlan(t, `echo "First"
echo "Second"
echo "Third"`, "")

	if plan.Root == nil {
		t.Fatal("expected non-nil root")
	}

	// Should have 3 shell command children
	if len(plan.Root.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(plan.Root.Children))
	}

	assertValidPlan(t, plan)
}

func TestEmptyInput(t *testing.T) {
	plan := mustPlan(t, "", "")

	if plan.Root != nil {
		t.Errorf("expected nil root for empty input, got %+v", plan.Root)
	}
}

// Function definition tests

func TestFunctionDefinition(t *testing.T) {
	plan := mustPlan(t, `fun hello = echo "Hello"`, "hello")

	if plan.Target != "hello" {
		t.Errorf("expected target=hello, got %q", plan.Target)
	}

	if plan.Root == nil {
		t.Fatal("expected non-nil root")
	}

	if plan.Root.Op != "shell" {
		t.Errorf("expected op=shell, got %q", plan.Root.Op)
	}

	assertValidPlan(t, plan)
}

func TestTargetSelection(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
	}{
		{
			name:   "first function",
			input:  "fun first = echo \"First\"\nfun second = echo \"Second\"",
			target: "first",
		},
		{
			name:   "second function",
			input:  "fun first = echo \"First\"\nfun second = echo \"Second\"",
			target: "second",
		},
		{
			name:   "middle function",
			input:  "fun a = echo \"A\"\nfun b = echo \"B\"\nfun c = echo \"C\"",
			target: "b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := mustPlan(t, tt.input, tt.target)

			if plan.Target != tt.target {
				t.Errorf("expected target=%q, got %q", tt.target, plan.Target)
			}

			if plan.Root == nil {
				t.Fatal("expected non-nil root")
			}

			assertValidPlan(t, plan)
		})
	}
}

func TestTargetNotFound(t *testing.T) {
	tree := parser.Parse([]byte(`fun hello = echo "Hello"`))

	_, err := Plan(tree.Events, tree.Tokens, Config{
		Target: "nonexistent",
		Debug:  DebugOff,
	})

	if err == nil {
		t.Fatal("expected error for nonexistent target")
	}
}

// Script mode vs command mode

func TestScriptMode(t *testing.T) {
	// Script mode: no target, execute all top-level commands
	plan := mustPlan(t, `fun helper = echo "Helper"
echo "Main"`, "")

	if plan.Target != "" {
		t.Errorf("expected empty target, got %q", plan.Target)
	}

	// Should only execute top-level commands, not function definitions
	if plan.Root == nil {
		t.Fatal("expected non-nil root")
	}

	if plan.Root.Op != "shell" {
		t.Errorf("expected op=shell, got %q", plan.Root.Op)
	}

	assertValidPlan(t, plan)
}

func TestCommandMode(t *testing.T) {
	// Command mode: target specific function
	plan := mustPlan(t, `fun helper = echo "Helper"
echo "Main"`, "helper")

	if plan.Target != "helper" {
		t.Errorf("expected target=helper, got %q", plan.Target)
	}

	assertValidPlan(t, plan)
}

// Plan invariants

func TestStepIDUniqueness(t *testing.T) {
	plan := mustPlan(t, `echo "First"
echo "Second"
echo "Third"`, "")

	// Collect all step IDs
	ids := make(map[uint64]bool)
	var collect func(*planfmt.Step)
	collect = func(step *planfmt.Step) {
		if step == nil {
			return
		}
		if ids[step.ID] {
			t.Errorf("duplicate step ID: %d", step.ID)
		}
		ids[step.ID] = true
		for _, child := range step.Children {
			collect(child)
		}
	}
	collect(plan.Root)

	// Verify we have the expected number of unique IDs
	expectedCount := countSteps(plan.Root)
	if len(ids) != expectedCount {
		t.Errorf("expected %d unique IDs, got %d", expectedCount, len(ids))
	}

	assertValidPlan(t, plan)
}

func TestArgsSorted(t *testing.T) {
	plan := mustPlan(t, `echo "Hello"`, "")

	if plan.Root == nil {
		t.Fatal("expected non-nil root")
	}

	// Verify args are sorted by key
	for i := 1; i < len(plan.Root.Args); i++ {
		if plan.Root.Args[i-1].Key >= plan.Root.Args[i].Key {
			t.Errorf("args not sorted: %q >= %q",
				plan.Root.Args[i-1].Key, plan.Root.Args[i].Key)
		}
	}

	assertValidPlan(t, plan)
}

// Determinism tests

func TestPlanDeterminism(t *testing.T) {
	input := `echo "Hello"
echo "World"`

	plan1 := mustPlan(t, input, "")
	plan2 := mustPlan(t, input, "")

	// Both plans should have same step count
	count1 := countSteps(plan1.Root)
	count2 := countSteps(plan2.Root)

	if count1 != count2 {
		t.Errorf("non-deterministic step count: %d vs %d", count1, count2)
	}

	// Both plans should validate
	assertValidPlan(t, plan1)
	assertValidPlan(t, plan2)
}

// Telemetry and debug tests

func TestTelemetryBasic(t *testing.T) {
	tree := parser.Parse([]byte(`echo "Hello"`))

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{
		Target:    "",
		Telemetry: TelemetryBasic,
		Debug:     DebugOff,
	})

	if err != nil {
		t.Fatalf("planning failed: %v", err)
	}

	if result.Telemetry == nil {
		t.Fatal("expected telemetry, got nil")
	}

	if result.Telemetry.EventCount == 0 {
		t.Error("expected non-zero event count")
	}

	// PlanTime is always collected (in result.PlanTime, not telemetry)
	if result.PlanTime == 0 {
		t.Error("expected non-zero plan time (always collected)")
	}
}

func TestPlanTimeAlwaysCollected(t *testing.T) {
	tree := parser.Parse([]byte(`echo "Hello"`))

	// Even with TelemetryOff, PlanTime should be collected
	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{
		Target:    "",
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	})

	if err != nil {
		t.Fatalf("planning failed: %v", err)
	}

	if result.PlanTime == 0 {
		t.Error("expected non-zero plan time (always collected)")
	}

	// But telemetry should be nil
	if result.Telemetry != nil {
		t.Error("expected nil telemetry for TelemetryOff")
	}
}

func TestDebugEvents(t *testing.T) {
	tree := parser.Parse([]byte(`echo "Hello"`))

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{
		Target:    "",
		Telemetry: TelemetryOff,
		Debug:     DebugPaths,
	})

	if err != nil {
		t.Fatalf("planning failed: %v", err)
	}

	if len(result.DebugEvents) == 0 {
		t.Error("expected debug events, got none")
	}

	// Should have at least enter_plan and exit_plan
	var hasEnter, hasExit bool
	for _, evt := range result.DebugEvents {
		if evt.Event == "enter_plan" {
			hasEnter = true
		}
		if evt.Event == "exit_plan" {
			hasExit = true
		}
	}

	if !hasEnter {
		t.Error("expected enter_plan debug event")
	}
	if !hasExit {
		t.Error("expected exit_plan debug event")
	}
}

func TestTelemetryOff(t *testing.T) {
	tree := parser.Parse([]byte(`echo "Hello"`))

	result, err := PlanWithObservability(tree.Events, tree.Tokens, Config{
		Target:    "",
		Telemetry: TelemetryOff,
		Debug:     DebugOff,
	})

	if err != nil {
		t.Fatalf("planning failed: %v", err)
	}

	if result.Telemetry != nil {
		t.Error("expected nil telemetry for TelemetryOff")
	}

	if len(result.DebugEvents) != 0 {
		t.Error("expected no debug events for DebugOff")
	}
}
