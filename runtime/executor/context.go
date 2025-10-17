package executor

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/planfmt"
)

// ExecutionContext provides execution environment for decorators
// Follows the Execution Context Pattern from DECORATOR_GUIDE.md
type ExecutionContext interface {
	// Execute nested block (callback to executor)
	ExecuteBlock(steps []planfmt.Step) (exitCode int, err error)

	// Context for cancellation and deadlines
	Context() context.Context

	// Decorator arguments (typed accessors)
	ArgString(key string) string
	ArgInt(key string) int64
	ArgBool(key string) bool
	ArgDuration(key string) time.Duration
	Args() map[string]interface{} // Snapshot for logging

	// Environment and working directory (immutable snapshots)
	Environ() map[string]string
	Workdir() string

	// Context wrapping (returns new context with modifications)
	WithContext(ctx context.Context) ExecutionContext
	WithEnviron(env map[string]string) ExecutionContext
	WithWorkdir(dir string) ExecutionContext
}

// executionContext implements ExecutionContext
// All fields are immutable - modifications create new contexts
type executionContext struct {
	executor *executor
	command  planfmt.Command
	ctx      context.Context
	environ  map[string]string // Immutable snapshot
	workdir  string            // Immutable snapshot
}

// newExecutionContext creates a new execution context for a decorator
// Captures current environment and working directory as immutable snapshots
func newExecutionContext(cmd planfmt.Command, exec *executor, ctx context.Context) ExecutionContext {
	invariant.NotNil(ctx, "context")

	// Capture current working directory at context creation time
	// This ensures isolation - changes to os.Getwd() won't affect this context
	wd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}

	return &executionContext{
		executor: exec,
		command:  cmd,
		ctx:      ctx,
		environ:  captureEnviron(), // Immutable snapshot
		workdir:  wd,               // Immutable snapshot
	}
}

// ExecuteBlock executes nested steps (callback to executor)
func (e *executionContext) ExecuteBlock(steps []planfmt.Step) (int, error) {
	// TODO: Implement in Phase 3C when we update executor
	panic("ExecuteBlock not yet implemented")
}

// Context returns the Go context for cancellation and deadlines
func (e *executionContext) Context() context.Context {
	return e.ctx
}

// ArgString retrieves a string argument
func (e *executionContext) ArgString(key string) string {
	for _, arg := range e.command.Args {
		if arg.Key == key && arg.Val.Kind == planfmt.ValueString {
			return arg.Val.Str
		}
	}
	return ""
}

// ArgInt retrieves an integer argument
func (e *executionContext) ArgInt(key string) int64 {
	for _, arg := range e.command.Args {
		if arg.Key == key && arg.Val.Kind == planfmt.ValueInt {
			return arg.Val.Int
		}
	}
	return 0
}

// ArgBool retrieves a boolean argument
func (e *executionContext) ArgBool(key string) bool {
	for _, arg := range e.command.Args {
		if arg.Key == key && arg.Val.Kind == planfmt.ValueBool {
			return arg.Val.Bool
		}
	}
	return false
}

// ArgDuration retrieves a duration argument
// TODO: Implement when planfmt.Value supports Duration type
func (e *executionContext) ArgDuration(key string) time.Duration {
	// For now, durations are stored as strings and parsed
	// This will be updated when planfmt.Value adds Duration support
	_ = key
	return 0
}

// Args returns a snapshot of all arguments for logging
func (e *executionContext) Args() map[string]interface{} {
	args := make(map[string]interface{})
	for _, arg := range e.command.Args {
		switch arg.Val.Kind {
		case planfmt.ValueString:
			args[arg.Key] = arg.Val.Str
		case planfmt.ValueInt:
			args[arg.Key] = arg.Val.Int
		case planfmt.ValueBool:
			args[arg.Key] = arg.Val.Bool
		}
	}
	return args
}

// Environ returns the environment variables (immutable snapshot)
func (e *executionContext) Environ() map[string]string {
	return e.environ
}

// Workdir returns the working directory (immutable snapshot)
func (e *executionContext) Workdir() string {
	return e.workdir
}

// WithContext returns a new context with the specified Go context
// Original context is unchanged (immutable)
func (e *executionContext) WithContext(ctx context.Context) ExecutionContext {
	invariant.NotNil(ctx, "context")

	return &executionContext{
		executor: e.executor,
		command:  e.command,
		ctx:      ctx,
		environ:  e.environ, // Share immutable snapshot
		workdir:  e.workdir, // Share immutable snapshot
	}
}

// WithEnviron returns a new context with the specified environment
// Original context is unchanged (immutable)
func (e *executionContext) WithEnviron(env map[string]string) ExecutionContext {
	invariant.NotNil(env, "environment")

	// Deep copy to ensure immutability
	envCopy := make(map[string]string, len(env))
	for k, v := range env {
		envCopy[k] = v
	}

	return &executionContext{
		executor: e.executor,
		command:  e.command,
		ctx:      e.ctx,
		environ:  envCopy,
		workdir:  e.workdir,
	}
}

// WithWorkdir returns a new context with the specified working directory
// Original context is unchanged (immutable)
func (e *executionContext) WithWorkdir(dir string) ExecutionContext {
	invariant.Precondition(dir != "", "working directory cannot be empty")

	return &executionContext{
		executor: e.executor,
		command:  e.command,
		ctx:      e.ctx,
		environ:  e.environ,
		workdir:  dir,
	}
}

// captureEnviron captures current environment as immutable snapshot
// Returns a new map that won't be affected by future os.Setenv() calls
func captureEnviron() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		// Split on first '=' only
		if idx := strings.IndexByte(e, '='); idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env
}
