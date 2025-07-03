package lexer

import (
	"fmt"
	"strings"
)

// TokenType represents the type of token in Devcmd
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL

	// Language structure tokens
	VAR    // var
	WATCH  // watch
	STOP   // stop
	AT     // @
	COLON  // :
	EQUALS // =
	COMMA  // ,
	LPAREN // (
	RPAREN // )
	LBRACE // {
	RBRACE // }

	// Literals and Content
	IDENTIFIER // command names, variable names, decorator names
	SHELL_TEXT // shell command text
	NUMBER     // 8080, 3.14, -100
	STRING     // "hello", 'world', `template`
	DURATION   // 30s, 5m, 1h

	// Continuation and structure
	LINE_CONT // \
	NEWLINE   // \n

	// Comments
	COMMENT           // #
	MULTILINE_COMMENT // /* */
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
	SHELL_TEXT:        "SHELL_TEXT",
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
	DoubleQuoted StringType = iota // "string"
	SingleQuoted                   // 'string'
	Backtick                       // `string`
)

// SemanticTokenType represents semantic categories for syntax highlighting
type SemanticTokenType int

const (
	SemKeyword   SemanticTokenType = iota // var, watch, stop
	SemCommand                            // command names
	SemVariable                           // variable names
	SemDecorator                          // decorator names
	SemString                             // string literals
	SemNumber                             // numeric literals
	SemComment                            // comments
	SemOperator                           // :, =, {, }, (, ), @
	SemParameter                          // decorator parameter names
	SemShellText                          // shell text content
)

// Token represents a single token with position information
type Token struct {
	Type      TokenType
	Semantic  SemanticTokenType
	Line      int
	Column    int
	EndLine   int
	EndColumn int
	Value     string
	Raw       string
	Scope     string
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
		Line:      uint32(t.Line - 1),
		Character: uint32(t.Column - 1),
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
	Name   string
	Value  string
	Line   int
	Column int
}

// ParseDecoratorArgs parses decorator arguments supporting Kotlin-like named parameters
func ParseDecoratorArgs(args string, line, column int) ([]DecoratorArg, error) {
	if len(strings.TrimSpace(args)) == 0 {
		return nil, nil
	}
	result := make([]DecoratorArg, 0, 3)
	parts := splitDecoratorArgsFast(args)
	foundNamed := false
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			continue
		}
		if eqIndex := strings.IndexByte(part, '='); eqIndex > 0 {
			foundNamed = true
			name := strings.TrimSpace(part[:eqIndex])
			value := strings.TrimSpace(part[eqIndex+1:])
			if !isValidParameterName(name) {
				return nil, fmt.Errorf("invalid parameter name '%s' at line %d, column %d", name, line, column)
			}
			result = append(result, DecoratorArg{Name: name, Value: unquoteIfNeeded(value), Line: line, Column: column})
		} else {
			if foundNamed {
				return nil, fmt.Errorf("positional argument follows named argument at position %d, line %d, column %d", i+1, line, column)
			}
			result = append(result, DecoratorArg{Value: unquoteIfNeeded(part), Line: line, Column: column})
		}
	}
	return result, nil
}

func isValidParameterName(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func splitDecoratorArgsFast(args string) []string {
	if len(args) == 0 {
		return nil
	}
	result := make([]string, 0, 4)
	start, inQuotes := 0, false
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
	if start < len(args) {
		result = append(result, args[start:])
	}
	return result
}

func unquoteIfNeeded(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') || (first == '`' && last == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// GetSemanticTokens extracts all tokens with semantic information for syntax highlighting
func GetSemanticTokens(input string) ([]Token, error) {
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()
	if len(tokens) > 0 && tokens[len(tokens)-1].Type == EOF {
		tokens = tokens[:len(tokens)-1]
	}
	return tokens, nil
}

// ToLSPSemanticTokensArray converts tokens to LSP semantic tokens array format
func ToLSPSemanticTokensArray(tokens []Token) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}
	result := make([]uint32, 0, len(tokens)*5)
	var prevLine, prevChar uint32
	for _, token := range tokens {
		line := uint32(token.Line - 1)
		char := uint32(token.Column - 1)
		length := uint32(len(token.Value))
		tokenType := uint32(token.Semantic)
		deltaLine := line - prevLine
		var deltaChar uint32
		if deltaLine == 0 {
			deltaChar = char - prevChar
		} else {
			deltaChar = char
		}
		result = append(result, deltaLine, deltaChar, length, tokenType, 0)
		prevLine = line
		prevChar = char
	}
	return result
}

// GetTextMateGrammarScopes returns all unique TextMate scopes used
func GetTextMateGrammarScopes(tokens []Token) []string {
	if len(tokens) == 0 {
		return nil
	}
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

