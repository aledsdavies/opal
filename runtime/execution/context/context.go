package context

import (
	"fmt"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
	"io"
	"os"
	"os/exec"
)

// ================================================================================================
// EXECUTION CONTEXT - Contains execution logic moved from core
// ================================================================================================

// Ctx carries state needed for command execution with execution methods
type Ctx struct {
	Env     ir.EnvSnapshot    `json:"env"`      // Frozen environment snapshot
	Vars    map[string]string `json:"vars"`     // CLI variables (@var) resolved to strings
	WorkDir string            `json:"work_dir"` // Current working directory

	// IO streams
	Stdout io.Writer `json:"-"` // Standard output
	Stderr io.Writer `json:"-"` // Standard error
	Stdin  io.Reader `json:"-"` // Standard input

	// System information
	NumCPU int `json:"num_cpu"` // Number of CPU cores available

	// Execution flags
	DryRun bool `json:"dry_run"` // Plan mode - don't actually execute
	Debug  bool `json:"debug"`   // Debug mode - verbose output

	// UI configuration from standardized flags
	UI       *UIConfig          `json:"ui,omitempty"`        // UI behavior configuration
	UIConfig *UIConfig          `json:"ui_config,omitempty"` // Alias for backward compatibility
	Commands map[string]ir.Node `json:"commands,omitempty"`  // Available commands for @cmd decorator

	// Execution delegate for action decorators
	Executor ExecutionDelegate `json:"-"` // Delegate for executing actions within decorators
}

// UIConfig contains standardized UI behavior flags
type UIConfig struct {
	ColorMode   string `json:"color_mode"`   // "auto", "always", "never"
	Quiet       bool   `json:"quiet"`        // minimal output (errors only)
	Verbose     bool   `json:"verbose"`      // extra debugging output
	Interactive string `json:"interactive"`  // "auto", "always", "never"
	AutoConfirm bool   `json:"auto_confirm"` // auto-confirm all prompts (--yes)
	CI          bool   `json:"ci"`           // CI mode (optimized for CI environments)
}

// ExecutionDelegate provides action execution capability for decorator contexts
type ExecutionDelegate interface {
	ExecuteAction(ctx *Ctx, name string, args []any) CommandResult
	ExecuteBlock(ctx *Ctx, name string, args []any, innerSteps []ir.CommandStep) CommandResult
	ExecuteCommand(ctx *Ctx, commandName string) CommandResult
}

// CommandResult represents the result of executing a command or action
type CommandResult struct {
	Stdout   string `json:"stdout"`    // Standard output as string
	Stderr   string `json:"stderr"`    // Standard error as string
	ExitCode int    `json:"exit_code"` // Exit code (0 = success)
}

// Implement decorators.CommandResult interface
func (r CommandResult) GetStdout() string { return r.Stdout }
func (r CommandResult) GetStderr() string { return r.Stderr }
func (r CommandResult) GetExitCode() int  { return r.ExitCode }
func (r CommandResult) IsSuccess() bool   { return r.ExitCode == 0 }

// Additional convenience methods
func (r CommandResult) Success() bool { return r.IsSuccess() }
func (r CommandResult) Failed() bool  { return !r.IsSuccess() }

// ================================================================================================
// INTERFACE COMPLIANCE CHECKS
// ================================================================================================

// Ensure Ctx implements ExecutionContext interface
var _ decorators.ExecutionContext = (*Ctx)(nil)

// Ensure CommandResult implements decorators.CommandResult interface
var _ decorators.CommandResult = (*CommandResult)(nil)

// ================================================================================================
// EXECUTION CONTEXT INTERFACE IMPLEMENTATION
// ================================================================================================

// ExecShell executes a shell command and returns the result
func (ctx *Ctx) ExecShell(command string) decorators.CommandResult {
	if ctx.DryRun {
		return &CommandResult{
			Stdout:   fmt.Sprintf("[DRY RUN] Would execute: %s", command),
			Stderr:   "",
			ExitCode: 0,
		}
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = ctx.WorkDir

	// Set up environment
	cmd.Env = os.Environ()

	var stdout, stderr []byte
	var err error

	stdout, err = cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr = exitError.Stderr
			return &CommandResult{
				Stdout:   string(stdout),
				Stderr:   string(stderr),
				ExitCode: exitError.ExitCode(),
			}
		}
		return &CommandResult{
			Stdout:   string(stdout),
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}

	return &CommandResult{
		Stdout:   string(stdout),
		Stderr:   "",
		ExitCode: 0,
	}
}

// GetEnv returns the value of an environment variable
func (ctx *Ctx) GetEnv(key string) string {
	if value, exists := ctx.Env.Get(key); exists {
		return value
	}
	return os.Getenv(key)
}

// SetEnv sets an environment variable (note: this only affects the context, not the actual process)
func (ctx *Ctx) SetEnv(key, value string) {
	if ctx.Env.Values == nil {
		ctx.Env.Values = make(map[string]string)
	}
	ctx.Env.Values[key] = value
}

// GetWorkingDir returns the current working directory
func (ctx *Ctx) GetWorkingDir() string {
	return ctx.WorkDir
}

// SetWorkingDir changes the working directory for subsequent operations
func (ctx *Ctx) SetWorkingDir(dir string) error {
	ctx.WorkDir = dir
	return nil
}

// Prompt asks the user for input (implementation depends on UI configuration)
func (ctx *Ctx) Prompt(message string) (string, error) {
	if ctx.UI != nil && ctx.UI.AutoConfirm {
		return "", fmt.Errorf("prompt not available in auto-confirm mode")
	}

	fmt.Fprintf(ctx.Stdout, "%s: ", message)
	// For now, return empty string - would need actual input handling
	return "", fmt.Errorf("interactive prompts not yet implemented")
}

// Confirm asks the user for yes/no confirmation
func (ctx *Ctx) Confirm(message string) (bool, error) {
	if ctx.UI != nil && ctx.UI.AutoConfirm {
		return true, nil
	}

	fmt.Fprintf(ctx.Stdout, "%s (y/N): ", message)
	// For now, return false - would need actual input handling
	return false, fmt.Errorf("interactive confirmation not yet implemented")
}

// Log outputs a log message at the specified level
func (ctx *Ctx) Log(level decorators.LogLevel, message string) {
	if ctx.UI != nil && ctx.UI.Quiet && level != decorators.LogLevelError {
		return
	}

	prefix := ""
	switch level {
	case decorators.LogLevelDebug:
		if ctx.Debug {
			prefix = "[DEBUG] "
		} else {
			return
		}
	case decorators.LogLevelInfo:
		prefix = "[INFO] "
	case decorators.LogLevelWarn:
		prefix = "[WARN] "
	case decorators.LogLevelError:
		prefix = "[ERROR] "
	}

	fmt.Fprintf(ctx.Stderr, "%s%s\n", prefix, message)
}

// Printf outputs a formatted message
func (ctx *Ctx) Printf(format string, args ...any) {
	if ctx.UI != nil && ctx.UI.Quiet {
		return
	}
	fmt.Fprintf(ctx.Stdout, format, args...)
}

// ================================================================================================
// COMMAND RESULT INTERFACE IMPLEMENTATION - Already implemented above
// ================================================================================================

// ================================================================================================
// CONTEXT METHODS
// ================================================================================================

// Clone creates a copy of the context for isolated execution
func (ctx *Ctx) Clone() *Ctx {
	newVars := make(map[string]string, len(ctx.Vars))
	for k, v := range ctx.Vars {
		newVars[k] = v
	}

	// Deep copy UIConfig if it exists
	var ui *UIConfig
	if ctx.UI != nil {
		ui = &UIConfig{
			ColorMode:   ctx.UI.ColorMode,
			Quiet:       ctx.UI.Quiet,
			Verbose:     ctx.UI.Verbose,
			Interactive: ctx.UI.Interactive,
			AutoConfirm: ctx.UI.AutoConfirm,
			CI:          ctx.UI.CI,
		}
	}

	return &Ctx{
		Env:      ctx.Env, // EnvSnapshot is immutable, safe to share
		Vars:     newVars,
		WorkDir:  ctx.WorkDir,
		NumCPU:   ctx.NumCPU,
		Stdout:   ctx.Stdout,
		Stderr:   ctx.Stderr,
		Stdin:    ctx.Stdin,
		DryRun:   ctx.DryRun,
		Debug:    ctx.Debug,
		UI:       ui,
		Executor: ctx.Executor, // Share the execution delegate
	}
}

// WithWorkDir returns a new context with updated working directory
// This is the correct pattern - never use os.Chdir()
func (ctx *Ctx) WithWorkDir(workDir string) *Ctx {
	newCtx := ctx.Clone()
	newCtx.WorkDir = workDir
	return newCtx
}

// ================================================================================================
// CONTEXT CREATION HELPERS
// ================================================================================================

// CtxOptions contains options for creating a new execution context
type CtxOptions struct {
	WorkDir    string
	EnvOptions EnvOptions
	NumCPU     int
	DryRun     bool
	Debug      bool
	UIConfig   *UIConfig
	Executor   ExecutionDelegate
	Vars       map[string]string  // CLI variables
	Commands   map[string]ir.Node // Available commands for @cmd decorator
}

// EnvOptions contains environment configuration
type EnvOptions struct {
	BlockList []string // Environment variables to exclude
}

// NewCtx creates a new execution context with the given options
func NewCtx(opts CtxOptions) (*Ctx, error) {
	// Create environment snapshot
	env := ir.EnvSnapshot{} // This would be implemented in core/ir if needed

	// Initialize vars map
	vars := opts.Vars
	if vars == nil {
		vars = make(map[string]string)
	}

	return &Ctx{
		Env:      env,
		Vars:     vars,
		WorkDir:  opts.WorkDir,
		NumCPU:   opts.NumCPU,
		DryRun:   opts.DryRun,
		Debug:    opts.Debug,
		UI:       opts.UIConfig,
		Executor: opts.Executor,
	}, nil
}
