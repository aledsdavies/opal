package lexer

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// Character classification lookup tables for 3-5x faster operations
var (
	isWhitespace [256]bool
	isLetter     [256]bool
	isDigit      [256]bool
	isIdentStart [256]bool
	isIdentPart  [256]bool
	isHexDigit   [256]bool
)

func init() {
	for i := 0; i < 256; i++ {
		ch := byte(i)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f'
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_' || ch >= 0x80
		isDigit[i] = '0' <= ch && ch <= '9'
		isIdentStart[i] = isLetter[i] || ch == '_' || ch >= 0x80
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		isHexDigit[i] = isDigit[i] || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
	}
}

// LexerMode represents the current parsing context in Devcmd
type LexerMode int

const (
	// LanguageMode: Top-level parsing and decorator parsing
	// Recognizes: var, watch, stop, @decorators, :, =, {, }, (, ), literals
	// Examples:
	//   var PORT = 8080;
	//   build: npm run build        # : → switches to CommandMode
	//   @timeout(30s) { ... }       # stays in LanguageMode for decorator parsing
	LanguageMode LexerMode = iota

	// CommandMode: Inside command bodies (after : or inside {})
	// Recognizes: shell text as complete units + decorators
	// Examples:
	//   build: echo hello; npm test     # "echo hello; npm test" as single shell unit
	//   deploy: { npm build }           # "npm build" as shell unit inside {}
	//   server: @timeout(30s) { ... }   # @ switches back to LanguageMode
	CommandMode
)

// Pool for working buffers only
var workBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 256)
	},
}

// Lexer tokenizes Devcmd source code with mode-based parsing
//
// Mode Rules:
// 1. LanguageMode: Parse language constructs and decorators
// 2. CommandMode: Parse shell text while recognizing decorators
//
// Syntax Sugar Handling:
// - Simple commands: "cmd: shell" automatically becomes "cmd: { shell }"
// - Decorators: NEVER get sugar, always require explicit braces
//
// Transition Rules:
// - LanguageMode → CommandMode: after : (simple) or { (block)
// - CommandMode → LanguageMode: on @ (decorator), } (end block), \n (end simple)
type Lexer struct {
	input     []byte    // Use []byte for faster operations
	position  int       // current position in input (points to current char)
	readPos   int       // current reading position (after current char)
	ch        byte      // current char under examination (byte for ASCII fast path)
	line      int       // current line number
	column    int       // current column number
	afterAt   bool      // Track if we're immediately after @ for decorator parsing
	lastToken TokenType // Track the last token type for context
	mode      LexerMode // Current lexer mode (LanguageMode or CommandMode)
}

// estimateTokenCount predicts slice capacity to avoid slice growth
func estimateTokenCount(inputSize int) int {
	// Devcmd typically has ~6-8 tokens per line, lines average ~40 chars
	// Add 20% buffer for complex decorators
	estimate := (inputSize / 6) * 12 / 10
	if estimate < 16 {
		estimate = 16
	}
	return estimate
}

// New creates a new lexer instance with optimized initialization
func New(input string) *Lexer {
	l := &Lexer{
		input: []byte(input), // Direct conversion for better performance
		line:  1,
		column: 0,
		mode:   LanguageMode, // Always start in LanguageMode
	}
	l.readChar() // initialize first character
	return l
}

// setMode allows changing the lexer mode for testing
func (l *Lexer) setMode(mode LexerMode) {
	l.mode = mode
}

// TokenizeToSlice tokenizes to pre-allocated slice for maximum performance
func (l *Lexer) TokenizeToSlice() []Token {
	estimatedTokens := estimateTokenCount(len(l.input))
	result := make([]Token, 0, estimatedTokens)

	for {
		tok := l.NextToken()
		result = append(result, tok)
		if tok.Type == EOF {
			break
		}
	}

	return result
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	tok := l.lexTokenFast()

	// Update context tracking for decorator names
	if tok.Type == AT {
		l.afterAt = true
	} else if l.afterAt && tok.Type == IDENTIFIER {
		l.afterAt = false // Reset after processing decorator name
	}

	// Track last token for context
	l.lastToken = tok.Type

	return tok
}

// lexTokenFast performs fast token lexing with mode-aware logic
func (l *Lexer) lexTokenFast() Token {
	l.skipWhitespaceFast()

	start := l.position

	// Handle different modes with explicit rules
	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode(start)
	case CommandMode:
		return l.lexCommandMode(start)
	default:
		return l.lexLanguageMode(start) // Default fallback
	}
}

// lexLanguageMode handles top-level language constructs and decorator parsing
// Recognizes: var, watch, stop, @decorators, :, =, {, }, (, ), literals
func (l *Lexer) lexLanguageMode(start int) Token {
	switch l.ch {
	case 0:
		return Token{
			Type:      EOF,
			Value:     "",
			Line:      l.line,
			Column:    l.column,
			EndLine:   l.line,
			EndColumn: l.column,
		}

	case '\n':
		tok := Token{
			Type:      NEWLINE,
			Value:     "\n",
			Line:      l.line,
			Column:    l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '@':
		// Always parse decorators in LanguageMode
		// Examples: @timeout, @var, @parallel, @sh
		tok := Token{
			Type:      AT,
			Value:     "@",
			Line:      l.line,
			Column:    l.column,
			Semantic:  SemOperator,
			Scope:     "punctuation.definition.decorator.devcmd",
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case ':':
		// SYNTAX SUGAR DECISION POINT:
		// After colon, determine if we switch to CommandMode
		tok := Token{
			Type:    COLON,
			Value:   ":",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()

		// Check if we should switch to CommandMode for simple commands
		// Rules: switch unless we see { (explicit block) or @ (decorator)
		if l.shouldSwitchToCommandMode() {
			l.mode = CommandMode
		}

		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '=':
		tok := Token{
			Type:    EQUALS,
			Value:   "=",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case ',':
		tok := Token{
			Type:    COMMA,
			Value:   ",",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '(':
		tok := Token{
			Type:    LPAREN,
			Value:   "(",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case ')':
		tok := Token{
			Type:    RPAREN,
			Value:   ")",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '{':
		// Explicit block start - switch to CommandMode
		// No syntax sugar here - explicit braces mean explicit intent
		tok := Token{
			Type:    LBRACE,
			Value:   "{",
			Line:    l.line,
			Column:  l.column,
		}
		l.mode = CommandMode
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '}':
		// Block end - should only happen in error cases in LanguageMode
		tok := Token{
			Type:    RBRACE,
			Value:   "}",
			Line:    l.line,
			Column:  l.column,
		}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '"':
		return l.lexStringFast('"', DoubleQuoted, start)
	case '\'':
		return l.lexStringFast('\'', SingleQuoted, start)
	case '`':
		return l.lexStringFast('`', Backtick, start)

	case '#':
		return l.lexCommentFast(start)

	case '/':
		if l.peekChar() == '*' {
			return l.lexMultilineCommentFast(start)
		}
		fallthrough

	case '\\':
		if l.peekChar() == '\n' {
			return l.lexLineContinuationFast(start)
		}
		fallthrough

	default:
		if isLetter[l.ch] {
			return l.lexIdentifierOrKeywordFast(start)
		} else if isDigit[l.ch] || l.ch == '-' {
			return l.lexNumberOrDurationFast(start)
		} else {
			// Single character tokens
			return l.lexSingleCharFast(start)
		}
	}
}

// lexCommandMode handles shell text and decorators inside command bodies
// Recognizes: shell text as complete units + decorators (switches back to LanguageMode)
func (l *Lexer) lexCommandMode(start int) Token {
	switch l.ch {
	case 0:
		// EOF in CommandMode - switch back to LanguageMode
		l.mode = LanguageMode
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}

	case '\n':
		// Newline in CommandMode - end of simple command, back to LanguageMode
		// This handles the syntax sugar: "cmd: shell" where \n ends the command
		l.mode = LanguageMode
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '}':
		// End of explicit block - back to LanguageMode
		l.mode = LanguageMode
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '@':
		// @ in CommandMode - always tokenize separately for parser to handle
		// This allows both decorators and inline @var() usage:
		// deploy: { @parallel { npm build } }  # @parallel is decorator
		// build: echo @var(PORT)               # @var is inline usage
		if l.isDecoratorStart() {
			// Switch to LanguageMode for decorator parsing
			l.mode = LanguageMode
		}
		// Always tokenize @ separately - let parser determine semantics
		tok := Token{Type: AT, Value: "@", Line: l.line, Column: l.column, Semantic: SemOperator}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok

	case '\\':
		if l.peekChar() == '\n' {
			// Line continuation - stay in CommandMode
			return l.lexLineContinuationFast(start)
		}
		// Otherwise treat as shell text
		return l.lexShellTextFast(start)

	case ' ', '\t', '\r', '\f':
		// Skip whitespace in CommandMode and look for actual shell content
		l.skipWhitespaceFast()
		if l.ch == 0 || l.ch == '\n' || l.ch == '}' || (l.ch == '@' && l.isDecoratorStart()) {
			// If we hit a boundary after whitespace, handle it normally
			return l.lexCommandMode(l.position)
		}
		// Otherwise start lexing shell text from current position
		return l.lexShellTextFast(l.position)

	default:
		// Everything else in CommandMode is shell text
		return l.lexShellTextFast(start)
	}
}

// shouldSwitchToCommandMode determines if we should switch to CommandMode after ':'
// This implements the syntax sugar logic:
// - "cmd: shell" → switch to CommandMode (sugar applies)
// - "cmd: { ... }" → don't switch (explicit braces, no sugar)
// - "cmd: @decorator ..." → don't switch (decorator, LanguageMode continues)
func (l *Lexer) shouldSwitchToCommandMode() bool {
	// Save current position to peek ahead
	savedPos := l.position
	savedReadPos := l.readPos
	savedCh := l.ch
	savedLine := l.line
	savedColumn := l.column

	// Skip whitespace to see what follows the colon
	l.skipWhitespaceFast()

	// Don't switch to CommandMode if we see:
	// - '{' (explicit block syntax, no sugar)
	// - '@' (decorator follows, stay in LanguageMode)
	// - '\n' or EOF (empty command)
	shouldSwitch := l.ch != '{' && l.ch != '@' && l.ch != '\n' && l.ch != 0

	// Restore position
	l.position = savedPos
	l.readPos = savedReadPos
	l.ch = savedCh
	l.line = savedLine
	l.column = savedColumn

	return shouldSwitch
}

// isDecoratorStart checks if @ starts a decorator vs inline @var() usage
// This determines whether to switch to LanguageMode for decorator parsing
// Decorators: @timeout, @parallel, @retry, @sh, @watch-files, etc.
// Inline usage: @var(NAME) within shell text
func (l *Lexer) isDecoratorStart() bool {
	// Look ahead to see if this is a decorator name
	if l.peekChar() == 0 || !isLetter[l.peekChar()] {
		return false
	}

	// Save position for lookahead
	savedPos := l.position
	savedReadPos := l.readPos
	savedCh := l.ch

	// Read the identifier after @
	l.readChar() // skip @
	start := l.position
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}
	identifier := string(l.input[start:l.position])

	// Restore position
	l.position = savedPos
	l.readPos = savedReadPos
	l.ch = savedCh

	// @var is inline usage (stays in CommandMode), everything else is a decorator (switches to LanguageMode)
	return identifier != "var"
}

// lexShellTextFast lexes shell command text as complete units
// This preserves shell semantics: "echo hello; npm test" is one command unit
func (l *Lexer) lexShellTextFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	// Don't skip leading whitespace here - it's handled by the caller
	// We want to preserve the exact shell command text

	for l.ch != 0 {
		// Stop at structural boundaries that end shell commands
		if l.ch == '\n' || l.ch == '}' {
			break
		}

		// Stop at ALL @ symbols (both decorators and inline @var usage)
		// This allows proper tokenization of both @timeout and @var
		if l.ch == '@' {
			break
		}

		// Stop at line continuation
		if l.ch == '\\' && l.peekChar() == '\n' {
			break
		}

		l.readChar()
	}

	// Handle case where we didn't consume any characters
	if l.position <= start {
		if l.ch != 0 {
			// Advance by one to avoid infinite loop
			l.readChar()
		}
	}

	value := string(l.input[start:l.position])

	// Only trim trailing whitespace, preserve leading space for accurate shell commands
	value = strings.TrimRight(value, " \t\r\f")

	return Token{
		Type:      IDENTIFIER,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemCommand,
		Scope:     "source.shell.embedded.devcmd",
	}
}

// lexSingleCharFast tokenizes single character symbols
func (l *Lexer) lexSingleCharFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	// Read single character
	char := l.ch
	l.readChar()

	return Token{
		Type:      IDENTIFIER, // Treat as identifier for parser flexibility
		Value:     string(char),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemOperator, // Single chars are typically operators/symbols
		Scope:     "punctuation.other.devcmd",
	}
}

// Fast string lexing with zero-copy optimization when possible
func (l *Lexer) lexStringFast(quote byte, stringType StringType, start int) Token {
	startLine := l.line
	startColumn := l.column

	l.readChar() // skip opening quote

	// Track if we need escape processing
	var escaped []byte
	valueStart := l.position

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			// Copy unescaped portion if this is our first escape
			if escaped == nil {
				escaped = make([]byte, 0, 64) // Reasonable initial capacity
				escaped = append(escaped, l.input[valueStart:l.position]...)
			} else {
				escaped = append(escaped, l.input[valueStart:l.position]...)
			}

			l.readChar()
			if l.ch == 0 {
				break
			}

			escapeBytes := l.handleEscapeFast(stringType)
			escaped = append(escaped, escapeBytes...)
			l.readChar()
			valueStart = l.position
		} else {
			l.readChar()
		}
	}

	var value string
	if escaped != nil {
		// Had escapes - append final portion and convert
		escaped = append(escaped, l.input[valueStart:l.position]...)
		value = string(escaped)
	} else {
		// No escapes - zero-copy slice reference
		value = string(l.input[valueStart:l.position])
	}

	if l.ch == quote {
		l.readChar() // skip closing quote
	}

	return Token{
		Type:       STRING,
		Value:      value,
		Line:       startLine,
		Column:     startColumn,
		EndLine:    l.line,
		EndColumn:  l.column,
		StringType: stringType,
		Raw:        string(l.input[start:l.position]),
		Semantic:   SemString,
		Scope:      getStringScope(stringType),
	}
}

// Fast escape handling with byte operations
func (l *Lexer) handleEscapeFast(stringType StringType) []byte {
	switch stringType {
	case SingleQuoted:
		if l.ch == '\'' {
			return []byte("'")
		}
		return []byte{byte('\\'), l.ch}

	case DoubleQuoted:
		switch l.ch {
		case 'n':
			return []byte{'\n'}
		case 't':
			return []byte{'\t'}
		case 'r':
			return []byte{'\r'}
		case '\\':
			return []byte{'\\'}
		case '"':
			return []byte{'"'}
		default:
			return []byte{byte('\\'), l.ch}
		}

	case Backtick:
		switch l.ch {
		case 'n':
			return []byte{'\n'}
		case 't':
			return []byte{'\t'}
		case 'r':
			return []byte{'\r'}
		case 'b':
			return []byte{'\b'}
		case 'f':
			return []byte{'\f'}
		case 'v':
			return []byte{'\v'}
		case '0':
			return []byte{'\x00'}
		case '\\':
			return []byte{'\\'}
		case '`':
			return []byte{'`'}
		case '"':
			return []byte{'"'}
		case '\'':
			return []byte{'\''}
		case 'x':
			return []byte(l.readHexEscape())
		case 'u':
			if l.peekChar() == '{' {
				return []byte(l.readUnicodeEscape())
			}
			return []byte("\\u")
		default:
			return []byte{byte('\\'), l.ch}
		}
	}

	return []byte{byte('\\'), l.ch}
}

func getStringScope(stringType StringType) string {
	switch stringType {
	case DoubleQuoted:
		return "string.quoted.double.devcmd"
	case SingleQuoted:
		return "string.quoted.single.devcmd"
	case Backtick:
		return "string.quoted.backtick.devcmd"
	default:
		return "string.quoted.devcmd"
	}
}

// Fast identifier/keyword lexing with simplified logic
func (l *Lexer) lexIdentifierOrKeywordFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	l.readIdentifierFast()

	// Zero-copy string slice
	value := string(l.input[start:l.position])

	// Simplified keyword detection
	var tokenType TokenType
	var semantic SemanticTokenType
	var scope string

	// Check if we're after an @ symbol for decorator names
	if l.afterAt {
		tokenType = IDENTIFIER
		semantic = SemDecorator
		scope = "entity.name.function.decorator.devcmd"
	} else if l.isInDecoratorParams() {
		// If we're inside decorator parentheses, this could be a parameter name
		tokenType = IDENTIFIER
		semantic = SemParameter
		scope = "variable.parameter.devcmd"
	} else {
		// Fast keyword detection using length
		switch len(value) {
		case 3:
			if value == "var" {
				tokenType = VAR
				semantic = SemKeyword
				scope = "keyword.control.var.devcmd"
			} else {
				tokenType = IDENTIFIER
				semantic = SemCommand
				scope = "entity.name.function.devcmd"
			}
		case 4:
			if value == "stop" {
				tokenType = STOP
				semantic = SemKeyword
				scope = "keyword.control.stop.devcmd"
			} else {
				tokenType = IDENTIFIER
				semantic = SemCommand
				scope = "entity.name.function.devcmd"
			}
		case 5:
			if value == "watch" {
				tokenType = WATCH
				semantic = SemKeyword
				scope = "keyword.control.watch.devcmd"
			} else {
				tokenType = IDENTIFIER
				semantic = SemCommand
				scope = "entity.name.function.devcmd"
			}
		default:
			tokenType = IDENTIFIER
			semantic = SemCommand
			scope = "entity.name.function.devcmd"
		}
	}

	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  semantic,
		Scope:     scope,
	}
}

// isInDecoratorParams checks if we're inside decorator parentheses (optimized)
func (l *Lexer) isInDecoratorParams() bool {
	// Fast path: if we haven't seen certain tokens recently, skip expensive check
	if l.lastToken != LPAREN && l.lastToken != COMMA && l.lastToken != EQUALS {
		return false
	}

	// Look backwards for decorator context
	pos := l.position - 1
	depth := 0
	maxLookback := 100 // Reasonable limit

	for pos >= 0 && maxLookback > 0 {
		ch := l.input[pos]
		if ch == ')' {
			depth++
		} else if ch == '(' {
			depth--
			if depth < 0 {
				// Found opening paren, look back for @identifier pattern
				pos--
				// Skip whitespace
				for pos >= 0 && (l.input[pos] == ' ' || l.input[pos] == '\t') {
					pos--
				}

				// Look for identifier before @
				identEnd := pos + 1
				for pos >= 0 && (isLetter[l.input[pos]] || isDigit[l.input[pos]] || l.input[pos] == '-' || l.input[pos] == '_') {
					pos--
				}

				// Check if we found @ right before the identifier
				if pos >= 0 && l.input[pos] == '@' && pos + 1 < identEnd {
					return true
				}
				return false
			}
		} else if ch == '\n' || ch == ';' || ch == '{' || ch == '}' {
			// Statement boundary - not in decorator params
			return false
		}
		pos--
		maxLookback--
	}

	return false
}

// Fast number or duration lexing with direct byte operations
func (l *Lexer) lexNumberOrDurationFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	// Handle negative numbers
	if l.ch == '-' {
		l.readChar()
	}

	// Read integer part using lookup table
	for l.ch != 0 && isDigit[l.ch] {
		l.readChar()
	}

	// Handle decimal point
	if l.ch == '.' && l.peekChar() != 0 && isDigit[l.peekChar()] {
		l.readChar() // consume '.'
		for l.ch != 0 && isDigit[l.ch] {
			l.readChar()
		}
	}

	// Check if this is followed by a duration unit
	if l.isDurationUnit() {
		l.readDurationUnit()

		return Token{
			Type:      DURATION,
			Value:     string(l.input[start:l.position]),
			Line:      startLine,
			Column:    startColumn,
			EndLine:   l.line,
			EndColumn: l.column,
			Semantic:  SemNumber, // Durations are semantically similar to numbers
			Scope:     "constant.numeric.duration.devcmd",
		}
	}

	// Not a duration, return as number token
	return Token{
		Type:      NUMBER,
		Value:     string(l.input[start:l.position]),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemNumber,
		Scope:     "constant.numeric.devcmd",
	}
}

// isDurationUnit checks if current position starts a duration unit
func (l *Lexer) isDurationUnit() bool {
	if l.ch == 0 {
		return false
	}

	// Check for common duration units: ns, us, ms, s, m, h
	switch l.ch {
	case 'n':
		return l.peekChar() == 's' // ns
	case 'u': // us (microseconds - ASCII version)
		return l.peekChar() == 's'
	case 'm':
		next := l.peekChar()
		return next == 's' || next == 0 || !isLetter[next] // ms or m (minutes)
	case 's':
		next := l.peekChar()
		return next == 0 || !isLetter[next] // s (seconds)
	case 'h':
		next := l.peekChar()
		return next == 0 || !isLetter[next] // h (hours)
	}

	// Check for Unicode microsecond symbol (μs) as multi-byte sequence
	if l.ch == 0xCE && l.peekChar() == 0xBC { // UTF-8 encoding of μ
		return l.peekCharAt(2) == 's'
	}

	return false
}

// readDurationUnit reads the duration unit (ns, us, ms, s, m, h)
func (l *Lexer) readDurationUnit() {
	switch l.ch {
	case 'n':
		if l.peekChar() == 's' {
			l.readChar() // consume 'n'
			l.readChar() // consume 's'
		}
	case 'u':
		if l.peekChar() == 's' {
			l.readChar() // consume 'u'
			l.readChar() // consume 's'
		}
	case 'm':
		l.readChar() // consume 'm'
		if l.ch == 's' {
			l.readChar() // consume 's' for 'ms'
		}
		// else it's just 'm' for minutes
	case 's', 'h':
		l.readChar() // consume 's' or 'h'
	case 0xCE: // First byte of UTF-8 μ
		if l.peekChar() == 0xBC && l.peekCharAt(2) == 's' {
			l.readChar() // consume first byte of μ
			l.readChar() // consume second byte of μ
			l.readChar() // consume 's'
		}
	}
}

// peekCharAt looks ahead n characters
func (l *Lexer) peekCharAt(n int) byte {
	pos := l.readPos + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// Fast comment lexing
func (l *Lexer) lexCommentFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	return Token{
		Type:      COMMENT,
		Value:     string(l.input[start:l.position]),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Scope:     "comment.line.hash.devcmd",
	}
}

// Fast multiline comment lexing
func (l *Lexer) lexMultilineCommentFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	l.readChar() // skip /
	l.readChar() // skip *

	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume *
			l.readChar() // consume /
			break
		}
		l.readChar()
	}

	return Token{
		Type:      MULTILINE_COMMENT,
		Value:     string(l.input[start:l.position]),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Scope:     "comment.block.devcmd",
	}
}

// Fast line continuation lexing
func (l *Lexer) lexLineContinuationFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	l.readChar() // skip \
	l.readChar() // skip \n

	return Token{
		Type:      LINE_CONT,
		Value:     "\\\n",
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemOperator,
		Scope:     "punctuation.separator.continuation.devcmd",
	}
}

// Fast identifier reading with lookup table
func (l *Lexer) readIdentifierFast() {
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}
}

// Fast whitespace skipping with lookup table
func (l *Lexer) skipWhitespaceFast() {
	for isWhitespace[l.ch] && l.ch != '\n' {
		l.readChar()
	}
}

// Optimized character reading with byte operations
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // represents EOF
		l.position = l.readPos
	} else {
		l.ch = l.input[l.readPos]
		l.position = l.readPos
		l.readPos++
	}

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// Optimized hex escape reading
func (l *Lexer) readHexEscape() string {
	// Read \xHH hex escape sequence
	if !isHexDigit[l.peekChar()] {
		return "\\x"
	}

	l.readChar() // consume 'x'
	hex1 := l.ch
	l.readChar()

	if !isHexDigit[l.ch] {
		// Get buffer from pool for this small operation
		workBuf := workBufPool.Get().([]byte)
		defer workBufPool.Put(workBuf[:0])

		workBuf = append(workBuf, '\\', 'x', hex1)
		return string(workBuf)
	}

	hex2 := l.ch
	value := hexValueFast(hex1)*16 + hexValueFast(hex2)
	return string(rune(value))
}

// Optimized unicode escape reading
func (l *Lexer) readUnicodeEscape() string {
	// Read \u{HHHH} unicode escape sequence
	l.readChar() // consume 'u'
	l.readChar() // consume '{'

	start := l.position
	for l.ch != '}' && l.ch != 0 && isHexDigit[l.ch] {
		l.readChar()
	}

	if l.ch != '}' {
		return "\\u{"
	}

	hexDigits := string(l.input[start:l.position])
	l.readChar() // consume '}'

	// Convert hex to unicode codepoint
	if len(hexDigits) == 0 {
		return "\\u{}"
	}

	var value rune
	for _, ch := range hexDigits {
		value = value*16 + rune(hexValueFast(byte(ch)))
	}

	if !utf8.ValidRune(value) {
		// Get buffer from pool for this operation
		workBuf := workBufPool.Get().([]byte)
		defer workBufPool.Put(workBuf[:0])

		workBuf = append(workBuf, "\\u{"...)
		workBuf = append(workBuf, hexDigits...)
		workBuf = append(workBuf, '}')
		return string(workBuf)
	}

	return string(value)
}

// Fast hex value conversion using lookup table approach
func hexValueFast(ch byte) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 0
}
