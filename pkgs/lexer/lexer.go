package lexer

import (
	"strings"
	"unicode/utf8"
)

// Character classification lookup tables for fast operations
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
		isIdentStart[i] = isLetter[i]
		isIdentPart[i] = isIdentStart[i] || isDigit[i] || ch == '-'
		isHexDigit[i] = isDigit[i] || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
	}
}

// LexerMode represents the current parsing context
type LexerMode int

const (
	// LanguageMode: Structural parsing of Devcmd syntax
	LanguageMode LexerMode = iota
	// CommandMode: Shell content capture
	CommandMode
	// PatternMode: Inside pattern-matching blocks (@when, @try, etc.)
	PatternMode
)

// Lexer tokenizes Devcmd source code with mode-based parsing
type Lexer struct {
	input        string // Changed from []byte to string for efficiency
	position     int
	readPos      int
	ch           byte
	line         int
	column       int
	mode         LexerMode
	braceLevel   int // Track brace nesting for command mode
	patternLevel int // Track pattern-matching decorator nesting
	modeStack    []LexerMode // Stack to track mode transitions
}

// New creates a new lexer instance
func New(input string) *Lexer {
	l := &Lexer{
		input:        input,
		line:         1,
		column:       0, // Start at column 0, will be incremented to 1 on first readChar
		mode:         LanguageMode,
		braceLevel:   0,
		patternLevel: 0,
		modeStack:    []LexerMode{},
	}
	l.readChar()
	return l
}

// pushMode saves current mode and switches to new mode
func (l *Lexer) pushMode(newMode LexerMode) {
	l.modeStack = append(l.modeStack, l.mode)
	l.mode = newMode
}

// popMode restores previous mode from stack
func (l *Lexer) popMode() {
	if len(l.modeStack) > 0 {
		l.mode = l.modeStack[len(l.modeStack)-1]
		l.modeStack = l.modeStack[:len(l.modeStack)-1]
	}
}

// TokenizeToSlice tokenizes to pre-allocated slice with memory optimization
func (l *Lexer) TokenizeToSlice() []Token {
	// More conservative estimate to reduce memory usage
	estimatedTokens := len(l.input) / 12 // More conservative estimate
	if estimatedTokens < 4 {
		estimatedTokens = 4
	}
	if estimatedTokens > 500 {
		estimatedTokens = 500 // Cap initial allocation
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
	return l.lexToken()
}

// lexToken performs token lexing with mode-aware logic
func (l *Lexer) lexToken() Token {
	// Skip whitespace in LanguageMode and PatternMode
	if l.mode == LanguageMode || l.mode == PatternMode {
		l.skipWhitespace()
	}

	start := l.position

	switch l.mode {
	case LanguageMode:
		return l.lexLanguageMode(start)
	case CommandMode:
		return l.lexCommandMode(start)
	case PatternMode:
		return l.lexPatternMode(start)
	default:
		return l.lexLanguageMode(start)
	}
}

// lexLanguageMode handles structural Devcmd syntax
func (l *Lexer) lexLanguageMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		return l.createSimpleToken(EOF, "", start, startLine, startColumn)
	case '\n':
		tok := l.createSimpleToken(NEWLINE, "\n", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		// Check if we're in a pattern-matching context
		if l.patternLevel > 0 {
			// In pattern mode, ':' doesn't switch to command mode
			l.updateTokenEnd(&tok)
			return tok
		}
		// Only switch to command mode if not in variable assignment context
		if l.shouldEnterCommandMode() {
			l.mode = CommandMode
			l.skipWhitespace() // Skip whitespace at mode boundary
		}
		l.updateTokenEnd(&tok)
		return tok
	case '=':
		tok := l.createSimpleToken(EQUALS, "=", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case ',':
		tok := l.createSimpleToken(COMMA, ",", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		if l.patternLevel > 0 {
			// Inside pattern-matching decorator
			l.mode = PatternMode
		} else {
			// Regular command block
			l.mode = CommandMode
		}
		l.braceLevel++
		l.readChar()
		l.skipWhitespace() // Skip whitespace after opening brace
		l.updateTokenEnd(&tok)
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}
		if l.braceLevel == 0 {
			l.mode = LanguageMode
			if l.patternLevel > 0 {
				l.patternLevel--
			}
		} else {
			// Pop mode from stack if we have one
			l.popMode()
		}
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '"':
		return l.lexString('"', DoubleQuoted, start)
	case '\'':
		return l.lexString('\'', SingleQuoted, start)
	case '`':
		return l.lexString('`', Backtick, start)
	case '#':
		return l.lexComment(start)
	case '/':
		if l.peekChar() == '*' {
			return l.lexMultilineComment(start)
		}
		fallthrough
	case '\\':
		if l.peekChar() == '\n' {
			// Line continuation in language mode - treat as single char
			return l.lexSingleChar(start)
		}
		fallthrough
	default:
		if l.ch < 128 && isIdentStart[l.ch] {
			return l.lexIdentifierOrKeyword(start)
		} else if l.ch >= 128 && isLetter[l.ch] {
			return l.lexIdentifierOrKeyword(start)
		} else if isDigit[l.ch] || (l.ch == '-' && l.peekChar() != 0 && isDigit[l.peekChar()]) {
			return l.lexNumberOrDuration(start)
		} else {
			return l.lexSingleChar(start)
		}
	}
}

// lexPatternMode handles pattern-matching decorator blocks (@when, @try, etc.)
func (l *Lexer) lexPatternMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return l.createSimpleToken(EOF, "", start, startLine, startColumn)
	case '\n':
		// In pattern mode, newlines are consumed but NOT emitted as tokens
		// This aligns with the shell behavior where newlines separate commands
		l.readChar()
		l.skipWhitespace()
		return l.lexToken() // Get the next meaningful token
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		// After ':' in pattern mode, check if we should enter command mode
		// Look ahead to see if we have a block '{' or direct shell content
		l.skipWhitespace()
		if l.ch == '{' {
			// Stay in PatternMode, the '{' will switch to CommandMode
		} else if l.ch == '@' {
			// Decorator after pattern - push current mode and switch to LanguageMode
			l.pushMode(PatternMode)
			l.mode = LanguageMode
		} else if l.shouldEnterCommandMode() {
			l.mode = CommandMode
		}
		l.updateTokenEnd(&tok)
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}
		if l.braceLevel == 0 {
			l.mode = LanguageMode
			if l.patternLevel > 0 {
				l.patternLevel--
			}
		} else {
			// Pop mode from stack
			l.popMode()
		}
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.mode = CommandMode
		l.braceLevel++
		l.readChar()
		l.skipWhitespace()
		l.updateTokenEnd(&tok)
		return tok
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '"':
		return l.lexString('"', DoubleQuoted, start)
	case '\'':
		return l.lexString('\'', SingleQuoted, start)
	case '`':
		return l.lexString('`', Backtick, start)
	default:
		if l.ch < 128 && isIdentStart[l.ch] {
			return l.lexIdentifierOrKeyword(start)
		} else if l.ch >= 128 && isLetter[l.ch] {
			return l.lexIdentifierOrKeyword(start)
		} else if isDigit[l.ch] || (l.ch == '-' && l.peekChar() != 0 && isDigit[l.peekChar()]) {
			return l.lexNumberOrDuration(start)
		} else {
			return l.lexSingleChar(start)
		}
	}
}

// lexCommandMode handles shell content capture with proper newline handling
// lexCommandMode handles shell content capture with proper newline handling
func (l *Lexer) lexCommandMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return l.createSimpleToken(EOF, "", start, startLine, startColumn)
	case '\n':
		if l.braceLevel > 0 {
			// Inside a command block `{}`, newlines separate commands but are NOT emitted as tokens
			// Only jump back to pattern mode at the top brace of a pattern branch
			if l.patternLevel > 0 && l.braceLevel == 1 {
				l.mode = PatternMode
			}
			l.readChar()       // Consume '\n'
			l.skipWhitespace() // Consume all whitespace before the next token.
			return l.lexToken() // Return the next meaningful token.
		}

		// Outside braces: a newline terminates the command line.
		l.mode = LanguageMode
		tok := l.createSimpleToken(NEWLINE, "\n", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		return tok
	case '}':
		// Only recognize } as structural if it closes a Devcmd brace
		if l.braceLevel > 0 {
			tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
			l.braceLevel--
			// Only decrement patternLevel when we close the outermost brace that opened the @when/@try block
			if l.braceLevel == 0 {
				l.mode = LanguageMode
				if l.patternLevel > 0 {
					l.patternLevel--
				}
			} else if l.patternLevel > 0 {
				// Pop mode from stack to return to pattern mode for nested braces
				l.popMode()
			}
			l.readChar()
			l.updateTokenEnd(&tok)
			return tok
		}
		// Otherwise, treat as shell content
		return l.lexShellText(start)
	case '@':
		// Handle decorator in command mode - switch back to LanguageMode temporarily
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		// Don't switch mode here - the decorator parsing will handle it
		l.updateTokenEnd(&tok)
		return tok
	default:
		// All other content is handled as shell text
		return l.lexShellText(start)
	}
}

// shouldEnterCommandMode determines if we should switch to command mode after ':'
func (l *Lexer) shouldEnterCommandMode() bool {
	// Save current state
	pos, readPos, ch := l.position, l.readPos, l.ch
	defer func() { l.position, l.readPos, l.ch = pos, readPos, ch }()

	l.skipWhitespace()

	// Don't enter command mode if we see structural tokens or EOF
	// But DO enter command mode if we see anything that looks like shell content
	switch l.ch {
	case '{', '@', '\n', 0, '}':
		return false
	case '(':
		// Check if it's part of a decorator like @timeout(30s)
		return false
	default:
		return true
	}
}

// lexShellText captures shell content as a single token
// It handles POSIX quoting rules and line continuations structurally
func (l *Lexer) lexShellText(start int) Token {
	startLine, startColumn := l.line, l.column
	startOffset := start

	var inSingleQuotes, inDoubleQuotes, inBackticks bool
	var prevWasBackslash bool

	for {
		switch l.ch {
		case 0:
			// EOF - return what we have
			return l.makeShellToken(start, startOffset, startLine, startColumn)

		case '\n':
			// Handle line continuation outside quotes
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks && prevWasBackslash {
				prevWasBackslash = false
				l.readChar()
				// Skip following whitespace per GNU make behavior
				for l.ch == ' ' || l.ch == '\t' {
					l.readChar()
				}
				continue
			}

			// Newlines inside quotes are part of shell text
			if inSingleQuotes || inDoubleQuotes || inBackticks {
				prevWasBackslash = false
				l.readChar()
				continue
			}

			// Otherwise, newline ends shell text
			prevWasBackslash = false
			return l.makeShellToken(start, startOffset, startLine, startColumn)

		case '\'':
			if !inDoubleQuotes && !inBackticks {
				inSingleQuotes = !inSingleQuotes
			}
			prevWasBackslash = false
			l.readChar()

		case '"':
			if !inSingleQuotes && !inBackticks {
				inDoubleQuotes = !inDoubleQuotes
			}
			prevWasBackslash = false
			l.readChar()

		case '`':
			if !inSingleQuotes && !inDoubleQuotes {
				inBackticks = !inBackticks
			}
			prevWasBackslash = false
			l.readChar()

		case '\\':
			if inSingleQuotes {
				// In single quotes, backslash is literal
				prevWasBackslash = false
				l.readChar()
			} else {
				// Mark potential line continuation
				prevWasBackslash = true
				l.readChar()
				// In double quotes or backticks, consume escaped character
				if (inDoubleQuotes || inBackticks) && l.ch != 0 {
					prevWasBackslash = false // Not a line continuation
					l.readChar()
				}
			}

		case '}':
			// Structural boundary only if not in quotes and we're in a block
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks && l.braceLevel > 0 {
				prevWasBackslash = false
				return l.makeShellToken(start, startOffset, startLine, startColumn)
			}
			prevWasBackslash = false
			l.readChar()

		case '@':
			// Decorator boundary only if not in quotes
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks {
				prevWasBackslash = false
				return l.makeShellToken(start, startOffset, startLine, startColumn)
			}
			prevWasBackslash = false
			l.readChar()

		case ';':
			// Pattern boundary check only if not in quotes
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks &&
			   l.patternLevel > 0 && l.isPatternBreak() {
				prevWasBackslash = false
				l.readChar() // include semicolon
				return l.makeShellTokenForPattern(start, startOffset, startLine, startColumn)
			}
			prevWasBackslash = false
			l.readChar()

		default:
			// Any other character resets line continuation
			if l.ch != ' ' && l.ch != '\t' {
				prevWasBackslash = false
			}
			l.readChar()
		}
	}
}

// makeShellToken creates a shell text token from the captured range
// makeShellToken creates a shell text token from the captured range
func (l *Lexer) makeShellToken(start, startOffset, startLine, startColumn int) Token {
	// Get the raw text
	rawText := l.input[start:l.position]

	// Process line continuations
	processedText := l.processLineContinuations(rawText)

	// Trim whitespace
	processedText = strings.TrimSpace(processedText)

	// Don't emit empty tokens - but ensure we've actually consumed something
	if processedText == "" {
		// If we haven't moved forward, we need to consume at least one character
		// to avoid infinite loops
		if l.position == start && l.ch != 0 {
			l.readChar()
		}
		return l.lexToken()
	}

	return Token{
		Type:      SHELL_TEXT,
		Value:     processedText,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Raw:       rawText,      // Keep original for formatting tools
		Semantic:  SemShellText,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: startOffset},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// processLineContinuations handles backslash-newline sequences in shell text
func (l *Lexer) processLineContinuations(text string) string {
	// Fast path: no backslashes means no continuations
	if !strings.Contains(text, "\\") {
		return text
	}

	var result strings.Builder
	result.Grow(len(text))

	i := 0
	inSingleQuotes := false

	for i < len(text) {
		ch := text[i]

		// Track single quote state
		if ch == '\'' {
			inSingleQuotes = !inSingleQuotes
			result.WriteByte(ch)
			i++
			continue
		}

		// In single quotes, everything is literal
		if inSingleQuotes {
			result.WriteByte(ch)
			i++
			continue
		}

		// Check for line continuation outside single quotes
		if ch == '\\' && i+1 < len(text) && text[i+1] == '\n' {
			// Skip the backslash and newline
			i += 2

			// Skip following whitespace
			for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
				i++
			}

			// Add a space to join the lines
			if result.Len() > 0 && i < len(text) {
				lastCh := result.String()[result.Len()-1]
				if lastCh != ' ' && lastCh != '\t' {
					result.WriteByte(' ')
				}
			}
		} else {
			result.WriteByte(ch)
			i++
		}
	}

	return result.String()
}

// makeShellTokenForPattern creates a shell token for pattern mode
func (l *Lexer) makeShellTokenForPattern(start, startOffset, startLine, startColumn int) Token {
	// Adjust position to exclude the semicolon we just consumed
	endPos := l.position - 1
	rawText := l.input[start:endPos]

	// Trim trailing whitespace
	rawText = strings.TrimSpace(rawText)

	// Don't emit empty tokens
	if rawText == "" {
		// Switch back to pattern mode for next token
		l.mode = PatternMode
		return l.lexToken()
	}

	tok := Token{
		Type:      SHELL_TEXT,
		Value:     rawText,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column - 1, // Adjust for semicolon
		Raw:       rawText,
		Semantic:  SemShellText,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: startOffset},
			End:   SourcePosition{Line: l.line, Column: l.column - 1, Offset: endPos},
		},
	}

	// Switch back to pattern mode
	l.mode = PatternMode

	return tok
}

// lexIdentifierOrKeyword lexes identifiers and keywords with optimized lookahead
func (l *Lexer) lexIdentifierOrKeyword(start int) Token {
	startLine, startColumn := l.line, l.column

	// Use readIdentifier to handle the full identifier
	l.readIdentifier()

	// Get value as byte slice first, then convert only once
	valueBytes := l.input[start:l.position]
	value := string(valueBytes) // Single allocation

	var tokenType TokenType
	var semantic SemanticTokenType

	// Check for boolean literals first
	if value == "true" || value == "false" {
		return Token{
			Type:      BOOLEAN,
			Value:     value,
			Line:      startLine,
			Column:    startColumn,
			EndLine:   l.line,
			EndColumn: l.column,
			Semantic:  SemBoolean,
			Span: SourceSpan{
				Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
				End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
			},
		}
	}

	// Check for keywords
	if keywordType, isKeyword := keywords[value]; isKeyword {
		tokenType = keywordType
		semantic = SemKeyword
		// Special handling for pattern-matching decorators
		if value == "when" || value == "try" {
			// Track that we're entering a pattern-matching decorator
			l.patternLevel++
		}
	} else {
		tokenType = IDENTIFIER
		semantic = SemCommand // Default to command name
	}

	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  semantic,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// Keywords map - includes pattern-matching decorator keywords
var keywords = map[string]TokenType{
	"var":   VAR,
	"stop":  STOP,
	"watch": WATCH,
	"when":  WHEN,
	"try":   TRY,
}

// lexNumberOrDuration lexes numbers and durations with optimized lookahead
func (l *Lexer) lexNumberOrDuration(start int) Token {
	startLine, startColumn := l.line, l.column

	// Fast path: use lookahead to scan number in one pass
	pos := l.position
	input := l.input
	inputLen := len(input)

	// Handle negative numbers
	if pos < inputLen && input[pos] == '-' {
		pos++
		l.readChar()
	}

	// Scan integer part using lookahead
	for pos < inputLen && input[pos] >= '0' && input[pos] <= '9' {
		pos++
	}

	// Check for decimal part
	if pos < inputLen && input[pos] == '.' && pos+1 < inputLen && input[pos+1] >= '0' && input[pos+1] <= '9' {
		pos++ // consume '.'
		for pos < inputLen && input[pos] >= '0' && input[pos] <= '9' {
			pos++
		}
	}

	// Update lexer position efficiently
	for l.position < pos {
		l.readChar()
	}

	// Check for duration unit using optimized lookahead
	isDuration := false
	if l.position < inputLen {
		ch := l.input[l.position]
		switch ch {
		case 'n':
			// nanoseconds: ns
			if l.position+1 < inputLen && l.input[l.position+1] == 's' {
				isDuration = true
				l.readChar()
				l.readChar()
			}
		case 'u':
			// microseconds: us (instead of μs)
			if l.position+1 < inputLen && l.input[l.position+1] == 's' {
				isDuration = true
				l.readChar()
				l.readChar()
			}
		case 'm':
			// milliseconds: ms OR minutes: m
			if l.position+1 < inputLen && l.input[l.position+1] == 's' {
				isDuration = true
				l.readChar()
				l.readChar()
			} else if l.position+1 >= inputLen || !isLetter[l.input[l.position+1]] {
				isDuration = true
				l.readChar()
			}
		case 's':
			// seconds: s
			if l.position+1 >= inputLen || !isLetter[l.input[l.position+1]] {
				isDuration = true
				l.readChar()
			}
		case 'h':
			// hours: h
			if l.position+1 >= inputLen || !isLetter[l.input[l.position+1]] {
				isDuration = true
				l.readChar()
			}
		}
	}

	// Single string allocation
	value := string(l.input[start:l.position])

	tokenType := NUMBER
	if isDuration {
		tokenType = DURATION
	}

	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemNumber,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexString lexes string literals with optimized lookahead and minimal allocations
func (l *Lexer) lexString(quote byte, stringType StringType, start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar() // consume opening quote

	input := l.input
	inputLen := len(input)

	// Fast path: scan for simple strings without escapes using lookahead
	pos := l.position
	hasEscapes := false

	// For single-quoted strings, `\` is not an escape character.
	if stringType != SingleQuoted {
		for pos < inputLen {
			ch := input[pos]
			if ch == quote {
				break
			}
			if ch == 0 {
				break
			}
			if ch == '\\' {
				hasEscapes = true
				break
			}
			pos++
		}
	} else {
		// For single-quoted strings, just find the next quote.
		for pos < inputLen {
			if input[pos] == quote {
				break
			}
			pos++
		}
	}

	var value string

	if !hasEscapes && pos < inputLen && input[pos] == quote {
		// Fast path: simple string without escapes
		value = string(input[l.position:pos]) // Single allocation

		// Update lexer position efficiently
		for l.position < pos {
			l.readChar()
		}

		if l.ch == quote {
			l.readChar() // consume closing quote
		}
	} else if stringType == SingleQuoted {
		// For single-quoted strings, no escape processing at all
		valueStart := l.position

		// Just consume characters until closing quote
		for l.ch != quote && l.ch != 0 {
			l.readChar()
		}

		value = l.input[valueStart:l.position]

		if l.ch == quote {
			l.readChar() // consume closing quote
		}
	} else {
		// Slow path: string with escapes or complex cases (not single-quoted)
		var escaped strings.Builder
		valueStart := l.position

		for l.ch != quote && l.ch != 0 {
			if l.ch == '\\' {
				if !hasEscapes {
					hasEscapes = true
					escaped.WriteString(l.input[valueStart:l.position])
				} else {
					escaped.WriteString(l.input[valueStart:l.position])
				}
				l.readChar()
				if l.ch == 0 {
					break
				}
				escapeStr := l.handleEscape(stringType)
				escaped.WriteString(escapeStr)
				l.readChar()
				valueStart = l.position
			} else {
				l.readChar()
			}
		}

		if hasEscapes {
			escaped.WriteString(l.input[valueStart:l.position])
			value = escaped.String()
		} else {
			value = l.input[valueStart:l.position] // String slicing
		}

		if l.ch == quote {
			l.readChar() // consume closing quote
		}
	}

	return Token{
		Type:       STRING,
		Value:      value,
		Line:       startLine,
		Column:     startColumn,
		EndLine:    l.line,
		EndColumn:  l.column,
		StringType: stringType,
		Raw:        l.input[start:l.position], // String slicing
		Semantic:   SemString,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexComment lexes single-line comments
func (l *Lexer) lexComment(start int) Token {
	startLine, startColumn := l.line, l.column
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return Token{
		Type:      COMMENT,
		Value:     l.input[start:l.position], // String slicing
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexMultilineComment lexes multi-line comments
func (l *Lexer) lexMultilineComment(start int) Token {
	startLine, startColumn := l.line, l.column
	l.readChar() // consume '/'
	l.readChar() // consume '*'

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
		Value:     l.input[start:l.position], // String slicing
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemComment,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
}

// lexSingleChar lexes single character tokens
func (l *Lexer) lexSingleChar(start int) Token {
	startLine, startColumn := l.line, l.column
	char := l.ch
	l.readChar()

	token := Token{
		Type:      IDENTIFIER,
		Value:     string(char),
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemOperator,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	return token
}

// isPatternBreak checks if we're at a pattern boundary (pattern identifier followed by ':')
func (l *Lexer) isPatternBreak() bool {
	// Save current state
	pos, readPos, ch := l.position, l.readPos, l.ch
	defer func() { l.position, l.readPos, l.ch = pos, readPos, ch }()

	// Skip the semicolon
	l.readChar()

	// Skip whitespace
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}

	// Check if we have an identifier or wildcard (*)
	if !isLetter[l.ch] && l.ch != '*' {
		return false
	}

	if l.ch == '*' {
		// Wildcard pattern
		l.readChar()
	} else {
		// Scan identifier - any identifier is valid for patterns
		for l.ch != 0 && isIdentPart[l.ch] {
			l.readChar()
		}
	}

	// Skip whitespace after identifier/wildcard
	for l.ch == ' ' || l.ch == '\t' {
		l.readChar()
	}

	// Check if followed by ':'
	return l.ch == ':'
}

// Helper methods for creating tokens with proper position tracking

// createSimpleToken creates a token with basic type and value
func (l *Lexer) createSimpleToken(tokenType TokenType, value string, start, startLine, startColumn int) Token {
	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   startLine, // Will be updated by updateTokenEnd
		EndColumn: startColumn, // Will be updated by updateTokenEnd
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: startLine, Column: startColumn, Offset: start}, // Will be updated
		},
	}
}

// createTokenWithSemantic creates a token with specific semantic type
func (l *Lexer) createTokenWithSemantic(tokenType TokenType, semantic SemanticTokenType, value string, start, startLine, startColumn int) Token {
	return Token{
		Type:      tokenType,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   startLine, // Will be updated by updateTokenEnd
		EndColumn: startColumn, // Will be updated by updateTokenEnd
		Semantic:  semantic,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: startLine, Column: startColumn, Offset: start}, // Will be updated
		},
	}
}

// updateTokenEnd updates the end position of a token
func (l *Lexer) updateTokenEnd(token *Token) {
	token.EndLine = l.line
	token.EndColumn = l.column
	token.Span.End = SourcePosition{Line: l.line, Column: l.column, Offset: l.position}
}

// Helper methods

func (l *Lexer) readIdentifier() {
	for l.ch != 0 && l.ch < 128 && isIdentPart[l.ch] {
		l.readChar()
	}
}

// skipWhitespace with optimized lookahead
func (l *Lexer) skipWhitespace() {
	input := l.input
	inputLen := len(input)

	// Fast lookahead for whitespace skipping
	for l.position < inputLen {
		ch := input[l.position]
		if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\f' {
			break
		}
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

	// Position tracking: increment column before handling newline
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0 // Reset to 0, will be incremented to 1 on next readChar
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) peekCharAt(n int) byte {
	pos := l.readPos + n - 1
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
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
	}
}

func (l *Lexer) handleEscape(stringType StringType) string {
	switch stringType {
	case SingleQuoted:
		// In single-quoted strings, a backslash is a literal character.
		// It does not escape anything. This function should not be called
		// for single-quoted strings if the logic in lexString is correct.
		return "\\" + string(l.ch)
	case DoubleQuoted:
		switch l.ch {
		case 'n':
			return "\n"
		case 't':
			return "\t"
		case 'r':
			return "\r"
		case '\\':
			return "\\"
		case '"':
			return "\""
		default:
			return "\\" + string(l.ch)
		}
	case Backtick:
		switch l.ch {
		case 'n':
			return "\n"
		case 't':
			return "\t"
		case 'r':
			return "\r"
		case 'b':
			return "\b"
		case 'f':
			return "\f"
		case 'v':
			return "\v"
		case '0':
			return "\x00"
		case '\\':
			return "\\"
		case '`':
			return "`"
		case '"':
			return "\""
		case '\'':
			return "'"
		case 'x':
			return l.readHexEscape()
		case 'u':
			if l.peekChar() == '{' {
				return l.readUnicodeEscape()
			}
			return "\\u"
		default:
			return "\\" + string(l.ch)
		}
	}
	return "\\" + string(l.ch)
}

func (l *Lexer) readHexEscape() string {
	if !isHexDigit[l.peekChar()] {
		return "\\x"
	}
	l.readChar()
	hex1 := l.ch
	l.readChar()
	if !isHexDigit[l.ch] {
		return "\\x" + string(hex1)
	}
	hex2 := l.ch
	value := hexValue(hex1)*16 + hexValue(hex2)
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
	hexDigits := l.input[start:l.position]
	l.readChar()
	if len(hexDigits) == 0 {
		return "\\u{}"
	}
	var value rune
	for _, ch := range hexDigits {
		value = value*16 + rune(hexValue(byte(ch)))
	}
	if !utf8.ValidRune(value) {
		return "\\u{" + hexDigits + "}"
	}
	return string(value)
}

func hexValue(ch byte) int {
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
