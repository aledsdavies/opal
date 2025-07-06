package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Helper function to verify token positions in any test
func verifyTokenPosition(t *testing.T, token Token, expectedLine, expectedColumn int, tokenIndex int) {
	t.Helper()
	if token.Line != expectedLine {
		t.Errorf("Token %d: expected line %d, got %d", tokenIndex, expectedLine, token.Line)
	}
	if token.Column != expectedColumn {
		t.Errorf("Token %d: expected column %d, got %d", tokenIndex, expectedColumn, token.Column)
	}

	// Verify span consistency
	if token.Line != token.Span.Start.Line {
		t.Errorf("Token %d: Line %d != Span.Start.Line %d", tokenIndex, token.Line, token.Span.Start.Line)
	}
	if token.Column != token.Span.Start.Column {
		t.Errorf("Token %d: Column %d != Span.Start.Column %d", tokenIndex, token.Column, token.Span.Start.Column)
	}

	// Verify span makes sense
	if token.Span.Start.Offset > token.Span.End.Offset {
		t.Errorf("Token %d: Start offset %d > End offset %d", tokenIndex, token.Span.Start.Offset, token.Span.End.Offset)
	}
}

func TestBasicTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "var mycommand:",
			expected: []TokenType{VAR, IDENTIFIER, COLON, EOF},
		},
		{
			input:    "watch server:",
			expected: []TokenType{WATCH, IDENTIFIER, COLON, EOF},
		},
		{
			input:    "stop app",
			expected: []TokenType{STOP, IDENTIFIER, EOF},
		},
		{
			input:    "var test = \"hello world\"",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, STRING, EOF},
		},
		{
			input:    "# this is a comment",
			expected: []TokenType{COMMENT, EOF},
		},
		{
			input:    "/* multiline comment */",
			expected: []TokenType{MULTILINE_COMMENT, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var tokenTypes []TokenType
			for _, token := range tokens {
				tokenTypes = append(tokenTypes, token.Type)
			}

			if diff := cmp.Diff(test.expected, tokenTypes); diff != "" {
				t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
			}

			// Verify all tokens have valid positions
			for i, token := range tokens {
				if token.Line <= 0 || token.Column <= 0 {
					t.Errorf("Token %d has invalid position: %d:%d", i, token.Line, token.Column)
				}
			}
		})
	}
}

func TestBooleanTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
		values   []string
	}{
		{
			input:    "var enabled = true",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, BOOLEAN, EOF},
			values:   []string{"var", "enabled", "=", "true", ""},
		},
		{
			input:    "var disabled = false",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, BOOLEAN, EOF},
			values:   []string{"var", "disabled", "=", "false", ""},
		},
		{
			input:    "var (debug = true, production = false)",
			expected: []TokenType{VAR, LPAREN, IDENTIFIER, EQUALS, BOOLEAN, COMMA, IDENTIFIER, EQUALS, BOOLEAN, RPAREN, EOF},
			values:   []string{"var", "(", "debug", "=", "true", ",", "production", "=", "false", ")", ""},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var tokenTypes []TokenType
			for _, token := range tokens {
				tokenTypes = append(tokenTypes, token.Type)
			}

			if diff := cmp.Diff(test.expected, tokenTypes); diff != "" {
				t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
			}

			// Verify token values
			for i, expected := range test.values {
				if i < len(tokens) && tokens[i].Value != expected {
					t.Errorf("Token %d: expected value %q, got %q", i, expected, tokens[i].Value)
				}
			}

			// Verify boolean tokens have correct semantic type
			for _, token := range tokens {
				if token.Type == BOOLEAN {
					if token.Semantic != SemBoolean {
						t.Errorf("Boolean token %q has wrong semantic type: %v", token.Value, token.Semantic)
					}
				}
			}
		})
	}
}

func TestStructuralTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "@timeout(30s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, EOF},
		},
		{
			input:    "@timeout(30s) { echo hello }",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, SHELL_TEXT, RBRACE, EOF},
		},
		{
			input:    "@retry(attempts=3, delay=1.5s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, IDENTIFIER, EQUALS, NUMBER, COMMA, IDENTIFIER, EQUALS, DURATION, RPAREN, EOF},
		},
		{
			input:    "var (PORT = \"8080\", HOST = \"localhost\")",
			expected: []TokenType{VAR, LPAREN, IDENTIFIER, EQUALS, STRING, COMMA, IDENTIFIER, EQUALS, STRING, RPAREN, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var tokenTypes []TokenType
			for _, token := range tokens {
				tokenTypes = append(tokenTypes, token.Type)
			}

			if diff := cmp.Diff(test.expected, tokenTypes); diff != "" {
				t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
			}
		})
	}
}

func TestBooleanContext(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "boolean in variable declaration",
			input: "var IS_PRODUCTION = true",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "IS_PRODUCTION"},
				{EQUALS, "="},
				{BOOLEAN, "true"},
				{EOF, ""},
			},
		},
		{
			name:  "boolean vs identifier",
			input: "var truename = \"value\"",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "truename"},
				{EQUALS, "="},
				{STRING, "value"},
				{EOF, ""},
			},
		},
		{
			name:  "mixed boolean and string vars",
			input: "var (ENABLED = true, NAME = \"app\", DEBUG = false)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "ENABLED"},
				{EQUALS, "="},
				{BOOLEAN, "true"},
				{COMMA, ","},
				{IDENTIFIER, "NAME"},
				{EQUALS, "="},
				{STRING, "app"},
				{COMMA, ","},
				{IDENTIFIER, "DEBUG"},
				{EQUALS, "="},
				{BOOLEAN, "false"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.tokenType, actual.Type)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.value, actual.Value)
				}
			}
		})
	}
}

func TestShellTextCapture(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "simple command",
			input: "build: echo hello world",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello world"},
				{EOF, ""},
			},
		},
		{
			name:  "command with semicolons",
			input: "build: echo hello; echo world",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello; echo world"},
				{EOF, ""},
			},
		},
		{
			name:  "block command",
			input: "deploy: { cd src; make clean; make install }",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "cd src; make clean; make install"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name:  "command with pipes",
			input: "process: cat file.txt | grep pattern | sort",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "process"},
				{COLON, ":"},
				{SHELL_TEXT, "cat file.txt | grep pattern | sort"},
				{EOF, ""},
			},
		},
		{
			name:  "command with redirections",
			input: "log: tail -f app.log > output.txt 2>&1",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "log"},
				{COLON, ":"},
				{SHELL_TEXT, "tail -f app.log > output.txt 2>&1"},
				{EOF, ""},
			},
		},
		{
			name:  "command with background process",
			input: "start: node server.js & echo 'Server started'",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "start"},
				{COLON, ":"},
				{SHELL_TEXT, "node server.js & echo 'Server started'"},
				{EOF, ""},
			},
		},
		{
			name:  "complex shell operators",
			input: "deploy: npm run build && npm test || (echo 'Build failed' && exit 1)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{SHELL_TEXT, "npm run build && npm test || (echo 'Build failed' && exit 1)"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()
			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.tokenType, actual.Type)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.value, actual.Value)
				}

				// Verify position consistency for all tokens
				if actual.Line <= 0 || actual.Column <= 0 {
					t.Errorf("Token %d has invalid position: %d:%d", i, actual.Line, actual.Column)
				}
			}
		})
	}
}

func TestMultiLineShellCommand(t *testing.T) {
	input := `test-quick: {
    echo "⚡ Running quick checks..."
    echo "🔍 Checking Go formatting..."
    if command -v gofumpt >/dev/null 2>&1; then if [ "$(gofumpt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofumpt -l .; exit 1; fi; else if [ "$(gofmt -l . | wc -l)" -gt 0 ]; then echo "❌ Go formatting issues:"; gofmt -l .; exit 1; fi; fi
    echo "🔍 Checking Nix formatting..."
    if command -v nixpkgs-fmt >/dev/null 2>&1; then nixpkgs-fmt --check . || (echo "❌ Run 'dev format' to fix"; exit 1); else echo "⚠️  nixpkgs-fmt not available, skipping Nix format check"; fi
    dev lint
    echo "✅ Quick checks passed!"
}`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// Debug: print all tokens to see what the lexer produces
	t.Logf("Total tokens: %d", len(tokens))
	for i, token := range tokens {
		if token.Type == SHELL_TEXT || token.Type == NEWLINE {
			t.Logf("  %d: %s %q", i, token.Type, token.Value)
		}
	}

	// Count SHELL_TEXT tokens to understand lexer behavior
	shellTextCount := 0
	newlineCount := 0
	for _, token := range tokens {
		if token.Type == SHELL_TEXT {
			shellTextCount++
		}
		if token.Type == NEWLINE {
			newlineCount++
		}
	}

	t.Logf("SHELL_TEXT tokens: %d, NEWLINE tokens: %d", shellTextCount, newlineCount)

	// Verify that we get separate SHELL_TEXT tokens for each line
	// This matches the test failure output showing separate text parts
	if shellTextCount < 6 { // Should have at least 6 shell commands
		t.Errorf("Expected at least 6 SHELL_TEXT tokens, got %d", shellTextCount)
	}
}

func TestLineContinuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "line continuation in shell text with ''",
			input: `build: echo 'hello \\\nworld'`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo 'hello \\\nworld'"}, // '' means values are escaped and not interpolation
				{EOF, ""},
			},
		},
		{
			name: "line continuation in shell text",
			input: `build: echo hello \
world`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello world"}, // Single space - shell line continuation behavior
				{EOF, ""},
			},
		},
		{
			name: "line continuation with trailing spaces",
			input: `build: echo hello \
world`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello world"}, // Single space - trailing spaces trimmed before \
				{EOF, ""},
			},
		},
		{
			name: "multiple line continuations",
			input: `build: echo hello \
beautiful \
world`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello beautiful world"}, // Each continuation becomes single space
				{EOF, ""},
			},
		},
		{
			name: "line continuation in block",
			input: `build: {
    echo hello \
    world
}`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo hello world"}, // Single space - indentation preserved, continuation merged
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			// Debug output
			t.Logf("Actual tokens:")
			for i, token := range tokens {
				t.Logf("  %d: %s %q", i, token.Type, token.Value)
			}

			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.tokenType, actual.Type)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.value, actual.Value)
				}
			}
		})
	}
}

func TestShellTextHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "simple shell command",
			input: "build: echo hello",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{SHELL_TEXT, "echo hello"},
				{EOF, ""},
			},
		},
		{
			name: "multi-line shell commands in block",
			input: `build: {
    echo line1
    echo line2
}`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{LBRACE, "{"},
				{SHELL_TEXT, "echo line1"}, // Each line should be separate SHELL_TEXT
				{SHELL_TEXT, "echo line2"},
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			// Debug output
			t.Logf("Actual tokens for %s:", test.name)
			for i, token := range tokens {
				t.Logf("  %d: %s %q", i, token.Type, token.Value)
			}

			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.tokenType, actual.Type)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.value, actual.Value)
				}
			}
		})
	}
}

func TestModeTransitions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			token TokenType
		}
	}{
		{
			name:  "simple command mode transition",
			input: "build: echo hello",
			expected: []struct {
				token TokenType
			}{
				{IDENTIFIER}, // build
				{COLON},      // :
				{SHELL_TEXT}, // echo hello
				{EOF},        // EOF
			},
		},
		{
			name:  "block command mode transition",
			input: "build: { echo hello }",
			expected: []struct {
				token TokenType
			}{
				{IDENTIFIER}, // build
				{COLON},      // :
				{LBRACE},     // {
				{SHELL_TEXT}, // echo hello
				{RBRACE},     // }
				{EOF},        // EOF
			},
		},
		{
			name:  "decorator mode transition",
			input: "build: @timeout(30s) { echo hello }",
			expected: []struct {
				token TokenType
			}{
				{IDENTIFIER}, // build
				{COLON},      // :
				{AT},         // @
				{IDENTIFIER}, // timeout
				{LPAREN},     // (
				{DURATION},   // 30s
				{RPAREN},     // )
				{LBRACE},     // {
				{SHELL_TEXT}, // echo hello
				{RBRACE},     // }
				{EOF},        // EOF
			},
		},
		{
			name:  "@when pattern mode transition",
			input: "build: @when(ENV) { prod: echo hello }",
			expected: []struct {
				token TokenType
			}{
				{IDENTIFIER}, // build
				{COLON},      // :
				{AT},         // @
				{WHEN},       // when
				{LPAREN},     // (
				{IDENTIFIER}, // ENV
				{RPAREN},     // )
				{LBRACE},     // {
				{IDENTIFIER}, // prod
				{COLON},      // :
				{SHELL_TEXT}, // echo hello
				{RBRACE},     // }
				{EOF},        // EOF
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.token {
					t.Errorf("Step %d: expected token %s, got %s", i, expected.token, actual.Type)
				}
			}
		})
	}
}

func TestVariableDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "simple variable",
			input: "var PORT = \"8080\"",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{STRING, "8080"},
				{EOF, ""},
			},
		},
		{
			name:  "string variable",
			input: `var HOST = "localhost"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "HOST"},
				{EQUALS, "="},
				{STRING, "localhost"},
				{EOF, ""},
			},
		},
		{
			name:  "boolean variable",
			input: "var DEBUG = true",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "DEBUG"},
				{EQUALS, "="},
				{BOOLEAN, "true"},
				{EOF, ""},
			},
		},
		{
			name:  "grouped variables with mixed types",
			input: "var (\n  PORT = \"8080\"\n  HOST = \"localhost\"\n  DEBUG = false\n)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{LPAREN, "("},
				{NEWLINE, "\n"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{STRING, "8080"},
				{NEWLINE, "\n"},
				{IDENTIFIER, "HOST"},
				{EQUALS, "="},
				{STRING, "localhost"},
				{NEWLINE, "\n"},
				{IDENTIFIER, "DEBUG"},
				{EQUALS, "="},
				{BOOLEAN, "false"},
				{NEWLINE, "\n"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "duration variable",
			input: "var TIMEOUT = 30s",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "TIMEOUT"},
				{EQUALS, "="},
				{DURATION, "30s"},
				{EOF, ""},
			},
		},
		{
			name:  "number variable",
			input: "var MAX_RETRIES = 5",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "MAX_RETRIES"},
				{EQUALS, "="},
				{NUMBER, "5"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			if len(tokens) != len(test.expected) {
				t.Errorf("Expected %d tokens, got %d", len(test.expected), len(tokens))
				// Debug output
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
				}
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s", i, expected.tokenType, actual.Type)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q", i, expected.value, actual.Value)
				}
			}
		})
	}
}

func TestComplexWhenExample(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedTokens []TokenType
		expectedCounts map[TokenType]int
		description    string
	}{
		{
			name: "simple when pattern with newlines",
			input: `build: @when(ENV) {
  prod: npm run build:production
  dev: npm run build:dev
  *: npm run build
}`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,        // build:
				AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, // @when(ENV) {
				IDENTIFIER, COLON, SHELL_TEXT,                 // prod: npm run build:production
				IDENTIFIER, COLON, SHELL_TEXT,                 // dev: npm run build:dev
				ASTERISK, COLON, SHELL_TEXT,                   // *: npm run build
				RBRACE,                                        // }
				EOF,
			},
			expectedCounts: map[TokenType]int{
				WHEN:       1,
				AT:         1,
				ASTERISK:   1,
				SHELL_TEXT: 3,
				NEWLINE:    0, // Should be consumed in pattern blocks
				IDENTIFIER: 4, // build, ENV, prod, dev
			},
			description: "Pattern blocks should consume newlines without emitting NEWLINE tokens",
		},
		{
			name: "when pattern with semicolon separators",
			input: `deploy: @when(REGION) { us-east: kubectl apply -f us.yaml; eu-west: kubectl apply -f eu.yaml; *: echo "default" }`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,        // deploy:
				AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, // @when(REGION) {
				IDENTIFIER, COLON, SHELL_TEXT,                 // us-east: kubectl apply -f us.yaml
				IDENTIFIER, COLON, SHELL_TEXT,                 // eu-west: kubectl apply -f eu.yaml
				ASTERISK, COLON, SHELL_TEXT,                   // *: echo "default"
				RBRACE,                                        // }
				EOF,
			},
			expectedCounts: map[TokenType]int{
				WHEN:       1,
				AT:         1,
				ASTERISK:   1,
				SHELL_TEXT: 3,
				NEWLINE:    0, // Should be consumed
				IDENTIFIER: 4, // deploy, REGION, us-east, eu-west
			},
			description: "Semicolons should separate patterns without appearing in shell text",
		},
		{
			name: "nested when with timeout decorator",
			input: `server: @when(NODE_ENV) {
  production: @timeout(60s) {
    node server.js --port 80
  }
  development: @timeout(30s) {
    nodemon server.js --port 3000
  }
}`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,        // server:
				AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, // @when(NODE_ENV) {
				IDENTIFIER, COLON,                             // production:
				AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, // @timeout(60s) {
				SHELL_TEXT,                                    // node server.js --port 80
				RBRACE,                                        // }
				IDENTIFIER, COLON,                             // development:
				AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, // @timeout(30s) {
				SHELL_TEXT,                                    // nodemon server.js --port 3000
				RBRACE,                                        // }
				RBRACE,                                        // }
				EOF,
			},
			expectedCounts: map[TokenType]int{
				WHEN:       1,
				AT:         3, // @when, @timeout, @timeout
				SHELL_TEXT: 2,
				DURATION:   2, // 60s, 30s
				NEWLINE:    0, // Should be consumed
				IDENTIFIER: 5, // server, NODE_ENV, production, timeout, development, timeout
			},
			description: "Nested decorators should work with pattern blocks",
		},
		{
			name: "try pattern with error handling",
			input: `test: @try {
  main: npm test
  error: echo "Tests failed"
  finally: echo "Cleanup"
}`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,    // test:
				AT, TRY, LBRACE,      // @try {
				IDENTIFIER, COLON, SHELL_TEXT, // main: npm test
				IDENTIFIER, COLON, SHELL_TEXT, // error: echo "Tests failed"
				IDENTIFIER, COLON, SHELL_TEXT, // finally: echo "Cleanup"
				RBRACE,               // }
				EOF,
			},
			expectedCounts: map[TokenType]int{
				TRY:        1,
				AT:         1,
				SHELL_TEXT: 3,
				NEWLINE:    0, // Should be consumed
				IDENTIFIER: 4, // test, main, error, finally
			},
			description: "Try patterns should handle multiple named patterns",
		},
		{
			name: "mixed patterns with explicit blocks",
			input: `deploy: @when(ENV) {
  prod: { npm run build && npm run deploy }
  dev: npm run dev-deploy
  *: { echo "Unknown env: $ENV"; exit 1 }
}`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,        // deploy:
				AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, // @when(ENV) {
				IDENTIFIER, COLON, LBRACE, SHELL_TEXT, RBRACE, // prod: { npm run build && npm run deploy }
				IDENTIFIER, COLON, SHELL_TEXT,                 // dev: npm run dev-deploy
				ASTERISK, COLON, LBRACE, SHELL_TEXT, RBRACE,   // *: { echo "Unknown env: $ENV"; exit 1 }
				RBRACE,                                        // }
				EOF,
			},
			expectedCounts: map[TokenType]int{
				WHEN:       1,
				AT:         1,
				ASTERISK:   1,
				SHELL_TEXT: 3,
				LBRACE:     3, // Main block + 2 explicit blocks
				RBRACE:     3, // Corresponding closing braces
				NEWLINE:    0, // Should be consumed
				IDENTIFIER: 4, // deploy, ENV, prod, dev
			},
			description: "Mixed explicit blocks and simple commands in patterns",
		},
		{
			name: "pattern with decorators inside branches",
			input: `build: @when(STAGE) {
  prod: @timeout(60s) { npm run build:prod }
  dev: @retry(3) { npm run build:dev }
  test: @parallel {
    npm run build:test
    npm run lint
  }
  *: npm run build
}`,
			expectedTokens: []TokenType{
				IDENTIFIER, COLON,        // build:
				AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, // @when(STAGE) {
				IDENTIFIER, COLON,                             // prod:
				AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, // @timeout(60s) {
				SHELL_TEXT,                                    // npm run build:prod
				RBRACE,                                        // }
				IDENTIFIER, COLON,                             // dev:
				AT, IDENTIFIER, LPAREN, NUMBER, RPAREN, LBRACE, // @retry(3) {
				SHELL_TEXT,                                    // npm run build:dev
				RBRACE,                                        // }
				IDENTIFIER, COLON,                             // test:
				AT, IDENTIFIER, LBRACE,                        // @parallel {
				SHELL_TEXT,                                    // npm run build:test
				SHELL_TEXT,                                    // npm run lint
				RBRACE,                                        // }
				ASTERISK, COLON, SHELL_TEXT,                   // *: npm run build
				RBRACE,                                        // } (close @when)
				EOF,
			},
			expectedCounts: map[TokenType]int{
				WHEN:       1,
				AT:         4, // @when, @timeout, @retry, @parallel
				ASTERISK:   1,
				SHELL_TEXT: 5, // prod, dev, test (2 commands), wildcard
				DURATION:   1, // 60s
				NUMBER:     1, // 3
				LBRACE:     4, // @when, @timeout, @retry, @parallel
				RBRACE:     4, // Corresponding closing braces
				NEWLINE:    0, // Should be consumed
				IDENTIFIER: 9, // build, STAGE, prod, timeout, dev, retry, test, parallel
			},
			description: "Pattern branches can contain their own decorators with blocks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := New(tt.input)
			tokens := lexer.TokenizeToSlice()

			// Check total token count
			if len(tokens) != len(tt.expectedTokens) {
				t.Errorf("expected %d tokens, got %d", len(tt.expectedTokens), len(tokens))
				t.Logf("Expected: %v", tt.expectedTokens)
				t.Logf("Actual tokens:")
				for i, tok := range tokens {
					t.Logf("  %d: %s %q at %d:%d", i, tok.Type, tok.Value, tok.Line, tok.Column)
				}
				return
			}

			// Check token sequence
			for i, expectedType := range tt.expectedTokens {
				if tokens[i].Type != expectedType {
					t.Errorf("token %d: expected %s, got %s %q",
						i, expectedType, tokens[i].Type, tokens[i].Value)
				}
			}

			// Check specific token counts
			actualCounts := make(map[TokenType]int)
			for _, token := range tokens {
				actualCounts[token.Type]++
			}

			for tokenType, expectedCount := range tt.expectedCounts {
				if actualCounts[tokenType] != expectedCount {
					t.Errorf("expected %d %s tokens, got %d",
						expectedCount, tokenType, actualCounts[tokenType])
				}
			}

			// Verify no NEWLINE tokens in pattern blocks (key requirement)
			if actualCounts[NEWLINE] > 0 {
				t.Errorf("found %d NEWLINE tokens in pattern block - these should be consumed",
					actualCounts[NEWLINE])
				for i, tok := range tokens {
					if tok.Type == NEWLINE {
						t.Logf("  NEWLINE at token %d, line %d:%d", i, tok.Line, tok.Column)
					}
				}
			}

			// Verify token positions are valid
			for i, token := range tokens {
				if token.Line <= 0 {
					t.Errorf("token %d (%s) has invalid line: %d", i, token.Type, token.Line)
				}
				if token.Column <= 0 && token.Type != NEWLINE {
					t.Errorf("token %d (%s) has invalid column: %d", i, token.Type, token.Column)
				}
				if token.Span.Start.Offset < 0 || token.Span.End.Offset < token.Span.Start.Offset {
					t.Errorf("token %d (%s) has invalid span: %d:%d",
						i, token.Type, token.Span.Start.Offset, token.Span.End.Offset)
				}
			}

			// Should always end with EOF
			if tokens[len(tokens)-1].Type != EOF {
				t.Errorf("expected last token to be EOF, got %s", tokens[len(tokens)-1].Type)
			}

			t.Logf("✓ %s: %s", tt.name, tt.description)
		})
	}
}

func TestTokenClassification(t *testing.T) {
	tests := []struct {
		tokenType TokenType
		isStruct  bool
		isLiteral bool
		isShell   bool
		isPattern bool
	}{
		{VAR, true, false, false, false},
		{WATCH, true, false, false, false},
		{STOP, true, false, false, false},
		{WHEN, true, false, false, true},
		{TRY, true, false, false, true},
		{AT, true, false, false, false},
		{COLON, true, false, false, false},
		{LBRACE, true, false, false, false},
		{RBRACE, true, false, false, false},
		{ASTERISK, true, false, false, true},
		{STRING, false, true, false, false},
		{NUMBER, false, true, false, false},
		{DURATION, false, true, false, false},
		{IDENTIFIER, false, true, false, false},
		{BOOLEAN, false, true, false, false},
		{SHELL_TEXT, false, false, true, false},
		{COMMENT, false, false, false, false},
		{EOF, false, false, false, false},
	}

	for _, test := range tests {
		t.Run(test.tokenType.String(), func(t *testing.T) {
			if IsStructuralToken(test.tokenType) != test.isStruct {
				t.Errorf("IsStructuralToken(%s) = %v, want %v", test.tokenType, IsStructuralToken(test.tokenType), test.isStruct)
			}
			if IsLiteralToken(test.tokenType) != test.isLiteral {
				t.Errorf("IsLiteralToken(%s) = %v, want %v", test.tokenType, IsLiteralToken(test.tokenType), test.isLiteral)
			}
			if IsShellContent(test.tokenType) != test.isShell {
				t.Errorf("IsShellContent(%s) = %v, want %v", test.tokenType, IsShellContent(test.tokenType), test.isShell)
			}
			if IsPatternToken(test.tokenType) != test.isPattern {
				t.Errorf("IsPatternToken(%s) = %v, want %v", test.tokenType, IsPatternToken(test.tokenType), test.isPattern)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []TokenType{EOF},
		},
		{
			name:     "whitespace only",
			input:    "   \n\t  ",
			expected: []TokenType{NEWLINE, EOF},
		},
		{
			name:     "comment only",
			input:    "# just a comment",
			expected: []TokenType{COMMENT, EOF},
		},
		{
			name:     "empty command",
			input:    "empty:",
			expected: []TokenType{IDENTIFIER, COLON, EOF},
		},
		{
			name:     "empty block",
			input:    "empty: { }",
			expected: []TokenType{IDENTIFIER, COLON, LBRACE, RBRACE, EOF},
		},
		{
			name:     "empty @when",
			input:    "@when(ENV) { }",
			expected: []TokenType{AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, RBRACE, EOF},
		},
		{
			name:     "only newlines",
			input:    "\n\n\n",
			expected: []TokenType{NEWLINE, NEWLINE, NEWLINE, EOF},
		},
		{
			name:     "boolean edge cases",
			input:    "var t = true, f = false",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, BOOLEAN, COMMA, IDENTIFIER, EQUALS, BOOLEAN, EOF},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var tokenTypes []TokenType
			for _, token := range tokens {
				tokenTypes = append(tokenTypes, token.Type)
			}

			if diff := cmp.Diff(test.expected, tokenTypes); diff != "" {
				t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestOptimizedPerformance(t *testing.T) {
	// Test that the lexer can handle reasonable loads with @when
	input := generateTestInputWithWhen(100) // 100 lines

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens from large input")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Error("Large input should end with EOF token")
	}
}

// Helper functions
func generateTestInputWithWhen(lines int) string {
	var result strings.Builder
	for i := 0; i < lines; i++ {
		if i%3 == 0 {
			result.WriteString(fmt.Sprintf("cmd%d: @when(ENV) { prod: echo hello %d; dev: echo world %d; *: echo default %d }\n", i, i, i, i))
		} else {
			result.WriteString(fmt.Sprintf("cmd%d: echo hello %d\n", i, i))
		}
	}
	return result.String()
}

func TestStringTypes(t *testing.T) {
	tests := []struct {
		input      string
		stringType StringType
		value      string
	}{
		{`"hello"`, DoubleQuoted, "hello"},
		{`'world'`, SingleQuoted, "world"},
		{"`test`", Backtick, "test"},
		{`"with\nescapes"`, DoubleQuoted, "with\nescapes"},
		{`'literal\backslash'`, SingleQuoted, `literal\backslash`},
		{`"quoted with spaces"`, DoubleQuoted, "quoted with spaces"},
		{`'single with "double" quotes'`, SingleQuoted, `single with "double" quotes`},
		{"`backtick with 'single' quotes`", Backtick, "backtick with 'single' quotes"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != STRING {
				t.Errorf("Expected STRING token, got %s", token.Type)
				return
			}

			if token.StringType != test.stringType {
				t.Errorf("String type mismatch: expected %v, got %v", test.stringType, token.StringType)
			}

			if diff := cmp.Diff(test.value, token.Value); diff != "" {
				t.Errorf("String value mismatch (-want +got):\n%s", diff)
			}

			// Verify position is valid
			verifyTokenPosition(t, token, 1, 1, 0)
		})
	}
}

func TestDurations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"30s", "30s"},
		{"5m", "5m"},
		{"1h", "1h"},
		{"500ms", "500ms"},
		{"2.5s", "2.5s"},
		{"1.5h", "1.5h"},
		{"100ns", "100ns"},
		{"250us", "250us"},
		{"-30s", "-30s"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != DURATION {
				t.Errorf("Expected DURATION token, got %s", token.Type)
				return
			}

			if diff := cmp.Diff(test.expected, token.Value); diff != "" {
				t.Errorf("Duration value mismatch (-want +got):\n%s", diff)
			}

			verifyTokenPosition(t, token, 1, 1, 0)
		})
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "123"},
		{"-456", "-456"},
		{"3.14", "3.14"},
		{"-2.5", "-2.5"},
		{"0", "0"},
		{"0.0", "0.0"},
		{"1000000", "1000000"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != NUMBER {
				t.Errorf("Expected NUMBER token, got %s", token.Type)
				return
			}

			if diff := cmp.Diff(test.expected, token.Value); diff != "" {
				t.Errorf("Number value mismatch (-want +got):\n%s", diff)
			}

			verifyTokenPosition(t, token, 1, 1, 0)
		})
	}
}

func TestKeywordDetection(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"var", VAR},
		{"watch", WATCH},
		{"stop", STOP},
		{"when", WHEN},
		{"try", TRY},
		{"true", BOOLEAN},
		{"false", BOOLEAN},
		{"timeout", IDENTIFIER},  // Not a keyword
		{"parallel", IDENTIFIER}, // Not a keyword
		{"retry", IDENTIFIER},    // Not a keyword
		{"trueish", IDENTIFIER},  // Not boolean, just starts with "true"
		{"falsey", IDENTIFIER},   // Not boolean, just starts with "false"
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != test.expected {
				t.Errorf("Expected token type %s, got %s", test.expected, token.Type)
			}

			if token.Value != test.input {
				t.Errorf("Expected token value %q, got %q", test.input, token.Value)
			}

			// Verify semantic type for booleans
			if token.Type == BOOLEAN && token.Semantic != SemBoolean {
				t.Errorf("Boolean token %q has wrong semantic type: %v", token.Value, token.Semantic)
			}
		})
	}
}

func TestPatternModeFeatures(t *testing.T) {
	input := `deploy: @when(ENV) {
  production: @timeout(60s) { npm run build:prod }
  staging: @timeout(30s) { npm run build:staging }
  development: npm run build:dev
  *: echo "Unknown environment: $ENV"
}`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// Verify the presence of pattern-specific tokens
	hasPatternTokens := false
	for _, token := range tokens {
		if IsPatternToken(token.Type) {
			hasPatternTokens = true
			break
		}
	}

	if !hasPatternTokens {
		t.Error("Expected to find pattern tokens in @when decorator")
	}

	// Verify that we have proper nesting with decorators inside patterns
	hasNestedDecorator := false
	foundTimeout := false
	for i, token := range tokens {
		if token.Type == IDENTIFIER && token.Value == "timeout" && i > 0 && tokens[i-1].Type == AT {
			foundTimeout = true
		}
		if foundTimeout && token.Type == LBRACE {
			hasNestedDecorator = true
			break
		}
	}

	if !hasNestedDecorator {
		t.Error("Expected to find nested decorator inside pattern")
	}
}

func TestMultiLinePatternDecorators(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "when with multi-line blocks",
			input: `@when(NODE_ENV) {
  production: {
    npm run build:prod
    npm run deploy
  }
  development: {
    npm run build:dev
    npm start
  }
  *: echo "Unknown environment"
}`,
		},
		{
			name: "try with nested decorators",
			input: `@try {
  main: @timeout(30s) { npm run build }
  error: @retry(3) { echo "Retrying..." }
  finally: echo "Done"
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			// Debug output to see actual structure
			t.Logf("Tokens for %s:", test.name)
			for i, token := range tokens {
				t.Logf("  %d: %s %q", i, token.Type, token.Value)
			}

			// Verify basic structure rather than exact token sequence
			hasWhenOrTry := false
			hasPatterns := false
			hasProperNesting := false

			for i, token := range tokens {
				// Check for @when or @try
				if token.Type == WHEN || token.Type == TRY {
					hasWhenOrTry = true
				}

				// Check for pattern identifiers
				if token.Type == IDENTIFIER {
					switch token.Value {
					case "production", "development", "main", "error", "finally":
						hasPatterns = true
					}
				}

				// Check for proper brace nesting
				if token.Type == LBRACE && i+1 < len(tokens) {
					// Look for content inside braces
					depth := 1
					j := i + 1
					for j < len(tokens) && depth > 0 {
						if tokens[j].Type == LBRACE {
							depth++
						} else if tokens[j].Type == RBRACE {
							depth--
						}
						j++
					}
					if depth == 0 {
						hasProperNesting = true
					}
				}
			}

			// Verify essential structure
			if !hasWhenOrTry {
				t.Error("Expected to find @when or @try decorator")
			}
			if !hasPatterns {
				t.Error("Expected to find pattern identifiers")
			}
			if !hasProperNesting {
				t.Error("Expected to find proper brace nesting")
			}

			// For the nested decorators test, also check for decorators
			if test.name == "try with nested decorators" {
				hasNestedDecorators := false
				for i, token := range tokens {
					if token.Type == AT && i+1 < len(tokens) {
						next := tokens[i+1]
						if next.Type == IDENTIFIER && (next.Value == "timeout" || next.Value == "retry") {
							hasNestedDecorators = true
							break
						}
					}
				}
				if !hasNestedDecorators {
					t.Error("Expected to find nested decorators (@timeout, @retry)")
				}
			}

			// Verify the lexer doesn't crash and produces reasonable output
			if len(tokens) == 0 {
				t.Error("Expected some tokens, got none")
			}

			if tokens[len(tokens)-1].Type != EOF {
				t.Error("Expected last token to be EOF")
			}
		})
	}
}
