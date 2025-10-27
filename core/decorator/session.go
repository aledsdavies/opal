package decorator

import (
	"io/fs"
)

// Session represents an execution context (local, SSH, Docker, K8s, etc.).
// All execution happens within a Session.
type Session interface {
	// Run executes a command with arguments
	Run(argv []string, opts RunOpts) (Result, error)

	// Put writes data to a file on the session's filesystem
	Put(data []byte, path string, mode fs.FileMode) error

	// Get reads data from a file on the session's filesystem
	Get(path string) ([]byte, error)

	// Env returns an immutable snapshot of environment variables
	Env() map[string]string

	// WithEnv returns a new Session with environment delta applied (copy-on-write)
	WithEnv(delta map[string]string) Session

	// Cwd returns the current working directory
	Cwd() string

	// Close cleans up the session
	Close() error
}

// RunOpts configures command execution.
type RunOpts struct {
	Stdin  []byte
	Stdout any // io.Writer or similar
	Stderr any // io.Writer or similar
}

// Result is the outcome of command execution.
type Result struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}
