package decorator

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aledsdavies/opal/core/invariant"
)

// LocalSession implements Session for local command execution.
// Uses os/exec to run commands on the local machine.
type LocalSession struct {
	env map[string]string // Environment variables (copy-on-write)
	cwd string            // Current working directory
}

// NewLocalSession creates a new local session with the current environment.
func NewLocalSession() *LocalSession {
	return &LocalSession{
		env: envToMap(os.Environ()),
		cwd: mustGetwd(),
	}
}

// Run executes a command locally using os/exec.
func (s *LocalSession) Run(argv []string, opts RunOpts) (Result, error) {
	invariant.Precondition(len(argv) > 0, "argv cannot be empty")

	// Create command
	ctx := context.Background() // TODO: Accept context from caller
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	// Set working directory
	if s.cwd != "" {
		cmd.Dir = s.cwd
	}

	// Set environment (merge session env)
	cmd.Env = mapToEnv(s.env)

	// Wire up I/O
	if opts.Stdin != nil {
		cmd.Stdin = bytes.NewReader(opts.Stdin)
	}

	var stdout, stderr bytes.Buffer
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = &stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	// Execute
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1 // Generic failure (e.g., command not found)
		}
	}

	return Result{
		ExitCode: exitCode,
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
	}, nil
}

// Put writes data to a file on the local filesystem.
func (s *LocalSession) Put(data []byte, path string, mode fs.FileMode) error {
	invariant.Precondition(path != "", "path cannot be empty")

	// Resolve relative paths against cwd
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.cwd, path)
	}

	// Create parent directories
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write file
	return os.WriteFile(path, data, mode)
}

// Get reads data from a file on the local filesystem.
func (s *LocalSession) Get(path string) ([]byte, error) {
	invariant.Precondition(path != "", "path cannot be empty")

	// Resolve relative paths against cwd
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.cwd, path)
	}

	return os.ReadFile(path)
}

// Env returns an immutable snapshot of environment variables.
func (s *LocalSession) Env() map[string]string {
	// Return a copy to prevent mutation
	envCopy := make(map[string]string, len(s.env))
	for k, v := range s.env {
		envCopy[k] = v
	}
	return envCopy
}

// WithEnv returns a new Session with environment delta applied (copy-on-write).
func (s *LocalSession) WithEnv(delta map[string]string) Session {
	// Create new session with merged environment
	newEnv := make(map[string]string, len(s.env)+len(delta))
	for k, v := range s.env {
		newEnv[k] = v
	}
	for k, v := range delta {
		newEnv[k] = v
	}

	return &LocalSession{
		env: newEnv,
		cwd: s.cwd, // Inherit cwd
	}
}

// Cwd returns the current working directory.
func (s *LocalSession) Cwd() string {
	return s.cwd
}

// Close is a no-op for LocalSession (no resources to clean up).
func (s *LocalSession) Close() error {
	return nil
}

// Helper functions

// envToMap converts os.Environ() format to map.
func envToMap(environ []string) map[string]string {
	envMap := make(map[string]string, len(environ))
	for _, kv := range environ {
		if idx := strings.IndexByte(kv, '='); idx > 0 {
			envMap[kv[:idx]] = kv[idx+1:]
		}
	}
	return envMap
}

// mapToEnv converts map to os.Environ() format.
func mapToEnv(envMap map[string]string) []string {
	environ := make([]string, 0, len(envMap))
	for k, v := range envMap {
		environ = append(environ, k+"="+v)
	}
	return environ
}

// mustGetwd returns the current working directory or panics.
func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory: " + err.Error())
	}
	return cwd
}
