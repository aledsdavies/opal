package stdlib

import (
	"fmt"
	"strings"
	"sync"
)

// DecoratorType represents the type of decorator
type DecoratorType int

const (
	// FunctionDecorator appears inline within shell content and returns values
	FunctionDecorator DecoratorType = iota
	// BlockDecorator modifies execution behavior and requires explicit blocks
	BlockDecorator
)

// SemanticType represents the semantic category for syntax highlighting
type SemanticType int

const (
	SemDecorator SemanticType = iota // Generic decorator
	SemVariable                      // Variable-related decorators (@var, @env)
	SemFunction                      // Function-related decorators (@sh, @now)
)

// ArgumentType represents the expected type of decorator arguments
type ArgumentType int

const (
	StringArg ArgumentType = iota
	NumberArg
	DurationArg
	IdentifierArg
	BooleanArg
	ExpressionArg // Can be any expression including @var() references
)

// DecoratorSignature defines the expected signature for a decorator
type DecoratorSignature struct {
	Name        string
	Type        DecoratorType
	Semantic    SemanticType
	Description string
	Args        []ArgumentSpec
	RequiresBlock bool // Only for BlockDecorator - whether it requires explicit {}
}

// ArgumentSpec defines an argument specification
type ArgumentSpec struct {
	Name     string
	Type     ArgumentType
	Optional bool
	Default  string
}

// DecoratorRegistry holds all valid decorators
type DecoratorRegistry struct {
	mu         sync.RWMutex
	decorators map[string]*DecoratorSignature
}

// NewDecoratorRegistry creates a new registry with all standard decorators
func NewDecoratorRegistry() *DecoratorRegistry {
	registry := &DecoratorRegistry{
		decorators: make(map[string]*DecoratorSignature),
	}

	registry.registerStandardDecorators()
	return registry
}

// registerStandardDecorators registers all standard Devcmd decorators
func (r *DecoratorRegistry) registerStandardDecorators() {
	// Function Decorators - appear inline within shell content
	r.register(&DecoratorSignature{
		Name:        "var",
		Type:        FunctionDecorator,
		Semantic:    SemVariable,
		Description: "Variable substitution - replaces with variable value",
		Args: []ArgumentSpec{
			{Name: "name", Type: IdentifierArg, Optional: false},
		},
	})

	// Block Decorators - modify execution behavior and require explicit blocks

	r.register(&DecoratorSignature{
		Name:          "parallel",
		Type:          BlockDecorator,
		Semantic:      SemDecorator,
		Description:   "Executes commands in parallel",
		RequiresBlock: true,
		Args:          []ArgumentSpec{}, // No arguments
	})
}

// Register adds a new decorator to the registry (thread-safe)
func (r *DecoratorRegistry) Register(signature *DecoratorSignature) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decorators[signature.Name] = signature
}

// register adds a decorator to the registry (internal, not thread-safe)
func (r *DecoratorRegistry) register(signature *DecoratorSignature) {
	r.decorators[signature.Name] = signature
}

// Get returns the decorator signature for a given name
func (r *DecoratorRegistry) Get(name string) (*DecoratorSignature, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decorator, exists := r.decorators[name]
	return decorator, exists
}

// IsValidDecorator checks if a decorator name is valid
func (r *DecoratorRegistry) IsValidDecorator(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.decorators[name]
	return exists
}

// IsFunctionDecorator checks if a decorator is a function decorator
func (r *DecoratorRegistry) IsFunctionDecorator(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if decorator, exists := r.decorators[name]; exists {
		return decorator.Type == FunctionDecorator
	}
	return false
}

// IsBlockDecorator checks if a decorator is a block decorator
func (r *DecoratorRegistry) IsBlockDecorator(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if decorator, exists := r.decorators[name]; exists {
		return decorator.Type == BlockDecorator
	}
	return false
}

// GetSemanticType returns the semantic type for a decorator
func (r *DecoratorRegistry) GetSemanticType(name string) SemanticType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if decorator, exists := r.decorators[name]; exists {
		return decorator.Semantic
	}
	return SemDecorator // Default to generic decorator
}

// RequiresBlock checks if a decorator requires an explicit block
func (r *DecoratorRegistry) RequiresBlock(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if decorator, exists := r.decorators[name]; exists {
		return decorator.RequiresBlock
	}
	return false
}

// ValidateArguments validates decorator arguments against the signature
func (r *DecoratorRegistry) ValidateArguments(name string, args []string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	decorator, exists := r.decorators[name]
	if !exists {
		return fmt.Errorf("unknown decorator: @%s", name)
	}

	// Check argument count
	requiredArgs := 0
	for _, arg := range decorator.Args {
		if !arg.Optional {
			requiredArgs++
		}
	}

	if len(args) < requiredArgs {
		return fmt.Errorf("@%s requires at least %d arguments, got %d", name, requiredArgs, len(args))
	}

	if len(args) > len(decorator.Args) {
		return fmt.Errorf("@%s accepts at most %d arguments, got %d", name, len(decorator.Args), len(args))
	}

	// TODO: Add type validation for arguments
	return nil
}

// GetAllDecorators returns all registered decorators
func (r *DecoratorRegistry) GetAllDecorators() []*DecoratorSignature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	decorators := make([]*DecoratorSignature, 0, len(r.decorators))
	for _, decorator := range r.decorators {
		decorators = append(decorators, decorator)
	}
	return decorators
}

// GetFunctionDecorators returns all function decorators
func (r *DecoratorRegistry) GetFunctionDecorators() []*DecoratorSignature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var decorators []*DecoratorSignature
	for _, decorator := range r.decorators {
		if decorator.Type == FunctionDecorator {
			decorators = append(decorators, decorator)
		}
	}
	return decorators
}

// GetBlockDecorators returns all block decorators
func (r *DecoratorRegistry) GetBlockDecorators() []*DecoratorSignature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var decorators []*DecoratorSignature
	for _, decorator := range r.decorators {
		if decorator.Type == BlockDecorator {
			decorators = append(decorators, decorator)
		}
	}
	return decorators
}

// GetDecoratorsBySemanticType returns decorators filtered by semantic type
func (r *DecoratorRegistry) GetDecoratorsBySemanticType(semanticType SemanticType) []*DecoratorSignature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var decorators []*DecoratorSignature
	for _, decorator := range r.decorators {
		if decorator.Semantic == semanticType {
			decorators = append(decorators, decorator)
		}
	}
	return decorators
}

// GetUsageString returns a usage string for a decorator
func (s *DecoratorSignature) GetUsageString() string {
	var parts []string
	parts = append(parts, "@"+s.Name)

	if len(s.Args) > 0 {
		var argStrs []string
		for _, arg := range s.Args {
			argStr := arg.Name
			if arg.Optional {
				argStr = "[" + argStr + "]"
			}
			argStrs = append(argStrs, argStr)
		}
		parts = append(parts, "("+strings.Join(argStrs, ", ")+")")
	}

	if s.RequiresBlock {
		parts = append(parts, " { ... }")
	}

	return strings.Join(parts, "")
}

// GetDocumentationString returns a documentation string for a decorator
func (s *DecoratorSignature) GetDocumentationString() string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("**@%s** - %s\n", s.Name, s.Description))
	doc.WriteString(fmt.Sprintf("Type: %s\n", s.getTypeString()))
	doc.WriteString(fmt.Sprintf("Semantic: %s\n", s.getSemanticString()))
	doc.WriteString(fmt.Sprintf("Usage: `%s`\n", s.GetUsageString()))

	if len(s.Args) > 0 {
		doc.WriteString("\nArguments:\n")
		for _, arg := range s.Args {
			optional := ""
			if arg.Optional {
				optional = " (optional"
				if arg.Default != "" {
					optional += fmt.Sprintf(", default: %s", arg.Default)
				}
				optional += ")"
			}
			doc.WriteString(fmt.Sprintf("- `%s`: %s%s\n", arg.Name, arg.getTypeString(), optional))
		}
	}

	return doc.String()
}

// getTypeString returns a human-readable type string
func (s *DecoratorSignature) getTypeString() string {
	switch s.Type {
	case FunctionDecorator:
		return "function"
	case BlockDecorator:
		return "block"
	default:
		return "unknown"
	}
}

// getSemanticString returns a human-readable semantic string
func (s *DecoratorSignature) getSemanticString() string {
	switch s.Semantic {
	case SemVariable:
		return "variable"
	case SemFunction:
		return "function"
	case SemDecorator:
		return "decorator"
	default:
		return "unknown"
	}
}

// getTypeString returns a human-readable type string for arguments
func (a *ArgumentSpec) getTypeString() string {
	switch a.Type {
	case StringArg:
		return "string"
	case NumberArg:
		return "number"
	case DurationArg:
		return "duration"
	case IdentifierArg:
		return "identifier"
	case BooleanArg:
		return "boolean"
	case ExpressionArg:
		return "expression"
	default:
		return "unknown"
	}
}

// Global registry instance
var StandardDecorators = NewDecoratorRegistry()

// Public API functions

// RegisterDecorator adds a new decorator to the global registry
func RegisterDecorator(signature *DecoratorSignature) {
	StandardDecorators.Register(signature)
}

// IsValidDecorator checks if a decorator name is valid
func IsValidDecorator(name string) bool {
	return StandardDecorators.IsValidDecorator(name)
}

// IsFunctionDecorator checks if a decorator is a function decorator
func IsFunctionDecorator(name string) bool {
	return StandardDecorators.IsFunctionDecorator(name)
}

// IsBlockDecorator checks if a decorator is a block decorator
func IsBlockDecorator(name string) bool {
	return StandardDecorators.IsBlockDecorator(name)
}

// GetDecoratorSemanticType returns the semantic type for a decorator
func GetDecoratorSemanticType(name string) SemanticType {
	return StandardDecorators.GetSemanticType(name)
}

// RequiresExplicitBlock checks if a decorator must have explicit braces
func RequiresExplicitBlock(name string) bool {
	return StandardDecorators.RequiresBlock(name)
}

// GetDecorator returns the decorator signature for a given name
func GetDecorator(name string) (*DecoratorSignature, bool) {
	return StandardDecorators.Get(name)
}

// ValidateDecorator validates that a decorator is used correctly
func ValidateDecorator(name string, args []string, hasBlock bool) error {
	decorator, exists := StandardDecorators.Get(name)
	if !exists {
		return fmt.Errorf("unknown decorator: @%s", name)
	}

	// Validate arguments
	if err := StandardDecorators.ValidateArguments(name, args); err != nil {
		return err
	}

	// Validate block usage
	if decorator.RequiresBlock && !hasBlock {
		return fmt.Errorf("@%s requires explicit block syntax: @%s { ... }", name, name)
	}

	return nil
}

// GetAllDecorators returns all registered decorators
func GetAllDecorators() []*DecoratorSignature {
	return StandardDecorators.GetAllDecorators()
}

// GetFunctionDecorators returns all function decorators
func GetFunctionDecorators() []*DecoratorSignature {
	return StandardDecorators.GetFunctionDecorators()
}

// GetBlockDecorators returns all block decorators
func GetBlockDecorators() []*DecoratorSignature {
	return StandardDecorators.GetBlockDecorators()
}

// GetDecoratorsBySemanticType returns decorators filtered by semantic type
func GetDecoratorsBySemanticType(semanticType SemanticType) []*DecoratorSignature {
	return StandardDecorators.GetDecoratorsBySemanticType(semanticType)
}

// GetDecoratorDocumentation returns documentation for all decorators
func GetDecoratorDocumentation() string {
	var doc strings.Builder

	doc.WriteString("# Devcmd Standard Library Decorators\n\n")

	// Function decorators
	functionDecorators := GetFunctionDecorators()
	if len(functionDecorators) > 0 {
		doc.WriteString("## Function Decorators\n\n")
		doc.WriteString("Function decorators appear inline within shell content and return values.\n\n")
		for _, decorator := range functionDecorators {
			doc.WriteString(decorator.GetDocumentationString())
			doc.WriteString("\n")
		}
	}

	// Block decorators
	blockDecorators := GetBlockDecorators()
	if len(blockDecorators) > 0 {
		doc.WriteString("## Block Decorators\n\n")
		doc.WriteString("Block decorators modify execution behavior and require explicit block syntax.\n\n")
		for _, decorator := range blockDecorators {
			doc.WriteString(decorator.GetDocumentationString())
			doc.WriteString("\n")
		}
	}

	return doc.String()
}
