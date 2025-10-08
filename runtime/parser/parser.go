package parser

import (
	"fmt"
	"time"

	"github.com/aledsdavies/opal/core/types"
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
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("file_loop_iteration", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Skip newlines at top level
		if p.at(lexer.NEWLINE) {
			p.advance()
			continue
		}

		if p.at(lexer.FUN) {
			p.function()
		} else if p.at(lexer.VAR) {
			p.varDecl()
		} else if p.at(lexer.AT) {
			// Decorator at top level (script mode)
			p.decorator()
		} else if p.at(lexer.IDENTIFIER) {
			// Shell command at top level
			p.shellCommand()
		} else {
			// Unknown token, skip for now
			p.advance()
		}

		// INVARIANT: Parser must make progress in each iteration
		if p.pos == prevPos && !p.at(lexer.EOF) {
			panic(fmt.Sprintf("parser stuck in file() at pos %d - no progress made", p.pos))
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

	// Parse parameter list (optional)
	if p.at(lexer.LPAREN) {
		p.paramList()
	}

	// Parse body: either = expression/shell or block (required)
	if p.at(lexer.EQUALS) {
		p.token() // Consume '='

		// After '=', could be shell command or expression
		if p.at(lexer.IDENTIFIER) {
			// Shell command
			p.shellCommand()
		} else {
			// Expression
			p.expression()
		}
	} else if p.at(lexer.LBRACE) {
		// Block
		p.block()
	} else {
		// Missing function body - report error
		p.errorExpected(lexer.LBRACE, "function body")
	}

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
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("block_loop_iteration", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		p.statement()

		// INVARIANT: Parser must make progress in each iteration
		// If statement() didn't advance, we need to force progress to avoid infinite loop
		if p.pos == prevPos && !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("block_force_progress", fmt.Sprintf("pos: %d, forcing advance on %v", p.pos, p.current().Type))
			}
			// Force progress by advancing past the problematic token
			p.advance()
		}
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
	// Skip newlines (statement separators)
	for p.at(lexer.NEWLINE) {
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("statement_skip_newline", fmt.Sprintf("pos: %d", p.pos))
		}
		p.advance()
	}

	if p.at(lexer.VAR) {
		p.varDecl()
	} else if p.at(lexer.IDENTIFIER) {
		// Shell command
		p.shellCommand()
	} else if !p.at(lexer.RBRACE) && !p.at(lexer.EOF) {
		// Unknown statement - error recovery
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("error_recovery_start", fmt.Sprintf("pos: %d, unexpected %v in statement", p.pos, p.current().Type))
		}

		p.errorUnexpected("statement")
		p.recover()

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("recovery_sync_found", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Consume separator to guarantee progress
		if p.at(lexer.NEWLINE) || p.at(lexer.SEMICOLON) {
			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("consumed_separator", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
			}
			p.advance()
		}
	}
}

// shellCommand parses a shell command and its arguments
// Uses HasSpaceBefore to determine argument boundaries
// Consumes tokens until a shell operator (&&, ||, |) or statement boundary
func (p *parser) shellCommand() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_shell_command", "parsing shell command")
	}

	kind := p.start(NodeShellCommand)

	// Parse shell arguments until we hit an operator or boundary
	for !p.isShellOperator() && !p.isStatementBoundary() {
		prevPos := p.pos

		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("shell_arg_start", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}

		// Parse a single shell argument (may be multiple tokens without spaces)
		p.shellArg()

		// INVARIANT: must make progress
		if p.pos == prevPos {
			panic(fmt.Sprintf("parser stuck in shellCommand() at pos %d, token: %v", p.pos, p.current().Type))
		}
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_shell_command", "shell command complete")
	}

	// If we stopped at a shell operator, consume it and parse next command
	if p.isShellOperator() {
		p.token() // Consume operator (&&, ||, |)

		// Parse next command after operator
		if !p.isStatementBoundary() && !p.at(lexer.EOF) {
			p.shellCommand()
		}
	}
}

// shellArg parses a single shell argument
// Consumes tokens until we hit a space (HasSpaceBefore on next token)
// or a shell operator or statement boundary
// PRECONDITION: Must NOT be called when at operator or boundary (caller's responsibility)
func (p *parser) shellArg() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_shell_arg", "parsing shell argument")
	}

	// PRECONDITION CHECK: shellArg should never be called at operator/boundary
	if p.isShellOperator() || p.isStatementBoundary() {
		panic(fmt.Sprintf("BUG: shellArg() called at operator/boundary, pos: %d, token: %v",
			p.pos, p.current().Type))
	}

	kind := p.start(NodeShellArg)

	// Check if first token is a STRING that needs interpolation
	if p.at(lexer.STRING) && p.stringNeedsInterpolation() {
		// Parse string with interpolation
		p.stringLiteral()
	} else {
		// Consume first token (guaranteed to exist due to precondition)
		if p.config.debug >= DebugDetailed {
			p.recordDebugEvent("shell_arg_first_token", fmt.Sprintf("pos: %d, token: %v", p.pos, p.current().Type))
		}
		p.token()

		// Consume additional tokens that form this argument (no space between them)
		// Loop continues while: not at operator, not at boundary, and no space before current token
		for !p.isShellOperator() && !p.isStatementBoundary() && !p.current().HasSpaceBefore {
			prevPos := p.pos

			if p.config.debug >= DebugDetailed {
				p.recordDebugEvent("shell_arg_continue_token", fmt.Sprintf("pos: %d, token: %v, hasSpace: %v",
					p.pos, p.current().Type, p.current().HasSpaceBefore))
			}

			p.token() // Consume token as part of this argument

			// INVARIANT: p.token() calls p.advance() which MUST increment p.pos
			if p.pos <= prevPos {
				panic(fmt.Sprintf("parser stuck in shellArg() at pos %d (was %d), token: %v - advance() failed to increment position",
					p.pos, prevPos, p.current().Type))
			}
		}
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_shell_arg", "shell argument complete")
	}
}

// isShellOperator checks if current token is a shell operator that splits commands
func (p *parser) isShellOperator() bool {
	return p.at(lexer.AND_AND) || // &&
		p.at(lexer.OR_OR) || // ||
		p.at(lexer.PIPE) // |
}

// isStatementBoundary checks if current token ends a statement
func (p *parser) isStatementBoundary() bool {
	return p.at(lexer.NEWLINE) ||
		p.at(lexer.SEMICOLON) ||
		p.at(lexer.RBRACE) ||
		p.at(lexer.EOF)
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
	case p.at(lexer.INTEGER), p.at(lexer.FLOAT), p.at(lexer.BOOLEAN):
		// Literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)

	case p.at(lexer.STRING):
		// String - check if it needs interpolation
		p.stringLiteral()

	case p.at(lexer.AT):
		// Decorator: @var.name, @env.HOME
		p.decorator()

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

// decorator parses @identifier.property
// Only creates decorator node if identifier is registered
func (p *parser) decorator() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_decorator", "parsing decorator")
	}

	// Look ahead to check if this is a registered decorator
	// (we need to peek before consuming @ to decide if it's a decorator)
	atPos := p.pos
	p.advance() // Move past @

	// Check if next token is an identifier or VAR keyword
	if !p.at(lexer.IDENTIFIER) && !p.at(lexer.VAR) {
		// Not a decorator, treat @ as literal
		// TODO: This needs better handling for literal @ in strings
		return
	}

	// Get the decorator name
	decoratorName := string(p.current().Text)

	// Check if it's a registered decorator
	if !types.Global().IsRegistered(decoratorName) {
		// Not a registered decorator, treat @ as literal
		// Don't consume the identifier, let it be parsed normally
		return
	}

	// Get the schema for validation
	schema, hasSchema := types.Global().GetSchema(decoratorName)

	// It's a registered decorator, parse it
	// Reset position to @ and start the node
	p.pos = atPos
	kind := p.start(NodeDecorator)

	// Consume @ token (emit it)
	p.token()

	// Consume decorator name (IDENTIFIER or VAR keyword)
	p.token()

	// Track if primary parameter was provided via dot syntax
	hasPrimaryViaDot := false

	// Parse property access: .property
	if p.at(lexer.DOT) {
		p.token() // Consume DOT
		if p.at(lexer.IDENTIFIER) {
			p.token() // Consume property name
			hasPrimaryViaDot = true
		}
	}

	// Track provided parameters for validation
	providedParams := make(map[string]bool)
	if hasPrimaryViaDot && hasSchema && schema.PrimaryParameter != "" {
		providedParams[schema.PrimaryParameter] = true
	}

	// Parse parameters: (param1=value1, param2=value2)
	if p.at(lexer.LPAREN) {
		p.decoratorParamsWithValidation(decoratorName, schema, providedParams)
	}

	// Validate required parameters
	if hasSchema {
		p.validateRequiredParameters(decoratorName, schema, providedParams)
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_decorator", "decorator complete")
	}
}

// decoratorParamsWithValidation parses and validates decorator parameters
func (p *parser) decoratorParamsWithValidation(decoratorName string, schema types.DecoratorSchema, providedParams map[string]bool) {
	if !p.at(lexer.LPAREN) {
		return
	}

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_decorator_params", fmt.Sprintf("decorator=%s, schema_params=%d", decoratorName, len(schema.Parameters)))
	}

	paramListKind := p.start(NodeParamList)
	p.token() // Consume (

	// Parse parameters until we hit )
	for !p.at(lexer.RPAREN) && !p.at(lexer.EOF) {
		paramKind := p.start(NodeParam)

		// Parse parameter name
		if !p.at(lexer.IDENTIFIER) {
			p.errorExpected(lexer.IDENTIFIER, "parameter name")
			p.finish(paramKind)
			break
		}

		paramNameToken := p.current()
		paramName := string(paramNameToken.Text)
		p.token() // Consume parameter name

		// Expect =
		if !p.at(lexer.EQUALS) {
			p.errorExpected(lexer.EQUALS, "'=' after parameter name")
			p.finish(paramKind)
			break
		}
		p.token() // Consume =

		// Check if parameter exists in schema
		paramSchema, paramExists := schema.Parameters[paramName]
		if !paramExists {
			// Unknown parameter
			p.errorWithDetails(
				fmt.Sprintf("unknown parameter '%s' for @%s", paramName, decoratorName),
				"decorator parameter",
				p.validParametersSuggestion(schema),
			)
		} else {
			// Mark parameter as provided
			providedParams[paramName] = true
		}

		// Parse and validate parameter value type
		valueToken := p.current()
		if p.at(lexer.STRING) || p.at(lexer.INTEGER) || p.at(lexer.FLOAT) ||
			p.at(lexer.BOOLEAN) || p.at(lexer.IDENTIFIER) {

			// Validate type if parameter exists in schema
			if paramExists {
				p.validateParameterType(paramName, paramSchema, valueToken)
			}

			p.token() // Consume value
		} else {
			p.errorUnexpected("parameter value")
			p.finish(paramKind)
			break
		}

		p.finish(paramKind)

		// Check for comma (more parameters)
		if p.at(lexer.COMMA) {
			p.token() // Consume comma
		} else if !p.at(lexer.RPAREN) {
			p.errorUnexpected("',' or ')'")
			break
		}
	}

	if !p.at(lexer.RPAREN) {
		p.errorExpected(lexer.RPAREN, "')'")
		p.finish(paramListKind)
		return
	}
	p.token() // Consume )
	p.finish(paramListKind)
}

// validateParameterType checks if the token type matches the expected parameter type
func (p *parser) validateParameterType(paramName string, paramSchema types.ParamSchema, valueToken lexer.Token) {
	expectedType := paramSchema.Type
	actualType := p.tokenToParamType(valueToken.Type)

	if p.config.debug >= DebugDetailed {
		p.recordDebugEvent("validate_param_type",
			fmt.Sprintf("param=%s, expected=%s, actual=%s, match=%v",
				paramName, expectedType, actualType, actualType == expectedType))
	}

	if actualType != expectedType {
		p.errorWithDetails(
			fmt.Sprintf("parameter '%s' expects %s, got %s", paramName, expectedType, actualType),
			"decorator parameter",
			fmt.Sprintf("Use a %s value like %s", expectedType, p.exampleForType(expectedType)),
		)
	}
}

// tokenToParamType converts a lexer token type to a ParamType
func (p *parser) tokenToParamType(tokType lexer.TokenType) types.ParamType {
	switch tokType {
	case lexer.STRING:
		return types.TypeString
	case lexer.INTEGER:
		return types.TypeInt
	case lexer.FLOAT:
		return types.TypeFloat
	case lexer.BOOLEAN:
		return types.TypeBool
	case lexer.IDENTIFIER:
		// Identifiers could be variable references, for now treat as string
		return types.TypeString
	default:
		return types.TypeString
	}
}

// exampleForType returns an example value for a given type
func (p *parser) exampleForType(typ types.ParamType) string {
	switch typ {
	case types.TypeString:
		return "\"value\""
	case types.TypeInt:
		return "42"
	case types.TypeFloat:
		return "3.14"
	case types.TypeBool:
		return "true"
	default:
		return "value"
	}
}

// validateRequiredParameters checks that all required parameters were provided
func (p *parser) validateRequiredParameters(decoratorName string, schema types.DecoratorSchema, providedParams map[string]bool) {
	for paramName, paramSchema := range schema.Parameters {
		if paramSchema.Required && !providedParams[paramName] {
			suggestion := fmt.Sprintf("Provide %s parameter", paramName)
			if paramName == schema.PrimaryParameter {
				// Use first example from schema if available, otherwise generic
				exampleValue := "VALUE"
				if len(paramSchema.Examples) > 0 && paramSchema.Examples[0] != "" {
					exampleValue = paramSchema.Examples[0]
				}
				suggestion = fmt.Sprintf("Use dot syntax like @%s.%s or provide %s=\"%s\"", decoratorName, exampleValue, paramName, exampleValue)
			}

			p.errorWithDetails(
				fmt.Sprintf("missing required parameter '%s'", paramName),
				"decorator parameters",
				suggestion,
			)
		}
	}
}

// validParametersSuggestion returns a suggestion listing valid parameters
func (p *parser) validParametersSuggestion(schema types.DecoratorSchema) string {
	if len(schema.Parameters) == 0 {
		return "This decorator accepts no parameters"
	}

	params := make([]string, 0, len(schema.Parameters))
	for name := range schema.Parameters {
		params = append(params, name)
	}

	// Simple alphabetical sort
	for i := 0; i < len(params); i++ {
		for j := i + 1; j < len(params); j++ {
			if params[i] > params[j] {
				params[i], params[j] = params[j], params[i]
			}
		}
	}

	result := "Valid parameters: "
	for i, param := range params {
		if i > 0 {
			result += ", "
		}
		result += param
	}
	return result
}

// errorWithDetails creates a parse error with full context
func (p *parser) errorWithDetails(message, context, suggestion string) {
	tok := p.current()
	p.errors = append(p.errors, ParseError{
		Position:   tok.Position,
		Message:    message,
		Context:    context,
		Got:        tok.Type,
		Suggestion: suggestion,
	})
}

// stringNeedsInterpolation checks if the current STRING token needs interpolation
func (p *parser) stringNeedsInterpolation() bool {
	tok := p.current()

	if len(tok.Text) == 0 {
		return false
	}

	quoteType := tok.Text[0]

	// Single quotes never interpolate
	if quoteType == '\'' {
		return false
	}

	// Extract content without quotes
	content := tok.Text
	if len(content) >= 2 {
		content = content[1 : len(content)-1]
	} else {
		return false
	}

	// Tokenize and check if there are multiple parts or decorator parts
	parts := TokenizeString(content, quoteType)

	// Needs interpolation if there are multiple parts or if the single part is a decorator
	return len(parts) > 1 || (len(parts) == 1 && !parts[0].IsLiteral)
}

// stringLiteral parses a string literal, checking for interpolation
func (p *parser) stringLiteral() {
	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("enter_string_literal", "parsing string")
	}

	tok := p.current()

	// Check quote type - single quotes have no interpolation
	if len(tok.Text) == 0 {
		// Empty string token, treat as simple literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	quoteType := tok.Text[0]

	// Single quotes never interpolate
	if quoteType == '\'' {
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	// Extract content without quotes
	content := tok.Text
	if len(content) >= 2 {
		content = content[1 : len(content)-1] // Remove surrounding quotes
	} else {
		// Malformed string, treat as simple literal
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	// Tokenize the string content
	parts := TokenizeString(content, quoteType)

	// If no parts or only one literal part, treat as simple literal
	if len(parts) == 0 || (len(parts) == 1 && parts[0].IsLiteral) {
		kind := p.start(NodeLiteral)
		p.token()
		p.finish(kind)
		return
	}

	// Has interpolation - create interpolated string node
	kind := p.start(NodeInterpolatedString)
	p.token() // Consume the STRING token

	// Create nodes for each part
	for _, part := range parts {
		partKind := p.start(NodeStringPart)

		if part.IsLiteral {
			// Literal part - no additional nodes needed
			// The part's byte offsets are stored in the StringPart
		} else {
			// Decorator part - create decorator node
			decoratorKind := p.start(NodeDecorator)
			// Note: We don't consume tokens here because the decorator is embedded in the string
			// The decorator name and property are in the string content at part.Start:part.End
			p.finish(decoratorKind)
		}

		p.finish(partKind)
	}

	p.finish(kind)

	if p.config.debug >= DebugPaths {
		p.recordDebugEvent("exit_string_literal", "string complete")
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
