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

// LexerConfig holds lexer configuration
type LexerConfig struct {
	debug bool
	mode  LexerMode
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

// TokenStats holds per-token timing statistics (debug mode only)
type TokenStats struct {
	Type      TokenType
	Count     int
	TotalTime time.Duration
	AvgTime   time.Duration
}

// Lexer represents the v2 lexer
type Lexer struct {
	input     []byte // Use []byte for zero-allocation performance
	position  int
	line      int
	column    int
	totalTime time.Duration // Cumulative time spent tokenizing

	// Debug telemetry (nil when debug disabled for zero allocation)
	debugEnabled bool
	tokenStats   map[TokenType]*TokenStats // Per-token timing stats (debug only)
}

// NewLexer creates a new lexer instance (debug disabled by default)
func NewLexer(input string) *Lexer {
	return NewLexerWithOpts(input)
}

// NewLexerWithOpts creates a new lexer instance with options
func NewLexerWithOpts(input string, opts ...LexerOpt) *Lexer {
	config := &LexerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexer := &Lexer{
		debugEnabled: config.debug,
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

// NextToken returns the next token from the input
func (l *Lexer) NextToken() Token {
	// Time each token individually
	start := time.Now()

	token := l.lexToken() // Do the actual lexing work

	elapsed := time.Since(start)

	// Always accumulate total time (zero-alloc)
	l.totalTime += elapsed

	// Debug telemetry (only when enabled, will allocate)
	if l.debugEnabled && l.tokenStats != nil {
		l.recordTokenStats(token.Type, elapsed)
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
			Text:     "",
			Position: Position{Line: l.line, Column: l.column},
		}
	}

	// For now, just return EOF - we'll add more token recognition incrementally
	return Token{
		Type:     EOF,
		Text:     "",
		Position: Position{Line: l.line, Column: l.column},
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
		if ch == '\t' {
			l.column += 4 // Tab counts as 4 spaces
		} else {
			l.column++
		}
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
			l.column += 4
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
