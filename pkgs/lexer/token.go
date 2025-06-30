package lexer

import (
	"fmt"
	"strings"
)

// TokenType represents the type of token
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Language structure tokens
	VAR    // var
	WATCH  // watch
	STOP   // stop
	COLON  // :
	EQUALS // =
	LPAREN // (
	RPAREN // )
	LBRACE // {
	RBRACE // }

	// Literals
	IDENTIFIER // command names, variable names
	NUMBER     // 8080, 3.14, -100
	STRING     // "quoted", 'single', `backtick`

	// Decorators (all three forms)
	DECORATOR_CALL       // @word(args)
	DECORATOR_BLOCK      // @word{ block }
	DECORATOR_CALL_BLOCK // @word(args) { block }

	// Shell content
	SHELL_TEXT // Raw shell command text
	LINE_CONT  // \ (line continuation)

	// Structure
	NEWLINE           // statement boundaries
	COMMENT           // # single line comments
	MULTILINE_COMMENT // /* multiline comments */
)

// Pre-computed token name lookup for fast debugging
var tokenNames = [...]string{
	EOF:                  "EOF",
	ILLEGAL:              "ILLEGAL",
	VAR:                  "VAR",
	WATCH:                "WATCH",
	STOP:                 "STOP",
	COLON:                "COLON",
	EQUALS:               "EQUALS",
	LPAREN:               "LPAREN",
	RPAREN:               "RPAREN",
	LBRACE:               "LBRACE",
	RBRACE:               "RBRACE",
	IDENTIFIER:           "IDENTIFIER",
	NUMBER:               "NUMBER",
	STRING:               "STRING",
	DECORATOR_CALL:       "DECORATOR_CALL",
	DECORATOR_BLOCK:      "DECORATOR_BLOCK",
	DECORATOR_CALL_BLOCK: "DECORATOR_CALL_BLOCK",
	SHELL_TEXT:           "SHELL_TEXT",
	LINE_CONT:            "LINE_CONT",
	NEWLINE:              "NEWLINE",
	COMMENT:              "COMMENT",
	MULTILINE_COMMENT:    "MULTILINE_COMMENT",
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
	DoubleQuoted StringType = iota // "string"
	SingleQuoted                   // 'string'
	Backtick                       // `string`
)

// SemanticTokenType represents semantic categories for syntax highlighting
type SemanticTokenType int

const (
	SemKeyword   SemanticTokenType = iota // var, watch, stop
	SemCommand                            // command names after var/watch/stop
	SemVariable                           // variable names in declarations
	SemDecorator                          // @timeout, @var, etc.
	SemString                             // string literals
	SemNumber                             // numeric literals
	SemComment                            // comments
	SemOperator                           // :, =, {, }, (, )
	SemShellText                          // shell command content
	SemParameter                          // decorator parameter names
)

// Token represents a single token with position information
// Optimized for memory layout and cache efficiency
type Token struct {
	Type      TokenType
	Semantic  SemanticTokenType
	Line      int
	Column    int
	EndLine   int
	EndColumn int

	// String fields grouped together for better cache locality
	Value         string
	DecoratorName string // "var", "timeout", "sh", etc.
	Args          string // raw content inside parentheses
	Block         string // content inside braces
	Raw           string // Raw string content before escape processing
	Scope         string // TextMate-style scope

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
type DecoratorArg struct {
	Name   string // empty string for positional args
	Value  string
	Line   int
	Column int
}

// ParseDecoratorArgs parses decorator arguments supporting both positional and named parameters
// Optimized version with minimal allocations
func ParseDecoratorArgs(args string, line, column int) ([]DecoratorArg, error) {
	if len(strings.TrimSpace(args)) == 0 {
		return nil, nil
	}

	// Pre-allocate for common case of 1-3 args
	result := make([]DecoratorArg, 0, 3)

	// Split by commas, but respect quoted strings - optimized version
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

	return result, nil
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
// Optimized with pre-allocation
func ToLSPSemanticTokensArray(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}

	// Each token produces 5 uint32 values
	result := make([]uint32, 0, len(tokens)*5)
	var prevLine, prevChar uint32

	for _, token := range tokens {
		line := uint32(token.Line - 1)   // LSP is 0-indexed
		char := uint32(token.Column - 1) // LSP is 0-indexed
		length := uint32(len(token.Value))
		tokenType := uint32(token.Semantic)

		// LSP uses delta encoding
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
// Optimized with pre-sized map
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
