package executor

import (
	"testing"

	"github.com/aledsdavies/opal/core/planfmt"
)

// TestConvertPlanStepToSDK_Basic verifies basic step conversion
func TestConvertPlanStepToSDK_Basic(t *testing.T) {
	planStep := planfmt.Step{
		ID: 42,
		Commands: []planfmt.Command{
			{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "echo hello"}},
				},
			},
		},
	}

	sdkStep := convertPlanStepToSDK(planStep)

	if sdkStep.ID != 42 {
		t.Errorf("expected ID 42, got %d", sdkStep.ID)
	}
	if len(sdkStep.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sdkStep.Commands))
	}
	if sdkStep.Commands[0].Name != "@shell" {
		t.Errorf("expected name '@shell', got %q", sdkStep.Commands[0].Name)
	}
	if sdkStep.Commands[0].Args["command"] != "echo hello" {
		t.Errorf("expected command arg 'echo hello', got %v", sdkStep.Commands[0].Args["command"])
	}
}

// TestConvertPlanStepToSDK_WithBlock verifies nested block conversion
func TestConvertPlanStepToSDK_WithBlock(t *testing.T) {
	planStep := planfmt.Step{
		ID: 1,
		Commands: []planfmt.Command{
			{
				Decorator: "@retry",
				Args: []planfmt.Arg{
					{Key: "times", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 3}},
				},
				Block: []planfmt.Step{
					{
						ID: 2,
						Commands: []planfmt.Command{
							{Decorator: "@shell", Args: []planfmt.Arg{}},
						},
					},
				},
			},
		},
	}

	sdkStep := convertPlanStepToSDK(planStep)

	if len(sdkStep.Commands[0].Block) != 1 {
		t.Fatalf("expected 1 block step, got %d", len(sdkStep.Commands[0].Block))
	}
	if sdkStep.Commands[0].Block[0].ID != 2 {
		t.Errorf("expected block step ID 2, got %d", sdkStep.Commands[0].Block[0].ID)
	}
}

// TestConvertPlanArgsToMap_AllTypes verifies all argument types convert correctly
func TestConvertPlanArgsToMap_AllTypes(t *testing.T) {
	planArgs := []planfmt.Arg{
		{Key: "str", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "hello"}},
		{Key: "num", Val: planfmt.Value{Kind: planfmt.ValueInt, Int: 42}},
		{Key: "flag", Val: planfmt.Value{Kind: planfmt.ValueBool, Bool: true}},
	}

	args := convertPlanArgsToMap(planArgs)

	if args["str"] != "hello" {
		t.Errorf("expected 'hello', got %v", args["str"])
	}
	if args["num"] != int64(42) {
		t.Errorf("expected 42, got %v", args["num"])
	}
	if args["flag"] != true {
		t.Errorf("expected true, got %v", args["flag"])
	}
}

// TestConvertSDKStepToPlan_RoundTrip verifies conversion round-trip
func TestConvertSDKStepToPlan_RoundTrip(t *testing.T) {
	original := planfmt.Step{
		ID: 100,
		Commands: []planfmt.Command{
			{
				Decorator: "@shell",
				Args: []planfmt.Arg{
					{Key: "command", Val: planfmt.Value{Kind: planfmt.ValueString, Str: "ls"}},
				},
				Operator: "&&",
			},
		},
	}

	// Convert to SDK and back
	sdkStep := convertPlanStepToSDK(original)
	converted := convertSDKStepToPlan(sdkStep)

	if converted.ID != original.ID {
		t.Errorf("ID mismatch: expected %d, got %d", original.ID, converted.ID)
	}
	if len(converted.Commands) != len(original.Commands) {
		t.Fatalf("command count mismatch: expected %d, got %d", len(original.Commands), len(converted.Commands))
	}
	if converted.Commands[0].Decorator != original.Commands[0].Decorator {
		t.Errorf("decorator mismatch: expected %q, got %q", original.Commands[0].Decorator, converted.Commands[0].Decorator)
	}
	if converted.Commands[0].Operator != original.Commands[0].Operator {
		t.Errorf("operator mismatch: expected %q, got %q", original.Commands[0].Operator, converted.Commands[0].Operator)
	}
}

// TestConvertPlanStepsToSDK_MultipleSteps verifies slice conversion
func TestConvertPlanStepsToSDK_MultipleSteps(t *testing.T) {
	planSteps := []planfmt.Step{
		{ID: 1, Commands: []planfmt.Command{{Decorator: "@shell"}}},
		{ID: 2, Commands: []planfmt.Command{{Decorator: "@shell"}}},
		{ID: 3, Commands: []planfmt.Command{{Decorator: "@shell"}}},
	}

	sdkSteps := convertPlanStepsToSDK(planSteps)

	if len(sdkSteps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(sdkSteps))
	}
	for i, step := range sdkSteps {
		expectedID := uint64(i + 1)
		if step.ID != expectedID {
			t.Errorf("step %d: expected ID %d, got %d", i, expectedID, step.ID)
		}
	}
}

// TestConvertPlanStepToSDK_MultipleCommands verifies operator chaining
func TestConvertPlanStepToSDK_MultipleCommands(t *testing.T) {
	planStep := planfmt.Step{
		ID: 1,
		Commands: []planfmt.Command{
			{Decorator: "@shell", Operator: "&&"},
			{Decorator: "@shell", Operator: "||"},
			{Decorator: "@shell", Operator: ""}, // Last has no operator
		},
	}

	sdkStep := convertPlanStepToSDK(planStep)

	if len(sdkStep.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(sdkStep.Commands))
	}
	if sdkStep.Commands[0].Operator != "&&" {
		t.Errorf("command 0: expected operator '&&', got %q", sdkStep.Commands[0].Operator)
	}
	if sdkStep.Commands[1].Operator != "||" {
		t.Errorf("command 1: expected operator '||', got %q", sdkStep.Commands[1].Operator)
	}
	if sdkStep.Commands[2].Operator != "" {
		t.Errorf("command 2: expected empty operator, got %q", sdkStep.Commands[2].Operator)
	}
}

// TestConvertMapToPlanArgs_TypeHandling verifies type conversion
func TestConvertMapToPlanArgs_TypeHandling(t *testing.T) {
	args := map[string]interface{}{
		"str":  "hello",
		"i64":  int64(42),
		"i":    int(99),
		"bool": true,
	}

	planArgs := convertMapToPlanArgs(args)

	// Note: map iteration order is random, so we need to search
	found := make(map[string]bool)
	for _, arg := range planArgs {
		found[arg.Key] = true
		switch arg.Key {
		case "str":
			if arg.Val.Kind != planfmt.ValueString || arg.Val.Str != "hello" {
				t.Errorf("str: wrong value")
			}
		case "i64":
			if arg.Val.Kind != planfmt.ValueInt || arg.Val.Int != 42 {
				t.Errorf("i64: wrong value")
			}
		case "i":
			if arg.Val.Kind != planfmt.ValueInt || arg.Val.Int != 99 {
				t.Errorf("i: wrong value")
			}
		case "bool":
			if arg.Val.Kind != planfmt.ValueBool || arg.Val.Bool != true {
				t.Errorf("bool: wrong value")
			}
		}
	}

	if !found["str"] || !found["i64"] || !found["i"] || !found["bool"] {
		t.Error("not all args were converted")
	}
}

// TestConvertPlanStepToSDK_EmptyBlock verifies empty blocks work
func TestConvertPlanStepToSDK_EmptyBlock(t *testing.T) {
	planStep := planfmt.Step{
		ID: 1,
		Commands: []planfmt.Command{
			{
				Decorator: "@shell",
				Block:     []planfmt.Step{}, // Empty block
			},
		},
	}

	sdkStep := convertPlanStepToSDK(planStep)

	if len(sdkStep.Commands[0].Block) != 0 {
		t.Errorf("expected empty block, got %d steps", len(sdkStep.Commands[0].Block))
	}
}
