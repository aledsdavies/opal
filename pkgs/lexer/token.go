package lexer

import (
	"fmt"
	"strings"
)

// TokenType represents the type of token in Devcmd
//
// Devcmd uses mode-based lexing with two primary contexts:
// - LanguageMode: Top-level constructs and decorator parsing
// - CommandMode: Shell text with decorator recognition
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Language structure tokens (LanguageMode and CommandMode boundaries)
	VAR    // var - keyword for variable declarations
	WATCH  // watch - keyword for process management commands
	STOP   // stop - keyword for cleanup commands
	AT     // @ - decorator prefix (switches CommandMode → LanguageMode)
	COLON  // : - command separator (LanguageMode → CommandMode transition point)
	EQUALS // = - assignment operator in variable declarations
	COMMA  // , - separator in decorator parameters and variable groups
	LPAREN // ( - decorator parameter start
	RPAREN // ) - decorator parameter end
	LBRACE // { - explicit block start (LanguageMode → CommandMode)
	RBRACE // } - block end (CommandMode → LanguageMode)

	// Literals (recognized in both modes with context-specific semantics)
	IDENTIFIER // command names, variable names, decorator names, shell text
	NUMBER     // numeric literals: 8080, 3.14, -100
	STRING     // quoted strings: "hello", 'world', `template`
	DURATION   // time literals: 30s, 5m, 1h, 500ms, 2.5s

	// Continuation and structure
	LINE_CONT // \ - line continuation (preserves mode)
	NEWLINE   // \n - statement boundary (CommandMode → LanguageMode for simple commands)

	// Comments (recognized in both modes)
	COMMENT           // # - single line comments
	MULTILINE_COMMENT // /* */ - multiline comments
)

// Pre-computed token name lookup for fast debugging
var tokenNames = [...]string{
	EOF:               "EOF",
	ILLEGAL:           "ILLEGAL",
	VAR:               "VAR",
	WATCH:             "WATCH",
	STOP:              "STOP",
	AT:                "AT",
	COLON:             "COLON",
	EQUALS:            "EQUALS",
	COMMA:             "COMMA",
	LPAREN:            "LPAREN",
	RPAREN:            "RPAREN",
	LBRACE:            "LBRACE",
	RBRACE:            "RBRACE",
	IDENTIFIER:        "IDENTIFIER",
	NUMBER:            "NUMBER",
	STRING:            "STRING",
	DURATION:          "DURATION",
	LINE_CONT:         "LINE_CONT",
	NEWLINE:           "NEWLINE",
	COMMENT:           "COMMENT",
	MULTILINE_COMMENT: "MULTILINE_COMMENT",
}

func (t TokenType) String() string {
	if int(t) < len(tokenNames) && int(t) >= 0 {
		return tokenNames[t]
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// StringType represents the type of string literal
type StringType int

const (
	DoubleQuoted StringType = iota // "string" - supports escape sequences
	SingleQuoted                   // 'string' - literal strings
	Backtick                       // `string` - template strings with extended escapes
)

// SemanticTokenType represents semantic categories for syntax highlighting
// These provide rich context for IDE features like syntax highlighting,
// go-to-definition, and hover information
type SemanticTokenType int

const (
	SemKeyword   SemanticTokenType = iota // var, watch, stop
	SemCommand                            // command names and shell text
	SemVariable                           // variable names in declarations
	SemDecorator                          // decorator names after @
	SemString                             // string literals
	SemNumber                             // numeric literals and durations
	SemComment                            // comments
	SemOperator                           // :, =, {, }, (, ), @, shell operators
	SemParameter                          // decorator parameter names
)

// Token represents a single token with position information
// Optimized for memory layout and cache efficiency
//
// Usage in Mode-Based Lexing:
// - LanguageMode: Produces structured tokens (VAR, WATCH, AT, etc.)
// - CommandMode: Produces shell text as IDENTIFIER tokens + structural boundaries
type Token struct {
	Type      TokenType
	Semantic  SemanticTokenType
	Line      int
	Column    int
	EndLine   int
	EndColumn int

	// String fields grouped together for better cache locality
	Value string // Actual token content
	Raw   string // Raw string content before escape processing (for strings)
	Scope string // TextMate-style scope for syntax highlighting

	// Enum fields at end for optimal packing
	StringType StringType
}

// Position returns a formatted position string for error reporting
func (t Token) Position() string {
	if t.Line == t.EndLine {
		return fmt.Sprintf("%d:%d-%d", t.Line, t.Column, t.EndColumn)
	}
	return fmt.Sprintf("%d:%d-%d:%d", t.Line, t.Column, t.EndLine, t.EndColumn)
}

// ToLSPSemanticToken converts to Language Server Protocol format
func (t Token) ToLSPSemanticToken() LSPSemanticToken {
	return LSPSemanticToken{
		Line:      uint32(t.Line - 1),   // LSP is 0-indexed
		Character: uint32(t.Column - 1), // LSP is 0-indexed
		Length:    uint32(len(t.Value)),
		TokenType: uint32(t.Semantic),
	}
}

// LSPSemanticToken represents a token in LSP format
type LSPSemanticToken struct {
	Line      uint32
	Character uint32
	Length    uint32
	TokenType uint32
}

// DecoratorArg represents a single decorator argument
// Supports Kotlin-like named parameters for enhanced readability
//
// Examples:
// - Positional: @timeout(30s) → DecoratorArg{Value: "30s"}
// - Named: @retry(attempts=3) → DecoratorArg{Name: "attempts", Value: "3"}
type DecoratorArg struct {
	Name   string // empty string for positional args
	Value  string // argument value (unquoted)
	Line   int    // source position for error reporting
	Column int
}

// ParseDecoratorArgs parses decorator arguments supporting Kotlin-like named parameters
//
// Supported Patterns:
// - Positional: @timeout(30s)
// - Named: @retry(attempts=3)
// - Mixed: @timeout(30s, graceful=true) - positional must come first
// - Reordered: @retry(delay=1s, attempts=3) - named can be in any order
//
// Rules (Kotlin-style):
// - Once a named parameter is used, all following parameters must be named
// - Parameter names must be valid identifiers (letters, digits, -, _)
// - Values can be quoted or unquoted
func ParseDecoratorArgs(args string, line, column int) ([]DecoratorArg, error) {
	if len(strings.TrimSpace(args)) == 0 {
		return nil, nil
	}

	// Pre-allocate for common case of 1-3 args
	result := make([]DecoratorArg, 0, 3)

	// Split by commas, but respect quoted strings
	parts := splitDecoratorArgsFast(args)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}

		// Check if it's a named parameter (contains =)
		if eqIndex := strings.IndexByte(part, '='); eqIndex > 0 {
			// Named parameter: name=value
			name := strings.TrimSpace(part[:eqIndex])
			value := strings.TrimSpace(part[eqIndex+1:])

			// Validate parameter name (must be valid identifier)
			if !isValidParameterName(name) {
				return nil, fmt.Errorf("invalid parameter name '%s' at line %d, column %d", name, line, column)
			}

			// Remove quotes from value if present
			value = unquoteIfNeeded(value)

			result = append(result, DecoratorArg{
				Name:   name,
				Value:  value,
				Line:   line,
				Column: column, // TODO: calculate exact column
			})
		} else {
			// Positional parameter
			value := unquoteIfNeeded(part)

			result = append(result, DecoratorArg{
				Value:  value,
				Line:   line,
				Column: column,
			})
		}
	}

	// Validate Kotlin-like parameter rules:
	// Once a named parameter is used, all following parameters must be named
	foundNamed := false
	for i, arg := range result {
		if arg.Name != "" {
			foundNamed = true
		} else if foundNamed {
			return nil, fmt.Errorf("positional argument follows named argument at position %d, line %d, column %d", i+1, line, column)
		}
	}

	return result, nil
}

// isValidParameterName checks if a string is a valid parameter name
// Rules: start with letter/underscore, contain letters/digits/hyphens/underscores
func isValidParameterName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Must start with letter or underscore
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest can be letters, digits, underscores, or hyphens
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			 (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}

	return true
}

// Fast splitting with minimal allocations
func splitDecoratorArgsFast(args string) []string {
	if len(args) == 0 {
		return nil
	}

	// Pre-allocate for common case
	result := make([]string, 0, 4)

	start := 0
	inQuotes := false
	var quoteChar byte

	for i := 0; i < len(args); i++ {
		ch := args[i]
		switch ch {
		case '"', '\'', '`':
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
			}
		case ',':
			if !inQuotes {
				if i > start {
					result = append(result, args[start:i])
				}
				start = i + 1
			}
		}
	}

	// Add final part
	if start < len(args) {
		result = append(result, args[start:])
	}

	return result
}

// unquoteIfNeeded removes surrounding quotes if present - optimized version
func unquoteIfNeeded(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') ||
			(first == '`' && last == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// Syntax highlighting utility functions - optimized versions

// GetSemanticTokens extracts all tokens with semantic information for syntax highlighting
// Excludes EOF token for cleaner IDE integration
func GetSemanticTokens(input string) ([]Token, error) {
	lexer := New(input)

	tokens := lexer.TokenizeToSlice()
	// Remove EOF token for cleaner API
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == EOF {
		tokens = tokens[:len(tokens)-1]
	}

	return tokens, nil
}

// ToLSPSemanticTokensArray converts tokens to LSP semantic tokens array format
// Uses delta encoding as required by the Language Server Protocol
func ToLSPSemanticTokensArray(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}

	// Each token produces 5 uint32 values: deltaLine, deltaChar, length, tokenType, modifiers
	result := make([]uint32, 0, len(tokens)*5)
	var prevLine, prevChar uint32

	for _, token := range tokens {
		line := uint32(token.Line - 1)   // LSP is 0-indexed
		char := uint32(token.Column - 1) // LSP is 0-indexed
		length := uint32(len(token.Value))
		tokenType := uint32(token.Semantic)

		// LSP uses delta encoding for efficiency
		deltaLine := line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = char - prevChar
		} else {
			deltaChar = char
		}

		result = append(result, deltaLine, deltaChar, length, tokenType, 0) // modifiers = 0

		prevLine = line
		prevChar = char
	}

	return result
}

// GetTextMateGrammarScopes returns all unique TextMate scopes used
// Useful for building syntax highlighting grammars
func GetTextMateGrammarScopes(tokens []Token) []string {
	if len(tokens) == 0 {
		return nil
	}

	// Pre-size map for expected scope count
	scopes := make(map[string]bool, 16)
	for _, token := range tokens {
		if len(token.Scope) > 0 {
			scopes[token.Scope] = true
		}
	}

	if len(scopes) == 0 {
		return nil
	}

	result := make([]string, 0, len(scopes))
	for scope := range scopes {
		result = append(result, scope)
	}
	return result
}
