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

// Lexer tokenizes Devcmd source code with state machine-based parsing
type Lexer struct {
	input        string // Changed from []byte to string for efficiency
	position     int
	readPos      int
	ch           byte
	line         int
	column       int
	stateMachine *StateMachine // State machine for parsing context
	braceLevel   int           // Track brace nesting for command mode
	patternLevel int           // Track pattern-matching decorator nesting
}

// New creates a new lexer instance with state machine
func New(input string) *Lexer {
	l := &Lexer{
		input:        input,
		line:         1,
		column:       0, // Start at column 0, will be incremented to 1 on first readChar
		stateMachine: NewStateMachine(),
		braceLevel:   0,
		patternLevel: 0,
	}
	l.readChar()
	return l
}

// NewWithDebug creates a new lexer instance with debugging enabled
func NewWithDebug(input string) *Lexer {
	l := New(input)
	l.stateMachine.SetDebug(true)
	return l
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

// lexToken performs token lexing with state machine-aware logic
func (l *Lexer) lexToken() Token {
	// Skip whitespace in most modes
	mode := l.stateMachine.GetMode()
	if mode == LanguageMode || mode == PatternMode {
		l.skipWhitespace()
	}

	start := l.position

	// Check if we should enter shell content mode based on current state
	currentState := l.stateMachine.Current()
	if l.shouldLexShellContent(currentState) {
		return l.lexShellText(start)
	}

	switch mode {
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

// shouldLexShellContent determines if we should lex shell content based on state
func (l *Lexer) shouldLexShellContent(state LexerState) bool {
	// Don't lex shell content if we're at structural tokens
	switch l.ch {
	case 0, '\n', '{', '}', '@':
		return false
	case ':':
		// Colon is structural in pattern mode
		if l.stateMachine.GetMode() == PatternMode {
			return false
		}
		return false
	case '*':
		// Asterisk is structural in pattern mode
		if l.stateMachine.GetMode() == PatternMode {
			return false
		}
		// In other modes, continue to check state
	}

	// Lex shell content in these states when we're not at structural boundaries
	switch state {
	case StateAfterColon:
		// After colon, if we see content that isn't a decorator or brace, it's shell content
		return l.ch != '@' && l.ch != '{'
	case StateAfterPatternColon:
		// After pattern colon, if we see content that isn't a decorator or brace, it's shell content
		return l.ch != '@' && l.ch != '{'
	case StateCommandContent:
		// In command content, everything except structural tokens is shell content
		return true
	case StateAfterDecorator:
		// After decorator, if we see content that isn't a brace, it might be shell content
		return l.ch != '{'
	case StatePatternBlock:
		// In pattern block, don't lex shell content - parse pattern structure
		return false
	default:
		return false
	}
}

// lexLanguageMode handles structural Devcmd syntax
func (l *Lexer) lexLanguageMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		tok := l.createSimpleToken(NEWLINE, "\n", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(NEWLINE, "\n")
		return tok
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COLON, ":")
		return tok
	case '=':
		tok := l.createSimpleToken(EQUALS, "=", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(EQUALS, "=")
		return tok
	case ',':
		tok := l.createSimpleToken(COMMA, ",", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COMMA, ",")
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LPAREN, "(")
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RPAREN, ")")
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++
		l.readChar()
		l.skipWhitespace() // Skip whitespace after opening brace
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RBRACE, "}")
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(ASTERISK, "*")
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
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		// In pattern mode, consume newlines but don't emit tokens
		l.readChar()
		l.skipWhitespace()
		l.updateStateMachine(NEWLINE, "\n")
		return l.lexToken() // Get the next meaningful token
	case ':':
		tok := l.createSimpleToken(COLON, ":", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(COLON, ":")
		return tok
	case '}':
		tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
		if l.braceLevel > 0 {
			l.braceLevel--
		}
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RBRACE, "}")
		return tok
	case '{':
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++
		l.readChar()
		l.skipWhitespace()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	case '@':
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case '*':
		// Always treat * as ASTERISK token for wildcard patterns
		tok := l.createSimpleToken(ASTERISK, "*", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(ASTERISK, "*")
		return tok
	case '(':
		tok := l.createSimpleToken(LPAREN, "(", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LPAREN, "(")
		return tok
	case ')':
		tok := l.createSimpleToken(RPAREN, ")", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(RPAREN, ")")
		return tok
	case '"':
		return l.lexString('"', DoubleQuoted, start)
	case '\'':
		return l.lexString('\'', SingleQuoted, start)
	case '`':
		return l.lexString('`', Backtick, start)
	default:
		// In pattern mode, identifiers should be treated as pattern identifiers
		if l.ch < 128 && isIdentStart[l.ch] {
			return l.lexPatternIdentifier(start)
		} else if l.ch >= 128 && isLetter[l.ch] {
			return l.lexPatternIdentifier(start)
		} else if isDigit[l.ch] || (l.ch == '-' && l.peekChar() != 0 && isDigit[l.peekChar()]) {
			return l.lexNumberOrDuration(start)
		} else {
			return l.lexSingleChar(start)
		}
	}
}

// lexPatternIdentifier lexes identifiers in pattern mode
func (l *Lexer) lexPatternIdentifier(start int) Token {
	startLine, startColumn := l.line, l.column

	// Use readIdentifier to handle the full identifier
	l.readIdentifier()

	value := string(l.input[start:l.position])

	tok := Token{
		Type:      IDENTIFIER,
		Value:     value,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Semantic:  SemPattern, // Mark as pattern semantic in pattern mode
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: start},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
	}
	l.updateStateMachine(IDENTIFIER, value)
	return tok
}

// lexCommandMode handles shell content capture with proper newline handling
func (l *Lexer) lexCommandMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		tok := l.createSimpleToken(EOF, "", start, startLine, startColumn)
		l.updateStateMachine(EOF, "")
		return tok
	case '\n':
		if l.braceLevel > 0 {
			// Inside a command block `{}`, consume newlines and continue lexing
			l.readChar()       // Consume '\n'
			l.skipWhitespace() // Consume all whitespace before the next token
			l.updateStateMachine(NEWLINE, "\n")
			return l.lexToken() // Return the next meaningful token
		}

		// Outside braces: a newline terminates the command line
		tok := l.createSimpleToken(NEWLINE, "\n", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(NEWLINE, "\n")
		return tok
	case '}':
		// Only recognize } as structural if it closes a Devcmd brace
		if l.braceLevel > 0 {
			tok := l.createSimpleToken(RBRACE, "}", start, startLine, startColumn)
			l.braceLevel--
			l.readChar()
			l.updateTokenEnd(&tok)
			l.updateStateMachine(RBRACE, "}")
			return tok
		}
		// Otherwise, treat as shell content
		return l.lexShellText(start)
	case '@':
		// Handle decorator in command mode - switch back to LanguageMode temporarily
		tok := l.createTokenWithSemantic(AT, SemOperator, "@", start, startLine, startColumn)
		l.readChar()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(AT, "@")
		return tok
	case '{':
		// Handle opening brace in command mode
		tok := l.createSimpleToken(LBRACE, "{", start, startLine, startColumn)
		l.braceLevel++
		l.readChar()
		l.skipWhitespace()
		l.updateTokenEnd(&tok)
		l.updateStateMachine(LBRACE, "{")
		return tok
	default:
		// All other content is handled as shell text
		return l.lexShellText(start)
	}
}

// updateStateMachine notifies the state machine about the current token
func (l *Lexer) updateStateMachine(tokenType TokenType, value string) {
	if _, err := l.stateMachine.HandleToken(tokenType, value); err != nil {
		// In production, you might want to handle this error differently
		// For debugging, log state machine errors
		if l.stateMachine.debug {
			println("State machine error:", err.Error(), "- token:", tokenType.String(), "value:", value)
		}
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
			tok := l.makeShellToken(start, startOffset, startLine, startColumn)
			l.updateStateMachine(SHELL_TEXT, tok.Value)
			return tok

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
			tok := l.makeShellToken(start, startOffset, startLine, startColumn)
			l.updateStateMachine(SHELL_TEXT, tok.Value)
			return tok

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
				tok := l.makeShellToken(start, startOffset, startLine, startColumn)
				l.updateStateMachine(SHELL_TEXT, tok.Value)
				return tok
			}
			prevWasBackslash = false
			l.readChar()

		case '@':
			// Decorator boundary only if not in quotes
			if !inSingleQuotes && !inDoubleQuotes && !inBackticks {
				prevWasBackslash = false
				tok := l.makeShellToken(start, startOffset, startLine, startColumn)
				l.updateStateMachine(SHELL_TEXT, tok.Value)
				return tok
			}
			prevWasBackslash = false
			l.readChar()

		default:
			// Any other character resets line continuation and continues as shell content
			// This includes semicolons - they are always part of shell content now
			if l.ch != ' ' && l.ch != '\t' {
				prevWasBackslash = false
			}
			l.readChar()
		}
	}
}

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
		tok := Token{
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
		l.updateStateMachine(BOOLEAN, value)
		return tok
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

	tok := Token{
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
	l.updateStateMachine(tokenType, value)
	return tok
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

	tok := Token{
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
	l.updateStateMachine(tokenType, value)
	return tok
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

	tok := Token{
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
	l.updateStateMachine(STRING, value)
	return tok
}

// lexComment lexes single-line comments
func (l *Lexer) lexComment(start int) Token {
	startLine, startColumn := l.line, l.column
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	tok := Token{
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
	l.updateStateMachine(COMMENT, tok.Value)
	return tok
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

	tok := Token{
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
	l.updateStateMachine(MULTILINE_COMMENT, tok.Value)
	return tok
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
	l.updateStateMachine(IDENTIFIER, token.Value)
	return token
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
