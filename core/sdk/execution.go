// Package sdk provides the execution SDK for Opal decorators.
//
// # Transport Pattern for Remote Execution
//
// The Transport abstraction (in core/sdk/executor) enables remote execution
// while preserving Opal's security model. Decorators like @ssh.connect,
// @docker.exec, and @aws.ssm.connect use this pattern to redirect @shell
// commands to remote systems.
//
// ## How It Works
//
// Transport is an implementation detail of decorators, NOT a first-class
// ExecutionContext member. Decorators wrap ExecutionContext and intercept
// @shell calls to redirect them through custom transports.
//
// ## Example: SSH Decorator Pattern
//
//	// @ssh.connect decorator wraps execution context
//	func sshConnectHandler(ctx sdk.ExecutionContext, block []sdk.Step) (int, error) {
//	    host := ctx.ArgString("host")
//	    user := ctx.ArgString("user")
//	    key := ctx.ArgString("key")
//
//	    // Establish SSH connection
//	    transport, err := executor.NewSSHTransport(host, user, key)
//	    if err != nil {
//	        return 127, err
//	    }
//	    defer transport.Close()
//
//	    // Wrap context to use SSH transport
//	    sshCtx := &sshExecutionContext{
//	        parent:    ctx,
//	        transport: transport,
//	    }
//
//	    // Execute block with SSH context
//	    return sshCtx.ExecuteBlock(block)
//	}
//
//	// sshExecutionContext wraps ExecutionContext to use SSH transport
//	type sshExecutionContext struct {
//	    parent    sdk.ExecutionContext
//	    transport executor.Transport
//	}
//
//	// ExecuteBlock intercepts @shell calls and redirects to SSH
//	func (s *sshExecutionContext) ExecuteBlock(steps []sdk.Step) (int, error) {
//	    for _, step := range steps {
//	        // If step is @shell, use SSH transport
//	        if isShellCommand(step) {
//	            exitCode, err := s.executeShellViaSSH(step)
//	            if exitCode != 0 || err != nil {
//	                return exitCode, err
//	            }
//	        } else {
//	            // Other decorators delegate to parent
//	            exitCode, err := s.parent.ExecuteBlock([]sdk.Step{step})
//	            if exitCode != 0 || err != nil {
//	                return exitCode, err
//	            }
//	        }
//	    }
//	    return 0, nil
//	}
//
//	// Delegate all other methods to parent
//	func (s *sshExecutionContext) Context() context.Context { return s.parent.Context() }
//	func (s *sshExecutionContext) ArgString(k string) string { return s.parent.ArgString(k) }
//	// ... etc
//
// ## Security Guarantees
//
// - Transport receives io.Writer for stdout/stderr - scrubber sits above
// - Decorators can't bypass scrubber by using transport directly
// - All I/O flows through executor for automatic secret scrubbing
// - Connection security (SSH keys, Docker sockets, AWS credentials) managed by transport
//
// ## Future Transports
//
// - SSHTransport: Execute commands on remote servers via SSH
// - DockerTransport: Execute commands inside Docker containers
// - SSMTransport: Execute commands on EC2 instances via AWS SSM
// - KubernetesTransport: Execute commands in Kubernetes pods
//
// See core/sdk/executor/transport.go for the Transport interface.
package sdk

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"time"

	"github.com/aledsdavies/opal/core/sdk/executor"
)

// Step represents a unit of work to execute (runtime execution model).
// This is separate from planfmt.Step (binary serialization format).
//
// Knowledge domain: How to execute work
// NOT: How to serialize/deserialize plans
//
// Example:
//
//	Step with single command:
//	  Step{ID: 1, Tree: &CommandNode{Name: "shell", Args: {"command": "echo hi"}}}
//
//	Step with operators:
//	  Step{ID: 2, Tree: &AndNode{
//	    Left:  &CommandNode{Name: "shell", Args: {"command": "npm build"}},
//	    Right: &CommandNode{Name: "shell", Args: {"command": "docker build"}},
//	  }}
type Step struct {
	ID   uint64   // Unique identifier (from plan)
	Tree TreeNode // Execution tree (operator precedence)
}

// TreeNode represents a node in the execution tree.
// The tree structure captures operator precedence within a step.
//
// Operator precedence (high to low): | > && > || > ;
//
// Example: echo "a" | grep "a" && echo "b" || echo "c"
// Parsed as: ((echo "a" | grep "a") && echo "b") || echo "c"
type TreeNode interface {
	isTreeNode()
}

// CommandNode is a leaf node - represents a single decorator invocation.
type CommandNode struct {
	Name  string                 // Decorator name: "shell", "retry", "parallel"
	Args  map[string]interface{} // Decorator arguments (typed values)
	Block []Step                 // Nested steps (for decorators with blocks)
}

func (*CommandNode) isTreeNode() {}

// PipelineNode executes a chain of piped commands (cmd1 | cmd2 | cmd3).
// All commands run concurrently with stdout→stdin streaming.
type PipelineNode struct {
	Commands []CommandNode // All commands in the pipeline
}

func (*PipelineNode) isTreeNode() {}

// AndNode executes left, then right only if left succeeded (exit 0).
// Implements bash && operator semantics.
type AndNode struct {
	Left  TreeNode
	Right TreeNode
}

func (*AndNode) isTreeNode() {}

// OrNode executes left, then right only if left failed (exit != 0).
// Implements bash || operator semantics.
type OrNode struct {
	Left  TreeNode
	Right TreeNode
}

func (*OrNode) isTreeNode() {}

// SequenceNode executes all nodes sequentially (semicolon operator).
// Always executes all nodes regardless of exit codes.
// Returns exit code of last node.
type SequenceNode struct {
	Nodes []TreeNode
}

func (*SequenceNode) isTreeNode() {}

// RedirectMode is defined in executor package to avoid import cycles.
// Re-export it here for convenience.
type RedirectMode = executor.RedirectMode

const (
	RedirectOverwrite = executor.RedirectOverwrite // > (truncate file)
	RedirectAppend    = executor.RedirectAppend    // >> (append to file)
)

// SinkCaps describes what operations a sink supports.
type SinkCaps struct {
	Overwrite      bool // Supports > (truncate and write)
	Append         bool // Supports >> (append to existing)
	Atomic         bool // Writes are atomic (readers see old-or-new, never partial)
	ConcurrentSafe bool // Multiple writers can safely write concurrently
}

// SinkProvider is an optional interface that decorators can implement
// to support being used as redirect targets.
//
// When a decorator is used as a redirect target (echo "data" > @decorator),
// the planner calls AsSink() to get a Sink implementation.
//
// Examples:
//   - @shell("output.txt") → FsPathSink{Path: "output.txt"}
//   - @aws.s3.object("bucket", "key") → S3Sink{Bucket: "bucket", Key: "key"}
//   - @http.post("url") → HTTPSink{URL: "url"}
//
// Decorators declare redirect support in their schema with WithRedirect(),
// and implement this interface to provide the actual sink.
type SinkProvider interface {
	// AsSink returns a Sink implementation for this decorator.
	// Called when decorator is used as redirect target.
	// Context provides access to decorator arguments.
	AsSink(ctx ExecutionContext) Sink
}

// Sink represents a destination for redirected output.
// Sinks are opened using the current execution context's transport,
// so files open in the right place (local/SSH/Docker/etc).
//
// Examples:
//   - FsPathSink: File on local or remote filesystem
//   - S3Sink: S3 object (future)
//   - HTTPSink: HTTP endpoint (future)
type Sink interface {
	// Caps returns what operations this sink supports
	Caps() SinkCaps

	// Open opens the sink for writing using the current context's transport.
	// The returned WriteCloser MUST be closed by the caller.
	//
	// For FsPathSink, this calls transport.OpenFileWriter() which:
	//   - LocalTransport: opens local file
	//   - SSHTransport: opens remote file via SSH
	//   - DockerTransport: opens file inside container
	//
	// meta is reserved for future use (e.g., S3 metadata, HTTP headers)
	Open(ctx ExecutionContext, mode RedirectMode, meta map[string]any) (io.WriteCloser, error)

	// Identity returns (kind, identifier) for error messages and logging.
	// kind: "fs.file", "s3.object", "http.post", etc.
	// identifier: human-readable sink identifier (e.g., "output.txt", "s3://bucket/key")
	//
	// Used in error messages like:
	//   "Error: failed to open sink fs.file (output.txt): permission denied"
	Identity() (kind, identifier string)
}

// FsPathSink is a sink that writes to a filesystem path.
// The path is opened using the current context's transport, so:
//   - Local execution: opens local file
//   - SSH execution: opens remote file
//   - Docker execution: opens file inside container
type FsPathSink struct {
	Path string      // File path (may contain resolved variables)
	Perm fs.FileMode // File permissions (e.g., 0644)
}

// Caps returns filesystem sink capabilities.
// Supports both overwrite and append, atomic writes (via temp+rename for >),
// but NOT concurrent-safe (OS doesn't guarantee linearizable appends).
func (s FsPathSink) Caps() SinkCaps {
	return SinkCaps{
		Overwrite:      true,
		Append:         true,
		Atomic:         true,
		ConcurrentSafe: false,
	}
}

// Open opens the file for writing using the current context's transport.
// This ensures the file opens in the right place (local/remote/container).
func (s FsPathSink) Open(ctx ExecutionContext, mode RedirectMode, _ map[string]any) (io.WriteCloser, error) {
	// Get transport from context
	transport := ctx.Transport()

	// Type assert to executor.Transport
	// This is safe because all contexts must provide a valid transport
	transportImpl, ok := transport.(executor.Transport)
	if !ok {
		return nil, errors.New("transport does not implement executor.Transport")
	}

	// Open file using transport (works for local/SSH/Docker/etc)
	return transportImpl.OpenFileWriter(ctx.Context(), s.Path, mode, s.Perm)
}

// Identity returns ("fs.file", path) for error messages.
func (s FsPathSink) Identity() (kind, identifier string) {
	return "fs.file", s.Path
}

// RedirectNode redirects stdout from Source to Sink.
// The sink is opened using the current context's transport, so files
// open in the right place (local/SSH/Docker/etc).
//
// Precedence: | > redirect > && > || > ;
type RedirectNode struct {
	Source TreeNode // Command/pipeline producing output
	Sink   Sink     // Where output goes (FsPathSink, S3Sink, etc.)
	Mode   RedirectMode
}

func (*RedirectNode) isTreeNode() {}

// ExecutionContext provides execution environment for decorators.
// This is the interface decorators receive - it abstracts away the executor implementation.
//
// Design: Decorators depend on this interface (in core/sdk), runtime implements it.
// This avoids circular dependencies: core/sdk ← runtime/executor implements.
//
// Security: Decorators have NO direct I/O access. All output flows through
// the executor which automatically scrubs secrets.
type ExecutionContext interface {
	// ExecuteBlock executes nested steps within this context.
	// This enables recursive composition: @retry { @timeout { @shell {...} } }
	//
	// The executor calls back into itself to execute the block, allowing
	// decorators to wrap execution without knowing executor internals.
	ExecuteBlock(steps []Step) (exitCode int, err error)

	// Context returns the Go context for cancellation and deadlines.
	// Decorators should pass this to long-running operations.
	Context() context.Context

	// Argument accessors - typed access to decorator parameters
	// Returns zero value if argument doesn't exist or has wrong type
	ArgString(key string) string
	ArgInt(key string) int64
	ArgBool(key string) bool
	ArgDuration(key string) time.Duration

	// Args returns a snapshot of all arguments for logging/debugging.
	// Modifications to the returned map do NOT affect the context.
	Args() map[string]interface{}

	// Environment and working directory (immutable snapshots)
	// These are captured at context creation time to ensure isolation.
	// Changes to os.Getwd() or os.Setenv() do NOT affect this context.
	Environ() map[string]string
	Workdir() string

	// Context wrapping - returns NEW context with modifications
	// Original context is unchanged (immutable, copy-on-write)
	//
	// This enables decorators to modify execution environment:
	//   @aws.auth(profile="prod") {
	//       // This block runs with prod auth in environment
	//   }
	WithContext(ctx context.Context) ExecutionContext
	WithEnviron(env map[string]string) ExecutionContext
	WithWorkdir(dir string) ExecutionContext

	// Pipe I/O for pipe operator support
	// These are nil when not piped - decorator should use default behavior
	//
	// Stdin returns piped input (nil if not piped).
	// When nil, decorator should use its default stdin behavior.
	Stdin() io.Reader

	// StdoutPipe returns piped output (nil if not piped).
	// When nil, decorator should write to its default stdout (which goes through scrubber).
	// When non-nil, decorator MUST write to this pipe.
	StdoutPipe() io.Writer

	// Clone creates a new context for a child command.
	// Inherits: Go context, environment, workdir
	// Replaces: args, stdin, stdoutPipe
	//
	// This is how executor creates contexts for each command in the tree.
	// Stdin and stdoutPipe may be nil (not piped).
	Clone(args map[string]interface{}, stdin io.Reader, stdoutPipe io.Writer) ExecutionContext

	// Transport returns the transport for command execution and file operations.
	// This enables redirect sinks to open files in the right place:
	//   - Local execution: LocalTransport opens local files
	//   - SSH execution: SSHTransport opens remote files
	//   - Docker execution: DockerTransport opens files inside container
	//
	// Decorators like @ssh.connect wrap ExecutionContext and return their own transport.
	// This is how "echo 'hello' > file.txt" works correctly in all contexts.
	//
	// Note: This is defined in core/sdk/executor to avoid import cycles.
	// The actual type is executor.Transport, but we can't import that here.
	// Callers should type-assert to executor.Transport.
	Transport() interface{}
}

// ExecutionHandler is the function signature for execution decorators.
// Decorators receive:
// - ctx: Execution context with args, environment, and ExecuteBlock callback
// - block: Child steps to execute (empty slice for leaf decorators)
//
// Block is optional - many decorators don't need it:
// - Leaf decorators: @shell("echo hi"), @file.write(...) - block is empty
// - Control flow: @retry(3) {...}, @parallel {...} - block has steps
//
// Returns:
// - exitCode: 0 for success, non-zero for failure
// - err: Error if decorator itself failed (not the command it ran)
//
// Error precedence (normative):
// 1. err != nil → Failure (exit code informational)
// 2. err == nil + exitCode == 0 → Success
// 3. err == nil + exitCode != 0 → Failure
type ExecutionHandler func(ctx ExecutionContext, block []Step) (exitCode int, err error)

// ValueHandler is the function signature for value decorators.
// Value decorators return data with no side effects - used at PLAN TIME.
//
// Key distinction:
// - Value decorators: Resolve values during planning (@env.HOME, @aws.secret.db_password)
// - Execution decorators: Perform/modify tasks during execution (@shell, @retry, @parallel)
//
// Value decorators can be interpolated in strings:
//
//	echo "Home: @env.HOME"  ← resolved at plan time
//
// Execution decorators cannot:
//
//	echo "@shell('ls')"  ← stays literal, not executed
//
// Examples: @env.HOME, @var.name, @aws.secret.db_password, @git.commit_hash
//
// Returns:
// - value: The resolved value (string, int, bool, etc.)
// - err: Error if resolution failed
type ValueHandler func(ctx ExecutionContext) (value interface{}, err error)
