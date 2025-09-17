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
	Describe(args []DecoratorParam) plan.ExecutionStep
}

// ActionDecorator - Standalone action decorators
type ActionDecorator interface {
	DecoratorBase
	// Plan generation - shows what action will be executed
	Describe(args []DecoratorParam) plan.ExecutionStep
}

// BlockDecorator - Execution wrapper decorators
type BlockDecorator interface {
	DecoratorBase
	// Plan generation - shows how inner commands will be wrapped
	Describe(args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}

// PatternDecorator - Conditional execution decorators
type PatternDecorator interface {
	DecoratorBase
	// Plan generation - shows which branch will be selected
	Describe(args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}
