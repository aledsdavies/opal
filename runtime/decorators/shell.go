package decorators

import (
	"context"
	"fmt"

	"github.com/aledsdavies/opal/core/decorator"
	"github.com/aledsdavies/opal/core/types"
)

// ShellDecorator implements the @shell decorator using the new decorator architecture.
// It executes shell commands via Session.Run() with bash -c wrapper.
type ShellDecorator struct{}

// Descriptor returns the decorator metadata.
func (d *ShellDecorator) Descriptor() decorator.Descriptor {
	return decorator.NewDescriptor("shell").
		Summary("Execute shell commands").
		Param("command", types.TypeString, "Shell command to execute", "echo hello", "npm run build", "kubectl get pods").
		Block(decorator.BlockForbidden).             // Leaf decorator - no blocks
		TransportScope(decorator.TransportScopeAny). // Works in any session
		Roles(decorator.RoleWrapper).                // Executes work
		Build()
}

// Wrap implements the Exec interface.
// @shell is a leaf decorator - it ignores the 'next' parameter and executes directly.
func (d *ShellDecorator) Wrap(next decorator.ExecNode, params map[string]any) decorator.ExecNode {
	return &shellNode{params: params}
}

// shellNode wraps shell command execution.
type shellNode struct {
	params map[string]any
}

// Execute implements the ExecNode interface.
// Executes the shell command via Session.Run() with bash -c wrapper.
func (n *shellNode) Execute(ctx decorator.ExecContext) (decorator.Result, error) {
	// Extract command from params
	command, ok := n.params["command"].(string)
	if !ok || command == "" {
		return decorator.Result{ExitCode: 127}, fmt.Errorf("@shell requires command parameter")
	}

	// Create context for execution
	execCtx := context.Background()

	// Apply deadline if set
	if !ctx.Deadline.IsZero() {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithDeadline(execCtx, ctx.Deadline)
		defer cancel()
	}

	// Execute command through session with bash -c wrapper
	argv := []string{"bash", "-c", command}

	// Configure I/O from ExecContext
	opts := decorator.RunOpts{
		Stdin:  ctx.Stdin,  // Piped input (nil if not piped)
		Stdout: ctx.Stdout, // Piped output (nil if not piped)
	}

	result, err := ctx.Session.Run(execCtx, argv, opts)

	return result, err
}

// Register @shell decorator with the global registry
func init() {
	if err := decorator.Register("shell", &ShellDecorator{}); err != nil {
		panic(fmt.Sprintf("failed to register @shell decorator: %v", err))
	}
}
