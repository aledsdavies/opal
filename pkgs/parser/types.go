package parser

import (
	"fmt"

	"github.com/aledsdavies/devcmd/pkgs/ast"
	"github.com/aledsdavies/devcmd/pkgs/lexer"
)

// Parser represents the main parser state
type Parser struct {
	tokens    []lexer.Token
	current   int
	structure *StructureMap
	errors    []ParseError
	config    ParserConfig

	// Fast lookups built during preprocessing
	decorators map[int]*ast.Decorator
}

// StructureMap holds preprocessed structural information
type StructureMap struct {
	Variables   []VariableSpan
	Commands    []CommandSpan
	Decorators  []DecoratorSpan
	BlockRanges []BlockRange
}

// VariableSpan represents a variable declaration location
type VariableSpan struct {
	NameToken   lexer.Token
	ValueStart  int // token index
	ValueEnd    int
	IsGrouped   bool
	GroupStart  int // for grouped variables
	GroupEnd    int
}

// CommandSpan represents a command declaration location
type CommandSpan struct {
	TypeToken   lexer.Token // var/watch/stop
	NameToken   lexer.Token
	ColonToken  lexer.Token
	BodyStart   int
	BodyEnd     int
	IsBlock     bool
	Decorators  []int // indices into DecoratorSpan slice
}

// DecoratorSpan represents a decorator location with unified args and block support
type DecoratorSpan struct {
	AtToken     lexer.Token
	NameToken   lexer.Token
	HasArgs     bool
	ArgsStart   int // ( token index
	ArgsEnd     int // ) token index
	HasBlock    bool
	BlockStart  int // { token index
	BlockEnd    int // } token index
	Args        []DecoratorArgSpan
	StartIndex  int
	EndIndex    int
}

// DecoratorArgSpan represents an argument within decorator parentheses
type DecoratorArgSpan struct {
	Name        string // empty for positional args
	ValueStart  int    // token index
	ValueEnd    int
	IsNamed     bool
}

// BlockRange represents a block command's boundaries
type BlockRange struct {
	OpenBrace   lexer.Token
	CloseBrace  lexer.Token
	StartIndex  int
	EndIndex    int
	Statements  []StatementSpan
}

// StatementSpan represents a statement within a block
type StatementSpan struct {
	Start       int
	End         int
	HasDecorator bool
	DecoratorIndex int // index into DecoratorSpan slice if HasDecorator
}

// TokenRange represents a range of tokens for AST nodes
type TokenRange struct {
	Start int // index into parser.tokens
	End   int // index into parser.tokens
}

// Tokens returns the tokens in this range
func (tr TokenRange) Tokens(p *Parser) []lexer.Token {
	if tr.Start < 0 || tr.End >= len(p.tokens) || tr.Start > tr.End {
		return nil
	}
	return p.tokens[tr.Start : tr.End+1]
}

// ErrorType categorizes different kinds of parse errors
type ErrorType int

const (
	SyntaxError ErrorType = iota
	SemanticError
	DuplicateError
	ReferenceError
)

// ParseError represents a parse error with user-friendly context
type ParseError struct {
	Type     ErrorType
	Token    lexer.Token
	Message  string
	Context  string
	Hint     string
	Related  []lexer.Token // Related tokens for better error messages
}

// Error implements the error interface
func (pe ParseError) Error() string {
	position := fmt.Sprintf("line %d, column %d", pe.Token.Line, pe.Token.Column)

	if pe.Context != "" {
		return fmt.Sprintf("%s at %s in %s: %s",
			pe.errorTypeString(), position, pe.Context, pe.Message)
	}

	return fmt.Sprintf("%s at %s: %s",
		pe.errorTypeString(), position, pe.Message)
}

// DetailedError returns a more detailed error message with hints
func (pe ParseError) DetailedError() string {
	base := pe.Error()
	if pe.Hint != "" {
		return fmt.Sprintf("%s\nHint: %s", base, pe.Hint)
	}
	return base
}

func (pe ParseError) errorTypeString() string {
	switch pe.Type {
	case SyntaxError:
		return "Syntax error"
	case SemanticError:
		return "Semantic error"
	case DuplicateError:
		return "Duplicate declaration"
	case ReferenceError:
		return "Reference error"
	default:
		return "Parse error"
	}
}

// ParseResult contains the result of parsing
type ParseResult struct {
	Program *ast.Program
	Errors  []ParseError
}

// HasErrors returns true if there are any parse errors
func (pr ParseResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

// ErrorSummary returns a summary of all errors
func (pr ParseResult) ErrorSummary() string {
	if !pr.HasErrors() {
		return "No errors"
	}

	syntaxCount := 0
	semanticCount := 0
	duplicateCount := 0
	referenceCount := 0

	for _, err := range pr.Errors {
		switch err.Type {
		case SyntaxError:
			syntaxCount++
		case SemanticError:
			semanticCount++
		case DuplicateError:
			duplicateCount++
		case ReferenceError:
			referenceCount++
		}
	}

	summary := fmt.Sprintf("Found %d error(s)", len(pr.Errors))
	details := []string{}

	if syntaxCount > 0 {
		details = append(details, fmt.Sprintf("%d syntax", syntaxCount))
	}
	if semanticCount > 0 {
		details = append(details, fmt.Sprintf("%d semantic", semanticCount))
	}
	if duplicateCount > 0 {
		details = append(details, fmt.Sprintf("%d duplicate", duplicateCount))
	}
	if referenceCount > 0 {
		details = append(details, fmt.Sprintf("%d reference", referenceCount))
	}

	if len(details) > 0 {
		summary += " (" + joinStrings(details, ", ") + ")"
	}

	return summary
}

// Helper function to join strings (avoiding external dependencies)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// ParserConfig holds configuration options for the parser
type ParserConfig struct {
	// MaxErrors limits the number of errors collected before stopping
	MaxErrors int

	// StrictMode enables additional validation
	StrictMode bool

	// AllowUndefinedVars allows references to undefined variables
	AllowUndefinedVars bool
}

// DefaultConfig returns the default parser configuration
func DefaultConfig() ParserConfig {
	return ParserConfig{
		MaxErrors:          50,
		StrictMode:         false,
		AllowUndefinedVars: false,
	}
}
