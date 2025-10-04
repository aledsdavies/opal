package parser

import (
	"time"

	"github.com/aledsdavies/opal/runtime/lexer"
)

// Parse parses the input bytes and returns a parse tree
// Takes []byte directly for zero-copy performance
func Parse(source []byte, opts ...ParserOpt) *ParseTree {
	config := &ParserConfig{}
	for _, opt := range opts {
		opt(config)
	}

	var telemetry *ParseTelemetry
	var debugEvents []DebugEvent
	var startTotal time.Time

	// Initialize telemetry if enabled
	if config.telemetry >= TelemetryBasic {
		telemetry = &ParseTelemetry{}
		if config.telemetry >= TelemetryTiming {
			startTotal = time.Now()
		}
	}

	// Initialize debug if enabled
	if config.debug > DebugOff {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	// Lex the input first
	var startLex time.Time
	if config.telemetry >= TelemetryTiming {
		startLex = time.Now()
	}

	lex := lexer.NewLexer()
	lex.Init(source)
	tokens := lex.GetTokens()

	if config.telemetry >= TelemetryBasic {
		telemetry.TokenCount = len(tokens)
		if config.telemetry >= TelemetryTiming {
			telemetry.LexTime = time.Since(startLex)
		}
	}

	// Create parser with pre-allocated buffers
	// Heuristic: ~3 events per token (Open, Token, Close for simple nodes)
	eventCap := len(tokens) * 3
	if eventCap < 16 {
		eventCap = 16
	}

	p := &parser{
		tokens:      tokens,
		pos:         0,
		events:      make([]Event, 0, eventCap),
		errors:      make([]ParseError, 0, 4), // Most parses have 0-4 errors
		config:      config,
		debugEvents: debugEvents,
	}

	// Parse the file
	var startParse time.Time
	if config.telemetry >= TelemetryTiming {
		startParse = time.Now()
	}

	p.file()

	if config.telemetry >= TelemetryBasic {
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
		if config.telemetry >= TelemetryTiming {
			telemetry.ParseTime = time.Since(startParse)
			telemetry.TotalTime = time.Since(startTotal)
		}
	}

	return &ParseTree{
		Source:      source,
		Tokens:      tokens,
		Events:      p.events,
		Errors:      p.errors,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}
}

// ParseString is a convenience wrapper for tests
func ParseString(input string, opts ...ParserOpt) *ParseTree {
	return Parse([]byte(input), opts...)
}

// ParseTokens parses pre-lexed tokens (for benchmarking pure parse performance)
func ParseTokens(source []byte, tokens []lexer.Token, opts ...ParserOpt) *ParseTree {
	config := &ParserConfig{}
	for _, opt := range opts {
		opt(config)
	}

	var telemetry *ParseTelemetry
	var debugEvents []DebugEvent
	var startTotal time.Time

	// Initialize telemetry if enabled
	if config.telemetry >= TelemetryBasic {
		telemetry = &ParseTelemetry{}
		if config.telemetry >= TelemetryTiming {
			startTotal = time.Now()
		}
	}

	// Initialize debug if enabled
	if config.debug > DebugOff {
		debugEvents = make([]DebugEvent, 0, 100)
	}

	// Create parser with pre-allocated buffers
	eventCap := len(tokens) * 3
	if eventCap < 16 {
		eventCap = 16
	}

	p := &parser{
		tokens:      tokens,
		pos:         0,
		events:      make([]Event, 0, eventCap),
		errors:      make([]ParseError, 0, 4),
		config:      config,
		debugEvents: debugEvents,
	}

	// Parse the file
	var startParse time.Time
	if config.telemetry >= TelemetryTiming {
		startParse = time.Now()
	}

	p.file()

	if config.telemetry >= TelemetryBasic {
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
		telemetry.TokenCount = len(tokens)
		if config.telemetry >= TelemetryTiming {
			telemetry.ParseTime = time.Since(startParse)
			telemetry.TotalTime = time.Since(startTotal)
		}
	}

	return &ParseTree{
		Source:      source,
		Tokens:      tokens,
		Events:      p.events,
		Errors:      p.errors,
		Telemetry:   telemetry,
		DebugEvents: p.debugEvents,
	}
}

// parser is the internal parser state
type parser struct {
	tokens      []lexer.Token
	pos         int
	events      []Event
	errors      []ParseError
	config      *ParserConfig
	debugEvents []DebugEvent
}

// recordDebugEvent records debug events when debug tracing is enabled
func (p *parser) recordDebugEvent(event, context string) {
	if p.config.debug == DebugOff || p.debugEvents == nil {
		return
	}

	p.debugEvents = append(p.debugEvents, DebugEvent{
		Timestamp: time.Now(),
		Event:     event,
		TokenPos:  p.pos,
		Context:   context,
	})
}

// file parses the top-level source structure (file, stdin, or string)
func (p *parser) file() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_source", "parsing source")
	}

	kind := p.start(NodeSource)

	// Parse top-level declarations
	for !p.at(lexer.EOF) {
		if p.at(lexer.FUN) {
			p.function()
		} else if p.at(lexer.VAR) {
			p.varDecl()
		} else {
			// Unknown token, skip for now
			p.advance()
		}
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_source", "source complete")
	}
}

// function parses a function declaration: fun IDENTIFIER ParamList Block
func (p *parser) function() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_function", "parsing function")
	}

	kind := p.start(NodeFunction)

	// Consume 'fun' keyword
	p.token()

	// Consume function name
	if p.at(lexer.IDENTIFIER) {
		p.token()
	}

	// Parse parameter list
	p.paramList()

	// Parse block
	p.block()

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_function", "function complete")
	}
}

// paramList parses a parameter list: ( params )
func (p *parser) paramList() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_paramList", "parsing param list")
	}

	kind := p.start(NodeParamList)

	// Expect '('
	p.expect(lexer.LPAREN, "parameter list")

	// Parse parameters (comma-separated)
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		p.param()

		// If there's a comma, consume it and continue
		if p.at(lexer.COMMA) {
			p.token()
		} else {
			// No comma means we're done with parameters
			break
		}
	}

	// Expect ')'
	p.expect(lexer.RPAREN, "parameter list")

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_paramList", "param list complete")
	}
}

// param parses a single parameter: IDENTIFIER (: Type)? (= expression)?
func (p *parser) param() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_param", "parsing parameter")
	}

	kind := p.start(NodeParam)

	// Consume parameter name
	if p.at(lexer.IDENTIFIER) {
		p.token()
	}

	// Parse optional type annotation
	if p.at(lexer.COLON) {
		p.typeAnnotation()
	}

	// Parse optional default value
	if p.at(lexer.EQUALS) {
		p.defaultValue()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_param", "parameter complete")
	}
}

// typeAnnotation parses a type annotation: : Type
func (p *parser) typeAnnotation() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_typeAnnotation", "parsing type annotation")
	}

	kind := p.start(NodeTypeAnnotation)

	// Consume ':'
	if p.at(lexer.COLON) {
		p.token()
	}

	// Consume type name
	if p.at(lexer.IDENTIFIER) {
		p.token()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_typeAnnotation", "type annotation complete")
	}
}

// defaultValue parses a default value: = expression
func (p *parser) defaultValue() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_defaultValue", "parsing default value")
	}

	kind := p.start(NodeDefaultValue)

	// Consume '='
	if p.at(lexer.EQUALS) {
		p.token()
	}

	// Parse expression (for now, just consume one token - string literal, number, etc.)
	// TODO: Full expression parsing in later iteration
	if !p.at(lexer.EOF) && !p.at(lexer.RPAREN) && !p.at(lexer.COMMA) {
		p.token()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_defaultValue", "default value complete")
	}
}

// block parses a block: { statements }
func (p *parser) block() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_block", "parsing block")
	}

	kind := p.start(NodeBlock)

	// Expect '{'
	p.expect(lexer.LBRACE, "function body")

	// Parse statements
	for !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		p.statement()
	}

	// Expect '}'
	p.expect(lexer.RBRACE, "function body")

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_block", "block complete")
	}
}

// statement parses a statement
func (p *parser) statement() {
	if p.at(lexer.VAR) {
		p.varDecl()
	} else {
		// For now, skip unknown statements
		p.advance()
	}
}

// varDecl parses a variable declaration: var IDENTIFIER = expression
func (p *parser) varDecl() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_var_decl", "parsing variable declaration")
	}

	kind := p.start(NodeVarDecl)

	// Consume 'var' keyword
	p.token()

	// Expect identifier
	if !p.expect(lexer.IDENTIFIER, "variable declaration") {
		p.finish(kind)
		return
	}

	// Expect '='
	if !p.expect(lexer.EQUALS, "variable declaration") {
		p.finish(kind)
		return
	}

	// Parse expression
	p.expression()

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_var_decl", "variable declaration complete")
	}
}

// expression parses an expression
func (p *parser) expression() {
	p.binaryExpr(0) // Start with lowest precedence
}

// binaryExpr parses binary expressions with precedence
func (p *parser) binaryExpr(minPrec int) {
	// Parse left side (primary expression)
	p.primary()

	// Parse binary operators
	for {
		prec := p.precedence()
		if prec == 0 || prec < minPrec {
			break
		}

		// We have a binary operator
		kind := p.start(NodeBinaryExpr)
		p.token() // Consume operator

		// Parse right side with higher precedence
		p.binaryExpr(prec + 1)

		p.finish(kind)
	}
}

// primary parses a primary expression (literal, identifier, etc.)
func (p *parser) primary() {
	switch {
	case p.at(lexer.INTEGER), p.at(lexer.FLOAT), p.at(lexer.STRING), p.at(lexer.BOOLEAN):
		// Literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)

	case p.at(lexer.IDENTIFIER):
		// Identifier
		kind := p.start(NodeIdentifier)
		p.token()
		p.finish(kind)

	default:
		// Unexpected token - report error and create error node
		p.errorUnexpected("expression")
		// Advance to prevent infinite loop
		if !p.at(lexer.EOF) {
			p.advance()
		}
	}
}

// precedence returns the precedence of the current token as a binary operator
func (p *parser) precedence() int {
	switch p.current().Type {
	case lexer.OR_OR:
		return 1
	case lexer.AND_AND:
		return 2
	case lexer.EQ_EQ, lexer.NOT_EQ:
		return 3
	case lexer.LT, lexer.LT_EQ, lexer.GT, lexer.GT_EQ:
		return 4
	case lexer.PLUS, lexer.MINUS:
		return 5
	case lexer.MULTIPLY, lexer.DIVIDE, lexer.MODULO:
		return 6
	default:
		return 0 // Not a binary operator
	}
}

// at checks if current token is of given type
func (p *parser) at(typ lexer.TokenType) bool {
	return p.current().Type == typ
}

// current returns the current token
func (p *parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		// Return EOF token if we're past the end
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token
func (p *parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// start emits an Open event with the given node kind and returns it for matching close
func (p *parser) start(kind NodeKind) NodeKind {
	p.events = append(p.events, Event{
		Kind: EventOpen,
		Data: uint32(kind),
	})
	return kind
}

// finish emits a Close event with the given node kind
func (p *parser) finish(kind NodeKind) {
	p.events = append(p.events, Event{
		Kind: EventClose,
		Data: uint32(kind),
	})
}

// token emits a Token event and advances
func (p *parser) token() {
	p.events = append(p.events, Event{
		Kind: EventToken,
		Data: uint32(p.pos),
	})
	p.advance()
}

// expect checks for expected token and reports error if not found
func (p *parser) expect(expected lexer.TokenType, context string) bool {
	if p.at(expected) {
		p.token()
		return true
	}
	p.errorExpected(expected, context)
	return false
}

// errorExpected reports an error for missing expected token
func (p *parser) errorExpected(expected lexer.TokenType, context string) {
	current := p.current()

	err := ParseError{
		Position: current.Position,
		Message:  "missing " + tokenName(expected),
		Context:  context,
		Expected: []lexer.TokenType{expected},
		Got:      current.Type,
	}

	// Add helpful suggestions based on context
	switch expected {
	case lexer.RPAREN:
		err.Suggestion = "Add ')' to close the " + context
		err.Example = "fun greet(name) {}"
	case lexer.RBRACE:
		err.Suggestion = "Add '}' to close the " + context
		err.Example = "fun greet() { echo \"hello\" }"
	case lexer.LBRACE:
		err.Suggestion = "Add '{' to start the function body"
		err.Example = "fun greet() {}"
	case lexer.IDENTIFIER:
		if context == "function declaration" {
			err.Suggestion = "Add a function name after 'fun'"
			err.Example = "fun greet() {}"
		} else if context == "parameter" {
			err.Suggestion = "Add a parameter name"
			err.Example = "fun greet(name) {}"
		}
	}

	p.errors = append(p.errors, err)
}

// errorUnexpected reports an error for unexpected token
func (p *parser) errorUnexpected(context string) {
	current := p.current()

	err := ParseError{
		Position: current.Position,
		Message:  "unexpected " + tokenName(current.Type),
		Context:  context,
		Got:      current.Type,
	}

	p.errors = append(p.errors, err)
}

// isSyncToken checks if current token is a synchronization point
func (p *parser) isSyncToken() bool {
	switch p.current().Type {
	case lexer.RBRACE, // End of block
		lexer.SEMICOLON, // Statement terminator
		lexer.FUN,       // Start of new function
		lexer.EOF:       // End of file
		return true
	}

	// Newline can be a sync point in some contexts
	// For now, we'll rely on explicit tokens
	return false
}

// recover skips tokens until we reach a synchronization point
// This allows the parser to continue after errors and report multiple issues
func (p *parser) recover() {
	for !p.isSyncToken() && !p.at(lexer.EOF) {
		p.advance()
	}
}
