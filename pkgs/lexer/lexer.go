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
	}
	l.readChar()
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
func (l *Lexer) lexCommandMode(start int) Token {
	startLine, startColumn := l.line, l.column

	switch l.ch {
	case 0:
		l.mode = LanguageMode
		return l.createSimpleToken(EOF, "", start, startLine, startColumn)
	case '\n':
		if l.braceLevel > 0 {
			// Inside a command block `{}`, newlines separate commands but are NOT emitted as tokens
			// This applies to both regular command blocks and pattern blocks
			if l.patternLevel > 0 {
				// If we're in a pattern block (like @when), the newline switches us
				// back to PatternMode to parse the next case.
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
			if l.braceLevel == 0 {
				l.mode = LanguageMode
				if l.patternLevel > 0 {
					l.patternLevel--
				}
			} else if l.patternLevel > 0 {
				l.mode = PatternMode
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

// lexShellText with optimized lookahead and proper line continuation handling
func (l *Lexer) lexShellText(start int) Token {
	startLine, startColumn := l.line, l.column
	startOffset := start

	// Build the processed text by handling line continuations properly
	var processedBuilder strings.Builder
	var segments []ShellSegment

	// Fast path check for simple shell text (no line continuations)
	pos := l.position
	input := l.input
	inputLen := len(input)
	hasLineContinuation := false

	// Quick scan to see if we have line continuations
	endPos := pos
	inSingleQuotes := false
	for endPos < inputLen {
		ch := input[endPos]

		// Track single quote state
		if ch == '\'' {
			inSingleQuotes = !inSingleQuotes
		}

		if ch == '}' && l.braceLevel > 0 {
			break
		}
		if ch == '\n' {
			break
		}
		if ch == '@' {
			// Stop at decorator start
			break
		}

		// Only process line continuations outside of single quotes
		if !inSingleQuotes && ch == '\\' && endPos+1 < inputLen && input[endPos+1] == '\n' {
			hasLineContinuation = true
			break
		}

		// Special handling for pattern-matching contexts
		if l.patternLevel > 0 && ch == ';' {
			// In pattern mode, semicolon separates patterns
			// Look ahead to see if we have a pattern (identifier or *) after whitespace
			lookaheadPos := endPos + 1

			// Skip whitespace
			for lookaheadPos < inputLen && (input[lookaheadPos] == ' ' || input[lookaheadPos] == '\t') {
				lookaheadPos++
			}

			// Check if we have an identifier or wildcard followed by ':'
			if lookaheadPos < inputLen {
				if input[lookaheadPos] == '*' {
					// Wildcard pattern
					lookaheadPos++
				} else if isLetter[input[lookaheadPos]] {
					// Scan identifier - any identifier is valid for patterns
					for lookaheadPos < inputLen && isIdentPart[input[lookaheadPos]] {
						lookaheadPos++
					}
				} else {
					// Not a pattern, continue normal shell parsing
					endPos++
					continue
				}

				// Skip whitespace after identifier/wildcard
				for lookaheadPos < inputLen && (input[lookaheadPos] == ' ' || input[lookaheadPos] == '\t') {
					lookaheadPos++
				}

				// If we find ':', this is a new pattern
				if lookaheadPos < inputLen && input[lookaheadPos] == ':' {
					endPos++ // Include the semicolon in current shell text
					break
				}
			}
		}

		endPos++
	}

	// Fast path for simple shell text without line continuations
	if !hasLineContinuation && endPos > pos {
		// Update lexer position efficiently
		for l.position < endPos {
			l.readChar()
		}

		// Single allocation for the token value
		finalText := string(input[start:endPos])

		// Trim trailing whitespace if we stopped at '}'
		if endPos < inputLen && input[endPos] == '}' && l.braceLevel > 0 {
			finalText = strings.TrimRight(finalText, " \t\r\f")
		}

		// Trim trailing semicolon if we're in pattern mode and found a pattern break
		if l.patternLevel > 0 && endPos < inputLen && strings.HasSuffix(finalText, ";") {
			finalText = strings.TrimSuffix(finalText, ";")
			finalText = strings.TrimSpace(finalText) // Also trim whitespace
			l.mode = PatternMode // Switch back to pattern mode for next token
		}

		// Don't emit empty tokens
		if strings.TrimSpace(finalText) == "" {
			return l.lexToken()
		}

		return Token{
			Type:      SHELL_TEXT,
			Value:     finalText,
			Line:      startLine,
			Column:    startColumn,
			EndLine:   l.line,
			EndColumn: l.column,
			Raw:       string(input[startOffset:l.position]),
			Semantic:  SemShellText,
			Span: SourceSpan{
				Start: SourcePosition{Line: startLine, Column: startColumn, Offset: startOffset},
				End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
			},
		}
	}

	// Slow path for shell text with line continuations
	processedOffset := 0
	segmentStart := l.position
	segmentStartLine, segmentStartColumn := l.line, l.column
	var segmentRaw strings.Builder
	var segmentProcessed strings.Builder
	inSingleQuotes = false

	for l.ch != 0 {
		// Stop at structural boundaries
		if l.ch == '}' && l.braceLevel > 0 {
			break
		}
		if l.ch == '\n' {
			// In command mode, newlines break shell text into separate tokens
			break
		}
		if l.ch == '@' {
			// Stop at decorator start
			break
		}

		// Track single quote state
		if l.ch == '\'' {
			inSingleQuotes = !inSingleQuotes
		}

		// Check for pattern breaks in pattern-matching mode
		if l.patternLevel > 0 && l.ch == ';' {
			// Look ahead for potential pattern
			if l.isPatternBreak() {
				// Include the semicolon in current segment but don't add to processed text
				// The semicolon acts as a separator, not content
				segmentRaw.WriteByte(l.ch)
				l.readChar()
				break
			}
		}

		// Only process line continuations outside of single quotes
		if !inSingleQuotes && l.ch == '\\' && l.peekChar() == '\n' {
			// Line continuation: handle properly per the spec - remove \\\n and following whitespace
			// Add the raw continuation characters for source tracking
			segmentRaw.WriteByte('\\')
			l.readChar() // consume '\'
			segmentRaw.WriteByte('\n')
			l.readChar() // consume '\n'

			// Skip any following whitespace and record it in raw but don't add to processed
			for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\f' {
				segmentRaw.WriteByte(l.ch)
				l.readChar()
			}

			// Finish current segment if it has content
			if segmentProcessed.Len() > 0 {
				segments = append(segments, ShellSegment{
					Text:    segmentProcessed.String(),
					RawText: segmentRaw.String(),
					Span: SourceSpan{
						Start: SourcePosition{Line: segmentStartLine, Column: segmentStartColumn, Offset: segmentStart},
						End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
					},
					Offset: processedOffset,
				})

				processedBuilder.WriteString(segmentProcessed.String())
				processedOffset += segmentProcessed.Len()
			}

			// Line continuation doesn't add space - it just joins lines
			// Reset for next segment
			segmentRaw.Reset()
			segmentProcessed.Reset()
			segmentStart = l.position
			segmentStartLine, segmentStartColumn = l.line, l.column
			continue
		}

		// Normal character
		segmentRaw.WriteByte(l.ch)
		segmentProcessed.WriteByte(l.ch)
		l.readChar()
	}

	// Finish final segment
	if segmentProcessed.Len() > 0 {
		segments = append(segments, ShellSegment{
			Text:    segmentProcessed.String(),
			RawText: segmentRaw.String(),
			Span: SourceSpan{
				Start: SourcePosition{Line: segmentStartLine, Column: segmentStartColumn, Offset: segmentStart},
				End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
			},
			Offset: processedOffset,
		})

		processedBuilder.WriteString(segmentProcessed.String())
	}

	finalText := processedBuilder.String()

	// Trim trailing whitespace if we stopped at '}'
	if l.ch == '}' && l.braceLevel > 0 {
		finalText = strings.TrimRight(finalText, " \t\r\f")
	}

	// Handle pattern breaks
	if l.patternLevel > 0 && strings.HasSuffix(finalText, ";") {
		finalText = strings.TrimSuffix(finalText, ";")
		finalText = strings.TrimSpace(finalText) // Also trim whitespace
		l.mode = PatternMode // Switch back to pattern mode for next token
	}

	// Don't emit empty tokens
	if strings.TrimSpace(finalText) == "" {
		return l.lexToken()
	}

	return Token{
		Type:      SHELL_TEXT,
		Value:     finalText,
		Line:      startLine,
		Column:    startColumn,
		EndLine:   l.line,
		EndColumn: l.column,
		Raw:       l.input[startOffset:l.position], // Use string slicing
		Semantic:  SemShellText,
		Span: SourceSpan{
			Start: SourcePosition{Line: startLine, Column: startColumn, Offset: startOffset},
			End:   SourcePosition{Line: l.line, Column: l.column, Offset: l.position},
		},
		ShellSegments: segments,
	}
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
	} else {
		// Slow path: string with escapes or complex cases
		var escaped strings.Builder
		valueStart := l.position

		for l.ch != quote && l.ch != 0 {
			// In single-quoted strings, backslash is a literal character and does not start an escape sequence.
			if stringType != SingleQuoted && l.ch == '\\' {
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
