package execution

import (
	"fmt"
	"strconv"

	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/execution/context"
)

// ================================================================================================
// RUNTIME EXECUTION INTERFACES - Extend core interfaces with execution behavior
// ================================================================================================

// ValueDecorator extends core.ValueDecorator with execution capability
type ValueDecorator interface {
	Name() string
	// Runtime execution - resolve value in execution context
	Render(ctx *context.Ctx, args []DecoratorParam) (string, error)
	// Plan generation - show how value will be resolved
	Describe(ctx *context.Ctx, args []DecoratorParam) plan.ExecutionStep
}

// ActionDecorator extends core.ActionDecorator with execution capability
type ActionDecorator interface {
	Name() string
	// Runtime execution - execute action and return result
	Run(ctx *context.Ctx, args []DecoratorParam) context.CommandResult
	// Plan generation - show what action will be executed
	Describe(ctx *context.Ctx, args []DecoratorParam) plan.ExecutionStep
}

// BlockDecorator extends core.BlockDecorator with execution capability
type BlockDecorator interface {
	Name() string
	// Runtime execution - wrap and execute inner commands
	WrapCommands(ctx *context.Ctx, args []DecoratorParam, commands ir.CommandSeq) context.CommandResult
	// Plan generation - show how inner commands will be wrapped
	Describe(ctx *context.Ctx, args []DecoratorParam, inner plan.ExecutionStep) plan.ExecutionStep
}

// PatternDecorator extends core.PatternDecorator with execution capability
type PatternDecorator interface {
	Name() string
	// Runtime execution - select and execute branch
	SelectBranch(ctx *context.Ctx, args []DecoratorParam, branches map[string]ir.CommandSeq) context.CommandResult
	// Plan generation - show which branch will be selected
	Describe(ctx *context.Ctx, args []DecoratorParam, branches map[string]plan.ExecutionStep) plan.ExecutionStep
}

// DecoratorParam represents a parameter passed to a decorator at runtime
type DecoratorParam struct {
	Name  string `json:"name"`  // Parameter name (empty for positional)
	Value any    `json:"value"` // Parameter value (AST-independent)
}

// Ensure DecoratorParam implements core interface
var _ decorators.DecoratorParam = (*DecoratorParam)(nil)

// GetName returns the parameter name (empty for positional parameters)
func (p DecoratorParam) GetName() string {
	return p.Name
}

// GetValue returns the raw parameter value
func (p DecoratorParam) GetValue() any {
	return p.Value
}

// AsString converts the parameter value to a string
func (p DecoratorParam) AsString() string {
	if str, ok := p.Value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", p.Value)
}

// AsBool converts the parameter value to a boolean
func (p DecoratorParam) AsBool() bool {
	switch v := p.Value.(type) {
	case bool:
		return v
	case string:
		val, _ := strconv.ParseBool(v)
		return val
	}
	return false
}

// AsInt converts the parameter value to an integer
func (p DecoratorParam) AsInt() int {
	switch v := p.Value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		val, _ := strconv.Atoi(v)
		return val
	}
	return 0
}

// AsFloat converts the parameter value to a float64
func (p DecoratorParam) AsFloat() float64 {
	switch v := p.Value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		val, _ := strconv.ParseFloat(v, 64)
		return val
	}
	return 0.0
}

// ImportRequirement describes external dependencies a decorator needs
type ImportRequirement struct {
	Packages []string          `json:"packages"` // Go packages to import
	Binaries []string          `json:"binaries"` // External binaries required
	Env      map[string]string `json:"env"`      // Environment variables required
}

// ================================================================================================
// TYPE ALIASES - Backward compatibility during migration
// ================================================================================================

// Type aliases for easier migration from old decorators package
type (
	// Core IR types (already in right place)
	CommandSeq   = ir.CommandSeq
	CommandStep  = ir.CommandStep
	ChainElement = ir.ChainElement
	ElementKind  = ir.ElementKind
	ChainOp      = ir.ChainOp

	// Execution types (from context package)
	Ctx           = context.Ctx
	CommandResult = context.CommandResult
	UIConfig      = context.UIConfig

	// Support for pattern decorators
	PatternSchema = map[string]any

	// Support for parallel decorators
	ParallelMode = string
)

// Parallel mode constants
const (
	ParallelModeFailFast      ParallelMode = "fail-fast"
	ParallelModeFailImmediate ParallelMode = "fail-immediate"
	ParallelModeAll           ParallelMode = "all"
)

// Element kind constants (from core/ir)
const (
	ElementKindShell   = ir.ElementKindShell
	ElementKindAction  = ir.ElementKindAction
	ElementKindBlock   = ir.ElementKindBlock
	ElementKindPattern = ir.ElementKindPattern
)

// Chain operation constants (from core/ir)
const (
	ChainOpAnd = ir.ChainOpAnd
	ChainOpOr  = ir.ChainOpOr
)
