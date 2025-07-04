package lexer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/stdlib"
	"github.com/google/go-cmp/cmp"
)

func init() {
	// Register test decorators that are referenced in tests but not in the minimal standard registry
	testDecorators := []*stdlib.DecoratorSignature{
		{
			Name:          "timeout",
			Type:          stdlib.BlockDecorator,
			Semantic:      stdlib.SemDecorator,
			Description:   "Sets execution timeout (test decorator)",
			RequiresBlock: true,
			Args: []stdlib.ArgumentSpec{
				{Name: "duration", Type: stdlib.DurationArg, Optional: false},
			},
		},
		{
			Name:        "now",
			Type:        stdlib.FunctionDecorator,
			Semantic:    stdlib.SemFunction,
			Description: "Current timestamp (test decorator)",
			Args:        []stdlib.ArgumentSpec{}, // No arguments
		},
		{
			Name:          "watch-files",
			Type:          stdlib.BlockDecorator,
			Semantic:      stdlib.SemDecorator,
			Description:   "Watches files for changes (test decorator)",
			RequiresBlock: true,
			Args: []stdlib.ArgumentSpec{
				{Name: "pattern", Type: stdlib.StringArg, Optional: false},
			},
		},
		{
			Name:          "retry",
			Type:          stdlib.BlockDecorator,
			Semantic:      stdlib.SemDecorator,
			Description:   "Retries command on failure (test decorator)",
			RequiresBlock: true,
			Args: []stdlib.ArgumentSpec{
				{Name: "attempts", Type: stdlib.NumberArg, Optional: true, Default: "3"},
			},
		},
	}

	for _, decorator := range testDecorators {
		stdlib.RegisterDecorator(decorator)
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
		})
	}
}

func TestAtToken(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{
			input:    "@timeout",
			expected: []TokenType{AT, IDENTIFIER, EOF},
		},
		{
			input:    "@timeout(30s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, EOF},
		},
		{
			input:    "@timeout(30s) { echo hello }",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, DURATION, RPAREN, LBRACE, IDENTIFIER, RBRACE, EOF},
		},
		{
			input:    "@retry(attempts=3, delay=1.5s)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, IDENTIFIER, EQUALS, NUMBER, COMMA, IDENTIFIER, EQUALS, DURATION, RPAREN, EOF},
		},
		{
			input:    "@watch-files(debounce=500ms)",
			expected: []TokenType{AT, IDENTIFIER, LPAREN, IDENTIFIER, EQUALS, DURATION, RPAREN, EOF},
		},
		{
			input:    "@var{ echo hello }",
			expected: []TokenType{AT, IDENTIFIER, LBRACE, IDENTIFIER, RBRACE, EOF},
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

func TestDecoratorSemanticTypes(t *testing.T) {
	tests := []struct {
		input            string
		identifierValues []string
		semanticTypes    []SemanticTokenType
	}{
		{
			input:            "@timeout(30s)",
			identifierValues: []string{"timeout"},
			semanticTypes:    []SemanticTokenType{SemDecorator},
		},
		{
			input:            "var server",
			identifierValues: []string{"server"},
			semanticTypes:    []SemanticTokenType{SemCommand},
		},
		{
			// FIXED: Shell text after : or { has no leading space
			input:            "server: @timeout(30s) { echo hello }",
			identifierValues: []string{"server", "timeout", "echo hello"}, // No leading space
			semanticTypes:    []SemanticTokenType{SemCommand, SemDecorator, SemShellText},
		},
		{
			// FIXED: Internal spaces are preserved
			input:            "server: echo port @var(PORT)",
			identifierValues: []string{"server", "echo port ", "var", "PORT"}, // Space after "port" preserved
			semanticTypes:    []SemanticTokenType{SemCommand, SemShellText, SemVariable, SemParameter},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			var identifierTokens []Token
			for _, token := range tokens {
				if token.Type == IDENTIFIER {
					identifierTokens = append(identifierTokens, token)
				}
			}

			if len(identifierTokens) != len(test.identifierValues) {
				t.Errorf("Expected %d identifiers, got %d", len(test.identifierValues), len(identifierTokens))

				// Debug: show all tokens
				t.Logf("All tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q (semantic: %v)", i, token.Type, token.Value, token.Semantic)
				}
				return
			}

			for i, token := range identifierTokens {
				if token.Value != test.identifierValues[i] {
					t.Errorf("Identifier %d: expected %q, got %q", i, test.identifierValues[i], token.Value)
				}
				if token.Semantic != test.semanticTypes[i] {
					t.Errorf("Identifier %d semantic: expected %v (%d), got %v (%d)", i, test.semanticTypes[i], test.semanticTypes[i], token.Semantic, token.Semantic)
				}
			}
		})
	}
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
		{"250us", "250us"}, // ASCII microsecond notation
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
		})
	}
}

func TestNumberVsDuration(t *testing.T) {
	tests := []struct {
		input         string
		expectedType  TokenType
		expectedValue string
	}{
		{"123", NUMBER, "123"},
		{"123s", DURATION, "123s"},
		{"3.14", NUMBER, "3.14"},
		{"3.14s", DURATION, "3.14s"},
		{"-42", NUMBER, "-42"},
		{"42ms", DURATION, "42ms"},
		{"8080", NUMBER, "8080"}, // Port number stays as NUMBER
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			token := lexer.NextToken()

			if token.Type != test.expectedType {
				t.Errorf("Expected %s token, got %s", test.expectedType, token.Type)
				return
			}

			if diff := cmp.Diff(test.expectedValue, token.Value); diff != "" {
				t.Errorf("Value mismatch (-want +got):\n%s", diff)
			}
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
		})
	}
}

func TestCommandMode(t *testing.T) {
	input := "build: echo hello world"
	lexer := New(input)

	// Should start in language mode
	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", ident.Type)
	}

	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// After colon, should switch to command mode for simple commands
	shellText := lexer.NextToken()
	if shellText.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText.Type)
	}

	// FIXED: No leading space after colon
	if diff := cmp.Diff("echo hello world", shellText.Value); diff != "" {
		t.Errorf("Shell text mismatch (-want +got):\n%s", diff)
	}
}

func TestSemicolonInShellCommand(t *testing.T) {
	// Test that semicolons are part of shell commands, not separators
	input := "build: echo hello; echo world"
	lexer := New(input)

	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER || ident.Value != "build" {
		t.Errorf("Expected IDENTIFIER 'build', got %s %q", ident.Type, ident.Value)
	}

	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// The entire shell command should be one token including the semicolon
	shellText := lexer.NextToken()
	if shellText.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellText.Type)
	}

	// FIXED: No leading space after colon
	expectedShellText := "echo hello; echo world"
	if diff := cmp.Diff(expectedShellText, shellText.Value); diff != "" {
		t.Errorf("Shell text mismatch (-want +got):\n%s", diff)
	}
}

func TestBlockWithSemicolons(t *testing.T) {
	// Test that semicolons inside blocks are part of shell commands
	input := "deploy: { cd src; make clean; make install }"
	lexer := New(input)

	// deploy
	ident := lexer.NextToken()
	if ident.Type != IDENTIFIER || ident.Value != "deploy" {
		t.Errorf("Expected IDENTIFIER 'deploy', got %s %q", ident.Type, ident.Value)
	}

	// :
	colon := lexer.NextToken()
	if colon.Type != COLON {
		t.Errorf("Expected COLON, got %s", colon.Type)
	}

	// {
	lbrace := lexer.NextToken()
	if lbrace.Type != LBRACE {
		t.Errorf("Expected LBRACE, got %s", lbrace.Type)
	}

	// The shell command with semicolons should be a single token
	shellCmd := lexer.NextToken()
	if shellCmd.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER, got %s", shellCmd.Type)
	}

	// FIXED: No leading/trailing space in shell command
	expectedCmd := "cd src; make clean; make install"
	if diff := cmp.Diff(expectedCmd, shellCmd.Value); diff != "" {
		t.Errorf("Shell command mismatch (-want +got):\n%s", diff)
	}

	// }
	rbrace := lexer.NextToken()
	if rbrace.Type != RBRACE {
		t.Errorf("Expected RBRACE, got %s", rbrace.Type)
	}
}

func TestLineContinuation(t *testing.T) {
	input := "echo hello \\\nworld"
	lexer := New(input)

	// Simulate being in command mode
	lexer.setMode(CommandMode)

	// Token 1: "echo hello "
	shellText1 := lexer.NextToken()
	if shellText1.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER for first part, got %s", shellText1.Type)
	}
	if diff := cmp.Diff("echo hello ", shellText1.Value); diff != "" {
		t.Errorf("First part mismatch (-want +got):\n%s", diff)
	}

	// Token 2: The space from the line continuation
	spaceToken := lexer.NextToken()
	if spaceToken.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER for line continuation space, got %s", spaceToken.Type)
	}
	if diff := cmp.Diff(" ", spaceToken.Value); diff != "" {
		t.Errorf("Line continuation space mismatch (-want +got):\n%s", diff)
	}

	// Token 3: "world"
	shellText2 := lexer.NextToken()
	if shellText2.Type != IDENTIFIER {
		t.Errorf("Expected IDENTIFIER for second part, got %s", shellText2.Type)
	}
	if diff := cmp.Diff("world", shellText2.Value); diff != "" {
		t.Errorf("Second part mismatch (-want +got):\n%s", diff)
	}
}

func TestPosition(t *testing.T) {
	input := "var\ntest:"
	lexer := New(input)

	var1 := lexer.NextToken()
	if var1.Line != 1 || var1.Column != 1 {
		t.Errorf("Expected position 1:1, got %d:%d", var1.Line, var1.Column)
	}

	newline := lexer.NextToken()
	if newline.Type != NEWLINE {
		t.Errorf("Expected NEWLINE, got %s", newline.Type)
	}

	ident := lexer.NextToken()
	if ident.Line != 2 || ident.Column != 1 {
		t.Errorf("Expected position 2:1, got %d:%d", ident.Line, ident.Column)
	}
}

func TestComplexExample(t *testing.T) {
	input := `
var PORT = 8080

server: @timeout(30s) {
	echo "Starting server..."
	node app.js
}

watch tests: npm test

stop all: pkill -f "node|npm"
`

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
	var varCount, watchCount, stopCount, atCount int
	for _, token := range tokens {
		switch token.Type {
		case VAR:
			varCount++
		case WATCH:
			watchCount++
		case STOP:
			stopCount++
		case AT:
			atCount++
		}
	}

	if varCount != 1 {
		t.Errorf("Expected 1 VAR token, got %d", varCount)
	}
	if watchCount != 1 {
		t.Errorf("Expected 1 WATCH token, got %d", watchCount)
	}
	if stopCount != 1 {
		t.Errorf("Expected 1 STOP token, got %d", stopCount)
	}
	if atCount != 1 {
		t.Errorf("Expected 1 AT token, got %d", atCount)
	}
}

func TestNestedDecorators(t *testing.T) {
	input := "@timeout(30s) { @parallel { npm run api } }"
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	// FIXED: No extra whitespace tokens
	expected := []TokenType{
		AT, IDENTIFIER, // @timeout
		LPAREN, DURATION, RPAREN, // (30s)
		LBRACE,         // {
		AT, IDENTIFIER, // @parallel
		LBRACE,     // {
		IDENTIFIER, // npm run api
		RBRACE,     // }
		RBRACE,     // }
		EOF,
	}

	var tokenTypes []TokenType
	for _, token := range tokens {
		tokenTypes = append(tokenTypes, token.Type)
	}

	if diff := cmp.Diff(expected, tokenTypes); diff != "" {
		t.Errorf("Nested decorator token sequence mismatch (-want +got):\n%s", diff)

		// Debug: show actual tokens
		t.Logf("Actual tokens:")
		for i, token := range tokens {
			t.Logf("  %d: %s %q", i, token.Type, token.Value)
		}
	}

	// Check that decorator names have correct semantic types
	decoratorCount := 0
	for _, token := range tokens {
		if token.Type == IDENTIFIER && token.Semantic == SemDecorator {
			if token.Value == "timeout" || token.Value == "parallel" {
				decoratorCount++
			}
		}
	}

	if decoratorCount != 2 {
		t.Errorf("Expected 2 decorator identifiers (timeout, parallel), found %d", decoratorCount)
	}
}

func TestModeTransitions(t *testing.T) {
	// Debug: Simple command case
	t.Run("debug simple command", func(t *testing.T) {
		input := "build: echo hello"
		lexer := New(input)

		// Step 0: build
		token := lexer.NextToken()
		t.Logf("Step 0: token=%s, mode=%v (LanguageMode=0, CommandMode=1)", token.Type, lexer.mode)
		if token.Type != IDENTIFIER || lexer.mode != LanguageMode {
			t.Errorf("Step 0: expected IDENTIFIER in LanguageMode, got %s in mode %v", token.Type, lexer.mode)
		}

		// Step 1: :
		token = lexer.NextToken()
		t.Logf("Step 1: token=%s, mode=%v (LanguageMode=0, CommandMode=1)", token.Type, lexer.mode)
		if token.Type != COLON {
			t.Errorf("Step 1: expected COLON, got %s", token.Type)
		}
		// For simple commands, COLON should switch to CommandMode
		if lexer.mode != CommandMode {
			t.Errorf("Step 1: expected CommandMode (1) after COLON, got mode %v", lexer.mode)
		}

		// Step 2: echo hello
		token = lexer.NextToken()
		t.Logf("Step 2: token=%s, mode=%v, value=%q", token.Type, lexer.mode, token.Value)
		if token.Type != IDENTIFIER || lexer.mode != CommandMode {
			t.Errorf("Step 2: expected IDENTIFIER in CommandMode, got %s in mode %v", token.Type, lexer.mode)
		}

		// Step 3: EOF
		token = lexer.NextToken()
		t.Logf("Step 3: token=%s, mode=%v", token.Type, lexer.mode)
		if token.Type != EOF {
			t.Errorf("Step 3: expected EOF, got %s", token.Type)
		}
		// EOF should switch back to LanguageMode
		if lexer.mode != LanguageMode {
			t.Errorf("Step 3: expected LanguageMode (0) after EOF, got mode %v", lexer.mode)
		}
	})

	tests := []struct {
		name     string
		input    string
		expected []struct {
			token TokenType
			mode  LexerMode
		}
	}{
		{
			name:  "simple command transitions",
			input: "build: echo hello",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, CommandMode},       // : (switches to CommandMode because shouldSwitchToCommandMode() = true)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{EOF, LanguageMode},        // EOF (switches back to LanguageMode on newline/EOF)
			},
		},
		{
			name:  "explicit block transitions",
			input: "build: { echo hello }",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, LanguageMode},      // : (stays in LanguageMode because shouldSwitchToCommandMode() = false due to {)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{RBRACE, LanguageMode},     // } (switches back to LanguageMode)
				{EOF, LanguageMode},        // EOF (stays in LanguageMode)
			},
		},
		{
			name:  "decorator in command",
			input: "build: { @timeout(30s) { echo hello } }",
			expected: []struct {
				token TokenType
				mode  LexerMode
			}{
				{IDENTIFIER, LanguageMode}, // build (starts in LanguageMode)
				{COLON, LanguageMode},      // : (stays in LanguageMode because { follows)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{AT, LanguageMode},         // @ (switches to LanguageMode for decorator parsing)
				{IDENTIFIER, LanguageMode}, // timeout (stays in LanguageMode)
				{LPAREN, LanguageMode},     // ( (stays in LanguageMode)
				{DURATION, LanguageMode},   // 30s (stays in LanguageMode)
				{RPAREN, LanguageMode},     // ) (stays in LanguageMode)
				{LBRACE, CommandMode},      // { (switches to CommandMode)
				{IDENTIFIER, CommandMode},  // echo hello (stays in CommandMode)
				{RBRACE, CommandMode},      // } (stays in CommandMode - nested brace)
				{RBRACE, LanguageMode},     // } (switches to LanguageMode - outer brace)
				{EOF, LanguageMode},        // EOF (stays in LanguageMode)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)

			for i, expected := range test.expected {
				token := lexer.NextToken()
				if token.Type != expected.token {
					t.Errorf("Step %d: expected token %s, got %s", i, expected.token, token.Type)
				}

				// Check mode after token is processed
				if lexer.mode != expected.mode {
					t.Errorf("Step %d: expected mode %v, got %v (after token %s)",
						i, expected.mode, lexer.mode, token.Type)
				}
			}
		})
	}
}

func TestSyntaxSugarDetection(t *testing.T) {
	tests := []struct {
		name                      string
		input                     string
		shouldSwitchToCommandMode bool
	}{
		{
			name:                      "simple command gets sugar",
			input:                     "build: echo hello",
			shouldSwitchToCommandMode: true,
		},
		{
			name:                      "explicit block no sugar",
			input:                     "build: { echo hello }",
			shouldSwitchToCommandMode: false,
		},
		{
			name:                      "decorator no sugar",
			input:                     "build: @timeout(30s) { echo hello }",
			shouldSwitchToCommandMode: false,
		},
		{
			name:                      "empty command no sugar",
			input:                     "build:",
			shouldSwitchToCommandMode: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)

			// Skip to colon
			lexer.NextToken() // identifier
			lexer.NextToken() // colon

			// After colon, check if we switched to CommandMode
			nextToken := lexer.NextToken()

			if test.shouldSwitchToCommandMode {
				if nextToken.Type != IDENTIFIER || nextToken.Semantic != SemShellText {
					t.Errorf("Expected shell text token with SemShellText, got %s with semantic %v",
						nextToken.Type, nextToken.Semantic)
				}
			} else {
				// Should not switch to CommandMode immediately
				// Next token should be structural (LBRACE, AT) or EOF/NEWLINE
				if nextToken.Type == IDENTIFIER && nextToken.Semantic == SemShellText {
					t.Errorf("Unexpected switch to CommandMode, got shell text token: %s %q",
						nextToken.Type, nextToken.Value)
				}
			}
		})
	}
}

func TestDecoratorShapeDetection(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		shouldTokenize bool
		description    string
	}{
		// ✅ Valid decorator shapes - should tokenize @ separately
		{
			name:           "decorator with parentheses",
			input:          "cmd: echo @timeout(30s)",
			shouldTokenize: true,
			description:    "@timeout(30s) follows decorator shape",
		},
		{
			name:           "decorator with block",
			input:          "cmd: echo @parallel { npm test }",
			shouldTokenize: true,
			description:    "@parallel { } follows decorator shape",
		},
		{
			name:           "var usage with parentheses",
			input:          "cmd: echo @var(PORT)",
			shouldTokenize: true,
			description:    "@var(PORT) follows decorator shape",
		},
		{
			name:           "decorator with hyphen",
			input:          "cmd: echo @watch-files(pattern='*.js')",
			shouldTokenize: true,
			description:    "@watch-files() follows decorator shape",
		},
		{
			name:           "zero-arg decorator",
			input:          "cmd: echo @now",
			shouldTokenize: true,
			description:    "@now follows decorator shape (zero-arg)",
		},

		// ❌ Invalid shapes - should NOT tokenize @ separately
		{
			name:           "email address",
			input:          "cmd: echo admin@company.com",
			shouldTokenize: false,
			description:    "admin@company.com is email, not decorator shape",
		},
		{
			name:           "docker image digest",
			input:          "cmd: docker run nginx@sha256:abc123def456",
			shouldTokenize: false,
			description:    "nginx@sha256:... is docker image, not decorator shape",
		},
		{
			name:           "npm package version",
			input:          "cmd: npm install react@18.2.0",
			shouldTokenize: false,
			description:    "react@18.2.0 is package version, not decorator shape",
		},
		{
			name:           "url with auth",
			input:          "cmd: curl user@domain:pass",
			shouldTokenize: false,
			description:    "user@domain:pass is URL auth, not decorator shape",
		},
		{
			name:           "at symbol followed by number",
			input:          "cmd: echo version@123",
			shouldTokenize: false,
			description:    "version@123 doesn't follow decorator shape",
		},

		// Special cases that need careful handling
		{
			name:           "git reference with braces",
			input:          "cmd: git show HEAD@{2.days.ago}",
			shouldTokenize: false,
			description:    "HEAD@{...} is git reference, not decorator shape",
		},
		{
			name:           "shell array syntax",
			input:          "cmd: echo ${array[@]}",
			shouldTokenize: false,
			description:    "${array[@]} is shell syntax, not decorator shape",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			hasATToken := false
			for _, token := range tokens {
				if token.Type == AT {
					hasATToken = true
					break
				}
			}

			if hasATToken != test.shouldTokenize {
				t.Errorf("%s: expected @ tokenization to be %v, got %v",
					test.description, test.shouldTokenize, hasATToken)

				// Debug output
				t.Logf("All tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}
		})
	}
}

func TestDecoratorVsInlineVar(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedAT  int // Number of AT tokens expected
		description string
	}{
		{
			name:        "decorator @timeout",
			input:       "@timeout(30s)",
			expectedAT:  1,
			description: "Should tokenize @timeout as decorator",
		},
		{
			name:        "inline @var usage",
			input:       "cmd: echo @var(PORT)",
			expectedAT:  1,
			description: "Should tokenize @ in @var() as AT token, let parser handle semantics",
		},
		{
			name:        "mixed usage",
			input:       "@timeout(30s) { echo @var(PORT) }",
			expectedAT:  2,
			description: "Both @ symbols should be tokenized",
		},
		{
			name:        "email should not tokenize @",
			input:       "cmd: echo admin@company.com",
			expectedAT:  0,
			description: "Email addresses should not trigger @ tokenization",
		},
		{
			name:        "docker image should not tokenize @",
			input:       "cmd: docker run nginx@sha256:abc123",
			expectedAT:  0,
			description: "Docker image digests should not trigger @ tokenization",
		},
		{
			name:        "shell array should not tokenize @",
			input:       "cmd: echo ${array[@]}",
			expectedAT:  0,
			description: "Shell array syntax should not trigger @ tokenization",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			atCount := 0
			for _, token := range tokens {
				if token.Type == AT {
					atCount++
				}
			}

			if atCount != test.expectedAT {
				t.Errorf("%s: expected %d AT tokens, got %d",
					test.description, test.expectedAT, atCount)

				// Debug output
				t.Logf("All tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}
		})
	}
}

func TestVarInMiddleOfText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "var in quoted string",
			input: `cmd: echo "Port is @var(PORT) and host is @var(HOST)"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "cmd"},
				{COLON, ":"},
				{IDENTIFIER, `echo "Port is `}, // FIXED: No leading space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, ` and host is `},
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "HOST"},
				{RPAREN, ")"},
				{IDENTIFIER, `"`},
				{EOF, ""},
			},
		},
		{
			name:  "var mixed with email",
			input: `cmd: echo "API: @var(API_URL), contact: admin@company.com"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "cmd"},
				{COLON, ":"},
				{IDENTIFIER, `echo "API: `}, // FIXED: No leading space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "API_URL"},
				{RPAREN, ")"},
				{IDENTIFIER, `, contact: admin@company.com"`},
				{EOF, ""},
			},
		},
		{
			name:  "var in script parameters",
			input: `cmd: node app.js --port @var(PORT) --email admin@company.com`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "cmd"},
				{COLON, ":"},
				{IDENTIFIER, `node app.js --port `}, // FIXED: No leading space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, ` --email admin@company.com`},
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
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

func TestShellCommandsWithAtSymbols(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedTokens []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "email addresses in shell commands",
			input: "notify: echo 'Alert sent to admin@company.com'",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "notify"},
				{COLON, ":"},
				{IDENTIFIER, "echo 'Alert sent to admin@company.com'"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "docker image with digest",
			input: "deploy: docker run nginx@sha256:abc123def456",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{IDENTIFIER, "docker run nginx@sha256:abc123def456"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "git operations with @",
			input: "backup: git show HEAD@{2.days.ago}",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "backup"},
				{COLON, ":"},
				{IDENTIFIER, "git show HEAD@{2.days.ago}"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "ssh commands",
			input: "connect: ssh user@hostname.com",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "connect"},
				{COLON, ":"},
				{IDENTIFIER, "ssh user@hostname.com"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "npm package versions",
			input: "setup: npm install react@18.2.0 typescript@^4.9.0",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "setup"},
				{COLON, ":"},
				{IDENTIFIER, "npm install react@18.2.0 typescript@^4.9.0"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "shell arrays and parameters",
			input: "test: echo ${array[@]} and $@",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{IDENTIFIER, "echo ${array[@]} and $@"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "mixed legitimate @ and decorator",
			input: "deploy: @timeout(30s) { echo 'Deploying to admin@prod.com' }",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "timeout"},
				{LPAREN, "("},
				{DURATION, "30s"},
				{RPAREN, ")"},
				{LBRACE, "{"},
				{IDENTIFIER, "echo 'Deploying to admin@prod.com'"}, // FIXED: No leading/trailing space
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name:  "inline @var usage vs email",
			input: "server: node app.js --port @var(PORT) --admin admin@company.com",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "server"},
				{COLON, ":"},
				{IDENTIFIER, "node app.js --port "}, // FIXED: No leading space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, " --admin admin@company.com"},
				{EOF, ""},
			},
		},
		{
			name:  "multiple emails in one command",
			input: "notify: mail -s 'Alert' admin@company.com,ops@company.com < log.txt",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "notify"},
				{COLON, ":"},
				{IDENTIFIER, "mail -s 'Alert' admin@company.com,ops@company.com < log.txt"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "URL with authentication",
			input: "fetch: curl -u user@domain:pass https://api.service.com",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "fetch"},
				{COLON, ":"},
				{IDENTIFIER, "curl -u user@domain:pass https://api.service.com"}, // FIXED: No leading space
				{EOF, ""},
			},
		},
		{
			name:  "decorator followed by email in same block",
			input: "deploy: @parallel { echo 'Notify admin@company.com'; kubectl apply -f k8s/ }",
			expectedTokens: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "parallel"},
				{LBRACE, "{"},
				{IDENTIFIER, "echo 'Notify admin@company.com'; kubectl apply -f k8s/"}, // FIXED: No leading/trailing space
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			if len(tokens) != len(test.expectedTokens) {
				t.Errorf("Expected %d tokens, got %d", len(test.expectedTokens), len(tokens))

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
				return
			}

			for i, expected := range test.expectedTokens {
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

func TestWhitespacePreservationInShellCommands(t *testing.T) {
	// Test cases that specifically verify whitespace preservation around @var() decorators
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "space after command before @var()",
			input: "build: cp -r @var(SRC)/* @var(DEST)/",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{IDENTIFIER, "cp -r "}, // FIXED: No leading space after colon
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, "/* "}, // MUST preserve trailing space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "DEST"},
				{RPAREN, ")"},
				{IDENTIFIER, "/"},
				{EOF, ""},
			},
		},
		{
			name:  "space before and after @var()",
			input: "serve: go run main.go --port= @var(PORT) --host= @var(HOST)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "serve"},
				{COLON, ":"},
				{IDENTIFIER, "go run main.go --port= "}, // FIXED: No leading space after colon
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, " --host= "}, // MUST preserve leading AND trailing space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "HOST"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "multiple spaces around @var()",
			input: "build: echo  @var(MESSAGE)  world",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{IDENTIFIER, "echo  "}, // FIXED: No leading space, but preserve double space before @var
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "MESSAGE"},
				{RPAREN, ")"},
				{IDENTIFIER, "  world"}, // MUST preserve multiple leading spaces
				{EOF, ""},
			},
		},
		{
			name:  "tabs and spaces around @var()",
			input: "build: echo\t@var(VAR)\t\tworld",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{IDENTIFIER, "echo\t"}, // FIXED: No leading space after colon, preserve tab
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR"},
				{RPAREN, ")"},
				{IDENTIFIER, "\t\tworld"}, // MUST preserve tabs
				{EOF, ""},
			},
		},
		{
			name:  "shell operators with @var()",
			input: "process: cat @var(FILE) | grep pattern | sort",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "process"},
				{COLON, ":"},
				{IDENTIFIER, "cat "}, // FIXED: No leading space, MUST preserve space before @var()
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "FILE"},
				{RPAREN, ")"},
				{IDENTIFIER, " | grep pattern | sort"}, // MUST preserve leading space
				{EOF, ""},
			},
		},
		{
			name:  "complex shell command with multiple @var()",
			input: "deploy: cd @var(SRC) && npm run build: @var(ENV) && cp -r dist/* @var(DEST)/",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{IDENTIFIER, "cd "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, " && npm run build: "}, // MUST preserve spaces
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "ENV"},
				{RPAREN, ")"},
				{IDENTIFIER, " && cp -r dist/* "}, // MUST preserve spaces
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "DEST"},
				{RPAREN, ")"},
				{IDENTIFIER, "/"},
				{EOF, ""},
			},
		},
		{
			name:  "quoted strings with @var()",
			input: `greet: echo "Hello @var(NAME)!"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "greet"},
				{COLON, ":"},
				{IDENTIFIER, `echo "Hello `}, // FIXED: No leading space, MUST preserve space inside quote
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "NAME"},
				{RPAREN, ")"},
				{IDENTIFIER, `!"`}, // MUST preserve quote structure
				{EOF, ""},
			},
		},
		{
			name:  "shell command with paths and @var()",
			input: "backup: cp important.txt @var(HOME)/backup/",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "backup"},
				{COLON, ":"},
				{IDENTIFIER, "cp important.txt "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "HOME"},
				{RPAREN, ")"},
				{IDENTIFIER, "/backup/"},
				{EOF, ""},
			},
		},
		{
			name:  "SSH command with @var()",
			input: "connect: ssh -p @var(PORT) user@ @var(HOST)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "connect"},
				{COLON, ":"},
				{IDENTIFIER, "ssh -p "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, " user@ "}, // MUST preserve spaces (note: user@ is not a decorator)
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "HOST"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "function decorator with shell command",
			input: `build: echo "Files: ls @var(SRC) | wc -l"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{IDENTIFIER, `echo "Files: ls `}, // MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, ` | wc -l"`}, // MUST preserve quote
				{EOF, ""},
			},
		},
		{
			name:  "pkill command with @var()",
			input: "stop: pkill -f @var(PROCESS)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{STOP, "stop"}, // FIXED: STOP keyword, not IDENTIFIER
				{COLON, ":"},
				{IDENTIFIER, "pkill -f "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PROCESS"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "nested shell content with @var()",
			input: "test: ssh -p @var(PORT) user@ @var(HOST) 'echo hello'",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{IDENTIFIER, "ssh -p "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "PORT"},
				{RPAREN, ")"},
				{IDENTIFIER, " user@ "}, // MUST preserve spaces
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "HOST"},
				{RPAREN, ")"},
				{IDENTIFIER, " 'echo hello'"}, // MUST preserve leading space
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}

				t.Logf("Expected tokens:")
				for i, expected := range test.expected {
					t.Logf("  %d: %s %q", i, expected.tokenType, expected.value)
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
					t.Logf("  This is a critical whitespace preservation issue!")
				}
			}
		})
	}
}

func TestWhitespacePreservationInBlockCommands(t *testing.T) {
	// Test whitespace preservation in block commands
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "block command with @var() and spaces",
			input: "deploy: { cd @var(SRC); make clean; make install }",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "deploy"},
				{COLON, ":"},
				{LBRACE, "{"},
				{IDENTIFIER, "cd "}, // FIXED: No leading space after brace, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, "; make clean; make install"}, // FIXED: No trailing space, MUST preserve semicolon structure
				{RBRACE, "}"},
				{EOF, ""},
			},
		},
		{
			name:  "block command with multiple @var() and complex spacing",
			input: "build: { echo 'Building from @var(SRC) to @var(DEST)' && cp -r @var(SRC)/* @var(DEST)/ }",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{LBRACE, "{"},
				{IDENTIFIER, "echo 'Building from "}, // FIXED: No leading space, MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, " to "}, // MUST preserve spaces
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "DEST"},
				{RPAREN, ")"},
				{IDENTIFIER, "' && cp -r "}, // MUST preserve spaces
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "SRC"},
				{RPAREN, ")"},
				{IDENTIFIER, "/* "}, // MUST preserve space
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "DEST"},
				{RPAREN, ")"},
				{IDENTIFIER, "/"}, // FIXED: No trailing space
				{RBRACE, "}"},
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
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
					t.Logf("  This is a critical whitespace preservation issue in block commands!")
				}
			}
		})
	}
}

func TestEdgeCaseWhitespacePreservation(t *testing.T) {
	// Test edge cases for whitespace preservation
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "only spaces around @var()",
			input: "test:   @var(VAR)   ",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				// FIXED: After colon, initial spaces are skipped
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR"},
				{RPAREN, ")"},
				// Trailing spaces are preserved
				{IDENTIFIER, "   "},
				{EOF, ""},
			},
		},
		{
			name:  "mixed whitespace around @var()",
			input: "test: \t @var(VAR)\t \n",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				// FIXED: After colon, initial whitespace is skipped
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR"},
				{RPAREN, ")"},
				// Trailing whitespace before newline
				{IDENTIFIER, "\t "},
				{NEWLINE, "\n"},
				{EOF, ""},
			},
		},
		{
			name:  "no spaces around @var()",
			input: "test:word@var(VAR)word",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{IDENTIFIER, "word"}, // No spaces - should be preserved as-is
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR"},
				{RPAREN, ")"},
				{IDENTIFIER, "word"}, // No spaces - should be preserved as-is
				{EOF, ""},
			},
		},
		{
			name:  "empty text parts around @var()",
			input: "test:@var(VAR1)@var(VAR2)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{IDENTIFIER, "test"},
				{COLON, ":"},
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR1"},
				{RPAREN, ")"},
				{AT, "@"},
				{IDENTIFIER, "var"},
				{LPAREN, "("},
				{IDENTIFIER, "VAR2"},
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
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
					t.Logf("  This is a critical whitespace preservation edge case!")
				}
			}
		})
	}
}

func TestGetSemanticTokens(t *testing.T) {
	input := `var server: echo "hello"`

	tokens, err := GetSemanticTokens(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should NOT include EOF token
	for _, token := range tokens {
		if token.Type == EOF {
			t.Error("GetSemanticTokens should not include EOF token")
		}
	}
}

func TestTokenizeToSlice(t *testing.T) {
	input := `var server: @timeout(30s) { echo "test"; node app.js }`

	// Test TokenizeToSlice method
	lexer := New(input)
	tokens := lexer.TokenizeToSlice()

	if len(tokens) == 0 {
		t.Error("Expected tokens, got none")
	}

	// Should end with EOF
	if tokens[len(tokens)-1].Type != EOF {
		t.Errorf("Expected last token to be EOF, got %s", tokens[len(tokens)-1].Type)
	}

	// Verify we get reasonable token count (adjusted for shell text consolidation)
	if len(tokens) < 7 {
		t.Errorf("Expected at least 7 tokens for this input, got %d", len(tokens))
	}
}

func TestAtTokenDetection(t *testing.T) {
	tests := []struct {
		input       string
		shouldMatch bool
		atCount     int
	}{
		{"@timeout(30s)", true, 1},
		{"@var{}", true, 1},
		{"@test( args )", true, 1},
		{"@timeout @retry", true, 2},
		{"email@domain.com", true, 1}, // @ will be tokenized, parser decides semantics
		{"@ timeout", true, 1},        // @ will be tokenized, parser decides semantics
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			atTokens := 0
			for _, token := range tokens {
				if token.Type == AT {
					atTokens++
				}
			}

			if (atTokens > 0) != test.shouldMatch {
				t.Errorf("Expected AT token detection to be %v, got %v for input %q",
					test.shouldMatch, atTokens > 0, test.input)

				// Debug: show all tokens
				t.Logf("All tokens for %q:", test.input)
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}

			if atTokens != test.atCount {
				t.Errorf("Expected %d AT tokens, got %d for input %q",
					test.atCount, atTokens, test.input)
			}
		})
	}
}

func TestParseDecoratorArgs(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []DecoratorArg
		expectError bool
	}{
		{
			name:  "positional argument",
			input: "30s",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "named argument",
			input: "timeout=30s",
			expected: []DecoratorArg{
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "mixed positional and named",
			input: "30s, graceful=true",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "all named arguments",
			input: "timeout=30s, graceful=true",
			expected: []DecoratorArg{
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "reordered named arguments",
			input: "graceful=true, timeout=30s",
			expected: []DecoratorArg{
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
				{Name: "timeout", Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "complex values with quotes",
			input: `port="8080", host="localhost"`,
			expected: []DecoratorArg{
				{Name: "port", Value: "8080", Line: 1, Column: 1},
				{Name: "host", Value: "localhost", Line: 1, Column: 1},
			},
		},
		{
			name:  "hyphenated parameter names",
			input: "retry-count=3, max-delay=10s",
			expected: []DecoratorArg{
				{Name: "retry-count", Value: "3", Line: 1, Column: 1},
				{Name: "max-delay", Value: "10s", Line: 1, Column: 1},
			},
		},
		{
			name:  "underscore parameter names",
			input: "max_attempts=5, wait_time=1s",
			expected: []DecoratorArg{
				{Name: "max_attempts", Value: "5", Line: 1, Column: 1},
				{Name: "wait_time", Value: "1s", Line: 1, Column: 1},
			},
		},
		{
			name:        "positional after named (should error)",
			input:       "timeout=30s, graceful",
			expectError: true,
		},
		{
			name:        "invalid parameter name with number start",
			input:       "1timeout=30s",
			expectError: true,
		},
		{
			name:        "invalid parameter name with special chars",
			input:       "time@out=30s",
			expectError: true,
		},
		{
			name:     "empty arguments",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:  "single quoted values",
			input: "command='echo hello'",
			expected: []DecoratorArg{
				{Name: "command", Value: "echo hello", Line: 1, Column: 1},
			},
		},
		{
			name:  "backtick quoted values",
			input: "script=`date +%Y-%m-%d`",
			expected: []DecoratorArg{
				{Name: "script", Value: "date +%Y-%m-%d", Line: 1, Column: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := ParseDecoratorArgs(test.input, 1, 1)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(test.expected, args); diff != "" {
				t.Errorf("Decorator args mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestKotlinLikeParameterRules(t *testing.T) {
	// Test Kotlin-like parameter validation rules
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid positional only",
			input:       "30s, true, 5",
			shouldError: false,
		},
		{
			name:        "valid named only",
			input:       "timeout=30s, graceful=true, attempts=5",
			shouldError: false,
		},
		{
			name:        "valid mixed (positional first)",
			input:       "30s, graceful=true, attempts=5",
			shouldError: false,
		},
		{
			name:        "invalid mixed (positional after named)",
			input:       "timeout=30s, true",
			shouldError: true,
			errorMsg:    "positional argument follows named argument",
		},
		{
			name:        "invalid parameter name (starts with number)",
			input:       "1timeout=30s",
			shouldError: true,
			errorMsg:    "invalid parameter name",
		},
		{
			name:        "invalid parameter name (special chars)",
			input:       "time@out=30s",
			shouldError: true,
			errorMsg:    "invalid parameter name",
		},
		{
			name:        "valid parameter name with underscore",
			input:       "_timeout=30s",
			shouldError: false,
		},
		{
			name:        "valid parameter name with hyphen",
			input:       "retry-count=3",
			shouldError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseDecoratorArgs(test.input, 1, 1)

			if test.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDecoratorExamples(t *testing.T) {
	// Test examples from the spec
	examples := []struct {
		name     string
		input    string
		expected []DecoratorArg
	}{
		{
			name:  "timeout with duration",
			input: "30s",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
			},
		},
		{
			name:  "retry with named attempts",
			input: "attempts=3",
			expected: []DecoratorArg{
				{Name: "attempts", Value: "3", Line: 1, Column: 1},
			},
		},
		{
			name:  "timeout with graceful shutdown",
			input: "30s, graceful=true",
			expected: []DecoratorArg{
				{Value: "30s", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "var with environment variable",
			input: "NODE_ENV=development",
			expected: []DecoratorArg{
				{Name: "NODE_ENV", Value: "development", Line: 1, Column: 1},
			},
		},
		{
			name:  "parallel with worker count",
			input: "workers=4",
			expected: []DecoratorArg{
				{Name: "workers", Value: "4", Line: 1, Column: 1},
			},
		},
		{
			name:  "watch-files with pattern",
			input: `pattern="src/**/*"`,
			expected: []DecoratorArg{
				{Name: "pattern", Value: "src/**/*", Line: 1, Column: 1},
			},
		},
		{
			name:  "complex timeout with named duration",
			input: "duration=5m, graceful=true",
			expected: []DecoratorArg{
				{Name: "duration", Value: "5m", Line: 1, Column: 1},
				{Name: "graceful", Value: "true", Line: 1, Column: 1},
			},
		},
		{
			name:  "retry with delay duration",
			input: "attempts=3, delay=1.5s",
			expected: []DecoratorArg{
				{Name: "attempts", Value: "3", Line: 1, Column: 1},
				{Name: "delay", Value: "1.5s", Line: 1, Column: 1},
			},
		},
		{
			name:  "debounce with milliseconds",
			input: "debounce=500ms",
			expected: []DecoratorArg{
				{Name: "debounce", Value: "500ms", Line: 1, Column: 1},
			},
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			args, err := ParseDecoratorArgs(example.input, 1, 1)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(example.expected, args); diff != "" {
				t.Errorf("Example mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestVariableValueTokenization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:  "simple string variable",
			input: "var SRC = ./src",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "SRC"},
				{EQUALS, "="},
				{IDENTIFIER, "./src"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "URL variable",
			input: "var API_URL = https://api.example.com",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "API_URL"},
				{EQUALS, "="},
				{IDENTIFIER, "https://api.example.com"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "URL with port",
			input: "var DATABASE_URL = postgresql://user:pass@localhost:5432/dbname",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "DATABASE_URL"},
				{EQUALS, "="},
				{IDENTIFIER, "postgresql://user:pass@localhost:5432/dbname"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "file path variable",
			input: "var CONFIG_FILE = /etc/myapp/config.json",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "CONFIG_FILE"},
				{EQUALS, "="},
				{IDENTIFIER, "/etc/myapp/config.json"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "complex value with equals",
			input: "var QUERY = name=value&other=data",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "QUERY"},
				{EQUALS, "="},
				{IDENTIFIER, "name=value&other=data"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "simple identifier variable",
			input: "var HOST = localhost",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "HOST"},
				{EQUALS, "="},
				{IDENTIFIER, "localhost"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "number variable",
			input: "var PORT = 8080",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"}, // Should be NUMBER token
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
				{DURATION, "30s"}, // Should be DURATION token
				{EOF, ""},
			},
		},
		{
			name:  "quoted string variable",
			input: `var MESSAGE = "Hello, World!"`,
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "MESSAGE"},
				{EQUALS, "="},
				{STRING, "Hello, World!"}, // Should be STRING token
				{EOF, ""},
			},
		},
		{
			name:  "grouped variables with complex values",
			input: "var (\n  SRC = ./src\n  API_URL = https://api.example.com\n  PORT = 8080\n)",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{LPAREN, "("},
				{NEWLINE, "\n"},
				{IDENTIFIER, "SRC"},
				{EQUALS, "="},
				{IDENTIFIER, "./src"}, // Should be one token
				{NEWLINE, "\n"},
				{IDENTIFIER, "API_URL"},
				{EQUALS, "="},
				{IDENTIFIER, "https://api.example.com"}, // Should be one token
				{NEWLINE, "\n"},
				{IDENTIFIER, "PORT"},
				{EQUALS, "="},
				{NUMBER, "8080"}, // Should be NUMBER token
				{NEWLINE, "\n"},
				{RPAREN, ")"},
				{EOF, ""},
			},
		},
		{
			name:  "variable with URL containing special characters",
			input: "var API_URL = https://api.example.com/v1?key=abc123",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "API_URL"},
				{EQUALS, "="},
				{IDENTIFIER, "https://api.example.com/v1?key=abc123"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "variable with environment style name",
			input: "var NODE_ENV = production",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "NODE_ENV"},
				{EQUALS, "="},
				{IDENTIFIER, "production"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "variable with boolean-like value",
			input: "var DEBUG = true",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "DEBUG"},
				{EQUALS, "="},
				{IDENTIFIER, "true"}, // Should be one token
				{EOF, ""},
			},
		},
		{
			name:  "variable with hyphenated value",
			input: "var LONG_VALUE = this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "LONG_VALUE"},
				{EQUALS, "="},
				{IDENTIFIER, "this-is-a-very-long-value-that-spans-multiple-words-and-contains-hyphens"}, // Should be one token
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}

				t.Logf("Expected tokens:")
				for i, expected := range test.expected {
					t.Logf("  %d: %s %q", i, expected.tokenType, expected.value)
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

func TestVariableValueContextualTokenization(t *testing.T) {
	// Test that the lexer behaves differently in variable value context vs command context
	tests := []struct {
		name        string
		input       string
		description string
		expected    []struct {
			tokenType TokenType
			value     string
		}
	}{
		{
			name:        "URL in variable vs command",
			input:       "var URL = https://api.com\nbuild: echo https://api.com",
			description: "URLs should be tokenized as single identifiers in variable context, but may be different in command context",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "URL"},
				{EQUALS, "="},
				{IDENTIFIER, "https://api.com"}, // Variable context: should be one token
				{NEWLINE, "\n"},
				{IDENTIFIER, "build"},
				{COLON, ":"},
				{IDENTIFIER, "echo https://api.com"}, // Command context: part of shell text (no leading space)
				{EOF, ""},
			},
		},
		{
			name:        "colon handling in different contexts",
			input:       "var URL = https://api.com:8080\nserver: node app.js",
			description: "Colons in URLs should not be treated as command separators in variable context",
			expected: []struct {
				tokenType TokenType
				value     string
			}{
				{VAR, "var"},
				{IDENTIFIER, "URL"},
				{EQUALS, "="},
				{IDENTIFIER, "https://api.com:8080"}, // Variable context: colon is part of value
				{NEWLINE, "\n"},
				{IDENTIFIER, "server"},
				{COLON, ":"},                // Command context: colon is separator
				{IDENTIFIER, "node app.js"}, // FIXED: No leading space
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

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}

				t.Logf("Expected tokens:")
				for i, expected := range test.expected {
					t.Logf("  %d: %s %q", i, expected.tokenType, expected.value)
				}
				return
			}

			for i, expected := range test.expected {
				actual := tokens[i]
				if actual.Type != expected.tokenType {
					t.Errorf("Token %d: expected type %s, got %s (description: %s)", i, expected.tokenType, actual.Type, test.description)
				}
				if actual.Value != expected.value {
					t.Errorf("Token %d: expected value %q, got %q (description: %s)", i, expected.value, actual.Value, test.description)
				}
			}
		})
	}
}

func TestCurrentBehaviorAnalysis(t *testing.T) {
	// This test is designed to show exactly what the current lexer is doing
	// We'll run it and see what tokens we get, then understand the problem

	problemCases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple URL",
			input: "var API_URL = https://api.example.com",
		},
		{
			name:  "file path",
			input: "var SRC = ./src",
		},
		{
			name:  "URL with port",
			input: "var DB_URL = postgresql://user:pass@localhost:5432/dbname",
		},
		{
			name:  "path with special chars",
			input: "var CONFIG = /etc/myapp/config.json",
		},
		{
			name:  "value with equals",
			input: "var QUERY = name=value&other=data",
		},
	}

	for _, test := range problemCases {
		t.Run(test.name, func(t *testing.T) {
			lexer := New(test.input)
			tokens := lexer.TokenizeToSlice()

			t.Logf("Input: %s", test.input)
			t.Logf("Tokens produced:")
			for i, token := range tokens {
				t.Logf("  %d: %s %q (line %d, col %d)", i, token.Type, token.Value, token.Line, token.Column)
			}

			// Count how many tokens we get after the equals sign
			equalsIndex := -1
			for i, token := range tokens {
				if token.Type == EQUALS {
					equalsIndex = i
					break
				}
			}

			if equalsIndex >= 0 {
				valueTokens := []Token{}
				for i := equalsIndex + 1; i < len(tokens) && tokens[i].Type != EOF; i++ {
					valueTokens = append(valueTokens, tokens[i])
				}
				t.Logf("Value tokens (after =): %d tokens", len(valueTokens))
				for i, token := range valueTokens {
					t.Logf("  Value[%d]: %s %q", i, token.Type, token.Value)
				}
			}
		})
	}
}

func TestVariableValueTermination(t *testing.T) {
	// Test what terminates variable values in different contexts
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "variable terminated by newline",
			input:    "var URL = https://api.com\nbuild: echo hello",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, IDENTIFIER, NEWLINE, IDENTIFIER, COLON, IDENTIFIER, EOF},
		},
		{
			name:     "variable terminated by EOF",
			input:    "var URL = https://api.com",
			expected: []TokenType{VAR, IDENTIFIER, EQUALS, IDENTIFIER, EOF},
		},
		{
			name:     "variable in group terminated by newline",
			input:    "var (\n  URL = https://api.com\n  PORT = 8080\n)",
			expected: []TokenType{VAR, LPAREN, NEWLINE, IDENTIFIER, EQUALS, IDENTIFIER, NEWLINE, IDENTIFIER, EQUALS, NUMBER, NEWLINE, RPAREN, EOF},
		},
		{
			name:     "variable in group terminated by closing paren",
			input:    "var (URL = https://api.com)",
			expected: []TokenType{VAR, LPAREN, IDENTIFIER, EQUALS, IDENTIFIER, RPAREN, EOF},
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
				t.Errorf("Token sequence mismatch (-expected +actual):\n%s", diff)

				// Debug: show all actual tokens
				t.Logf("Actual tokens:")
				for i, token := range tokens {
					t.Logf("  %d: %s %q", i, token.Type, token.Value)
				}
			}
		})
	}
}

func TestZeroCopyBehavior(t *testing.T) {
	input := `var test: echo hello world`
	lexer := New(input)

	tokens := lexer.TokenizeToSlice()

	// Verify that simple tokens reference the original string
	for _, token := range tokens {
		if token.Type == IDENTIFIER {
			if token.Value != "" && !contains(input, token.Value) {
				t.Errorf("Token value %q should be zero-copy reference to original input", token.Value)
			}
		}
	}
}

func TestOptimizedPerformance(t *testing.T) {
	// Test that the optimized version can handle reasonable loads
	input := generateTestInput(1000) // 1000 lines

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
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || findInString(s, substr) >= 0)
}

func findInString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func generateTestInput(lines int) string {
	var result strings.Builder
	for i := 0; i < lines; i++ {
		result.WriteString(fmt.Sprintf("var cmd%d: echo hello %d\n", i, i))
	}
	return result.String()
}
