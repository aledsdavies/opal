package decorator

import (
	"context"
	"io"
)

// Exec is the interface for decorators that wrap execution.
// Exec decorators use the middleware pattern to compose behavior.
// Examples: @retry, @timeout, @parallel
type Exec interface {
	Decorator
	Wrap(next ExecNode, params map[string]any) ExecNode
}

// ExecNode represents an executable node in the execution tree.
type ExecNode interface {
	Execute(ctx ExecContext) (Result, error)
}

// ExecContext provides the execution context for command execution.
type ExecContext struct {
	// Context is the parent context for cancellation and deadlines
	// Decorators should pass this to Session.Run() and other operations
	Context context.Context

	// Session is the ambient execution context (env, cwd, transport)
	Session Session

	// Stdin provides input data for piped commands (nil if not piped)
	// Used for pipe operators: cmd1 | cmd2
	Stdin []byte

	// Stdout captures output for piped commands (nil if not piped)
	// Used for pipe operators: cmd1 | cmd2
	Stdout io.Writer

	// Trace is the telemetry span for observability
	// Opal runtime creates parent span automatically
	// Decorators can create child spans for internal tracking
	Trace Span
}
