package context

import (
	"github.com/aledsdavies/devcmd/core/ir"
	"io"
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
	UI *UIConfig `json:"ui,omitempty"` // UI behavior configuration

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

// Success returns true if the command executed successfully (exit code 0)
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.ExitCode != 0
}

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
