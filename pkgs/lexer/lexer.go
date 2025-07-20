package lexer

import (
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/types"
)

// ASCII character lookup tables for fast classification
var (
	isWhitespace      [128]bool // Only ASCII range
	isLetter          [128]bool
	isDigit           [128]bool
	isIdentStart      [128]bool
	isIdentPart       [128]bool
	singleCharTokens  [128]types.TokenType // Fast lookup for single-char tokens
	singleCharStrings [128]string          // Pre-allocated single-char strings
)

func init() {
	for i := 0; i < 128; i++ {
		ch := byte(i)
		isWhitespace[i] = ch == ' ' || ch == '\t' || ch == '\r' || ch == '\f'
		isLetter[i] = ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
		isDigit[i] = '0' <= ch && ch <= '9'
		isIdentStart[i] = isLetter[i] || ch == '_'
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		singleCharTokens[i] = types.ILLEGAL // Default to ILLEGAL for non-single-char tokens
		singleCharStrings[i] = string(ch)   // Pre-allocate single char strings
	}

	// Initialize single character token mappings
	singleCharTokens['@'] = types.AT
	singleCharTokens[':'] = types.COLON
	singleCharTokens['='] = types.EQUALS
	singleCharTokens[','] = types.COMMA
	singleCharTokens['('] = types.LPAREN
	singleCharTokens[')'] = types.RPAREN
	singleCharTokens['{'] = types.LBRACE
	singleCharTokens['}'] = types.RBRACE
	singleCharTokens['*'] = types.ASTERISK
}

// LexerMode represents the lexer's parsing modes
type LexerMode int

const (
	LanguageMode LexerMode = iota // Top-level parsing and decorator parsing
	CommandMode                   // Shell content parsing inside command bodies
	PatternMode                   // Pattern decorator parsing (@when, @try blocks)
)

// Lexer follows the specification's two-mode system
type Lexer struct {
	input    string // Complete input (read once from Reader)
	position int    // Current position in input (byte offset)
	readPos  int    // Current reading position in input (byte offset)
	ch       rune   // Current rune under examination
	line     int    // Current line number
	column   int    // Current column number

	// Two-mode system
	mode LexerMode

	// Context tracking
	braceLevel int   // Track brace nesting
	braceStack []int // Stack of brace positions for structural tracking

	// Decorator context tracking
	currentDecorator   string    // Track current decorator name for mode switching
	previousMode       LexerMode // Track previous mode for returns from CommandMode
	inPatternDecorator bool      // Track if we're inside a pattern decorator block
	decoratorStack     []string  // Stack of decorator names to track nesting

	// Position tracking for error reporting
	lastPosition int
	lastLine     int
	lastColumn   int
}

// New creates a new Lexer from an io.Reader
func New(reader io.Reader) *Lexer {
	// Read entire input into string (simpler approach for now)
	data, err := io.ReadAll(reader)
	if err != nil {
		// Handle error by creating empty lexer
		data = []byte{}
	}

	l := &Lexer{
		input:          string(data),
		line:           1,
		column:         0,            // Will be incremented to 1 by initial readChar()
		mode:           LanguageMode, // Start in LanguageMode
		braceStack:     make([]int, 0, 8),
		decoratorStack: make([]string, 0, 8),
	}
	l.readChar()
	return l
}

// readChar reads the next character and advances position
func (l *Lexer) readChar() {
	l.position = l.readPos

	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		var size int
		l.ch, size = utf8.DecodeRuneInString(l.input[l.readPos:])
		if l.ch == utf8.RuneError {
			l.ch = rune(l.input[l.readPos])
			size = 1
		}
		l.readPos += size
	}

	// Track line/column for current character
	if l.ch == '\n' {
		l.line++
		l.column = 0 // Will be incremented to 1 for next character
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	ch, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
	return ch
}

// skipWhitespace skips whitespace characters except newlines (using fast ASCII lookups)
func (l *Lexer) skipWhitespace() {
	for l.ch != '\n' && l.ch != 0 {
		// Fast path for ASCII
		if l.ch < 128 && isWhitespace[l.ch] {
			l.readChar()
		} else if l.ch >= 128 && unicode.IsSpace(l.ch) {
			// Fallback for non-ASCII
			l.readChar()
		} else {
			break
		}
	}
}

// TokenizeToSlice tokenizes the entire input and returns a slice of tokens
func (l *Lexer) TokenizeToSlice() []types.Token {
	var tokens []types.Token
	for {
		token := l.NextToken()
		tokens = append(tokens, token)
		if token.Type == types.EOF {
			break
		}
	}
	return tokens
}

// NextToken returns the next token from the input
func (l *Lexer) NextToken() types.Token {
	// Prevent infinite loops
	if l.position == l.lastPosition && l.line == l.lastLine && l.column == l.lastColumn {
		// We haven't advanced - force EOF to prevent infinite loop
		return l.createToken(types.EOF, "", l.position, l.line, l.column)
	}
	l.lastPosition = l.position
	l.lastLine = l.line
	l.lastColumn = l.column

	// Dispatch based on current mode
	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode()
	case CommandMode:
		return l.lexCommandMode()
	case PatternMode:
		return l.lexPatternMode()
	default:
		return l.createToken(types.EOF, "", l.position, l.line, l.column)
	}
}

// createToken creates a token with position information
func (l *Lexer) createToken(tokenType types.TokenType, value string, start, line, column int) types.Token {
	return types.Token{
		Type:   tokenType,
		Value:  value,
		Line:   line,
		Column: column,
		Span: types.SourceSpan{
			Start: types.SourcePosition{Line: line, Column: column, Offset: start},
			End:   types.SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexLanguageMode handles top-level parsing and decorator parsing
func (l *Lexer) lexLanguageMode() types.Token {
	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Skip newlines in language mode
		l.readChar()
		return l.NextToken()

	case ':':
		l.readChar()
		// Transition to CommandMode after colon
		l.mode = CommandMode
		return l.createToken(types.COLON, ":", start, startLine, startColumn)

	case '=':
		l.readChar()
		return l.createToken(types.EQUALS, "=", start, startLine, startColumn)

	case ',':
		l.readChar()
		return l.createToken(types.COMMA, ",", start, startLine, startColumn)

	case '(':
		l.readChar()
		return l.createToken(types.LPAREN, "(", start, startLine, startColumn)

	case ')':
		l.readChar()
		return l.createToken(types.RPAREN, ")", start, startLine, startColumn)

	case '{':
		l.readChar()
		l.braceLevel++
		l.braceStack = append(l.braceStack, start)
		// Transition to appropriate mode for block content
		if decorators.IsPatternDecorator(l.currentDecorator) {
			l.mode = PatternMode
			l.inPatternDecorator = true
		} else {
			l.mode = CommandMode
		}
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	case '}':
		l.readChar()
		l.braceLevel--
		if len(l.braceStack) > 0 {
			l.braceStack = l.braceStack[:len(l.braceStack)-1]
		}
		// Stay in LanguageMode or transition based on context
		if l.braceLevel <= 0 {
			l.mode = LanguageMode
		}
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case '*':
		l.readChar()
		return l.createToken(types.ASTERISK, "*", start, startLine, startColumn)

	case '@':
		return l.lexDecorator(start, startLine, startColumn)

	case '"', '\'', '`':
		return l.lexString(l.ch, start, startLine, startColumn)

	case '#':
		return l.lexComment(start, startLine, startColumn)

	case '/':
		// Check for multi-line comment /* */
		if l.peekChar() == '*' {
			return l.lexMultilineComment(start, startLine, startColumn)
		}
		// Not a comment - treat as unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)

	case '-':
		// Check if this is a negative number
		if l.readPos < len(l.input) {
			nextCh, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
			if (nextCh < 128 && isDigit[nextCh]) || (nextCh >= 128 && unicode.IsDigit(nextCh)) {
				return l.lexNumber(start, startLine, startColumn)
			}
		}
		// Not a negative number - treat as unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)

	default:
		// Fast path for ASCII identifier start
		if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
			return l.lexIdentifierOrKeyword(start, startLine, startColumn)
		}
		// Fast path for ASCII digits
		if (l.ch < 128 && isDigit[l.ch]) || (l.ch >= 128 && unicode.IsDigit(l.ch)) {
			return l.lexNumber(start, startLine, startColumn)
		}

		// Unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)
	}
}

// lexCommandMode handles shell content parsing inside command bodies
// Recognizes: Shell text, Line continuations, Decorators, Block boundaries
func (l *Lexer) lexCommandMode() types.Token {
	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Newlines end shell content in command mode (unless line continuation)
		l.readChar()
		// Return to appropriate mode based on context
		if l.inPatternDecorator && l.previousMode == PatternMode {
			// Check if we should return to PatternMode
			// For nested patterns (braceLevel > 1), we always return to PatternMode
			// For top-level patterns (braceLevel == 1), we also return to PatternMode
			if l.braceLevel >= 1 {
				l.mode = PatternMode
			}
		} else if l.braceLevel == 0 {
			// Return to LanguageMode for simple commands without braces
			l.mode = LanguageMode
		} else {
			// We're inside braces but not in a pattern decorator - stay in CommandMode
			// This ensures we don't accidentally switch modes
		}
		return l.NextToken()

	case '}':
		// Closing brace - exit command mode
		l.readChar()
		l.braceLevel--
		if len(l.braceStack) > 0 {
			l.braceStack = l.braceStack[:len(l.braceStack)-1]
		}
		// Return to appropriate mode when exiting blocks
		if l.braceLevel <= 0 {
			if l.inPatternDecorator {
				l.mode = LanguageMode
				l.inPatternDecorator = false
				l.currentDecorator = ""
			} else {
				l.mode = LanguageMode
			}
		} else {
			// We're still inside nested braces - return to CommandMode
			l.mode = CommandMode
		}
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case '{':
		// Opening brace in command mode - start new block
		l.readChar()
		l.braceLevel++
		l.braceStack = append(l.braceStack, start)
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	case '@':
		// Handle Decorator path: check if Block or Pattern decorator
		return l.lexDecoratorInCommand(start, startLine, startColumn)

	default:
		// Handle Shell path: all other content as shell text
		return l.lexShellText(start, startLine, startColumn)
	}
}

// lexDecorator handles decorator parsing in LanguageMode
func (l *Lexer) lexDecorator(start, startLine, startColumn int) types.Token {
	// Skip @ character
	l.readChar()

	// Skip whitespace after @
	l.skipWhitespace()

	// Read decorator identifier using fast ASCII lookups
	if (l.ch >= 128 || !isIdentStart[l.ch]) && (l.ch < 128 || (!unicode.IsLetter(l.ch) && l.ch != '_')) {
		return l.createToken(types.ILLEGAL, "@", start, startLine, startColumn)
	}

	// Return AT token, let next token be the identifier
	return l.createToken(types.AT, "@", start, startLine, startColumn)
}

// lexDecoratorInCommand checks if @identifier is a decorator in CommandMode
func (l *Lexer) lexDecoratorInCommand(start, startLine, startColumn int) types.Token {
	// Look ahead to check if this is @identifier pattern
	savedPos := l.position
	savedReadPos := l.readPos
	savedCh := l.ch
	savedLine := l.line
	savedColumn := l.column

	// Skip @
	l.readChar()
	l.skipWhitespace()

	// Check if followed by identifier using fast ASCII lookups
	if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
		// Read the identifier to check if it's a decorator
		identStart := l.position
		for {
			if l.ch < 128 && isIdentPart[l.ch] {
				l.readChar()
			} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
				l.readChar()
			} else {
				break
			}
		}
		identifier := l.input[identStart:l.position]

		// Check if it's a registered decorator
		if decorators.IsDecorator(identifier) {
			// Check decorator type from registry - only block/pattern decorators need LanguageMode
			// Function decorators (@var, @env) should remain as shell text for parser processing
			if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
				// Switch to LanguageMode for decorator parsing
				l.mode = LanguageMode

				// Advance past @ character (don't restore position)
				l.position = savedPos
				l.readPos = savedReadPos
				l.ch = savedCh
				l.line = savedLine
				l.column = savedColumn
				l.readChar() // Skip the @ character

				return l.createToken(types.AT, "@", start, startLine, startColumn)
			}
		}
	}

	// Restore position - this is shell text starting with @
	l.position = savedPos
	l.readPos = savedReadPos
	l.ch = savedCh
	l.line = savedLine
	l.column = savedColumn

	return l.lexShellText(start, startLine, startColumn)
}

// lexPatternMode handles pattern decorator content (@when, @try blocks)
func (l *Lexer) lexPatternMode() types.Token {
	l.skipWhitespace()

	start := l.position
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createToken(types.EOF, "", start, startLine, startColumn)

	case '\n':
		// Skip newlines in pattern mode
		l.readChar()
		return l.NextToken()

	case '}':
		// Closing brace - exit pattern mode
		l.readChar()
		l.braceLevel--
		if len(l.braceStack) > 0 {
			l.braceStack = l.braceStack[:len(l.braceStack)-1]
		}
		// Return to appropriate mode when exiting pattern blocks
		if l.braceLevel <= 0 {
			l.mode = LanguageMode
			l.currentDecorator = "" // Clear decorator context
			l.inPatternDecorator = false
		} else {
			// We're still inside outer braces, return to CommandMode
			l.mode = CommandMode
			l.inPatternDecorator = false
			l.currentDecorator = ""
		}
		return l.createToken(types.RBRACE, "}", start, startLine, startColumn)

	case ':':
		l.readChar()
		// After colon in pattern mode, switch to CommandMode for shell content
		l.previousMode = PatternMode
		l.mode = CommandMode
		return l.createToken(types.COLON, ":", start, startLine, startColumn)

	case '{':
		l.readChar()
		l.braceLevel++
		l.braceStack = append(l.braceStack, start)
		// Transition to CommandMode for block content inside patterns
		l.previousMode = PatternMode
		l.mode = CommandMode
		return l.createToken(types.LBRACE, "{", start, startLine, startColumn)

	default:
		// Pattern identifiers (prod, dev, main, error, finally, default)
		if (l.ch < 128 && isIdentStart[l.ch]) || (l.ch >= 128 && (unicode.IsLetter(l.ch) || l.ch == '_')) {
			return l.lexIdentifierOrKeyword(start, startLine, startColumn)
		}

		// Unknown character
		char := string(l.ch)
		l.readChar()
		return l.createToken(types.ILLEGAL, char, start, startLine, startColumn)
	}
}

// lexShellText handles shell content in CommandMode
func (l *Lexer) lexShellText(start, startLine, startColumn int) types.Token {
	var result strings.Builder
	var inSingleQuote, inDoubleQuote, inBacktick bool
	var shellBraceLevel int // Track ${...} parameter expansion braces
	var parenLevel int      // Track $(...) command substitution
	var anyBraceLevel int   // Track any {...} constructs in shell context

	for l.ch != 0 {

		// Stop at newline (unless line continuation or inside quotes)
		if l.ch == '\n' {
			// Check for line continuation (backslash before newline)
			// Process line continuation when NOT inside single quotes (but do process in double quotes and backticks)
			if l.position > 0 && l.input[l.position-1] == '\\' && !inSingleQuote {
				// Line continuation - remove the backslash
				text := result.String()
				if len(text) > 0 && text[len(text)-1] == '\\' {
					result.Reset()
					result.WriteString(text[:len(text)-1]) // Remove the backslash
				}
				l.readChar() // Skip newline
				// Skip leading whitespace on the next line
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				continue
			}

			// If inside single quotes, include the newline literally
			if inSingleQuote {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// If inside double quotes or backticks without line continuation, include newline
			if inDoubleQuote || inBacktick {
				result.WriteRune(l.ch)
				l.readChar()
				continue
			}

			// Not in quotes and no line continuation - end of shell text
			break
		}

		// Stop at closing brace (block boundary) - unless inside quotes or shell constructs
		if l.ch == '}' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			if shellBraceLevel > 0 {
				// This is closing a shell parameter expansion ${...}
				shellBraceLevel--
			} else if anyBraceLevel > 0 {
				// This is closing some other shell brace construct
				anyBraceLevel--
			} else {
				// This is a block boundary - only break if we're not inside any shell constructs
				break
			}
		}

		// Stop at @ if it starts a block/pattern decorator - unless inside quotes
		if l.ch == '@' && !inSingleQuote && !inDoubleQuote && !inBacktick {
			// Look ahead to see if this is @identifier for a block/pattern decorator
			if l.readPos < len(l.input) {
				nextCh, _ := utf8.DecodeRuneInString(l.input[l.readPos:])
				if (nextCh < 128 && isIdentStart[nextCh]) || (nextCh >= 128 && (unicode.IsLetter(nextCh) || nextCh == '_')) {
					// Check if this is a block/pattern decorator by reading ahead
					savedPos := l.position
					savedReadPos := l.readPos
					savedCh := l.ch

					// Skip @ and read identifier
					l.readChar()
					identStart := l.position
					for {
						if l.ch < 128 && isIdentPart[l.ch] {
							l.readChar()
						} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
							l.readChar()
						} else {
							break
						}
					}
					identifier := l.input[identStart:l.position]

					// Restore position
					l.position = savedPos
					l.readPos = savedReadPos
					l.ch = savedCh

					// Only break for block/pattern decorators
					if decorators.IsBlockDecorator(identifier) || decorators.IsPatternDecorator(identifier) {
						break
					}
				}
			}
		}

		// Track various shell constructs BEFORE adding character
		if !inSingleQuote && !inDoubleQuote && !inBacktick {
			// Track shell parameter expansion ${...}
			if l.ch == '$' && l.peekChar() == '{' {
				shellBraceLevel++
			}
			// Track command substitution $(...)
			if l.ch == '$' && l.peekChar() == '(' {
				parenLevel++
			}
			// Track closing parentheses for command substitution
			if l.ch == ')' && parenLevel > 0 {
				parenLevel--
			}
			// Track standalone braces in shell context (brace expansion, find {})
			if l.ch == '{' {
				// Check if this is part of ${...} (already handled above)
				if result.Len() > 0 {
					lastChar := result.String()[result.Len()-1]
					if lastChar != '$' {
						anyBraceLevel++
					}
				} else {
					anyBraceLevel++
				}
			}
		}

		// Add character to result
		result.WriteRune(l.ch)

		// Track quote state AFTER adding character
		if l.ch == '\'' && !inDoubleQuote && !inBacktick {
			inSingleQuote = !inSingleQuote
		} else if l.ch == '"' && !inSingleQuote && !inBacktick {
			inDoubleQuote = !inDoubleQuote
		} else if l.ch == '`' && !inSingleQuote && !inDoubleQuote {
			inBacktick = !inBacktick
		}

		l.readChar()
	}

	text := strings.TrimSpace(result.String())
	if text == "" {
		return l.createToken(types.ILLEGAL, "", start, startLine, startColumn)
	}

	return l.createToken(types.SHELL_TEXT, text, start, startLine, startColumn)
}

// lexIdentifierOrKeyword handles identifiers and keywords (using fast ASCII lookups)
func (l *Lexer) lexIdentifierOrKeyword(start, startLine, startColumn int) types.Token {
	for {
		// Fast path for ASCII
		if l.ch < 128 && isIdentPart[l.ch] {
			l.readChar()
		} else if l.ch >= 128 && (unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch)) {
			// Fallback for non-ASCII
			l.readChar()
		} else {
			break
		}
	}

	value := l.input[start:l.position]

	// Track decorator names for mode switching
	if decorators.IsDecorator(value) {
		l.currentDecorator = value
	}

	// Check for keywords
	switch value {
	case "var":
		return l.createToken(types.VAR, value, start, startLine, startColumn)
	case "watch":
		return l.createToken(types.WATCH, value, start, startLine, startColumn)
	case "stop":
		return l.createToken(types.STOP, value, start, startLine, startColumn)
	case "true", "false":
		return l.createToken(types.BOOLEAN, value, start, startLine, startColumn)
	default:
		return l.createToken(types.IDENTIFIER, value, start, startLine, startColumn)
	}
}

// lexString handles string literals (quoted strings)
func (l *Lexer) lexString(quote rune, start, startLine, startColumn int) types.Token {
	// Skip opening quote
	l.readChar()
	contentStart := l.position

	for l.ch != quote && l.ch != 0 {
		if l.ch == '\\' {
			// Handle escape sequences
			l.readChar() // Skip backslash
			if l.ch != 0 {
				l.readChar() // Skip escaped character
			}
		} else {
			l.readChar()
		}
	}

	if l.ch == 0 {
		// Unterminated string
		return l.createToken(types.ILLEGAL, "unterminated string", start, startLine, startColumn)
	}

	// Extract content without quotes
	value := l.input[contentStart:l.position]
	l.readChar() // Skip closing quote

	return l.createToken(types.STRING, value, start, startLine, startColumn)
}

// lexNumber handles number literals (using fast ASCII lookups)
func (l *Lexer) lexNumber(start, startLine, startColumn int) types.Token {
	hasDecimal := false

	// Handle negative sign if present
	if l.ch == '-' {
		l.readChar()
	}

	for {
		// Fast path for ASCII digits
		if l.ch < 128 && isDigit[l.ch] {
			l.readChar()
		} else if l.ch == '.' && !hasDecimal {
			hasDecimal = true
			l.readChar()
		} else if l.ch >= 128 && unicode.IsDigit(l.ch) {
			// Fallback for non-ASCII digits
			l.readChar()
		} else {
			break
		}
	}

	// Check for duration suffix using fast ASCII lookups
	if (l.ch < 128 && isLetter[l.ch]) || (l.ch >= 128 && unicode.IsLetter(l.ch)) {
		durStart := l.position
		for {
			if l.ch < 128 && isLetter[l.ch] {
				l.readChar()
			} else if l.ch >= 128 && unicode.IsLetter(l.ch) {
				l.readChar()
			} else {
				break
			}
		}
		suffix := l.input[durStart:l.position]

		// Valid duration suffixes
		switch suffix {
		case "ns", "us", "ms", "s", "m", "h":
			value := l.input[start:l.position]
			return l.createToken(types.DURATION, value, start, startLine, startColumn)
		default:
			// Invalid suffix - treat as separate tokens
			l.position = durStart
			l.readPos = durStart + utf8.RuneLen(l.ch)
			l.ch, _ = utf8.DecodeRuneInString(l.input[durStart:])
		}
	}

	value := l.input[start:l.position]
	return l.createToken(types.NUMBER, value, start, startLine, startColumn)
}

// lexComment handles comment lines starting with #
func (l *Lexer) lexComment(start, startLine, startColumn int) types.Token {
	// Read from # to end of line
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	value := l.input[start:l.position]
	return l.createToken(types.COMMENT, value, start, startLine, startColumn)
}

// lexMultilineComment handles multi-line comments /* */
func (l *Lexer) lexMultilineComment(start, startLine, startColumn int) types.Token {
	// Skip /*
	l.readChar() // Skip /
	l.readChar() // Skip *

	// Read until */
	for {
		if l.ch == 0 {
			// Unterminated comment
			return l.createToken(types.ILLEGAL, "unterminated comment", start, startLine, startColumn)
		}

		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // Skip *
			l.readChar() // Skip /
			break
		}

		l.readChar()
	}

	value := l.input[start:l.position]
	return l.createToken(types.MULTILINE_COMMENT, value, start, startLine, startColumn)
}
