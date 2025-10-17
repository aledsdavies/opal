package decorators

import (
	"fmt"

	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk/executor"
	"github.com/aledsdavies/opal/core/types"
	execruntime "github.com/aledsdavies/opal/runtime/executor"
)

func init() {
	// Register the @shell decorator with schema
	schema := types.NewSchema("shell", types.KindExecution).
		Description("Execute shell commands").
		Param("command", types.TypeString).
		Description("Shell command to execute").
		Required().
		Done().
		Build()

	if err := types.Global().RegisterExecutionWithSchema(schema, shellHandlerAdapter); err != nil {
		panic(fmt.Sprintf("failed to register @shell decorator: %v", err))
	}
}

// shellHandlerAdapter adapts the old registry signature to the new ExecutionContext signature
// This is a temporary bridge until we update the registry to support the new signature
func shellHandlerAdapter(ctx types.Context, args types.Args) error {
	// For now, this is a placeholder
	// We'll implement the real handler once we update the registry
	panic("@shell decorator not yet implemented - registry needs ExecutionContext support")
}

// shellHandler implements the @shell decorator using ExecutionContext
// This is the real implementation that will be used once registry is updated
// CRITICAL: Uses context workdir and environ, NOT os globals
func shellHandler(ctx execruntime.ExecutionContext, block []planfmt.Step) (int, error) {
	// Get command string from context args
	cmdStr := ctx.ArgString("command")
	if cmdStr == "" {
		return 127, fmt.Errorf("@shell requires command argument")
	}

	// Create command using SDK (automatically routes through scrubber)
	cmd := executor.BashContext(ctx.Context(), cmdStr)

	// CRITICAL: Use context state, not os state
	// This ensures isolation for @parallel, @ssh, @docker, etc.
	cmd.SetDir(ctx.Workdir())

	// For environment, we need to convert map to slice
	// Use AppendEnv for now (adds to existing env)
	// TODO: Consider if we need full replacement instead
	cmd.AppendEnv(ctx.Environ())

	// Execute and return exit code
	return cmd.Run()
}
