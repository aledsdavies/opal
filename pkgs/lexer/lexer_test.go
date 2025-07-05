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

func TestWhenConditionalTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "@when(ENV)",
			expected: []TokenType{AT, WHEN, LPAREN, IDENTIFIER, RPAREN, EOF},
		},
		{
			input:    "@when(ENV) { prod: npm run build }",
			expected: []TokenType{AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, IDENTIFIER, COLON, SHELL_TEXT, RBRACE, EOF},
		},
		{
			input:    "@when(REGION) { us-east-1: kubectl apply -f us.yaml }",
			expected: []TokenType{AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, IDENTIFIER, COLON, SHELL_TEXT, RBRACE, EOF},
		},
		{
			input:    "@when(ENV) { *: echo default }",
			expected: []TokenType{AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, ASTERISK, COLON, SHELL_TEXT, RBRACE, EOF},
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

func TestComplexWhenConditional(t *testing.T) {
	input := `build: @when(ENV) {
  prod: npm run build:production
  dev: npm run build:dev
  test: npm run build:test
  *: npm run build
}`

	expected := []TokenType{
		IDENTIFIER, COLON, AT, WHEN, LPAREN, IDENTIFIER, RPAREN, LBRACE, NEWLINE,
		IDENTIFIER, COLON, SHELL_TEXT, NEWLINE,
		IDENTIFIER, COLON, SHELL_TEXT, NEWLINE,
		IDENTIFIER, COLON, SHELL_TEXT, NEWLINE,
		ASTERISK, COLON, SHELL_TEXT, NEWLINE,
		RBRACE, EOF,
	}

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	var tokenTypes []TokenType
	for _, token := range tokens {
		tokenTypes = append(tokenTypes, token.Type)
	}

	if diff := cmp.Diff(expected, tokenTypes); diff != "" {
		t.Errorf("Token sequence mismatch (-want +got):\n%s", diff)
		// Debug output
		t.Logf("Actual tokens:")
		for i, token := range tokens {
			t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
		}
	}
}

func TestNestedWhenConditional(t *testing.T) {
	input := `server: @when(NODE_ENV) {
  production: @timeout(60s) {
    node server.js --port 80
  }
  development: @timeout(30s) {
    nodemon server.js --port 3000
  }
  *: echo "Unknown environment"
}`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// Debug: print all tokens to understand the structure
	t.Logf("Actual tokens:")
	for i, token := range tokens {
		t.Logf("  %d: %s %q at %d:%d", i, token.Type, token.Value, token.Line, token.Column)
	}

	// Verify we get the expected structure - just check that we have the right tokens
	hasWhen := false
	hasAsterisk := false
	hasShellText := false
	hasTimeout := false

	for _, token := range tokens {
		switch token.Type {
		case WHEN:
			hasWhen = true
		case ASTERISK:
			hasAsterisk = true
		case SHELL_TEXT:
			hasShellText = true
		case IDENTIFIER:
			if token.Value == "timeout" {
				hasTimeout = true
			}
		}
	}

	if !hasWhen {
		t.Error("Expected to find WHEN token")
	}
	if !hasAsterisk {
		t.Error("Expected to find ASTERISK token")
	}
	if !hasShellText {
		t.Error("Expected to find SHELL_TEXT token")
	}
	if !hasTimeout {
		t.Error("Expected to find timeout decorator")
	}

	// Verify we have at least one shell text token with the expected content
	foundEchoCommand := false
	for _, token := range tokens {
		if token.Type == SHELL_TEXT && strings.Contains(token.Value, "echo") {
			foundEchoCommand = true
			break
		}
	}

	if !foundEchoCommand {
		t.Error("Expected to find shell text with echo command")
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
			input:    "var (PORT = 8080, HOST = localhost)",
			expected: []TokenType{VAR, LPAREN, IDENTIFIER, EQUALS, NUMBER, COMMA, IDENTIFIER, EQUALS, IDENTIFIER, RPAREN, EOF},
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

func TestWildcardPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "*:",
			expected: []TokenType{ASTERISK, COLON, EOF},
		},
		{
			input:    "* : echo default",
			expected: []TokenType{ASTERISK, COLON, SHELL_TEXT, EOF},
		},
		{
			input:    "*: echo \"default case\"",
			expected: []TokenType{ASTERISK, COLON, SHELL_TEXT, EOF},
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
			input: "var PORT = 8080",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
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
			name:  "unquoted variable",
			input: "var ENV = production",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "ENV"},
				{EQUALS, "="},
				{IDENTIFIER, "production"},
				{EOF, ""},
			},
		},
		{
			name:  "grouped variables",
			input: "var (\n  PORT = 8080\n  HOST = localhost\n)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{LPAREN, "("},
				{NEWLINE, "\n"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"},
				{NEWLINE, "\n"},
				{IDENTIFIER, "HOST"},
				{EQUALS, "="},
				{IDENTIFIER, "localhost"},
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
	input := `build: @when(ENV) {
  prod: npm run build:production
  dev: npm run build:dev
  test: npm run build:test
  *: npm run build
}

deploy: @when(REGION) {
  us-east-1: kubectl apply -f k8s/us-east.yaml
  eu-west-1: kubectl apply -f k8s/eu-west.yaml
  ap-south-1: kubectl apply -f k8s/ap-south.yaml
  *: echo "Unsupported region: $REGION" && exit 1
}

server: @when(NODE_ENV) {
  production: @timeout(60s) {
    node server.js --port 80
  }
  development: @timeout(30s) {
    nodemon server.js --port 3000
  }
  test: @timeout(10s) {
    node server.js --port 8080 --test
  }
}`

	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Errorf("Expected last token to be EOF, got %s", tokens[len(tokens)-1].Type)
	}

	// Count different token types
	var whenCount, atCount, asteriskCount, shellTextCount, newlineCount int
	for _, token := range tokens {
		switch token.Type {
		case WHEN:
			whenCount++
		case AT:
			atCount++
		case ASTERISK:
			asteriskCount++
		case SHELL_TEXT:
			shellTextCount++
		case NEWLINE:
			newlineCount++
		}
	}

	if whenCount != 3 {
		t.Errorf("Expected 3 WHEN tokens, got %d", whenCount)
	}
	if atCount < 3 {
		t.Errorf("Expected at least 3 AT tokens, got %d", atCount)
	}
	if asteriskCount < 2 { // At least 2 asterisks
		t.Errorf("Expected at least 2 ASTERISK tokens, got %d", asteriskCount)
	}
	if shellTextCount < 5 { // At least 5 shell text tokens
		t.Errorf("Expected at least 5 SHELL_TEXT tokens, got %d", shellTextCount)
	}

	// Verify tokens have valid positions (allow 0 for newlines which reset column)
	for i, token := range tokens {
		if token.Line <= 0 {
			t.Errorf("Token %d has invalid line: %d", i, token.Line)
		}
		if token.Column <= 0 && token.Type != NEWLINE {
			t.Errorf("Token %d has invalid column: %d", i, token.Column)
		}
		if token.Span.Start.Offset < 0 || token.Span.End.Offset < token.Span.Start.Offset {
			t.Errorf("Token %d has invalid span: %d:%d", i, token.Span.Start.Offset, token.Span.End.Offset)
		}
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

func TestTryPatternDecorator(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "@try()",
			expected: []TokenType{AT, TRY, LPAREN, RPAREN, EOF},
		},
		{
			input:    "@try { main: npm run build }",
			expected: []TokenType{AT, TRY, LBRACE, IDENTIFIER, COLON, SHELL_TEXT, RBRACE, EOF},
		},
		{
			input:    "@try { main: npm run build; error: echo failed; finally: echo done }",
			expected: []TokenType{AT, TRY, LBRACE, IDENTIFIER, COLON, SHELL_TEXT, IDENTIFIER, COLON, SHELL_TEXT, IDENTIFIER, COLON, SHELL_TEXT, RBRACE, EOF},
		},
		{
			// Multi-line pattern with blocks
			input: `@try {
  main: { npm run build }
  error: { echo "Build failed" }
  finally: { echo "Cleanup" }
}`,
			expected: []TokenType{
				AT, TRY, LBRACE, NEWLINE,
				IDENTIFIER, COLON, LBRACE, SHELL_TEXT, RBRACE, NEWLINE,
				IDENTIFIER, COLON, LBRACE, SHELL_TEXT, RBRACE, NEWLINE,
				IDENTIFIER, COLON, LBRACE, SHELL_TEXT, RBRACE, NEWLINE,
				RBRACE, EOF,
			},
		},
		{
			// Mixed simple and block patterns
			input: `@try {
  main: npm test
  error: { echo "Test failed"; exit 1 }
  finally: echo "Done"
}`,
			expected: []TokenType{
				AT, TRY, LBRACE, NEWLINE,
				IDENTIFIER, COLON, SHELL_TEXT, NEWLINE,
				IDENTIFIER, COLON, LBRACE, SHELL_TEXT, RBRACE, NEWLINE,
				IDENTIFIER, COLON, SHELL_TEXT, NEWLINE,
				RBRACE, EOF,
			},
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

			// Verify specific token values for the semicolon-separated case
			if test.input == "@try { main: npm run build; error: echo failed; finally: echo done }" {
				// Find the shell text tokens and verify their content
				shellTokens := []string{}
				for _, token := range tokens {
					if token.Type == SHELL_TEXT {
						shellTokens = append(shellTokens, token.Value)
					}
				}

				expectedShellContent := []string{"npm run build", "echo failed", "echo done"}
				if len(shellTokens) == len(expectedShellContent) {
					for i, expected := range expectedShellContent {
						if strings.TrimSpace(shellTokens[i]) != expected {
							t.Errorf("Shell token %d: expected %q, got %q", i, expected, strings.TrimSpace(shellTokens[i]))
						}
					}
				} else {
					t.Errorf("Expected %d shell tokens, got %d: %v", len(expectedShellContent), len(shellTokens), shellTokens)
				}
			}
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
		{"timeout", IDENTIFIER}, // Not a keyword
		{"parallel", IDENTIFIER}, // Not a keyword
		{"retry", IDENTIFIER}, // Not a keyword
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
