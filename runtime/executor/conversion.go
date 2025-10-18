package executor

import (
	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk"
)

// convertPlanStepsToSDK converts planfmt.Step slice to sdk.Step slice.
// This separates the binary format (planfmt) from the execution model (sdk).
func convertPlanStepsToSDK(planSteps []planfmt.Step) []sdk.Step {
	sdkSteps := make([]sdk.Step, len(planSteps))
	for i, planStep := range planSteps {
		sdkSteps[i] = convertPlanStepToSDK(planStep)
	}
	return sdkSteps
}

// convertPlanStepToSDK converts a single planfmt.Step to sdk.Step.
func convertPlanStepToSDK(planStep planfmt.Step) sdk.Step {
	return sdk.Step{
		ID:       planStep.ID,
		Commands: convertPlanCommandsToSDK(planStep.Commands),
	}
}

// convertPlanCommandsToSDK converts planfmt.Command slice to sdk.Command slice.
func convertPlanCommandsToSDK(planCmds []planfmt.Command) []sdk.Command {
	sdkCmds := make([]sdk.Command, len(planCmds))
	for i, planCmd := range planCmds {
		sdkCmds[i] = sdk.Command{
			Name:     planCmd.Decorator,
			Args:     convertPlanArgsToMap(planCmd.Args),
			Block:    convertPlanStepsToSDK(planCmd.Block), // Recursive
			Operator: planCmd.Operator,
		}
	}
	return sdkCmds
}

// convertPlanArgsToMap converts []planfmt.Arg to map[string]interface{}.
// This provides a cleaner interface for decorators to access arguments.
func convertPlanArgsToMap(planArgs []planfmt.Arg) map[string]interface{} {
	args := make(map[string]interface{})
	for _, arg := range planArgs {
		switch arg.Val.Kind {
		case planfmt.ValueString:
			args[arg.Key] = arg.Val.Str
		case planfmt.ValueInt:
			args[arg.Key] = arg.Val.Int
		case planfmt.ValueBool:
			args[arg.Key] = arg.Val.Bool
			// TODO: Handle other value types (float, duration, etc.) as needed
		}
	}
	return args
}

// convertSDKStepsToPlan converts sdk.Step slice back to planfmt.Step slice.
// This is needed for ExecuteBlock callback - decorators work with sdk.Step,
// but executor internally still uses planfmt.Step.
//
// TODO: Eventually executor should work with sdk.Step natively.
func convertSDKStepsToPlan(sdkSteps []sdk.Step) []planfmt.Step {
	planSteps := make([]planfmt.Step, len(sdkSteps))
	for i, sdkStep := range sdkSteps {
		planSteps[i] = convertSDKStepToPlan(sdkStep)
	}
	return planSteps
}

// convertSDKStepToPlan converts a single sdk.Step to planfmt.Step.
func convertSDKStepToPlan(sdkStep sdk.Step) planfmt.Step {
	return planfmt.Step{
		ID:       sdkStep.ID,
		Commands: convertSDKCommandsToPlan(sdkStep.Commands),
	}
}

// convertSDKCommandsToPlan converts sdk.Command slice to planfmt.Command slice.
func convertSDKCommandsToPlan(sdkCmds []sdk.Command) []planfmt.Command {
	planCmds := make([]planfmt.Command, len(sdkCmds))
	for i, sdkCmd := range sdkCmds {
		planCmds[i] = planfmt.Command{
			Decorator: sdkCmd.Name,
			Args:      convertMapToPlanArgs(sdkCmd.Args),
			Block:     convertSDKStepsToPlan(sdkCmd.Block), // Recursive
			Operator:  sdkCmd.Operator,
		}
	}
	return planCmds
}

// convertMapToPlanArgs converts map[string]interface{} to []planfmt.Arg.
func convertMapToPlanArgs(args map[string]interface{}) []planfmt.Arg {
	planArgs := make([]planfmt.Arg, 0, len(args))
	for key, val := range args {
		var planVal planfmt.Value
		switch v := val.(type) {
		case string:
			planVal = planfmt.Value{Kind: planfmt.ValueString, Str: v}
		case int64:
			planVal = planfmt.Value{Kind: planfmt.ValueInt, Int: v}
		case int:
			planVal = planfmt.Value{Kind: planfmt.ValueInt, Int: int64(v)}
		case bool:
			planVal = planfmt.Value{Kind: planfmt.ValueBool, Bool: v}
		// TODO: Handle other types as needed
		default:
			// Skip unknown types for now
			continue
		}
		planArgs = append(planArgs, planfmt.Arg{Key: key, Val: planVal})
	}
	return planArgs
}
