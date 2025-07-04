package lexer

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/aledsdavies/devcmd/pkgs/stdlib"
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
	// VariableValueMode: Parsing variable values (after var NAME =)
	VariableValueMode
)

// Pool for working buffers only
var workBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 256)
	},
}

// BraceContext represents a brace nesting level with its mode
type BraceContext struct {
	level int
	mode  LexerMode
}

// VariableContext tracks variable parsing state
type VariableContext struct {
	inVarDecl       bool // Are we inside a var declaration?
	inVarGroup      bool // Are we inside var ( ... )?
	expectingValue  bool // Are we expecting a variable value (after =)?
	varGroupLevel   int  // Nesting level of var groups
	valueStarted    bool // Have we started parsing a value?
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
	braceStack []BraceContext // Stack-based bracket tracking
	varContext VariableContext // Variable parsing context
}

// New creates a new lexer instance with optimized initialization
func New(input string) *Lexer {
	l := &Lexer{
		input:      []byte(input),
		line:       1,
		column:     0,
		mode:       LanguageMode,
		braceStack: make([]BraceContext, 0, 16), // Pre-allocate for common nesting
		varContext: VariableContext{},
	}
	l.readChar()
	return l
}

// setMode allows changing the lexer mode for testing
func (l *Lexer) setMode(mode LexerMode) {
	l.mode = mode
}

// getCurrentBraceLevel returns the current brace nesting level
func (l *Lexer) getCurrentBraceLevel() int {
	if len(l.braceStack) == 0 {
		return 0
	}
	return l.braceStack[len(l.braceStack)-1].level
}

// pushBraceContext adds a new brace context to the stack
func (l *Lexer) pushBraceContext(mode LexerMode) {
	level := 1
	if len(l.braceStack) > 0 {
		level = l.braceStack[len(l.braceStack)-1].level + 1
	}
	l.braceStack = append(l.braceStack, BraceContext{level: level, mode: mode})
}

// popBraceContext removes the top brace context from the stack
func (l *Lexer) popBraceContext() LexerMode {
	if len(l.braceStack) == 0 {
		return LanguageMode
	}

	l.braceStack = l.braceStack[:len(l.braceStack)-1]

	// Return the mode we should switch to after popping
	if len(l.braceStack) == 0 {
		return LanguageMode
	}
	return l.braceStack[len(l.braceStack)-1].mode
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

	// Update variable context based on tokens
	l.updateVariableContext(tok)

	if tok.Type == AT {
		l.afterAt = true
	} else if l.afterAt && tok.Type == IDENTIFIER {
		// After parsing a decorator name, reset the flag
		if tok.Semantic == SemDecorator || tok.Semantic == SemVariable {
			l.afterAt = false
		}
	}

	l.lastToken = tok.Type
	return tok
}

// updateVariableContext updates the variable parsing context based on the current token
func (l *Lexer) updateVariableContext(tok Token) {
	switch tok.Type {
	case VAR:
		l.varContext.inVarDecl = true
		l.varContext.expectingValue = false
		l.varContext.valueStarted = false
	case LPAREN:
		if l.varContext.inVarDecl {
			l.varContext.inVarGroup = true
			l.varContext.varGroupLevel++
		}
	case RPAREN:
		if l.varContext.inVarGroup {
			l.varContext.varGroupLevel--
			if l.varContext.varGroupLevel == 0 {
				l.varContext.inVarGroup = false
				l.varContext.inVarDecl = false
				l.varContext.expectingValue = false
				l.varContext.valueStarted = false
			}
		}
	case EQUALS:
		if l.varContext.inVarDecl {
			l.varContext.expectingValue = true
			l.varContext.valueStarted = false
			l.mode = VariableValueMode
		}
	case NEWLINE:
		// Newline terminates variable declaration unless we're in a var group
		if l.varContext.inVarDecl && !l.varContext.inVarGroup {
			l.varContext.inVarDecl = false
			l.varContext.expectingValue = false
			l.varContext.valueStarted = false
			l.mode = LanguageMode
		}
		// In var group, newline just ends the current variable value
		if l.varContext.expectingValue {
			l.varContext.expectingValue = false
			l.varContext.valueStarted = false
			l.mode = LanguageMode
		}
	case COMMA:
		// Comma terminates variable value in var groups
		if l.varContext.inVarGroup && l.varContext.expectingValue {
			l.varContext.expectingValue = false
			l.varContext.valueStarted = false
			l.mode = LanguageMode
		}
	case EOF:
		// EOF terminates everything
		l.varContext = VariableContext{}
		l.mode = LanguageMode
	default:
		// Mark that we've started parsing a value
		if l.varContext.expectingValue && l.mode == VariableValueMode {
			// Check if this token looks like a variable value
			if tok.Type == IDENTIFIER || tok.Type == STRING || tok.Type == NUMBER || tok.Type == DURATION {
				l.varContext.valueStarted = true
				// Don't switch mode yet - let the value parsing continue until terminated
			}
		}
	}
}

// lexTokenFast performs fast token lexing with mode-aware logic
func (l *Lexer) lexTokenFast() Token {
	// Only skip whitespace in language mode and variable value mode
	// In command mode, whitespace should be preserved as shell text tokens
	if l.mode == LanguageMode || l.mode == VariableValueMode {
		l.skipWhitespaceFast()
	}
	start := l.position

	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode(start)
	case CommandMode:
		return l.lexCommandMode(start)
	case VariableValueMode:
		return l.lexVariableValueMode(start)
	default:
		return l.lexLanguageMode(start)
	}
}

// lexVariableValueMode handles variable value parsing with complex identifier support
func (l *Lexer) lexVariableValueMode(start int) Token {
	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}
	case '\n':
		// Newline terminates variable value
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '"':
		return l.lexStringFast('"', DoubleQuoted, start)
	case '\'':
		return l.lexStringFast('\'', SingleQuoted, start)
	case '`':
		return l.lexStringFast('`', Backtick, start)
	case ')':
		// Closing paren terminates variable value in var groups
		if l.varContext.inVarGroup {
			tok := Token{Type: RPAREN, Value: ")", Line: l.line, Column: l.column}
			l.readChar()
			tok.EndLine, tok.EndColumn = l.line, l.column
			return tok
		}
		// Otherwise, it's part of the variable value
		return l.lexComplexVariableValue(start)
	case ',':
		// Comma terminates variable value in var groups
		if l.varContext.inVarGroup {
			tok := Token{Type: COMMA, Value: ",", Line: l.line, Column: l.column}
			l.readChar()
			tok.EndLine, tok.EndColumn = l.line, l.column
			return tok
		}
		// Otherwise, it's part of the variable value
		return l.lexComplexVariableValue(start)
	default:
		// Check if it's a simple number or duration first
		if isDigit[l.ch] || l.ch == '-' {
			return l.lexNumberOrDurationFast(start)
		}
		// Otherwise, treat as complex variable value
		return l.lexComplexVariableValue(start)
	}
}

// lexComplexVariableValue lexes complex variable values like URLs, paths, etc.
func (l *Lexer) lexComplexVariableValue(start int) Token {
	startLine := l.line
	startColumn := l.column

	for l.ch != 0 && !l.isVariableValueTerminator() {
		l.readChar()
	}

	value := strings.TrimRight(string(l.input[start:l.position]), " \t\r\f")

	return Token{
		Type:      IDENTIFIER,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemString, // Complex variable values should be SemString
		Scope:     "string.unquoted.devcmd",
	}
}

// isVariableValueTerminator checks if the current character terminates a variable value
func (l *Lexer) isVariableValueTerminator() bool {
	switch l.ch {
	case '\n', 0: // Newline or EOF always terminates
		return true
	case ')':
		// Closing paren terminates if we're in a var group
		return l.varContext.inVarGroup
	case ',':
		// Comma terminates if we're in a var group
		return l.varContext.inVarGroup
	default:
		return false
	}
}

// **FIX 1: Helper function to peek ahead without consuming input**
// This is crucial for deciding mode transitions without eating significant whitespace.
func (l *Lexer) peekNextNonWhitespace() byte {
	// Save current state
	pos, readPos, ch, col := l.position, l.readPos, l.ch, l.column
	defer func() {
		// Restore state
		l.position, l.readPos, l.ch, l.column = pos, readPos, ch, col
	}()

	l.skipWhitespaceFast()
	return l.ch
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
			// CRITICAL: Skip whitespace after colon ONLY when entering command mode
			l.skipWhitespaceFast()
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

		// **FIX 1 (continued): Use the peeker to fix whitespace consumption.**
		// After a decorator's `(...)`, if it's not followed by a `{`, it was an
		// inline decorator (like @var). We must switch back to CommandMode
		// to continue parsing the shell text, and we must *not* consume the
		// whitespace that follows.
		if l.peekNextNonWhitespace() != '{' {
			l.mode = CommandMode
		}
		// If a `{` *does* follow, we stay in LanguageMode to parse it correctly.
		return tok
	case '{':
		tok := Token{Type: LBRACE, Value: "{", Line: l.line, Column: l.column}
		l.mode = CommandMode
		l.pushBraceContext(CommandMode)
		l.readChar()
		// CRITICAL: Skip whitespace after opening brace when entering command mode
		l.skipWhitespaceFast()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '}':
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.mode = l.popBraceContext()
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
	// If we are starting at whitespace, let lexShellTextFast handle it
	// to correctly preserve it as part of a shell text token.
	if isWhitespace[l.ch] {
		return l.lexShellTextFast(start)
	}

	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return Token{Type: EOF, Value: "", Line: l.line, Column: l.column}
	case '\n':
		// A newline only terminates a simple command (no braces on stack)
		if len(l.braceStack) == 0 {
			l.mode = LanguageMode
		}
		tok := Token{Type: NEWLINE, Value: "\n", Line: l.line, Column: l.column}
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '{':
		// Nested braces within command mode
		tok := Token{Type: LBRACE, Value: "{", Line: l.line, Column: l.column}
		l.pushBraceContext(CommandMode)
		l.readChar()
		l.skipWhitespaceFast()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '}':
		tok := Token{Type: RBRACE, Value: "}", Line: l.line, Column: l.column}
		l.mode = l.popBraceContext()
		l.readChar()
		tok.EndLine, tok.EndColumn = l.line, l.column
		return tok
	case '@':
		if l.isDecoratorShape() {
			l.mode = LanguageMode
			return l.lexLanguageMode(start)
		}
		return l.lexShellTextFast(start)
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
		if l.ch == '}' && len(l.braceStack) > 0 {
			break
		}
		if l.ch == '\n' && len(l.braceStack) == 0 {
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

	// Trim trailing whitespace if we stopped at '}' to avoid "text }" pattern.
	// This fulfills the "don't care about postfix whitespace on the last token" requirement.
	if l.ch == '}' && len(value) > 0 {
		value = strings.TrimRight(value, " \t\r\f")
	}

	// **FIX 2: Prevent empty IDENTIFIER tokens.**
	// If the value is empty after trimming (e.g., just space between a command
	// and a `}`), don't emit a token. Instead, get the next real token.
	if value == "" {
		return l.lexTokenFast()
	}

	return Token{
		Type:      IDENTIFIER,
		Value:     value, // Preserve all internal and leading whitespace
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemShellText,
		Scope:     "source.shell.embedded.devcmd",
	}
}

// isDecoratorShape checks if '@' starts a valid decorator using the decorator registry.
func (l *Lexer) isDecoratorShape() bool {
	// Save current state
	pos, readPos, ch := l.position, l.readPos, l.ch
	defer func() { l.position, l.readPos, l.ch = pos, readPos, ch }()

	if l.position > 0 {
		prevCh := l.input[l.position-1]
		if isIdentPart[prevCh] {
			return l.isValidDecoratorWithParentheses()
		}
	}

	l.readChar() // Skip '@'

	// Must be followed by letter or underscore
	if !isLetter[l.ch] && l.ch != '_' {
		return false
	}

	// Read the identifier part
	identStart := l.position
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}

	identName := string(l.input[identStart:l.position])

	if !stdlib.IsValidDecorator(identName) {
		return false
	}

	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\f' {
		l.readChar()
	}

	if stdlib.IsFunctionDecorator(identName) {
		if l.ch == '(' {
			return true
		}
		return l.isAtWordBoundary()
	}

	if stdlib.IsBlockDecorator(identName) {
		if l.ch == '(' || l.ch == '{' {
			return true
		}
		return l.isAtWordBoundary()
	}

	return false
}

func (l *Lexer) isValidDecoratorWithParentheses() bool {
	pos, readPos, ch := l.position, l.readPos, l.ch
	defer func() { l.position, l.readPos, l.ch = pos, readPos, ch }()

	l.readChar() // Skip '@'

	if !isLetter[l.ch] && l.ch != '_' {
		return false
	}

	identStart := l.position
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}

	identName := string(l.input[identStart:l.position])

	if !stdlib.IsValidDecorator(identName) {
		return false
	}

	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\f' {
		l.readChar()
	}

	return l.ch == '('
}

func (l *Lexer) isAtWordBoundary() bool {
	switch l.ch {
	case '\n', 0, '}', ';', '&', '|', '>', '<':
		return true
	case ' ', '\t', '\r', '\f':
		return true
	case '@':
		return true
	default:
		return false
	}
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
		// Use stdlib registry to determine semantic type
		stdlibSemanticType := stdlib.GetDecoratorSemanticType(value)
		switch stdlibSemanticType {
		case stdlib.SemVariable:
			semantic = SemVariable
		case stdlib.SemFunction:
			semantic = SemVariable // Map function decorators to SemVariable for now
		default:
			semantic = SemDecorator
		}
		scope = "entity.name.function.decorator.devcmd"
	} else if l.isInDecoratorParams() {
		tokenType = IDENTIFIER
		semantic = SemParameter
		scope = "variable.parameter.devcmd"
	} else {
		keywordType, isKeyword := keywords[value]
		if isKeyword {
			tokenType, semantic, scope = keywordType, SemKeyword, "keyword.control."+value+".devcmd"
		} else {
			// By default, an identifier at the start of a line is a command name
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

var keywords = map[string]TokenType{
	"var":   VAR,
	"stop":  STOP,
	"watch": WATCH,
}

func (l *Lexer) isInDecoratorParams() bool {
	if l.lastToken != LPAREN && l.lastToken != COMMA && l.lastToken != EQUALS {
		return false
	}
	pos, depth, maxLookback := l.position-1, 0, 200
	for pos >= 0 && maxLookback > 0 {
		ch := l.input[pos]
		if ch == ')' {
			depth++
		} else if ch == '(' {
			depth--
			if depth < 0 {
				// We found the opening parenthesis. Now look for the '@' before it.
				pos--
				for pos >= 0 && (l.input[pos] == ' ' || l.input[pos] == '\t') {
					pos--
				}
				// Go backwards over the identifier
				for pos >= 0 && (isIdentPart[l.input[pos]] || l.input[pos] == '-') {
					pos--
				}
				// Check if the preceding character is '@'
				if pos >= 0 && l.input[pos] == '@' {
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
	next := l.peekChar()
	switch l.ch {
	case 'n', 'u':
		return next == 's'
	case 'm':
		return next == 's' || next == 0 || !isLetter[next]
	case 's', 'h':
		return next == 0 || !isLetter[next]
	}
	if l.ch == 0xCE && next == 0xBC { // UTF-8 "μ"
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
	l.readChar() // consume '\'
	l.readChar() // consume '\n'
	// Per user request, replace line continuation with a single space.
	// This simplifies the parser, which no longer needs to handle LINE_CONT tokens.
	return Token{
		Type:      IDENTIFIER, // Treat it as part of shell text
		Value:     " ",
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemShellText,
		Scope:     "source.shell.embedded.devcmd",
	}
}

func (l *Lexer) readIdentifierFast() {
	for l.ch != 0 && (isIdentPart[l.ch] || l.ch == '-') {
		l.readChar()
	}
}

// skipWhitespaceFast skips whitespace with optimized early return
func (l *Lexer) skipWhitespaceFast() {
	for isWhitespace[l.ch] {
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
