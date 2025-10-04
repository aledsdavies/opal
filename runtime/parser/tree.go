package parser

import (
	"github.com/aledsdavies/opal/runtime/lexer"
)

// ParseTree represents the result of parsing
type ParseTree struct {
	Source      []byte          // Original source (for reference)
	Tokens      []lexer.Token   // Tokens from lexer
	Events      []Event         // Parse events
	Errors      []ParseError    // Parse errors
	Telemetry   *ParseTelemetry // Performance metrics (nil if disabled)
	DebugEvents []DebugEvent    // Debug events (nil if disabled)
}

// Event represents a parse tree construction event
type Event struct {
	Kind EventKind
	Data uint32
}

// EventKind represents the type of parse event
type EventKind uint8

const (
	EventOpen  EventKind = iota // Open syntax node
	EventClose                  // Close syntax node
	EventToken                  // Consume token
)

// NodeKind represents syntax node types
type NodeKind uint32

const (
	NodeSource NodeKind = iota // Top-level source (file, stdin, string)
	NodeFunction
	NodeParamList
	NodeBlock
	NodeParam          // Function parameter
	NodeTypeAnnotation // Type annotation (: Type)
	NodeDefaultValue   // Default value (= expression)

	// Statements
	NodeVarDecl // Variable declaration

	// Expressions
	NodeLiteral    // Literal value (int, string, bool, duration)
	NodeIdentifier // Identifier reference
	NodeBinaryExpr // Binary expression (a + b, a == b, etc.)
)

// ParseError represents a parse error with rich context for user-friendly messages
type ParseError struct {
	// Location
	Filename string         // Source filename (empty for stdin/string)
	Position lexer.Position // Line, column, offset

	// Core error info
	Message string // Clear, specific: "missing closing parenthesis"
	Context string // What we were parsing: "parameter list"

	// What went wrong
	Expected []lexer.TokenType // What tokens would be valid
	Got      lexer.TokenType   // What we found instead

	// How to fix it (educational)
	Suggestion string // Actionable fix: "Add ')' after the last parameter"
	Example    string // Valid syntax: "fun greet(name) {}"
	Note       string // Optional explanation for learning
}
