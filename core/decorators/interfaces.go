package decorators

import (
	"github.com/aledsdavies/devcmd/core/plan"
)

// ================================================================================================
// CORE DECORATOR INTERFACES - Structural definitions only, no execution logic
// ================================================================================================

// DecoratorBase provides common metadata for all decorators
type DecoratorBase interface {
	Name() string
	Description() string
	ParameterSchema() []ParameterSchema
	Examples() []Example
}

// ArgType represents parameter types independent of AST
type ArgType string

const (
	ArgTypeString     ArgType = "string"
	ArgTypeBool       ArgType = "bool"
	ArgTypeInt        ArgType = "int"
	ArgTypeFloat      ArgType = "float"
	ArgTypeDuration   ArgType = "duration"   // Duration strings like "30s", "5m", "1h"
	ArgTypeIdentifier ArgType = "identifier" // Variable/command identifiers
	ArgTypeList       ArgType = "list"
	ArgTypeMap        ArgType = "map"
	ArgTypeAny        ArgType = "any"
)

// ParameterSchema describes a decorator parameter
type ParameterSchema struct {
	Name        string  `json:"name"`        // Parameter name
	Type        ArgType `json:"type"`        // Parameter type (AST-independent)
	Required    bool    `json:"required"`    // Whether required
	Description string  `json:"description"` // Human-readable description
	Default     any     `json:"default"`     // Default value if not provided
}

// Example provides usage examples
type Example struct {
	Code        string `json:"code"`        // Example code
	Description string `json:"description"` // What it demonstrates
}

// ================================================================================================
// CORE DECORATOR INTERFACES - Plan generation only (execution in runtime)
// ================================================================================================

// ValueDecorator - Inline value substitution decorators
type ValueDecorator interface {
	DecoratorBase
	// Plan generation - shows how value will be resolved
	Describe(args []Param) plan.ExecutionStep
}

// ActionDecorator - Standalone action decorators
type ActionDecorator interface {
	DecoratorBase
	// Plan generation - shows what action will be executed
	Describe(args []Param) plan.ExecutionStep
	// Execution method - runtime implementations provide this
	Run(ctx ExecutionContext, args []Param) CommandResult
}

// BlockDecorator - Execution wrapper decorators
type BlockDecorator interface {
	DecoratorBase
	// Plan generation - shows how inner commands will be wrapped
	Describe(args []Param, inner plan.ExecutionStep) plan.ExecutionStep
}

// PatternDecorator - Conditional execution decorators
type PatternDecorator interface {
	DecoratorBase
	// Plan generation - shows which branch will be selected
	Describe(args []Param, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}

// ================================================================================================
// EXECUTION INTERFACES - Runtime contracts that decorators depend on
// ================================================================================================

// ExecutionContext provides the runtime environment for decorator execution
// Runtime implementations will provide concrete types that implement this interface
type ExecutionContext interface {
	// Shell execution
	ExecShell(command string) CommandResult

	// Environment access
	GetEnv(key string) string
	SetEnv(key, value string)
	GetWorkingDir() string
	SetWorkingDir(dir string) error

	// UI interaction
	Prompt(message string) (string, error)
	Confirm(message string) (bool, error)

	// Logging
	Log(level LogLevel, message string)
	Printf(format string, args ...any)
}

// CommandResult represents the result of executing a command
type CommandResult interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
	IsSuccess() bool
}

// LogLevel for logging operations
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// ImportRequirement describes external dependencies a decorator needs
type ImportRequirement interface {
	GetPackages() []string     // Go packages to import
	GetBinaries() []string     // External binaries required
	GetEnv() map[string]string // Environment variables required
}
