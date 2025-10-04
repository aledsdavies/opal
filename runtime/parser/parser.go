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
	if config.telemetry >= TelemetryTiming {
		startTotal = time.Now()
		telemetry = &ParseTelemetry{}
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

	if config.telemetry >= TelemetryTiming {
		telemetry.LexTime = time.Since(startLex)
		telemetry.TokenCount = len(tokens)
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

	if config.telemetry >= TelemetryTiming {
		telemetry.ParseTime = time.Since(startParse)
		telemetry.TotalTime = time.Since(startTotal)
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
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
	if config.telemetry >= TelemetryTiming {
		startTotal = time.Now()
		telemetry = &ParseTelemetry{}
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

	if config.telemetry >= TelemetryTiming {
		telemetry.ParseTime = time.Since(startParse)
		telemetry.TotalTime = time.Since(startTotal)
		telemetry.EventCount = len(p.events)
		telemetry.ErrorCount = len(p.errors)
		telemetry.TokenCount = len(tokens)
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

	// Consume '('
	if p.at(lexer.LPAREN) {
		p.token()
	}

	// Parse parameters
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		p.param()

		// TODO: Handle comma-separated parameters in next iteration
		// For now, just parse one parameter
		break
	}

	// Consume ')'
	if p.at(lexer.RPAREN) {
		p.token()
	}

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

	// TODO: Parse default value in next iteration

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

// block parses a block: { statements }
func (p *parser) block() {
	if p.config.debug > DebugOff {
		p.recordDebugEvent("enter_block", "parsing block")
	}

	kind := p.start(NodeBlock)

	// Consume '{'
	if p.at(lexer.LBRACE) {
		p.token()
	}

	// TODO: Parse statements (for now just skip to '}')

	// Consume '}'
	if p.at(lexer.RBRACE) {
		p.token()
	}

	p.finish(kind)

	if p.config.debug > DebugOff {
		p.recordDebugEvent("exit_block", "block complete")
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
