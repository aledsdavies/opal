package parser

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/google/go-cmp/cmp"
	"github.com/aledsdavies/devcmd/pkgs/stdlib"
)

// init registers any test-specific decorators not in stdlib
func init() {
	registerTestOnlyDecorators()
}

// registerTestOnlyDecorators registers decorators that are only used for testing
// and not part of the standard library
func registerTestOnlyDecorators() {
	// Register all decorators used in tests to ensure they are available

	// Function decorators (inline within shell content)
	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:        "var",
		Type:        stdlib.FunctionDecorator,
		Semantic:    stdlib.SemVariable,
		Description: "Variable substitution - replaces with variable value",
		Args: []stdlib.ArgumentSpec{
			{Name: "name", Type: stdlib.IdentifierArg, Optional: false},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:        "env",
		Type:        stdlib.FunctionDecorator,
		Semantic:    stdlib.SemFunction,
		Description: "Environment variable substitution",
		Args: []stdlib.ArgumentSpec{
			{Name: "name", Type: stdlib.IdentifierArg, Optional: false},
		},
	})

	// Block decorators (require explicit braces)
	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "env",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Sets environment variables for command execution",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "vars", Type: stdlib.StringArg, Optional: false},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "timeout",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Sets execution timeout for command blocks",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "duration", Type: stdlib.DurationArg, Optional: false},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "confirm",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Prompts for user confirmation before executing commands",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "message", Type: stdlib.StringArg, Optional: true, Default: "Are you sure?"},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "debounce",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Debounces command execution with specified delay",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "delay", Type: stdlib.DurationArg, Optional: false},
			{Name: "pattern", Type: stdlib.StringArg, Optional: true},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "cwd",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Changes working directory for command execution",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "directory", Type: stdlib.ExpressionArg, Optional: false}, // Can be @var() expression
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "parallel",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Executes commands in parallel",
		RequiresBlock: true,
		Args:          []stdlib.ArgumentSpec{}, // No arguments
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "retry",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Retries command execution on failure",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "attempts", Type: stdlib.NumberArg, Optional: true, Default: "3"},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "watch-files",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Watches files for changes and executes commands",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "pattern", Type: stdlib.ExpressionArg, Optional: true},        // Can be @var() expression
			{Name: "interval", Type: stdlib.DurationArg, Optional: true, Default: "1s"},
			{Name: "recursive", Type: stdlib.BooleanArg, Optional: true, Default: "true"},
		},
	})

	// Pattern decorators (handle pattern matching with specific syntax)
	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "when",
		Type:          stdlib.PatternDecorator,
		Semantic:      stdlib.SemPattern,
		Description:   "Pattern matching based on variable value - supports any identifier patterns",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "variable", Type: stdlib.IdentifierArg, Optional: false},
		},
		PatternSpec: &stdlib.PatternSpec{
			AllowedPatterns:  nil,  // nil means any identifier is allowed
			AllowWildcard:    true, // * wildcard is allowed
			RequiredPatterns: nil,  // No required patterns
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "try",
		Type:          stdlib.PatternDecorator,
		Semantic:      stdlib.SemPattern,
		Description:   "Exception handling with main, error, and finally blocks",
		RequiresBlock: true,
		Args:          []stdlib.ArgumentSpec{}, // No arguments
		PatternSpec: &stdlib.PatternSpec{
			AllowedPatterns:  []string{"main", "error", "finally"}, // Only these patterns allowed
			AllowWildcard:    false,                                 // No wildcard
			RequiredPatterns: []string{"main"},                     // main is required
		},
	})

	// Test-specific decorators for edge cases
	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "offset",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Test decorator - applies numeric offset to command execution",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "value", Type: stdlib.NumberArg, Optional: false},
		},
	})

	stdlib.RegisterDecorator(&stdlib.DecoratorSignature{
		Name:          "factor",
		Type:          stdlib.BlockDecorator,
		Semantic:      stdlib.SemDecorator,
		Description:   "Test decorator - applies scaling factor to command execution",
		RequiresBlock: true,
		Args: []stdlib.ArgumentSpec{
			{Name: "multiplier", Type: stdlib.NumberArg, Optional: false},
		},
	})
}

// DSL for building expected test results using natural language

// Program creates an expected program
func Program(items ...interface{}) ExpectedProgram {
	var variables []ExpectedVariable
	var commands []ExpectedCommand

	for _, item := range items {
		switch v := item.(type) {
		case ExpectedVariable:
			variables = append(variables, v)
		case ExpectedCommand:
			commands = append(commands, v)
		}
	}

	return ExpectedProgram{
		Variables: variables,
		Commands:  commands,
	}
}

// Var creates a variable declaration: var NAME = VALUE
func Var(name string, value interface{}) ExpectedVariable {
	return ExpectedVariable{
		Name:  name,
		Value: toExpression(value),
	}
}

// DurationExpr creates a duration expression for test expectations
func DurationExpr(value string) ExpectedExpression {
	return ExpectedExpression{
		Type:  "duration",
		Value: value,
	}
}

// BooleanExpr creates a boolean expression for test expectations
func BooleanExpr(value bool) ExpectedExpression {
	return ExpectedExpression{
		Type:  "boolean",
		Value: strconv.FormatBool(value),
	}
}

// Cmd creates a simple command: NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Cmd(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	// Validate syntax sugar rules: only simple shell content gets automatic braces
	if cmdBody.IsBlock {
		// Instead of panic, return an error command that will fail the test gracefully
		return ExpectedCommand{
			Name: name,
			Type: ast.Command,
			Body: ExpectedCommandBody{
				IsBlock: false,
				Content: ExpectedShellContent{
					Parts: []ExpectedShellPart{
						Text("ERROR: Cmd() is for simple commands only. Use CmdBlock() for explicit block syntax"),
					},
				},
			},
		}
	}

	// Check if the content contains BLOCK decorators - this would violate syntax sugar rules
	// Function decorators (@var) are allowed in simple commands
	if shellContent, ok := cmdBody.Content.(ExpectedShellContent); ok {
		for _, part := range shellContent.Parts {
			if part.Type == "function_decorator" {
				// Function decorators are allowed in simple commands - they get syntax sugar
				if part.FunctionDecorator != nil && !stdlib.IsFunctionDecorator(part.FunctionDecorator.Name) {
					// Instead of panic, return an error command
					return ExpectedCommand{
						Name: name,
						Type: ast.Command,
						Body: ExpectedCommandBody{
							IsBlock: false,
							Content: ExpectedShellContent{
								Parts: []ExpectedShellPart{
									Text("ERROR: Cmd() cannot contain block decorators. Block decorators require explicit block syntax - use CmdBlock() instead"),
								},
							},
						},
					}
				}
			}
		}
	}

	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: cmdBody,
	}
}

// Watch creates a watch command: watch NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Watch(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	// Validate syntax sugar rules: only simple shell content gets automatic braces
	if cmdBody.IsBlock {
		// Instead of panic, return an error command
		return ExpectedCommand{
			Name: name,
			Type: ast.WatchCommand,
			Body: ExpectedCommandBody{
				IsBlock: false,
				Content: ExpectedShellContent{
					Parts: []ExpectedShellPart{
						Text("ERROR: Watch() is for simple commands only. Use WatchBlock() for explicit block syntax"),
					},
				},
			},
		}
	}

	// Check if the content contains BLOCK decorators - this would violate syntax sugar rules
	// Function decorators (@var) are allowed in simple commands
	if shellContent, ok := cmdBody.Content.(ExpectedShellContent); ok {
		for _, part := range shellContent.Parts {
			if part.Type == "function_decorator" {
				// Function decorators are allowed in simple commands - they get syntax sugar
				if part.FunctionDecorator != nil && !stdlib.IsFunctionDecorator(part.FunctionDecorator.Name) {
					// Instead of panic, return an error command
					return ExpectedCommand{
						Name: name,
						Type: ast.WatchCommand,
						Body: ExpectedCommandBody{
							IsBlock: false,
							Content: ExpectedShellContent{
								Parts: []ExpectedShellPart{
									Text("ERROR: Watch() cannot contain block decorators. Block decorators require explicit block syntax - use WatchBlock() instead"),
								},
							},
						},
					}
				}
			}
		}
	}

	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: cmdBody,
	}
}

// WatchBlock creates a watch command with explicit block syntax
func WatchBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.WatchCommand,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContent(content...),
		},
	}
}

// Stop creates a stop command: stop NAME: BODY
// This applies syntax sugar for simple shell commands with or without function decorators
func Stop(name string, body interface{}) ExpectedCommand {
	cmdBody := toCommandBody(body)

	// Validate syntax sugar rules: only simple shell content gets automatic braces
	if cmdBody.IsBlock {
		// Instead of panic, return an error command
		return ExpectedCommand{
			Name: name,
			Type: ast.StopCommand,
			Body: ExpectedCommandBody{
				IsBlock: false,
				Content: ExpectedShellContent{
					Parts: []ExpectedShellPart{
						Text("ERROR: Stop() is for simple commands only. Use StopBlock() for explicit block syntax"),
					},
				},
			},
		}
	}

	// Check if the content contains BLOCK decorators - this would violate syntax sugar rules
	// Function decorators (@var) are allowed in simple commands
	if shellContent, ok := cmdBody.Content.(ExpectedShellContent); ok {
		for _, part := range shellContent.Parts {
			if part.Type == "function_decorator" {
				// Function decorators are allowed in simple commands - they get syntax sugar
				if part.FunctionDecorator != nil && !stdlib.IsFunctionDecorator(part.FunctionDecorator.Name) {
					// Instead of panic, return an error command
					return ExpectedCommand{
						Name: name,
						Type: ast.StopCommand,
						Body: ExpectedCommandBody{
							IsBlock: false,
							Content: ExpectedShellContent{
								Parts: []ExpectedShellPart{
									Text("ERROR: Stop() cannot contain block decorators. Block decorators require explicit block syntax - use StopBlock() instead"),
								},
							},
						},
					}
				}
			}
		}
	}

	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: cmdBody,
	}
}

// StopBlock creates a stop command with explicit block syntax
func StopBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.StopCommand,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContent(content...),
		},
	}
}

// CmdBlock creates a command with explicit block syntax: NAME: { content }
func CmdBlock(name string, content ...interface{}) ExpectedCommand {
	return ExpectedCommand{
		Name: name,
		Type: ast.Command,
		Body: ExpectedCommandBody{
			IsBlock: true,
			Content: toCommandContent(content...),
		},
	}
}

// Block creates a block command body: { content }
func Block(content ...interface{}) ExpectedCommandBody {
	return ExpectedCommandBody{
		IsBlock: true,
		Content: toCommandContent(content...),
	}
}

// Simple creates a simple command body (single line)
// This enforces that simple commands cannot contain BLOCK decorators (per syntax sugar rules)
// Function decorators (@var) are allowed and get syntax sugar
func Simple(parts ...interface{}) ExpectedCommandBody {
	shellParts := toShellParts(parts...)

	// Validate that simple commands don't contain BLOCK decorators
	// Function decorators are allowed in simple commands
	for _, part := range shellParts {
		if part.Type == "function_decorator" {
			if part.FunctionDecorator != nil && !stdlib.IsFunctionDecorator(part.FunctionDecorator.Name) {
				// Instead of panic, return an error body
				return ExpectedCommandBody{
					IsBlock: false,
					Content: ExpectedShellContent{
						Parts: []ExpectedShellPart{
							Text("ERROR: Simple() command bodies cannot contain block decorators. Per spec: 'Block decorators require explicit braces' - use Block() instead"),
						},
					},
				}
			}
		}
	}

	return ExpectedCommandBody{
		IsBlock: false,
		Content: ExpectedShellContent{
			Parts: shellParts,
		},
	}
}

// Text creates a text part
func Text(text string) ExpectedShellPart {
	return ExpectedShellPart{
		Type: "text",
		Text: text,
	}
}

// At creates a function decorator within shell content: @var(NAME)
// Only valid for function decorators like @var()
func At(name string, args ...interface{}) ExpectedShellPart {
	// Validate that this is a function decorator
	if !stdlib.IsFunctionDecorator(name) {
		// Instead of panic, return an error shell part
		return ExpectedShellPart{
			Type: "text",
			Text: fmt.Sprintf("ERROR: At() can only be used with function decorators, but '%s' is not a function decorator", name),
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedShellPart{
		Type: "function_decorator",
		FunctionDecorator: &ExpectedFunctionDecorator{
			Name: name,
			Args: decoratorArgs,
		},
	}
}

// FuncDecorator creates a function decorator expression for use in decorator arguments
// This is different from At() which creates shell parts
func FuncDecorator(name string, args ...interface{}) ExpectedExpression {
	// Validate that this is a function decorator
	if !stdlib.IsFunctionDecorator(name) {
		return ExpectedExpression{
			Type:  "identifier",
			Value: fmt.Sprintf("ERROR: FuncDecorator() can only be used with function decorators, but '%s' is not a function decorator", name),
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedExpression{
		Type: "function_decorator",
		Name: name,
		Args: decoratorArgs,
	}
}

// Decorator creates a block decorator: @timeout(30s)
// Only valid for block decorators that require explicit braces
func Decorator(name string, args ...interface{}) ExpectedDecorator {
	// Validate that this is a block decorator
	if !stdlib.IsBlockDecorator(name) {
		// Instead of panic, we'll return a decorator with an error name
		// This will cause tests to fail but not panic
		return ExpectedDecorator{
			Name: fmt.Sprintf("ERROR_NOT_BLOCK_DECORATOR_%s", name),
			Args: []ExpectedExpression{},
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedDecorator{
		Name: name,
		Args: decoratorArgs,
	}
}

// PatternDecorator creates a pattern decorator: @when(VAR) or @try
// Only valid for pattern decorators that handle pattern matching
func PatternDecorator(name string, args ...interface{}) ExpectedDecorator {
	// Validate that this is a pattern decorator
	if !stdlib.IsPatternDecorator(name) {
		// Instead of panic, we'll return a decorator with an error name
		return ExpectedDecorator{
			Name: fmt.Sprintf("ERROR_NOT_PATTERN_DECORATOR_%s", name),
			Args: []ExpectedExpression{},
		}
	}

	var decoratorArgs []ExpectedExpression
	for _, arg := range args {
		decoratorArgs = append(decoratorArgs, toDecoratorArgument(name, arg))
	}

	return ExpectedDecorator{
		Name: name,
		Args: decoratorArgs,
	}
}

// Pattern creates a pattern content with branches: @when(VAR) { pattern: command }
func Pattern(decorator ExpectedDecorator, branches ...ExpectedPatternBranch) ExpectedPatternContent {
	return ExpectedPatternContent{
		Decorator: decorator,
		Branches:  branches,
	}
}

// Branch creates a pattern branch: pattern: command
func Branch(pattern interface{}, command interface{}) ExpectedPatternBranch {
	var patternObj ExpectedPattern

	switch p := pattern.(type) {
	case string:
		if p == "*" {
			patternObj = ExpectedWildcardPattern{}
		} else {
			patternObj = ExpectedIdentifierPattern{Name: p}
		}
	case ExpectedPattern:
		patternObj = p
	default:
		patternObj = ExpectedIdentifierPattern{Name: fmt.Sprintf("%v", p)}
	}

	return ExpectedPatternBranch{
		Pattern: patternObj,
		Command: toCommandContent(command),
	}
}

// Wildcard creates a wildcard pattern: *
func Wildcard() ExpectedPattern {
	return ExpectedWildcardPattern{}
}

// PatternId creates an identifier pattern: production, main, etc.
func PatternId(name string) ExpectedPattern {
	return ExpectedIdentifierPattern{Name: name}
}

// validateDecoratorsForNestedUsage validates that decorators can only be used in explicit blocks
func validateDecoratorsForNestedUsage(decorators []ExpectedDecorator) {
	// Note: Multiple decorators are allowed in CmdBlock context as they represent
	// explicit nesting like: @timeout(30s) { @retry(2) { ... } }
	// The parser will handle the actual nesting validation

	for _, decorator := range decorators {
		if stdlib.RequiresExplicitBlock(decorator.Name) {
			// This will be validated at the call site to ensure explicit braces
		}
	}
}

// toDecoratorArgument converts arguments based on the decorator type
func toDecoratorArgument(decoratorName string, arg interface{}) ExpectedExpression {
	// Handle special cases for specific decorators
	switch decoratorName {
	case "var":
		// @var() takes identifier arguments (variable names)
		if str, ok := arg.(string); ok {
			return ExpectedExpression{Type: "identifier", Value: str}
		}
		return toExpression(arg)
	case "when":
		// @when() takes identifier arguments (variable names)
		if str, ok := arg.(string); ok {
			return ExpectedExpression{Type: "identifier", Value: str}
		}
		return toExpression(arg)
	default:
		// For other decorators, use the default conversion
		return toExpression(arg)
	}
}

// Helper conversion functions
func toExpression(v interface{}) ExpectedExpression {
	switch val := v.(type) {
	case string:
		if strings.HasPrefix(val, "@") {
			return ExpectedExpression{Type: "decorator", Value: val}
		}
		// Check if it's a duration pattern and return as duration
		if isDurationString(val) {
			return ExpectedExpression{Type: "duration", Value: val}
		}
		// Check if it's a numeric string
		if isNumericString(val) {
			return ExpectedExpression{Type: "number", Value: val}
		}
		return ExpectedExpression{Type: "string", Value: val}
	case int:
		return ExpectedExpression{Type: "number", Value: strconv.Itoa(val)}
	case bool:
		return ExpectedExpression{Type: "boolean", Value: strconv.FormatBool(val)}
	case ExpectedExpression:
		return val
	case ExpectedFunctionDecorator:
		return ExpectedExpression{
			Type: "function_decorator",
			Name: val.Name,
			Args: val.Args,
		}
	default:
		// Try to convert to string and handle as identifier
		str := fmt.Sprintf("%v", val)
		return ExpectedExpression{Type: "identifier", Value: strings.Trim(str, "\"")}
	}
}

func toCommandBody(v interface{}) ExpectedCommandBody {
	switch val := v.(type) {
	case ExpectedCommandBody:
		return val
	case string:
		// Empty string should create empty shell content
		if val == "" {
			return ExpectedCommandBody{
				IsBlock: false,
				Content: ExpectedShellContent{Parts: []ExpectedShellPart{}},
			}
		}
		// Simple string becomes simple command body (gets syntax sugar)
		return Simple(Text(val))
	case ExpectedShellContent:
		// Shell content that doesn't explicitly specify block structure
		// Check if it contains BLOCK decorators - if so, it needs explicit blocks
		// Function decorators are allowed and get syntax sugar
		for _, part := range val.Parts {
			if part.Type == "function_decorator" {
				if part.FunctionDecorator != nil && !stdlib.IsFunctionDecorator(part.FunctionDecorator.Name) {
					// Instead of panic, return an error body
					return ExpectedCommandBody{
						IsBlock: true, // Force block to avoid syntax sugar issues
						Content: ExpectedShellContent{
							Parts: []ExpectedShellPart{
								Text("ERROR: Shell content with block decorators requires explicit block syntax"),
							},
						},
					}
				}
			}
		}
		return ExpectedCommandBody{
			IsBlock: false,
			Content: val,
		}
	case ExpectedDecoratedContent:
		// Decorated content ALWAYS requires explicit blocks per spec
		return ExpectedCommandBody{
			IsBlock: true,
			Content: val,
		}
	case ExpectedPatternContent:
		// Pattern content ALWAYS requires explicit blocks per spec
		return ExpectedCommandBody{
			IsBlock: true,
			Content: val,
		}
	default:
		return ExpectedCommandBody{
			IsBlock: false,
			Content: ExpectedShellContent{
				Parts: []ExpectedShellPart{},
			},
		}
	}
}

func toCommandContent(items ...interface{}) ExpectedCommandContent {
	if len(items) == 0 {
		return ExpectedShellContent{Parts: []ExpectedShellPart{}}
	}

	var decorators []ExpectedDecorator
	var contentStart int

	// Check if the first item is a pattern decorator
	if len(items) > 0 {
		if patternContent, ok := items[0].(ExpectedPatternContent); ok {
			return patternContent
		}
	}

	// Collect all leading decorators
	for i, item := range items {
		if dec, ok := item.(ExpectedDecorator); ok {
			decorators = append(decorators, dec)
			contentStart = i + 1
		} else {
			break
		}
	}

	// If we have decorators, create decorated content
	if len(decorators) > 0 {
		// Validate decorator usage - but allow multiple decorators in CmdBlock context
		// This is for explicit nesting like: @timeout(30s) { @retry(2) { ... } }
		// which becomes: CmdBlock("cmd", Decorator("timeout", "30s"), Decorator("retry", "2"), Text(...))

		// Rest is the content
		var content ExpectedCommandContent
		if contentStart < len(items) {
			content = toCommandContent(items[contentStart:]...)
		} else {
			content = ExpectedShellContent{Parts: []ExpectedShellPart{}}
		}

		return ExpectedDecoratedContent{
			Decorators: decorators,
			Content:    content,
		}
	}

	// This is shell content
	return ExpectedShellContent{
		Parts: toShellParts(items...),
	}
}

func toShellParts(items ...interface{}) []ExpectedShellPart {
	var parts []ExpectedShellPart
	for _, item := range items {
		switch v := item.(type) {
		case ExpectedShellPart:
			parts = append(parts, v)
		case string:
			parts = append(parts, Text(v))
		case ExpectedFunctionDecorator:
			// Validate that function decorators are only used inline
			if !stdlib.IsFunctionDecorator(v.Name) {
				// Instead of panic, create an error text part
				parts = append(parts, Text(fmt.Sprintf("ERROR: '%s' is not a function decorator and cannot be used inline in shell content", v.Name)))
			} else {
				parts = append(parts, ExpectedShellPart{
					Type: "function_decorator",
					FunctionDecorator: &v,
				})
			}
		default:
			parts = append(parts, Text(fmt.Sprintf("%v", v)))
		}
	}
	return parts
}

func toDecorators(v interface{}) []ExpectedDecorator {
	switch val := v.(type) {
	case ExpectedDecorator:
		return []ExpectedDecorator{val}
	case []ExpectedDecorator:
		// Allow multiple decorators - this represents explicit nesting
		return val
	case []interface{}:
		var decorators []ExpectedDecorator
		for _, item := range val {
			if dec, ok := item.(ExpectedDecorator); ok {
				decorators = append(decorators, dec)
			}
		}
		// Allow multiple decorators - parser will handle nesting validation
		return decorators
	default:
		return []ExpectedDecorator{}
	}
}

// Helper functions for string validation
func isDurationString(s string) bool {
	if len(s) < 2 {
		return false
	}

	suffixes := []string{"ns", "us", "μs", "ms", "s", "m", "h"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			prefix := s[:len(s)-len(suffix)]
			if _, err := strconv.ParseFloat(prefix, 64); err == nil {
				return true
			}
		}
	}
	return false
}

func isNumericString(s string) bool {
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

// Test helper types for the new CST structure
type ExpectedProgram struct {
	Variables []ExpectedVariable
	Commands  []ExpectedCommand
}

type ExpectedVariable struct {
	Name  string
	Value ExpectedExpression
}

type ExpectedCommand struct {
	Name string
	Type ast.CommandType
	Body ExpectedCommandBody
}

type ExpectedCommandBody struct {
	IsBlock bool
	Content ExpectedCommandContent
}

type ExpectedCommandContent interface {
	IsExpectedCommandContent() bool
}

type ExpectedShellContent struct {
	Parts []ExpectedShellPart
}

func (s ExpectedShellContent) IsExpectedCommandContent() bool { return true }

type ExpectedDecoratedContent struct {
	Decorators []ExpectedDecorator
	Content    ExpectedCommandContent
}

func (d ExpectedDecoratedContent) IsExpectedCommandContent() bool { return true }

type ExpectedPatternContent struct {
	Decorator ExpectedDecorator
	Branches  []ExpectedPatternBranch
}

func (p ExpectedPatternContent) IsExpectedCommandContent() bool { return true }

type ExpectedPatternBranch struct {
	Pattern ExpectedPattern
	Command ExpectedCommandContent
}

type ExpectedPattern interface {
	IsExpectedPattern() bool
}

type ExpectedIdentifierPattern struct {
	Name string
}

func (i ExpectedIdentifierPattern) IsExpectedPattern() bool { return true }

type ExpectedWildcardPattern struct{}

func (w ExpectedWildcardPattern) IsExpectedPattern() bool { return true }

type ExpectedShellPart struct {
	Type              string
	Text              string
	FunctionDecorator *ExpectedFunctionDecorator
}

type ExpectedDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedFunctionDecorator struct {
	Name string
	Args []ExpectedExpression
}

type ExpectedExpression struct {
	Type  string
	Value string
	// For function decorators
	Name string                `json:"name,omitempty"`
	Args []ExpectedExpression `json:"args,omitempty"`
}

// Test case structure
type TestCase struct {
	Name        string
	Input       string
	WantErr     bool
	ErrorSubstr string
	Expected    ExpectedProgram
}

// flattenVariables collects all variables from individual and grouped declarations
func flattenVariables(program *ast.Program) []ast.VariableDecl {
	var allVariables []ast.VariableDecl

	// Add individual variables
	allVariables = append(allVariables, program.Variables...)

	// Add variables from groups
	for _, group := range program.VarGroups {
		allVariables = append(allVariables, group.Variables...)
	}

	return allVariables
}

// Comparison helpers for the new CST structure
func expressionToComparable(expr ast.Expression) interface{} {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return map[string]interface{}{
			"Type":  "string",
			"Value": e.Value,
		}
	case *ast.NumberLiteral:
		return map[string]interface{}{
			"Type":  "number",
			"Value": e.Value,
		}
	case *ast.DurationLiteral:
		return map[string]interface{}{
			"Type":  "duration",
			"Value": e.Value,
		}
	case *ast.BooleanLiteral:
		return map[string]interface{}{
			"Type":  "boolean",
			"Value": strconv.FormatBool(e.Value),
		}
	case *ast.Identifier:
		return map[string]interface{}{
			"Type":  "identifier",
			"Value": e.Name,
		}
	case *ast.FunctionDecorator:
		args := make([]interface{}, len(e.Args))
		for i, arg := range e.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "function_decorator",
			"Name": e.Name,
			"Args": args,
		}
	default:
		return map[string]interface{}{
			"Type":  "unknown",
			"Value": expr.String(),
		}
	}
}

func expectedExpressionToComparable(expr ExpectedExpression) interface{} {
	if expr.Type == "function_decorator" {
		args := make([]interface{}, len(expr.Args))
		for i, arg := range expr.Args {
			args[i] = expectedExpressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "function_decorator",
			"Name": expr.Name,
			"Args": args,
		}
	}
	return map[string]interface{}{
		"Type":  expr.Type,
		"Value": expr.Value,
	}
}

func shellPartToComparable(part ast.ShellPart) interface{} {
	switch p := part.(type) {
	case *ast.TextPart:
		return map[string]interface{}{
			"Type": "text",
			"Text": p.Text,
		}
	case *ast.FunctionDecorator:
		args := make([]interface{}, len(p.Args))
		for i, arg := range p.Args {
			args[i] = expressionToComparable(arg)
		}
		return map[string]interface{}{
			"Type": "function_decorator",
			"FunctionDecorator": map[string]interface{}{
				"Name": p.Name,
				"Args": args,
			},
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
			"Text": part.String(),
		}
	}
}

func expectedShellPartToComparable(part ExpectedShellPart) interface{} {
	result := map[string]interface{}{
		"Type": part.Type,
	}

	switch part.Type {
	case "text":
		result["Text"] = part.Text
	case "function_decorator":
		if part.FunctionDecorator != nil {
			args := make([]interface{}, len(part.FunctionDecorator.Args))
			for i, arg := range part.FunctionDecorator.Args {
				args[i] = expectedExpressionToComparable(arg)
			}
			result["FunctionDecorator"] = map[string]interface{}{
				"Name": part.FunctionDecorator.Name,
				"Args": args,
			}
		}
	}

	return result
}

func patternToComparable(pattern ast.Pattern) interface{} {
	switch p := pattern.(type) {
	case *ast.IdentifierPattern:
		return map[string]interface{}{
			"Type": "identifier",
			"Name": p.Name,
		}
	case *ast.WildcardPattern:
		return map[string]interface{}{
			"Type": "wildcard",
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func expectedPatternToComparable(pattern ExpectedPattern) interface{} {
	switch p := pattern.(type) {
	case ExpectedIdentifierPattern:
		return map[string]interface{}{
			"Type": "identifier",
			"Name": p.Name,
		}
	case ExpectedWildcardPattern:
		return map[string]interface{}{
			"Type": "wildcard",
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func commandContentToComparable(content ast.CommandContent) interface{} {
	switch c := content.(type) {
	case *ast.ShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = shellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case *ast.DecoratedContent:
		decorators := make([]interface{}, len(c.Decorators))
		for i, decorator := range c.Decorators {
			args := make([]interface{}, len(decorator.Args))
			for j, arg := range decorator.Args {
				args[j] = expressionToComparable(arg)
			}
			decorators[i] = map[string]interface{}{
				"Name": decorator.Name,
				"Args": args,
			}
		}
		return map[string]interface{}{
			"Type":       "decorated",
			"Decorators": decorators,
			"Content":    commandContentToComparable(c.Content),
		}
	case *ast.PatternContent:
		decorator := map[string]interface{}{
			"Name": c.Decorator.Name,
			"Args": make([]interface{}, len(c.Decorator.Args)),
		}
		for i, arg := range c.Decorator.Args {
			decorator["Args"].([]interface{})[i] = expressionToComparable(arg)
		}

		branches := make([]interface{}, len(c.Patterns))
		for i, branch := range c.Patterns {
			branches[i] = map[string]interface{}{
				"Pattern": patternToComparable(branch.Pattern),
				"Command": commandContentToComparable(branch.Command),
			}
		}

		return map[string]interface{}{
			"Type":      "pattern",
			"Decorator": decorator,
			"Branches":  branches,
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func expectedCommandContentToComparable(content ExpectedCommandContent) interface{} {
	switch c := content.(type) {
	case ExpectedShellContent:
		parts := make([]interface{}, len(c.Parts))
		for i, part := range c.Parts {
			parts[i] = expectedShellPartToComparable(part)
		}
		return map[string]interface{}{
			"Type":  "shell",
			"Parts": parts,
		}
	case ExpectedDecoratedContent:
		decorators := make([]interface{}, len(c.Decorators))
		for i, decorator := range c.Decorators {
			args := make([]interface{}, len(decorator.Args))
			for j, arg := range decorator.Args {
				args[j] = expectedExpressionToComparable(arg)
			}
			decorators[i] = map[string]interface{}{
				"Name": decorator.Name,
				"Args": args,
			}
		}
		return map[string]interface{}{
			"Type":       "decorated",
			"Decorators": decorators,
			"Content":    expectedCommandContentToComparable(c.Content),
		}
	case ExpectedPatternContent:
		decorator := map[string]interface{}{
			"Name": c.Decorator.Name,
			"Args": make([]interface{}, len(c.Decorator.Args)),
		}
		for i, arg := range c.Decorator.Args {
			decorator["Args"].([]interface{})[i] = expectedExpressionToComparable(arg)
		}

		branches := make([]interface{}, len(c.Branches))
		for i, branch := range c.Branches {
			branches[i] = map[string]interface{}{
				"Pattern": expectedPatternToComparable(branch.Pattern),
				"Command": expectedCommandContentToComparable(branch.Command),
			}
		}

		return map[string]interface{}{
			"Type":      "pattern",
			"Decorator": decorator,
			"Branches":  branches,
		}
	default:
		return map[string]interface{}{
			"Type": "unknown",
		}
	}
}

func commandBodyToComparable(body ast.CommandBody) interface{} {
	return map[string]interface{}{
		"IsBlock": body.IsBlock,
		"Content": commandContentToComparable(body.Content),
	}
}

func expectedCommandBodyToComparable(body ExpectedCommandBody) interface{} {
	return map[string]interface{}{
		"IsBlock": body.IsBlock,
		"Content": expectedCommandContentToComparable(body.Content),
	}
}

func RunTestCase(t *testing.T, tc TestCase) {
	t.Run(tc.Name, func(t *testing.T) {
		program, err := Parse(tc.Input)

		if tc.WantErr {
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.ErrorSubstr != "" && !strings.Contains(err.Error(), tc.ErrorSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.ErrorSubstr, err.Error())
			}
			return
		}

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Flatten all variables (individual and grouped) for comparison
		allVariables := flattenVariables(program)

		// Verify variables
		if len(allVariables) != len(tc.Expected.Variables) {
			t.Errorf("expected %d variables, got %d", len(tc.Expected.Variables), len(allVariables))
		} else {
			for i, expectedVar := range tc.Expected.Variables {
				actualVar := allVariables[i]

				actualComparable := map[string]interface{}{
					"Name":  actualVar.Name,
					"Value": expressionToComparable(actualVar.Value),
				}

				expectedComparable := map[string]interface{}{
					"Name":  expectedVar.Name,
					"Value": expectedExpressionToComparable(expectedVar.Value),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Variable[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}

		// Verify commands
		if len(program.Commands) != len(tc.Expected.Commands) {
			t.Errorf("expected %d commands, got %d", len(tc.Expected.Commands), len(program.Commands))
		} else {
			for i, expectedCmd := range tc.Expected.Commands {
				actualCmd := program.Commands[i]

				actualComparable := map[string]interface{}{
					"Name": actualCmd.Name,
					"Type": actualCmd.Type,
					"Body": commandBodyToComparable(actualCmd.Body),
				}

				expectedComparable := map[string]interface{}{
					"Name": expectedCmd.Name,
					"Type": expectedCmd.Type,
					"Body": expectedCommandBodyToComparable(expectedCmd.Body),
				}

				if diff := cmp.Diff(expectedComparable, actualComparable); diff != "" {
					t.Errorf("Command[%d] mismatch (-expected +actual):\n%s", i, diff)
				}
			}
		}
	})
}
