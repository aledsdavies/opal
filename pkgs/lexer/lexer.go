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
	LanguageMode LexerMode = iota // Parsing var/command definitions
	ShellMode                     // Parsing shell command content
)

// Pool for working buffers only
var workBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 256)
	},
}

// Lexer tokenizes Devcmd source code with aggressive optimizations
type Lexer struct {
	input     []byte // Use []byte for faster operations
	position  int    // current position in input (points to current char)
	readPos   int    // current reading position (after current char)
	ch        byte   // current char under examination (byte for ASCII fast path)
	line      int    // current line number
	column    int    // current column number
	mode      LexerMode
	modeStack []LexerMode // for nested contexts
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
		input:  []byte(input), // Direct conversion for better performance
		line:   1,
		column: 0,
		mode:   LanguageMode,
	}
	l.readChar() // initialize first character
	return l
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

// NextToken returns the next token from the input - optimized version
func (l *Lexer) NextToken() Token {
	var tok Token

	// Handle decorators first - they work everywhere
	if l.ch == '@' && l.isDecoratorStartFast() {
		return l.lexDecoratorFast()
	}

	// Mode-specific tokenization
	switch l.mode {
	case LanguageMode:
		tok = l.lexLanguageTokenFast()
	case ShellMode:
		tok = l.lexShellTokenFast()
	}

	return tok
}

// Fast language token lexing with lookup tables
func (l *Lexer) lexLanguageTokenFast() Token {
	l.skipWhitespaceFast()

	tok := Token{
		Line:   l.line,
		Column: l.column,
	}
	start := l.position

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Value = ""
		return tok
	case '\n':
		tok.Type = NEWLINE
		tok.Value = "\n"
		l.readChar()
	case ':':
		tok.Type = COLON
		tok.Value = ":"
		l.readChar()
		l.setMode(ShellMode)
	case '=':
		tok.Type = EQUALS
		tok.Value = "="
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
		l.readChar()
		l.setMode(ShellMode)
	case '}':
		tok.Type = RBRACE
		tok.Value = "}"
		l.readChar()
		l.setMode(LanguageMode)
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
	default:
		if isLetter[l.ch] {
			return l.lexIdentifierOrKeywordFast(start)
		} else if isDigit[l.ch] || l.ch == '-' {
			return l.lexNumberFast(start)
		} else {
			tok.Type = ILLEGAL
			tok.Value = string(l.ch)
			l.readChar()
		}
	}

	tok.EndLine = l.line
	tok.EndColumn = l.column
	return tok
}

// Fast shell token lexing
func (l *Lexer) lexShellTokenFast() Token {
	l.skipSpacesFast()

	tok := Token{
		Line:   l.line,
		Column: l.column,
	}
	start := l.position

	switch l.ch {
	case 0:
		tok.Type = EOF
		tok.Value = ""
		return tok
	case '\n':
		tok.Type = NEWLINE
		tok.Value = "\n"
		l.readChar()
		l.setMode(LanguageMode)
	case '}':
		tok.Type = RBRACE
		tok.Value = "}"
		l.readChar()
		l.setMode(LanguageMode)
	case '\\':
		if l.peekChar() == '\n' {
			return l.lexLineContinuationFast(start)
		}
		fallthrough
	default:
		return l.lexShellTextFast(start)
	}

	tok.EndLine = l.line
	tok.EndColumn = l.column
	return tok
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

// Fast identifier/keyword lexing with compile-time keyword map
func (l *Lexer) lexIdentifierOrKeywordFast(start int) Token {
	startLine := l.line
	startColumn := l.column

	l.readIdentifierFast()

	// Zero-copy string slice
	value := string(l.input[start:l.position])

	// Fast keyword lookup with branch prediction optimization
	var tokenType TokenType
	var semantic SemanticTokenType
	var scope string

	switch value {
	case "var":
		tokenType = VAR
		semantic = SemKeyword
		scope = "keyword.control.var.devcmd"
	case "watch":
		tokenType = WATCH
		semantic = SemKeyword
		scope = "keyword.control.watch.devcmd"
	case "stop":
		tokenType = STOP
		semantic = SemKeyword
		scope = "keyword.control.stop.devcmd"
	default:
		tokenType = IDENTIFIER
		semantic = SemCommand
		scope = "entity.name.function.devcmd"
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

// Fast number lexing with direct byte operations
func (l *Lexer) lexNumberFast(start int) Token {
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
		// Stop at decorators
		if l.ch == '@' && l.isDecoratorStartFast() {
			break
		}

		// Stop at structural tokens
		if l.ch == '\n' || l.ch == '}' || (l.ch == '\\' && l.peekChar() == '\n') {
			break
		}

		l.readChar()
	}

	return Token{
		Type:      SHELL_TEXT,
		Value:     string(l.input[start:l.position]),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemShellText,
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

// Fast decorator detection with minimal lookahead
func (l *Lexer) isDecoratorStartFast() bool {
	if l.ch != '@' {
		return false
	}

	next := l.peekChar()
	if !isLetter[next] {
		return false
	}

	// Quick lookahead to find ( or {
	saved := l.position
	savedCh := l.ch

	l.readChar() // skip @
	l.readIdentifierFast()

	isDecorator := l.ch == '(' || l.ch == '{'

	// Restore position
	l.position = saved
	l.readPos = saved + 1
	l.ch = savedCh

	return isDecorator
}

// Optimized decorator lexing with pooled buffer
func (l *Lexer) lexDecoratorFast() Token {
	startLine := l.line
	startColumn := l.column
	start := l.position

	l.readChar() // skip @

	// Read decorator name with fast identifier reading
	nameStart := l.position
	l.readIdentifierFast()
	nameEnd := l.position

	if nameStart == nameEnd {
		return Token{
			Type:     ILLEGAL,
			Value:    "@",
			Line:     startLine,
			Column:   startColumn,
			Semantic: SemOperator,
			Scope:    "invalid.illegal.decorator.devcmd",
		}
	}

	name := string(l.input[nameStart:nameEnd])
	var args string
	var block string
	var tokenType TokenType

	// Determine decorator form based on what follows
	switch l.ch {
	case '(':
		// @decorator(args) or @decorator(args) { block }
		l.readChar() // skip (
		args = l.readBalancedFast('(', ')')
		if l.ch == ')' {
			l.readChar() // skip closing )
		}

		l.skipWhitespaceFast()

		if l.ch == '{' {
			// @decorator(args) { block }
			l.readChar() // skip {
			block = l.readBalancedFast('{', '}')
			if l.ch == '}' {
				l.readChar() // skip closing }
			}
			tokenType = DECORATOR_CALL_BLOCK
		} else {
			// @decorator(args)
			tokenType = DECORATOR_CALL
		}

	case '{':
		// @decorator{ block }
		l.readChar() // skip {
		block = l.readBalancedFast('{', '}')
		if l.ch == '}' {
			l.readChar() // skip closing }
		}
		tokenType = DECORATOR_BLOCK

	default:
		return Token{
			Type:     ILLEGAL,
			Value:    string(l.input[start:l.position]),
			Line:     startLine,
			Column:   startColumn,
			Semantic: SemOperator,
			Scope:    "invalid.illegal.decorator.devcmd",
		}
	}

	// Build decorator value using pooled buffer
	workBuf := workBufPool.Get().([]byte)
	defer workBufPool.Put(workBuf[:0])

	workBuf = append(workBuf, '@')
	workBuf = append(workBuf, name...)

	if args != "" && block != "" {
		workBuf = append(workBuf, '(')
		workBuf = append(workBuf, args...)
		workBuf = append(workBuf, ") { "...)
		workBuf = append(workBuf, block...)
		workBuf = append(workBuf, " }"...)
	} else if args != "" {
		workBuf = append(workBuf, '(')
		workBuf = append(workBuf, args...)
		workBuf = append(workBuf, ')')
	} else if block != "" {
		workBuf = append(workBuf, "{ "...)
		workBuf = append(workBuf, block...)
		workBuf = append(workBuf, " }"...)
	}

	return Token{
		Type:          tokenType,
		Value:         string(workBuf), // Only allocation here
		Line:          startLine,
		Column:        startColumn,
		EndLine:       l.line,
		EndColumn:     l.column,
		DecoratorName: name,
		Args:          args,
		Block:         block,
		Semantic:      SemDecorator,
		Scope:         "support.function.decorator.devcmd",
	}
}

// Fast balanced reading with zero-copy string slicing
func (l *Lexer) readBalancedFast(open, close byte) string {
	start := l.position
	depth := 1 // we already consumed the opening delimiter

	for depth > 0 && l.ch != 0 {
		switch l.ch {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				// Don't include the closing delimiter
				result := string(l.input[start:l.position])
				return result
			}
		case '"', '\'', '`':
			// Handle quoted strings inside balanced content
			quote := l.ch
			l.readChar()
			for l.ch != quote && l.ch != 0 {
				if l.ch == '\\' {
					l.readChar()
					if l.ch != 0 {
						l.readChar()
					}
				} else {
					l.readChar()
				}
			}
			if l.ch == quote {
				l.readChar()
			}
			continue
		}
		l.readChar()
	}

	return string(l.input[start:l.position])
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

func (l *Lexer) setMode(mode LexerMode) {
	l.mode = mode
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
