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
	LanguageMode LexerMode = iota
	// CommandMode: Inside command bodies (after : or inside {})
	CommandMode
)

// Pool for working buffers only
var workBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 256)
	},
}

// Lexer tokenizes Devcmd source code with mode-based parsing
type Lexer struct {
	input      []byte
	position   int
	readPos    int
	ch         byte
	line       int
	column     int
	afterAt    bool
	lastToken  TokenType
	mode       LexerMode
	braceLevel int // **NEW**: Track block nesting level
}

// New creates a new lexer instance with optimized initialization
func New(input string) *Lexer {
	l := &Lexer{
		input:  []byte(input),
		line:   1,
		column: 0,
		mode:   LanguageMode,
	}
	l.readChar()
	return l
}

// setMode allows changing the lexer mode for testing
func (l *Lexer) setMode(mode LexerMode) {
	l.mode = mode
}

// TokenizeToSlice tokenizes to pre-allocated slice for maximum performance
func (l *Lexer) TokenizeToSlice() []Token {
	estimatedTokens := (len(l.input) / 5)
	if estimatedTokens < 16 {
		estimatedTokens = 16
	}
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

	if tok.Type == AT {
		l.afterAt = true
	} else if l.afterAt && tok.Type == IDENTIFIER {
		l.afterAt = false
	}

	l.lastToken = tok.Type
	return tok
}

// lexTokenFast performs fast token lexing with mode-aware logic
func (l *Lexer) lexTokenFast() Token {
	l.skipWhitespaceFast()
	start := l.position

	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode(start)
	case CommandMode:
		return l.lexCommandMode(start)
	default:
		return l.lexLanguageMode(start)
	}
}

// lexLanguageMode handles top-level language constructs and decorator parsing
func (l *Lexer) lexLanguageMode(start int) Token {
	switch l.ch {
	case 0:
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}
	case '\n':
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '@':
		tok := Token{Type: AT, Value: "@", Line: l.line, Column: l.column, Semantic: SemOperator, Scope: "punctuation.definition.decorator.devcmd"}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case ':':
		tok := Token{Type: COLON, Value: ":", Line: l.line, Column: l.column}
		l.readChar()
		if l.shouldSwitchToCommandMode() {
			l.mode = CommandMode
		}
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '=':
		tok := Token{Type: EQUALS, Value: "=", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case ',':
		tok := Token{Type: COMMA, Value: ",", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '(':
		tok := Token{Type: LPAREN, Value: "(", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case ')':
		tok := Token{Type: RPAREN, Value: ")", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		l.skipWhitespaceFast()
		if l.ch != '{' {
			l.mode = CommandMode
		}
		return tok
	case '{':
		tok := Token{Type: LBRACE, Value: "{", Line: l.line, Column: l.column}
		l.mode = CommandMode
		l.braceLevel++ // **MODIFIED**: Increment brace level
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '}':
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
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
			return l.lexSingleCharFast(start)
		}
	}
}

// lexCommandMode handles shell text and decorators inside command bodies
func (l *Lexer) lexCommandMode(start int) Token {
	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}
	case '\n':
		// A newline only terminates a simple command (braceLevel == 0).
		if l.braceLevel == 0 {
			l.mode = LanguageMode
		}
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '}':
		// **MODIFIED**: Decrement brace level and only switch mode if it's the last brace.
		l.braceLevel--
		if l.braceLevel < 0 {
			l.braceLevel = 0 // Should not happen in valid code
		}
		if l.braceLevel == 0 {
			l.mode = LanguageMode
		}
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '@':
		l.mode = LanguageMode
		return l.lexLanguageMode(start)
	case '\\':
		if l.peekChar() == '\n' {
			return l.lexLineContinuationFast(start)
		}
		return l.lexShellTextFast(start)
	default:
		return l.lexShellTextFast(start)
	}
}

// shouldSwitchToCommandMode determines if we apply syntax sugar for simple commands.
func (l *Lexer) shouldSwitchToCommandMode() bool {
	pos, readPos, ch, line, col := l.position, l.readPos, l.ch, l.line, l.column
	defer func() { l.position, l.readPos, l.ch, l.line, l.column = pos, readPos, ch, line, col }()
	l.skipWhitespaceFast()
	return l.ch != '{' && l.ch != '@' && l.ch != '\n' && l.ch != 0
}

// lexShellTextFast lexes shell command text as a single unit.
func (l *Lexer) lexShellTextFast(start int) Token {
	startLine, startColumn := l.line, l.column
	for l.ch != 0 {
		// **MODIFIED**: Use braceLevel to decide if '}' or '\n' are terminators.
		if (l.ch == '}' && l.braceLevel > 0) || (l.ch == '\n' && l.braceLevel == 0) {
			break
		}
		if l.ch == '@' && l.isDecoratorShape() {
			break
		}
		if l.ch == '\\' && l.peekChar() == '\n' {
			break
		}
		l.readChar()
	}

	value := string(l.input[start:l.position])
	return Token{
		Type:      IDENTIFIER,
		Value:     strings.TrimRight(value, " \t\r\f"),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemCommand,
		Scope:     "source.shell.embedded.devcmd",
	}
}

// isDecoratorShape checks if '@' starts a decorator or inline variable.
func (l *Lexer) isDecoratorShape() bool {
	if l.position > 0 && isIdentPart[l.input[l.position-1]] {
		return false
	}

	pos, readPos, ch := l.position, l.readPos, l.ch
	defer func() { l.position, l.readPos, l.ch = pos, readPos, ch }()

	l.readChar() // Skip '@'
	if !isLetter[l.ch] && l.ch != '_' {
		return false
	}
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}
	l.skipWhitespaceFast()
	return l.ch == '(' || l.ch == '{'
}

// --- Utility and unchanged functions from here ---

func (l *Lexer) lexSingleCharFast(start int) Token {
	startLine, startColumn := l.line, l.column
	char := l.ch
	l.readChar()
	return Token{
		Type:      IDENTIFIER,
		Value:     string(char),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemOperator,
		Scope:     "punctuation.other.devcmd",
	}
}

func (l *Lexer) lexStringFast(quote byte, stringType StringType, start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar()
	var escaped []byte
	valueStart := l.position
	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			if escaped == nil {
				escaped = make([]byte, 0, 64)
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
		escaped = append(escaped, l.input[valueStart:l.position]...)
		value = string(escaped)
	} else {
		value = string(l.input[valueStart:l.position])
	}
	if l.ch == quote {
		l.readChar()
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

func (l *Lexer) lexIdentifierOrKeywordFast(start int) Token {
	startLine, startColumn := l.line, l.column
	l.readIdentifierFast()
	value := string(l.input[start:l.position])
	var tokenType TokenType
	var semantic SemanticTokenType
	var scope string
	if l.afterAt {
		tokenType = IDENTIFIER
		semantic = SemDecorator
		scope = "entity.name.function.decorator.devcmd"
	} else if l.isInDecoratorParams() {
		tokenType = IDENTIFIER
		semantic = SemParameter
		scope = "variable.parameter.devcmd"
	} else {
		switch value {
		case "var":
			tokenType, semantic, scope = VAR, SemKeyword, "keyword.control.var.devcmd"
		case "stop":
			tokenType, semantic, scope = STOP, SemKeyword, "keyword.control.stop.devcmd"
		case "watch":
			tokenType, semantic, scope = WATCH, SemKeyword, "keyword.control.watch.devcmd"
		default:
			tokenType, semantic, scope = IDENTIFIER, SemCommand, "entity.name.function.devcmd"
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

func (l *Lexer) isInDecoratorParams() bool {
	if l.lastToken != LPAREN && l.lastToken != COMMA && l.lastToken != EQUALS {
		return false
	}
	pos, depth, maxLookback := l.position-1, 0, 100
	for pos >= 0 && maxLookback > 0 {
		ch := l.input[pos]
		if ch == ')' {
			depth++
		} else if ch == '(' {
			depth--
			if depth < 0 {
				pos--
				for pos >= 0 && (l.input[pos] == ' ' || l.input[pos] == '\t') {
					pos--
				}
				identEnd := pos + 1
				for pos >= 0 && (isLetter[l.input[pos]] || isDigit[l.input[pos]] || l.input[pos] == '-' || l.input[pos] == '_') {
					pos--
				}
				if pos >= 0 && l.input[pos] == '@' && pos+1 < identEnd {
					return true
				}
				return false
			}
		} else if ch == '\n' || ch == ';' || ch == '{' || ch == '}' {
			return false
		}
		pos--
		maxLookback--
	}
	return false
}

func (l *Lexer) lexNumberOrDurationFast(start int) Token {
	startLine, startColumn := l.line, l.column
	if l.ch == '-' {
		l.readChar()
	}
	for l.ch != 0 && isDigit[l.ch] {
		l.readChar()
	}
	if l.ch == '.' && l.peekChar() != 0 && isDigit[l.peekChar()] {
		l.readChar()
		for l.ch != 0 && isDigit[l.ch] {
			l.readChar()
		}
	}
	if l.isDurationUnit() {
		l.readDurationUnit()
		return Token{
			Type:      DURATION,
			Value:     string(l.input[start:l.position]),
			Line:      startLine,
			Column:    startColumn,
			EndLine:   l.line,
			EndColumn: l.column,
			Semantic:  SemNumber,
			Scope:     "constant.numeric.duration.devcmd",
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

func (l *Lexer) isDurationUnit() bool {
	if l.ch == 0 {
		return false
	}
	switch l.ch {
	case 'n':
		return l.peekChar() == 's'
	case 'u':
		return l.peekChar() == 's'
	case 'm':
		next := l.peekChar()
		return next == 's' || next == 0 || !isLetter[next]
	case 's', 'h':
		next := l.peekChar()
		return next == 0 || !isLetter[next]
	}
	if l.ch == 0xCE && l.peekChar() == 0xBC {
		return l.peekCharAt(2) == 's'
	}
	return false
}

func (l *Lexer) readDurationUnit() {
	switch l.ch {
	case 'n', 'u':
		if l.peekChar() == 's' {
			l.readChar()
			l.readChar()
		}
	case 'm':
		l.readChar()
		if l.ch == 's' {
			l.readChar()
		}
	case 's', 'h':
		l.readChar()
	case 0xCE:
		if l.peekChar() == 0xBC && l.peekCharAt(2) == 's' {
			l.readChar()
			l.readChar()
			l.readChar()
		}
	}
}

func (l *Lexer) peekCharAt(n int) byte {
	pos := l.readPos + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

func (l *Lexer) lexCommentFast(start int) Token {
	startLine, startColumn := l.line, l.column
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

func (l *Lexer) lexMultilineCommentFast(start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar()
	l.readChar()
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
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

func (l *Lexer) lexLineContinuationFast(start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar()
	l.readChar()
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

func (l *Lexer) readIdentifierFast() {
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}
}

func (l *Lexer) skipWhitespaceFast() {
	for isWhitespace[l.ch] && l.ch != '\n' {
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
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

func (l *Lexer) readHexEscape() string {
	if !isHexDigit[l.peekChar()] {
		return "\\x"
	}
	l.readChar()
	hex1 := l.ch
	l.readChar()
	if !isHexDigit[l.ch] {
		workBuf := workBufPool.Get().([]byte)
		defer workBufPool.Put(workBuf[:0])
		workBuf = append(workBuf, '\\', 'x', hex1)
		return string(workBuf)
	}
	hex2 := l.ch
	value := hexValueFast(hex1)*16 + hexValueFast(hex2)
	return string(rune(value))
}

func (l *Lexer) readUnicodeEscape() string {
	l.readChar()
	l.readChar()
	start := l.position
	for l.ch != '}' && l.ch != 0 && isHexDigit[l.ch] {
		l.readChar()
	}
	if l.ch != '}' {
		return "\\u{"
	}
	hexDigits := string(l.input[start:l.position])
	l.readChar()
	if len(hexDigits) == 0 {
		return "\\u{}"
	}
	var value rune
	for _, ch := range hexDigits {
		value = value*16 + rune(hexValueFast(byte(ch)))
	}
	if !utf8.ValidRune(value) {
		workBuf := workBufPool.Get().([]byte)
		defer workBufPool.Put(workBuf[:0])
		workBuf = append(workBuf, "\\u{"...)
		workBuf = append(workBuf, hexDigits...)
		workBuf = append(workBuf, '}')
		return string(workBuf)
	}
	return string(value)
}

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

