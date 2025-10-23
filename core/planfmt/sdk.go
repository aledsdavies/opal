package planfmt

import (
	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/sdk"
)

// ToSDKSteps converts planfmt.Step slice to sdk.Step slice.
// This is the boundary between binary format (planfmt) and execution model (sdk).
//
// The executor only sees SDK types - it has no knowledge of planfmt.
// All value interpolation is already done by the planner, so steps contain
// actual values (not placeholders), except for value decorator results which
// use DisplayID placeholders that get scrubbed during output.
func ToSDKSteps(planSteps []Step) []sdk.Step {
	sdkSteps := make([]sdk.Step, len(planSteps))
	for i, planStep := range planSteps {
		sdkSteps[i] = ToSDKStep(planStep)
	}
	return sdkSteps
}

// ToSDKStep converts a single planfmt.Step to sdk.Step.
func ToSDKStep(planStep Step) sdk.Step {
	return sdk.Step{
		ID:   planStep.ID,
		Tree: toSDKTree(planStep.Tree),
	}
}

// toSDKTree converts planfmt.ExecutionNode to sdk.TreeNode.
// This recursively converts the entire tree structure.
func toSDKTree(node ExecutionNode) sdk.TreeNode {
	switch n := node.(type) {
	case *CommandNode:
		return &sdk.CommandNode{
			Name:  n.Decorator,
			Args:  ToSDKArgs(n.Args),
			Block: ToSDKSteps(n.Block), // Recursive for nested steps
		}
	case *PipelineNode:
		commands := make([]sdk.CommandNode, len(n.Commands))
		for i, cmd := range n.Commands {
			commands[i] = sdk.CommandNode{
				Name:  cmd.Decorator,
				Args:  ToSDKArgs(cmd.Args),
				Block: ToSDKSteps(cmd.Block),
			}
		}
		return &sdk.PipelineNode{Commands: commands}
	case *AndNode:
		return &sdk.AndNode{
			Left:  toSDKTree(n.Left),
			Right: toSDKTree(n.Right),
		}
	case *OrNode:
		return &sdk.OrNode{
			Left:  toSDKTree(n.Left),
			Right: toSDKTree(n.Right),
		}
	case *SequenceNode:
		nodes := make([]sdk.TreeNode, len(n.Nodes))
		for i, child := range n.Nodes {
			nodes[i] = toSDKTree(child)
		}
		return &sdk.SequenceNode{Nodes: nodes}
	case *RedirectNode:
		// Convert Target CommandNode to Sink
		// For now, we only support @shell("path") which becomes FsPathSink
		// Future: support decorator sinks like @s3.object(), @http.post(), etc.
		var sink sdk.Sink

		invariant.Precondition(n.Target.Decorator == "@shell",
			"redirect target must be @shell for now (decorator sinks not yet implemented)")

		// Extract path from @shell("path") args
		path := ""
		for _, arg := range n.Target.Args {
			if arg.Key == "command" && arg.Val.Kind == ValueString {
				path = arg.Val.Str
				break
			}
		}
		invariant.Precondition(path != "", "redirect target @shell missing 'command' argument")

		// Create FsPathSink with default permissions (0644)
		sink = sdk.FsPathSink{
			Path: path,
			Perm: 0644,
		}

		return &sdk.RedirectNode{
			Source: toSDKTree(n.Source),
			Sink:   sink,
			Mode:   sdk.RedirectMode(n.Mode), // Convert planfmt.RedirectMode to sdk.RedirectMode
		}
	default:
		panic("unknown ExecutionNode type")
	}
}

// ToSDKArgs converts []planfmt.Arg to map[string]interface{}.
// This provides a cleaner interface for decorators to access arguments.
func ToSDKArgs(planArgs []Arg) map[string]interface{} {
	args := make(map[string]interface{})
	for _, arg := range planArgs {
		switch arg.Val.Kind {
		case ValueString:
			args[arg.Key] = arg.Val.Str
		case ValueInt:
			args[arg.Key] = arg.Val.Int
		case ValueBool:
			args[arg.Key] = arg.Val.Bool
			// TODO: Handle other value types (float, duration, etc.) as needed
		}
	}
	return args
}
