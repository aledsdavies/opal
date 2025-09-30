package parser

import (
	"github.com/aledsdavies/opal/runtime/lexer"
)

// Parse parses the input bytes and returns a parse tree
// Takes []byte directly for zero-copy performance
func Parse(source []byte) *ParseTree {
	// Lex the input first
	lex := lexer.NewLexer("")
	lex.Init(source)
	tokens := lex.GetTokens()

	// Create parser
	p := &parser{
		tokens: tokens,
		pos:    0,
		events: []Event{},
		errors: []ParseError{},
	}

	// Parse the file
	p.file()

	return &ParseTree{
		Source: source,
		Tokens: tokens,
		Events: p.events,
		Errors: p.errors,
	}
}

// ParseString is a convenience wrapper for tests
func ParseString(input string) *ParseTree {
	return Parse([]byte(input))
}

// parser is the internal parser state
type parser struct {
	tokens []lexer.Token
	pos    int
	events []Event
	errors []ParseError
}

// file parses the top-level file structure
func (p *parser) file() {
	// For now, just create an empty File node
	p.open()
	p.close()
}

// open emits an Open event
func (p *parser) open() {
	p.events = append(p.events, Event{
		Kind: EventOpen,
		Data: 0, // NodeKind will go here later
	})
}

// close emits a Close event
func (p *parser) close() {
	p.events = append(p.events, Event{
		Kind: EventClose,
		Data: 0,
	})
}
