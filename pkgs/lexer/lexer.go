package lexer

import (
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

// LexerMode represents the current parsing context
type LexerMode int

const (
	LanguageMode LexerMode = iota // Standard parsing mode
	ShellMode                     // Shell command parsing mode
)

// Pool for working buffers only
var workBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 256)
	},
}

// Lexer tokenizes Devcmd source code with aggressive optimizations
type Lexer struct {
	input     []byte    // Use []byte for faster operations
	position  int       // current position in input (points to current char)
	readPos   int       // current reading position (after current char)
	ch        byte      // current char under examination (byte for ASCII fast path)
	line      int       // current line number
	column    int       // current column number
	afterAt   bool      // Track if we're immediately after @
	lastToken TokenType // Track the last token type for context
	mode      LexerMode // Current lexer mode
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
		mode:   LanguageMode,
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

// NextToken returns the next token from the input - simplified version
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

// Fast token lexing with unified logic
func (l *Lexer) lexTokenFast() Token {
	l.skipWhitespaceFast()

	tok := Token{
		Line:   l.line,
		Column: l.column,
	}
	start := l.position

	// Handle shell mode differently
	if l.mode == ShellMode {
		return l.lexShellModeFast(start)
	}

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Value = ""
		return tok
	case '\n':
		tok.Type = NEWLINE
		tok.Value = "\n"
		l.readChar()
	case '@':
		tok.Type = AT
		tok.Value = "@"
		tok.Semantic = SemOperator
		tok.Scope = "punctuation.definition.decorator.devcmd"
		l.readChar()
	case ':':
		tok.Type = COLON
		tok.Value = ":"
		// Switch to shell mode after colon in language constructs
		if l.lastToken == IDENTIFIER {
			l.mode = ShellMode
		}
		l.readChar()
	case '=':
		tok.Type = EQUALS
		tok.Value = "="
		l.readChar()
	case ',':
		tok.Type = COMMA
		tok.Value = ","
		l.readChar()
	case '(':
		tok.Type = LPAREN
		tok.Value = "("
		l.readChar()
	case ')':
		tok.Type = RPAREN
		tok.Value = ")"
		l.readChar()
	case '{':
		tok.Type = LBRACE
		tok.Value = "{"
		// Switch to shell mode inside braces
		l.mode = ShellMode
		l.readChar()
	case '}':
		tok.Type = RBRACE
		tok.Value = "}"
		// Switch back to language mode after closing brace
		l.mode = LanguageMode
		l.readChar()
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
			// For any other character, tokenize it as an identifier (shell symbol)
			return l.lexSingleCharFast(start)
		}
	}

	tok.EndLine = l.line
	tok.EndColumn = l.column
	return tok
}

// lexShellModeFast handles shell command tokenization
func (l *Lexer) lexShellModeFast(start int) Token {
	// In shell mode, we primarily emit shell text until we hit structural tokens
	switch l.ch {
	case 0:
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}
	case '\n':
		l.mode = LanguageMode // Return to language mode on newline
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok
	case '}':
		l.mode = LanguageMode // Return to language mode
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok
	case '@':
		// Always create AT token, let parser decide semantics
		l.mode = LanguageMode
		tok := Token{Type: AT, Value: "@", Line: l.line, Column: l.column, Semantic: SemOperator}
		l.readChar()
		tok.EndLine = l.line
		tok.EndColumn = l.column
		return tok
	case '\\':
		if l.peekChar() == '\n' {
			return l.lexLineContinuationFast(start)
		}
		fallthrough
	default:
		return l.lexShellTextFast(start)
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

// afterAtOrWhitespace checks if we're after @ or significant whitespace (performance optimized)
func (l *Lexer) afterAtOrWhitespace() bool {
	// Quick check - if we just saw @, definitely parse as decorator
	if l.lastToken == AT {
		return true
	}

	// Otherwise be conservative - only if we have clear decorator context
	return l.position > 0 && isWhitespace[l.input[l.position-1]]
}

// isLikelyDecoratorParam is a fast heuristic for decorator parameters
func (l *Lexer) isLikelyDecoratorParam() bool {
	// Only if we recently saw a decorator identifier
	return l.lastToken == IDENTIFIER && l.afterAt
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

// Fast shell text lexing
func (l *Lexer) lexShellTextFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	for l.ch != 0 {
		// Stop at @ symbol (let parser decide if it's a decorator)
		if l.ch == '@' {
			break
		}

		// Stop at structural tokens that end shell commands
		if l.ch == '\n' || l.ch == '}' {
			break
		}

		// Stop at line continuation
		if l.ch == '\\' && l.peekChar() == '\n' {
			break
		}

		l.readChar()
	}

	// Don't return empty shell text tokens
	if l.position == start {
		// If we haven't consumed any characters, advance by one to avoid infinite loop
		l.readChar()
	}

	return Token{
		Type:      IDENTIFIER, // Changed from SHELL_TEXT to IDENTIFIER
		Value:     string(l.input[start:l.position]),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemCommand, // Changed from SemShellText to SemCommand
		Scope:     "source.shell.embedded.devcmd",
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

// Fast space skipping (preserves newlines)
func (l *Lexer) skipSpacesFast() {
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}
}

// Optimized character reading with byte operations
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // represents EOF
		l.position = l.readPos // FIX: Update position even at EOF
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
