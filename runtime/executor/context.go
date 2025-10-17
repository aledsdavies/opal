package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aledsdavies/opal/core/invariant"
	"github.com/aledsdavies/opal/core/planfmt"
	"github.com/aledsdavies/opal/core/sdk"
)

// executionContext implements sdk.ExecutionContext
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
func newExecutionContext(cmd planfmt.Command, exec *executor, ctx context.Context) sdk.ExecutionContext {
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
// Converts sdk.Step to planfmt.Step internally for execution
func (e *executionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
	// TODO: Implement conversion and execution in Phase 3E
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
func (e *executionContext) WithContext(ctx context.Context) sdk.ExecutionContext {
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
func (e *executionContext) WithEnviron(env map[string]string) sdk.ExecutionContext {
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
//
// Accepts both absolute and relative paths:
// - Absolute paths (e.g., "/tmp") are used as-is
// - Relative paths (e.g., "subdir", "../other", "./foo") are resolved against current context workdir
//
// Examples:
//
//	ctx.WithWorkdir("/tmp")           // → /tmp
//	ctx.WithWorkdir("subdir")         // → /current/subdir
//	ctx.WithWorkdir("../other")       // → /other (if current is /current)
//	ctx.WithWorkdir("foo/bar")        // → /current/foo/bar
//
// Chaining works intuitively:
//
//	ctx.WithWorkdir("foo").WithWorkdir("bar")  // → /current/foo/bar
//	ctx.WithWorkdir("foo").WithWorkdir("..")   // → /current
func (e *executionContext) WithWorkdir(dir string) sdk.ExecutionContext {
	invariant.Precondition(dir != "", "working directory cannot be empty")

	var resolved string
	if filepath.IsAbs(dir) {
		// Absolute path - use as-is
		resolved = dir
	} else {
		// Relative path - resolve against current context workdir
		resolved = filepath.Join(e.workdir, dir)
	}

	// Clean the path (remove . and .. components, collapse multiple slashes)
	resolved = filepath.Clean(resolved)

	return &executionContext{
		executor: e.executor,
		command:  e.command,
		ctx:      e.ctx,
		environ:  e.environ,
		workdir:  resolved,
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
