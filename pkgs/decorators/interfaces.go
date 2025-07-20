package decorators

import (
	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/plan"
	"github.com/aledsdavies/devcmd/pkgs/types"
)

// ParameterSchema describes a decorator parameter
type ParameterSchema struct {
	Name        string               // Parameter name (e.g., "key", "default")
	Type        types.ExpressionType // Parameter type (StringType, NumberType, etc.)
	Required    bool                 // Whether this parameter is required
	Description string               // Human-readable description
}

// ImportRequirement describes dependencies needed for code generation
type ImportRequirement struct {
	StandardLibrary []string          // Standard library imports (e.g., "time", "context", "sync")
	ThirdParty      []string          // Third-party imports (e.g., "github.com/pkg/errors")
	GoModules       map[string]string // Module dependencies for go.mod (module -> version)
}

// Decorator is a union interface for all decorator types
// Used for registry and common operations
type Decorator interface {
	Name() string
	Description() string
	Validate(ctx *ExecutionContext, params []ast.NamedParameter) error
	ParameterSchema() []ParameterSchema
	
	// ImportRequirements returns the dependencies needed for code generation
	ImportRequirements() ImportRequirement
}

// FunctionDecorator represents decorators that transform input arguments to output strings
// Examples: @env, @cd, @time, @var
type FunctionDecorator interface {
	Decorator

	// Run executes the decorator at runtime and returns the transformed value
	Run(ctx *ExecutionContext, params []ast.NamedParameter) (string, error)

	// Generate produces Go code for the decorator in compiled mode
	Generate(ctx *ExecutionContext, params []ast.NamedParameter) (string, error)

	// Plan creates a plan element describing what this decorator would do in dry run mode  
	Plan(ctx *ExecutionContext, params []ast.NamedParameter) (plan.PlanElement, error)
}

// BlockDecorator represents decorators that modify command execution behavior
// Examples: @watch, @stop, @parallel
type BlockDecorator interface {
	Decorator

	// Run executes the decorator at runtime with the given command content
	Run(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) error

	// Generate produces Go code for the decorator in compiled mode
	Generate(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (string, error)

	// Plan creates a plan element describing what this decorator would do in dry run mode with the given content
	Plan(ctx *ExecutionContext, params []ast.NamedParameter, content []ast.CommandContent) (plan.PlanElement, error)
}

// PatternDecorator represents decorators that handle pattern matching
// Examples: @when, @try
type PatternDecorator interface {
	Decorator

	// Run executes the decorator at runtime with pattern branches
	Run(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) error

	// Generate produces Go code for the decorator in compiled mode
	Generate(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (string, error)

	// Plan creates a plan element describing what this decorator would do in dry run mode with the given patterns
	Plan(ctx *ExecutionContext, params []ast.NamedParameter, patterns []ast.PatternBranch) (plan.PlanElement, error)
}

// DecoratorType represents the type of decorator
type DecoratorType int

const (
	FunctionType DecoratorType = iota
	BlockType
	PatternType
)

// GetDecoratorType returns the type of a decorator
func GetDecoratorType(d Decorator) DecoratorType {
	switch d.(type) {
	case FunctionDecorator:
		return FunctionType
	case BlockDecorator:
		return BlockType
	case PatternDecorator:
		return PatternType
	default:
		panic("unknown decorator type")
	}
}