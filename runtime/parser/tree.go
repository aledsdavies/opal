package parser

import (
	"github.com/aledsdavies/opal/runtime/lexer"
)

// ParseTree represents the result of parsing
type ParseTree struct {
	Source []byte        // Original source (for reference)
	Tokens []lexer.Token // Tokens from lexer
	Events []Event       // Parse events
	Errors []ParseError  // Parse errors
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

// ParseError represents a parse error
type ParseError struct {
	Message string
}
