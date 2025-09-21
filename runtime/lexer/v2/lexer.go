package v2

import (
	"time"
	"unicode/utf8"
)

// LexerMode represents the lexing mode
type LexerMode int

const (
	ModeCommand LexerMode = iota // Command mode: organized tasks (commands.cli)
	ModeScript                   // Script mode: direct execution scripts
)

// LexerOpt represents a lexer configuration option
type LexerOpt func(*LexerConfig)

// TimingMode controls timing behavior
type TimingMode int

const (
	TimingOff  TimingMode = iota // No timing (fastest, default)
	TimingOn                     // Basic timing for buffer operations
	TimingFine                   // Fine-grained timing with debug stats
)

// LexerConfig holds lexer configuration
type LexerConfig struct {
	debug  bool
	mode   LexerMode
	timing TimingMode
}

// WithDebug enables debug telemetry (will allocate for token stats)
func WithDebug() LexerOpt {
	return func(c *LexerConfig) {
		c.debug = true
	}
}

// WithScriptMode sets the lexer to script mode (direct execution)
func WithScriptMode() LexerOpt {
	return func(c *LexerConfig) {
		c.mode = ModeScript
	}
}

// WithCommandMode sets the lexer to command mode (organized tasks) - default
func WithCommandMode() LexerOpt {
	return func(c *LexerConfig) {
		c.mode = ModeCommand
	}
}

// WithTiming enables basic timing (buffer-level)
func WithTiming() LexerOpt {
	return func(c *LexerConfig) {
		c.timing = TimingOn
	}
}

// WithFineGrainTiming enables detailed timing with debug stats
func WithFineGrainTiming() LexerOpt {
	return func(c *LexerConfig) {
		c.timing = TimingFine
	}
}

// WithNoTiming disables all timing (fastest performance, default)
func WithNoTiming() LexerOpt {
	return func(c *LexerConfig) {
		c.timing = TimingOff
	}
}

// TokenStats holds per-token timing statistics (debug mode only)
type TokenStats struct {
	Type      TokenType
	Count     int
	TotalTime time.Duration
	AvgTime   time.Duration
}

// Lexer represents the v2 lexer
type Lexer struct {
	// Core lexing state
	input    []byte // Use []byte for zero-allocation performance
	position int
	line     int
	column   int

	// Buffering for efficient token access
	tokens     []Token // Internal token buffer
	tokenIndex int     // Current position in buffer
	bufferSize int     // Number of tokens to buffer at once (default: 2500)

	// Timing (configurable mode)
	totalTime  time.Duration // Total time for entire lexing process
	timingMode TimingMode    // How much timing to collect

	// Debug telemetry (nil when debug disabled for zero allocation)
	debugEnabled bool
	tokenStats   map[TokenType]*TokenStats // Per-token timing stats (debug only)
}

// NewLexer creates a new lexer instance with optional configuration
func NewLexer(input string, opts ...LexerOpt) *Lexer {
	config := &LexerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexer := &Lexer{
		debugEnabled: config.debug,
		bufferSize:   2500,                   // Large enough for 90%+ of devcmd files
		tokens:       make([]Token, 0, 2500), // Pre-allocate capacity
		timingMode:   config.timing,          // Default is TimingOff (0)
	}

	// Only allocate debug structures when needed
	if config.debug {
		lexer.tokenStats = make(map[TokenType]*TokenStats)
	}

	lexer.Init([]byte(input))
	return lexer
}

// Init resets the lexer with new input (following Go scanner pattern)
func (l *Lexer) Init(input []byte) {
	l.input = input
	l.position = 0
	l.line = 1
	l.column = 1
	l.totalTime = 0 // Reset timing

	// Reset buffering state
	l.tokens = l.tokens[:0] // Reset slice but keep capacity
	l.tokenIndex = 0

	// Reset debug stats if enabled
	if l.debugEnabled && l.tokenStats != nil {
		// Clear existing stats without reallocating map
		for k := range l.tokenStats {
			delete(l.tokenStats, k)
		}
	}
}

// Duration returns the total cumulative time spent tokenizing
func (l *Lexer) Duration() time.Duration {
	return l.totalTime
}

// HasDebugTelemetry returns true if debug telemetry is enabled
func (l *Lexer) HasDebugTelemetry() bool {
	return l.debugEnabled
}

// GetTokenStats returns per-token timing statistics (debug mode only)
func (l *Lexer) GetTokenStats() map[TokenType]*TokenStats {
	if !l.debugEnabled || l.tokenStats == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make(map[TokenType]*TokenStats, len(l.tokenStats))
	for k, v := range l.tokenStats {
		// Copy the stats struct
		statsCopy := *v
		result[k] = &statsCopy
	}
	return result
}

// NextToken returns the next token using streaming interface
func (l *Lexer) NextToken() Token {
	// Ensure buffer has tokens
	if l.tokenIndex >= len(l.tokens) {
		l.fillBuffer()
	}

	// If still no tokens, return EOF
	if l.tokenIndex >= len(l.tokens) {
		return Token{Type: EOF, Text: nil, Position: Position{Line: l.line, Column: l.column}}
	}

	token := l.tokens[l.tokenIndex]
	l.tokenIndex++
	return token
}

// GetTokens returns all tokens using batch interface
// If tokens have already been consumed via NextToken(), this includes those tokens
// No timing logic - timing is handled by NextToken() calls
func (l *Lexer) GetTokens() []Token {
	var tokens []Token

	// First, collect any tokens already consumed via NextToken()
	for i := 0; i < l.tokenIndex; i++ {
		tokens = append(tokens, l.tokens[i])
	}

	// Then continue collecting remaining tokens via NextToken()
	for {
		token := l.NextToken()
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	return tokens
}

// fillBuffer fills the internal token buffer with the next batch of tokens
func (l *Lexer) fillBuffer() {
	var start time.Time
	if l.timingMode >= TimingOn {
		start = time.Now()
	}

	// Reset buffer but keep capacity
	l.tokens = l.tokens[:0]
	l.tokenIndex = 0

	// Fill buffer up to current capacity
	targetSize := cap(l.tokens)
	for len(l.tokens) < targetSize {
		token := l.nextToken()

		// Check if we need to grow the buffer
		if len(l.tokens) == cap(l.tokens) {
			// Double the capacity for very large files
			newCapacity := cap(l.tokens) * 2
			newTokens := make([]Token, len(l.tokens), newCapacity)
			copy(newTokens, l.tokens)
			l.tokens = newTokens
		}

		l.tokens = append(l.tokens, token)

		if token.Type == EOF {
			break
		}
	}

	// Update timing (accumulate across buffer fills)
	if l.timingMode >= TimingOn {
		l.totalTime += time.Since(start)
	}
}

// nextToken returns the next token from the input (internal implementation)
func (l *Lexer) nextToken() Token {
	token := l.lexToken() // Do the actual lexing work

	// Debug telemetry (only when enabled with fine-grain timing, will allocate)
	if l.debugEnabled && l.timingMode >= TimingFine && l.tokenStats != nil {
		// Record token count for debug stats
		l.recordTokenStats(token.Type, 0) // No per-token timing
	}

	return token
}

// recordTokenStats records per-token timing statistics (debug mode only)
func (l *Lexer) recordTokenStats(tokenType TokenType, elapsed time.Duration) {
	stats, exists := l.tokenStats[tokenType]
	if !exists {
		// Allocate new stats (only in debug mode)
		stats = &TokenStats{
			Type:      tokenType,
			Count:     0,
			TotalTime: 0,
		}
		l.tokenStats[tokenType] = stats
	}

	stats.Count++
	stats.TotalTime += elapsed
	stats.AvgTime = stats.TotalTime / time.Duration(stats.Count)
}

// lexToken performs the actual tokenization work
func (l *Lexer) lexToken() Token {
	// Skip whitespace (except newlines which are significant)
	l.skipWhitespace()

	// Check for EOF
	if l.position >= len(l.input) {
		return Token{
			Type:     EOF,
			Text:     nil,
			Position: Position{Line: l.line, Column: l.column},
		}
	}

	// Capture current position for token
	start := Position{Line: l.line, Column: l.column}
	ch := l.currentChar()

	// Identifier or keyword
	if ch < 128 && isIdentStart[ch] {
		return l.lexIdentifier(start)
	}

	// String literals
	if ch == '"' || ch == '\'' || ch == '`' {
		return l.lexString(start, ch)
	}

	// Numbers (integers, floats, etc.) - no longer handle negative sign here
	if ch < 128 && isDigit[ch] {
		return l.lexNumber(start)
	}

	// Decimal numbers starting with dot (.5, .123)
	if ch == '.' && l.position+1 < len(l.input) && l.input[l.position+1] < 128 && isDigit[l.input[l.position+1]] {
		return l.lexNumber(start)
	}

	// Single character punctuation
	switch ch {
	case '=':
		l.advanceChar()
		return Token{Type: EQUALS, Text: []byte{'='}, Position: start}
	case ':':
		l.advanceChar()
		return Token{Type: COLON, Text: []byte{':'}, Position: start}
	case '{':
		l.advanceChar()
		return Token{Type: LBRACE, Text: []byte{'{'}, Position: start}
	case '}':
		l.advanceChar()
		return Token{Type: RBRACE, Text: []byte{'}'}, Position: start}
	case '(':
		l.advanceChar()
		return Token{Type: LPAREN, Text: []byte{'('}, Position: start}
	case ')':
		l.advanceChar()
		return Token{Type: RPAREN, Text: []byte{')'}, Position: start}
	case '[':
		l.advanceChar()
		return Token{Type: LSQUARE, Text: []byte{'['}, Position: start}
	case ']':
		l.advanceChar()
		return Token{Type: RSQUARE, Text: []byte{']'}, Position: start}
	case ',':
		l.advanceChar()
		return Token{Type: COMMA, Text: []byte{','}, Position: start}
	case ';':
		l.advanceChar()
		return Token{Type: SEMICOLON, Text: []byte{';'}, Position: start}
	case '-':
		l.advanceChar()
		return Token{Type: MINUS, Text: []byte{'-'}, Position: start}
		// NOTE: '\n' is now handled as whitespace and skipped
		// Meaningful newlines will be implemented when we add statement parsing
	}

	// Unrecognized character - advance and mark as illegal
	l.advanceChar()
	return Token{
		Type:     ILLEGAL,
		Text:     []byte{ch},
		Position: start,
	}
}

// skipWhitespace skips whitespace characters except newlines
func (l *Lexer) skipWhitespace() {
	start := l.position

	// Array jumping: fast scan for non-whitespace
	for l.position < len(l.input) {
		ch := l.input[l.position]
		if ch >= 128 || !isWhitespace[ch] {
			break
		}
		l.position++
	}

	// Update column position based on characters skipped
	l.updateColumnFromWhitespace(start, l.position)
}

// updateColumnFromWhitespace updates column position after array jumping
func (l *Lexer) updateColumnFromWhitespace(start, end int) {
	for i := start; i < end; i++ {
		ch := l.input[i]
		if ch == '\n' {
			l.line++
			l.column = 1
		} else if ch == '\t' {
			l.column++ // Go standard: column = byte count, tab = 1 byte
		} else {
			l.column++
		}
	}
}

// lexIdentifier reads an identifier or keyword starting at current position
func (l *Lexer) lexIdentifier(start Position) Token {
	startPos := l.position

	// Read all identifier characters
	for l.position < len(l.input) {
		ch := l.input[l.position]
		if ch >= 128 || !isIdentPart[ch] {
			break
		}
		l.advanceChar()
	}

	// Extract the text as byte slice (zero allocation)
	text := l.input[startPos:l.position]

	// Check if it's a keyword (need string for map lookup)
	tokenType := l.lookupKeyword(string(text))

	return Token{
		Type:     tokenType,
		Text:     text,
		Position: start,
	}
}

// lexString reads a string literal starting at current position
func (l *Lexer) lexString(start Position, quote byte) Token {
	startPos := l.position
	l.advanceChar() // Skip opening quote

	// Read until closing quote
	for l.position < len(l.input) {
		ch := l.currentChar()

		// Found closing quote
		if ch == quote {
			l.advanceChar() // Include closing quote
			break
		}

		// Handle escape sequences
		if ch == '\\' && l.position+1 < len(l.input) {
			l.advanceChar() // Skip backslash
			l.advanceChar() // Skip escaped character
			continue
		}

		// For backticks, newlines are allowed
		if quote == '`' && ch == '\n' {
			l.advanceChar()
			continue
		}

		// For double/single quotes, newlines end the string (error case)
		if ch == '\n' && quote != '`' {
			break // Unterminated string
		}

		l.advanceChar()
	}

	// Extract the full string including quotes as byte slice (zero allocation)
	text := l.input[startPos:l.position]

	return Token{
		Type:     STRING,
		Text:     text,
		Position: start,
	}
}

// lookupKeyword returns the appropriate token type for keywords, or IDENTIFIER
func (l *Lexer) lookupKeyword(text string) TokenType {
	switch text {
	case "var":
		return VAR
	case "for":
		return FOR
	case "in":
		return IN
	case "if":
		return IF
	case "else":
		return ELSE
	case "when":
		return WHEN
	case "try":
		return TRY
	case "catch":
		return CATCH
	case "finally":
		return FINALLY
	default:
		return IDENTIFIER
	}
}

// currentChar returns the current character being examined (ASCII fast path)
func (l *Lexer) currentChar() byte {
	if l.position >= len(l.input) {
		return 0 // EOF
	}
	return l.input[l.position]
}

// peekChar returns the character at offset from current position without advancing
func (l *Lexer) peekChar(offset int) byte {
	pos := l.position + offset
	if pos >= len(l.input) {
		return 0 // EOF
	}
	return l.input[pos]
}

// advanceChar moves to the next character, handling Unicode for position tracking only
func (l *Lexer) advanceChar() {
	if l.position >= len(l.input) {
		return
	}

	ch := l.input[l.position]

	// Fast path for ASCII (majority case)
	if ch < 128 {
		if ch == '\n' {
			l.line++
			l.column = 1
		} else if ch == '\t' {
			l.column++ // Go standard: column = byte count, tab = 1 byte
		} else {
			l.column++
		}
		l.position++
		return
	}

	// Unicode character - we only need size for position tracking
	// Content goes into tokens as raw bytes
	_, size := utf8.DecodeRune(l.input[l.position:])
	if size <= 0 {
		size = 1 // Invalid UTF-8, treat as single byte
	}

	l.position += size
	l.column++ // Unicode characters count as 1 column for display
}

// lexNumber tokenizes numeric literals (integers, floats, scientific notation)
func (l *Lexer) lexNumber(start Position) Token {
	startPos := l.position

	// Check if starting with decimal point
	if l.currentChar() == '.' {
		l.advanceChar()
		if !l.readDigits() {
			// No digits after decimal - shouldn't happen given our caller check
			return Token{Type: ILLEGAL, Text: l.input[startPos:l.position], Position: start}
		}
		// This is a decimal number like .5
		return Token{
			Type:     FLOAT,
			Text:     l.input[startPos:l.position],
			Position: start,
		}
	}

	// Read integer part
	if !l.readDigits() {
		// No digits found - this shouldn't happen given our caller check
		return Token{Type: ILLEGAL, Text: l.input[startPos:l.position], Position: start}
	}

	// Check for decimal point
	if l.position < len(l.input) && l.currentChar() == '.' {
		l.advanceChar()
		// Read decimal part (optional - Go allows 5.)
		l.readDigits()
		return Token{
			Type:     FLOAT,
			Text:     l.input[startPos:l.position],
			Position: start,
		}
	}

	// Just an integer
	return Token{
		Type:     INTEGER,
		Text:     l.input[startPos:l.position],
		Position: start,
	}
}

// readDigits reads a sequence of digits and returns true if at least one was found
func (l *Lexer) readDigits() bool {
	startPos := l.position

	for l.position < len(l.input) {
		ch := l.currentChar()
		if ch >= 128 || !isDigit[ch] {
			break
		}
		l.advanceChar()
	}

	return l.position > startPos
}
