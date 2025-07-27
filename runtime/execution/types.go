package execution

import "fmt"

// ExecutionMode represents the different modes of execution
type ExecutionMode int

const (
	InterpreterMode ExecutionMode = iota // Run commands directly
	GeneratorMode                        // Generate Go code for compilation
	PlanMode                             // Generate execution plan for dry-run
)

// String returns a string representation of the execution mode
func (m ExecutionMode) String() string {
	switch m {
	case InterpreterMode:
		return "interpreter"
	case GeneratorMode:
		return "generator"
	case PlanMode:
		return "plan"
	default:
		return "unknown"
	}
}

// ExecutionResult represents the result of executing shell content in different modes
type ExecutionResult struct {
	// Mode is the execution mode that produced this result
	Mode ExecutionMode

	// Data contains the mode-specific result:
	// - InterpreterMode: nil (execution happens directly)
	// - GeneratorMode: string (Go code)
	// - PlanMode: plan.PlanElement (plan element)
	Data interface{}

	// Error contains any execution error
	Error error
}


// CommandResult represents the structured output from command execution
// Used by ActionDecorators to enable proper piping and chaining
type CommandResult struct {
	Stdout   string // Standard output as string
	Stderr   string // Standard error as string  
	ExitCode int    // Exit code (0 = success)
}

// Success returns true if the command executed successfully (exit code 0)
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.ExitCode != 0
}

// Error returns an error representation when the command failed
func (r CommandResult) Error() error {
	if r.Success() {
		return nil
	}
	if r.Stderr != "" {
		return fmt.Errorf("exit code %d: %s", r.ExitCode, r.Stderr)
	}
	return fmt.Errorf("exit code %d", r.ExitCode)
}
